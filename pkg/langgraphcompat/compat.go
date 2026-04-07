package langgraphcompat

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/easyspace-ai/minote/pkg/agent"
	"github.com/easyspace-ai/minote/pkg/checkpoint"
	"github.com/easyspace-ai/minote/pkg/clarification"
	"github.com/easyspace-ai/minote/pkg/llm"
	"github.com/easyspace-ai/minote/pkg/memory"
	"github.com/easyspace-ai/minote/pkg/models"
	"github.com/easyspace-ai/minote/pkg/persistence"
	"github.com/easyspace-ai/minote/pkg/sandbox"
	"github.com/easyspace-ai/minote/pkg/subagent"
	"github.com/easyspace-ai/minote/pkg/tools"
	"github.com/easyspace-ai/minote/pkg/tools/builtin"
	"github.com/easyspace-ai/minote/pkg/tracing"
)

// LangGraph API-compatible server wrapper for deerflow-go
// Implements the endpoints expected by @langchain/langgraph-sdk

type Server struct {
	httpServer        *http.Server
	logger            *log.Logger
	llmProvider       llm.LLMProvider
	tools             *tools.Registry
	sandbox           *sandbox.Sandbox
	sandboxMu         sync.Mutex
	sandboxRoot       string
	subagents         *subagent.Pool
	clarify           *clarification.Manager
	clarifyAPI        *clarification.API
	defaultModel      string
	maxTurns          int
	store             *checkpoint.PostgresStore
	persistence       *persistence.PostgresStore
	startedAt         time.Time
	sessions          map[string]*Session
	sessionsMu        sync.RWMutex
	runs              map[string]*Run
	runsMu            sync.RWMutex
	runStreams        map[string]map[uint64]chan StreamEvent
	runStreamSeq      uint64
	dataRoot          string
	compatFSManaged   bool
	uiStateMu         sync.RWMutex
	skills            map[string]GatewaySkill
	mcpConfig         gatewayMCPConfig
	channelConfig     gatewayChannelsConfig
	agents            map[string]GatewayAgent
	userProfile       string
	memory            gatewayMemoryResponse
	memoryStore       memory.Storage
	memoryStoreCloser interface{ Close() }
	memorySvc         *memory.Service
	memoryThread      string
	mcpMu             sync.Mutex
	mcpClients        map[string]gatewayMCPClient
	mcpToolNames      map[string]struct{}
	mcpDeferredTools  []models.Tool
	mcpConnector      gatewayMCPConnector
	channelMu         sync.Mutex
	channelService    *gatewayChannelService
	summarizer        *llm.ConversationSummarizer
	tracer            tracing.Tracer
	backgroundTasks   sync.WaitGroup
	frontend          http.Handler
	// Optional: inject library document text (notex) when context carries studio_document_ids.
	studioDocInject func(ctx context.Context, userID, conversationID int64, docIDs []int64) string
}

type HealthStatus struct {
	Status     string            `json:"status"`
	Components map[string]string `json:"components"`
	Uptime     time.Duration     `json:"uptime"`
}

type Session struct {
	CheckpointID string
	ThreadID     string
	Messages     []models.Message
	Todos        []Todo
	Values       map[string]any
	Metadata     map[string]any
	Configurable map[string]any
	Status       string
	PresentFiles *tools.PresentFileRegistry
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Todo struct {
	Content string `json:"content,omitempty"`
	Status  string `json:"status,omitempty"`
}

type ThreadState struct {
	CheckpointID string         `json:"checkpoint_id,omitempty"`
	Values       map[string]any `json:"values"`
	Next         []string       `json:"next"`
	Tasks        []any          `json:"tasks"`
	Metadata     map[string]any `json:"metadata"`
	Config       map[string]any `json:"config,omitempty"`
	CreatedAt    string         `json:"created_at,omitempty"`
}

type Run struct {
	RunID        string
	ThreadID     string
	AssistantID  string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Events       []StreamEvent
	Error        string
	cancel       context.CancelFunc
	abandonTimer *time.Timer
}

const generalPurposeSubagentPrompt = "You are a general-purpose subagent working on a delegated task. Complete it autonomously and return a clear, actionable result.\n\n" +
	"<guidelines>\n" +
	"- Focus on completing the delegated task efficiently\n" +
	"- Use available tools as needed to accomplish the goal\n" +
	"- Think step by step but act decisively\n" +
	"- If you hit issues, explain them clearly in your response\n" +
	"- Do not ask for clarification; work with the provided information\n" +
	"</guidelines>"

const bashSubagentPrompt = "You are a bash command execution specialist. Execute the requested commands carefully and report results clearly.\n\n" +
	"<guidelines>\n" +
	"- Execute commands one at a time when they depend on each other\n" +
	"- Use parallel execution only when commands are independent\n" +
	"- Report both success/failure and relevant output\n" +
	"- Use absolute paths for file operations\n" +
	"- Be cautious with destructive operations\n" +
	"</guidelines>"

// LangGraph API types
type RunCreateRequest struct {
	AssistantID      string         `json:"assistant_id"`
	ThreadID         string         `json:"thread_id,omitempty"`
	Input            map[string]any `json:"input,omitempty"`
	Config           map[string]any `json:"config,omitempty"`
	Context          map[string]any `json:"context,omitempty"`
	AutoAcceptedPlan *bool          `json:"auto_accepted_plan,omitempty"`
	Feedback         string         `json:"feedback,omitempty"`
}

// Message represents a LangGraph-compatible message
type Message struct {
	Type             string         `json:"type"`
	ID               string         `json:"id"`
	Role             string         `json:"role,omitempty"`
	Content          any            `json:"content,omitempty"`
	Name             string         `json:"name,omitempty"`
	Data             map[string]any `json:"data,omitempty"`
	AdditionalKwargs map[string]any `json:"additional_kwargs,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
	ToolCalls        []ToolCall     `json:"tool_calls,omitempty"`
	UsageMetadata    map[string]int `json:"usage_metadata,omitempty"`
}

// ToolCall represents a LangGraph-compatible tool call
type ToolCall struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Args     map[string]any `json:"args"`
	RootID   string         `json:"root_id,omitempty"`
	ParentID string         `json:"parent_id,omitempty"`
}

type StreamEvent struct {
	ID       string
	Event    string
	Data     any
	RunID    string
	ThreadID string
}

const defaultGatewaySubagentMaxConcurrent = 3

type ServerOption func(*Server) error

func WithFrontendFS(frontend fs.FS) ServerOption {
	return func(s *Server) error {
		if s == nil {
			return errors.New("server is nil")
		}
		if frontend == nil {
			return nil
		}

		s.frontend = spaFileServer(frontend)
		return nil
	}
}

// spaFileServer serves a static site and falls back to the root document so
// client-side routers (e.g. React Router) work for deep links. It uses URL path
// "/" for the index (not "/index.html") because net/http FileServer redirects
// "/index.html" in recent Go versions.
func spaFileServer(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil {
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fileServer.ServeHTTP(w, r)
			return
		}

		cleaned := path.Clean(r.URL.Path)
		if cleaned == "/" || cleaned == "." {
			fileServer.ServeHTTP(w, fileServerRequest(r, "/"))
			return
		}

		rel := strings.TrimPrefix(cleaned, "/")
		if !fs.ValidPath(rel) {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		if _, err := fs.Stat(fsys, rel); err != nil {
			fileServer.ServeHTTP(w, fileServerRequest(r, "/"))
			return
		}
		fileServer.ServeHTTP(w, fileServerRequest(r, cleaned))
	})
}

func fileServerRequest(base *http.Request, urlPath string) *http.Request {
	r2 := base.Clone(base.Context())
	r2.RequestURI = ""
	u := cloneURL(base.URL)
	u.Path = urlPath
	u.RawPath = ""
	r2.URL = u
	return r2
}

func NewServer(addr string, dbURL string, defaultModel string, opts ...ServerOption) (*Server, error) {
	logger := log.Default()
	ctx := context.Background()

	// Load config.yaml if present
	gatewayConfig := loadGatewayConfig()

	// Apply log level from config
	applyLogLevel(gatewayConfig, logger)

	// Apply PDF converter mode and optional MarkItDown HTTP service from config
	if uploads := configUploads(gatewayConfig); uploads != nil {
		if uploads.PDFConverter != "" {
			SetPDFConverterMode(uploads.PDFConverter)
		}
		if strings.TrimSpace(uploads.MarkitdownURL) != "" {
			SetMarkitdownHTTPURL(uploads.MarkitdownURL)
		}
	}
	if envURL := strings.TrimSpace(os.Getenv("MARKITDOWN_URL")); envURL != "" {
		SetMarkitdownHTTPURL(envURL)
	}

	// Apply sandbox output truncation config
	if gatewayConfig != nil && gatewayConfig.Sandbox != nil {
		if gatewayConfig.Sandbox.BashOutputMaxChars > 0 {
			builtin.BashOutputMaxChars = gatewayConfig.Sandbox.BashOutputMaxChars
		}
	}

	// Create LLM provider (reads DEFAULT_LLM_PROVIDER env, defaults to "openai")
	providerName := strings.TrimSpace(os.Getenv("DEFAULT_LLM_PROVIDER"))
	provider := llm.NewProvider(providerName)
	logger.Printf("llm: %s", llm.DescribeProviderEnv(providerName, ""))

	// Wrap with token usage tracking if enabled
	tokenUsageEnabled := configTokenUsageEnabled(gatewayConfig)
	tokenTracker := llm.NewTokenUsageTracker(tokenUsageEnabled)
	provider = llm.NewTrackingProvider(provider, tokenTracker)

	// Create tool registry with built-in tools
	registry := tools.NewRegistry()
	clarifyManager := clarification.NewManager(32)
	registry.Register(builtin.BashTool())
	for _, tool := range builtin.FileTools() {
		registry.Register(tool)
	}
	for _, tool := range builtin.WebTools() {
		registry.Register(tool)
	}
	registry.Register(builtin.ViewImageTool())
	registry.Register(clarification.AskClarificationTool(clarifyManager))
	if acpAgents := loadACPAgentConfigs(); len(acpAgents) > 0 {
		registry.Register(tools.InvokeACPAgentTool(acpAgents))
	}
	subagentAppCfg := loadSubagentsAppConfig()
	subagentExecutor := agent.NewSubagentExecutor(provider, registry, nil)
	subagentPool := subagent.NewPool(subagentExecutor, subagent.PoolConfig{
		MaxConcurrent: defaultGatewaySubagentMaxConcurrent,
		Timeout:       subagentAppCfg.timeoutFor(subagent.SubagentGeneralPurpose),
		Defaults:      gatewayDefaultSubagentConfigs(subagentAppCfg),
	})
	registry.Register(tools.TaskTool(subagentPool))

	// Create checkpoint store
	var store *checkpoint.PostgresStore
	if dbURL != "" {
		var err error
		store, err = checkpoint.NewPostgresStore(ctx, dbURL)
		if err != nil {
			logger.Printf("Warning: failed to create Postgres store: %v", err)
		}
	}

	// Create persistence store for agents/skills/channels
	var persistenceStore *persistence.PostgresStore
	if dbURL != "" && store != nil {
		// Reuse the existing connection pool from checkpoint store if possible, but for now just create a new pool
		var err error
		persistenceStore, err = persistence.NewPostgresStore(ctx, dbURL)
		if err != nil {
			logger.Printf("Warning: failed to create persistence store: %v", err)
		}
	}

	dataRoot := strings.TrimSpace(os.Getenv("DEERFLOW_DATA_ROOT"))
	if dataRoot == "" {
		dataRoot = filepath.Join(os.TempDir(), "deerflow-go-data")
	}
	dataRootAbs, err := filepath.Abs(dataRoot)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dataRootAbs, 0o755); err != nil {
		return nil, err
	}

	var memoryStore memory.Storage
	var memoryStoreCloser interface{ Close() }
	var memorySvc *memory.Service

	memoryCfg := configMemory(gatewayConfig)
	memoryEnabled := true
	if memoryCfg != nil && !memoryCfg.Enabled {
		memoryEnabled = false
	}

	if memoryEnabled && dbURL != "" {
		postgresMemoryStore, err := memory.NewPostgresStore(ctx, dbURL)
		if err != nil {
			logger.Printf("Warning: failed to create memory store: %v", err)
		} else {
			memoryStore = postgresMemoryStore
			memoryStoreCloser = postgresMemoryStore
			timeout := resolveMemoryTimeout(memoryCfg)
			memoryModel := resolveMemoryModel(memoryCfg, defaultModel)
			memorySvc = memory.NewService(memoryStore, memory.NewLLMClient(provider, memoryModel)).WithUpdateTimeout(timeout)
		}
	}
	if memoryEnabled && memoryStore == nil {
		storagePath := filepath.Join(dataRootAbs, "memory")
		if memoryCfg != nil && memoryCfg.StoragePath != "" {
			storagePath = memoryCfg.StoragePath
		}
		fileMemoryStore, err := memory.NewFileStore(storagePath)
		if err != nil {
			logger.Printf("Warning: failed to create file-backed memory store: %v", err)
		} else {
			if err := fileMemoryStore.AutoMigrate(ctx); err != nil {
				logger.Printf("Warning: failed to initialize file-backed memory store: %v", err)
			} else {
				memoryStore = fileMemoryStore
				timeout := resolveMemoryTimeout(memoryCfg)
				memoryModel := resolveMemoryModel(memoryCfg, defaultModel)
				memorySvc = memory.NewService(memoryStore, memory.NewLLMClient(provider, memoryModel)).WithUpdateTimeout(timeout)
			}
		}
	}

	s := &Server{
		logger:            logger,
		llmProvider:       provider,
		tools:             registry,
		sandboxRoot:       filepath.Join(os.TempDir(), "deerflow-langgraph-sandbox"),
		subagents:         subagentPool,
		clarify:           clarifyManager,
		clarifyAPI:        clarification.NewAPI(clarifyManager),
		defaultModel:      defaultModel,
		maxTurns:          8,
		store:             store,
		persistence:       persistenceStore,
		startedAt:         time.Now().UTC(),
		sessions:          make(map[string]*Session),
		runs:              make(map[string]*Run),
		runStreams:        make(map[string]map[uint64]chan StreamEvent),
		dataRoot:          dataRootAbs,
		skills:            defaultGatewaySkills(),
		mcpConfig:         defaultGatewayMCPConfig(),
		agents:            map[string]GatewayAgent{},
		memory:            defaultGatewayMemory(),
		memoryStore:       memoryStore,
		memoryStoreCloser: memoryStoreCloser,
		memorySvc:         memorySvc,
		mcpClients:        map[string]gatewayMCPClient{},
		mcpToolNames:      map[string]struct{}{},
		mcpConnector:      defaultGatewayMCPConnector,
	}
	s.channelService = newGatewayChannelService(s)
	s.channelService.start()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	subagentExecutor.SetSandboxProvider(s.getOrCreateSandbox)
	registry.Register(s.setupAgentTool())
	registry.Register(s.todoTool())

	// Initialize conversation summarizer
	if sumCfg := configSummarization(gatewayConfig); sumCfg != nil && sumCfg.Enabled {
		ApplySummarizationConfig(sumCfg)
		s.summarizer = llm.NewConversationSummarizer(provider, buildSummarizationConfig(sumCfg, defaultModel))
	}

	// Initialize tracer
	if gatewayConfig != nil && gatewayConfig.Tracing != nil && gatewayConfig.Tracing.Enabled {
		s.tracer = tracing.NewTracer(tracing.Config{
			Enabled:  true,
			Provider: gatewayConfig.Tracing.Provider,
			Endpoint: gatewayConfig.Tracing.Endpoint,
			APIKey:   gatewayConfig.Tracing.APIKey,
		})
	}

	if err := s.loadGatewayState(); err != nil {
		logger.Printf("Warning: failed to load gateway state: %v", err)
	}
	if err := s.loadGatewayCompatFiles(); err != nil {
		logger.Printf("Warning: failed to load gateway compatibility files: %v", err)
	}
	s.loadMCPServersFromConfig(gatewayConfig)
	s.applyGatewayMCPConfig(ctx, s.mcpConfig)
	if err := s.loadPersistedSessions(); err != nil {
		logger.Printf("Warning: failed to load persisted sessions: %v", err)
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	var handler http.Handler = wrapAuth(wrapTrailingSlashCompat(wrapCORSCompat(mux)), defaultAuthConfig())
	if s.tracer != nil {
		handler = tracing.Middleware(s.tracer)(handler)
	}

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// 启动定期清理过期数据的goroutine
	go s.startCleanupTicker()

	return s, nil
}

func wrapTrailingSlashCompat(next http.Handler) http.Handler {
	if next == nil {
		return http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil {
			next.ServeHTTP(w, r)
			return
		}
		path := r.URL.Path
		if path == "" || path == "/" || !strings.HasSuffix(path, "/") {
			next.ServeHTTP(w, r)
			return
		}

		trimmedPath := strings.TrimRight(path, "/")
		if trimmedPath == "" {
			trimmedPath = "/"
		}

		cloned := r.Clone(r.Context())
		cloned.URL = cloneURL(r.URL)
		cloned.URL.Path = trimmedPath
		if rawPath := cloned.URL.RawPath; rawPath != "" {
			cloned.URL.RawPath = strings.TrimRight(rawPath, "/")
			if cloned.URL.RawPath == "" {
				cloned.URL.RawPath = "/"
			}
		}
		next.ServeHTTP(w, cloned)
	})
}

func wrapCORSCompat(next http.Handler) http.Handler {
	if next == nil {
		return http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w, r)
		if r != nil && r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	header := w.Header()
	header.Add("Vary", "Origin")
	header.Add("Vary", "Access-Control-Request-Method")
	header.Add("Vary", "Access-Control-Request-Headers")

	origin := "*"
	if r != nil {
		if candidate := strings.TrimSpace(r.Header.Get("Origin")); candidate != "" {
			origin = candidate
		}
	}
	header.Set("Access-Control-Allow-Origin", origin)
	header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
	header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Last-Event-ID, X-Requested-With")
	header.Set("Access-Control-Expose-Headers", "Content-Disposition, Content-Length, Content-Type, Content-Location, Location")
	header.Set("Access-Control-Max-Age", "600")
}

func cloneURL(src *url.URL) *url.URL {
	if src == nil {
		return &url.URL{}
	}
	dst := *src
	return &dst
}

func (s *Server) newAgent(cfg agent.AgentConfig) *agent.Agent {
	sandboxRef := cfg.Sandbox
	if sandboxRef == nil {
		if sb, err := s.getOrCreateSandbox(); err == nil {
			sandboxRef = sb
		} else if s.logger != nil {
			s.logger.Printf("Warning: failed to initialize sandbox: %v", err)
		}
	}
	registry := cfg.Tools
	if registry == nil {
		registry = s.tools
	}
	return agent.New(agent.AgentConfig{
		LLMProvider:     s.llmProvider,
		Tools:           registry,
		DeferredTools:   s.currentDeferredMCPTools(),
		PresentFiles:    cfg.PresentFiles,
		MaxTurns:        s.maxTurns,
		AgentType:       cfg.AgentType,
		Model:           cfg.Model,
		ReasoningEffort: cfg.ReasoningEffort,
		SystemPrompt:    cfg.SystemPrompt,
		Temperature:     cfg.Temperature,
		MaxTokens:       cfg.MaxTokens,
		Sandbox:         sandboxRef,
		RequestTimeout:  cfg.RequestTimeout,
	})
}

func (s *Server) getOrCreateSandbox() (*sandbox.Sandbox, error) {
	if s == nil {
		return nil, errors.New("server is nil")
	}

	s.sandboxMu.Lock()
	defer s.sandboxMu.Unlock()

	if s.sandbox != nil {
		return s.sandbox, nil
	}

	root := strings.TrimSpace(s.sandboxRoot)
	if root == "" {
		root = filepath.Join(os.TempDir(), "deerflow-langgraph-sandbox")
		s.sandboxRoot = root
	}

	sb, err := sandbox.New("langgraph", root)
	if err != nil {
		return nil, err
	}
	s.sandbox = sb
	return sb, nil
}

func (s *Server) currentDeferredMCPTools() []models.Tool {
	if s == nil {
		return nil
	}
	s.mcpMu.Lock()
	defer s.mcpMu.Unlock()
	if len(s.mcpDeferredTools) == 0 {
		return nil
	}
	out := make([]models.Tool, 0, len(s.mcpDeferredTools))
	for _, tool := range s.mcpDeferredTools {
		out = append(out, tool)
	}
	return out
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	s.registerLangGraphRoutes(mux, "")
	s.registerLangGraphRoutes(mux, "/api/langgraph")
	s.registerGatewayRoutes(mux)
	s.registerDocsRoutes(mux)

	// Health check
	mux.HandleFunc("GET /health", s.handleHealth)
	if s.frontend != nil {
		mux.Handle("/", s.frontend)
	}
}

func (s *Server) registerLangGraphRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("POST "+prefix+"/runs/stream", s.handleRunsStream)
	mux.HandleFunc("POST "+prefix+"/runs", s.handleRunsCreate)
	mux.HandleFunc("GET "+prefix+"/runs/{run_id}", s.handleRunGet)
	mux.HandleFunc("GET "+prefix+"/runs/{run_id}/stream", s.handleRunStream)
	mux.HandleFunc("POST "+prefix+"/runs/{run_id}/cancel", s.handleRunCancel)

	mux.HandleFunc("GET "+prefix+"/threads", s.handleThreadsList)
	mux.HandleFunc("GET "+prefix+"/threads/{thread_id}", s.handleThreadGet)
	mux.HandleFunc("POST "+prefix+"/threads", s.handleThreadCreate)
	mux.HandleFunc("PUT "+prefix+"/threads/{thread_id}", s.handleThreadUpdate)
	mux.HandleFunc("PATCH "+prefix+"/threads/{thread_id}", s.handleThreadUpdate)
	mux.HandleFunc("DELETE "+prefix+"/threads/{thread_id}", s.handleThreadDelete)
	mux.HandleFunc("POST "+prefix+"/threads/search", s.handleThreadSearch)
	mux.HandleFunc("GET "+prefix+"/threads/{thread_id}/files", s.handleThreadFiles)
	mux.HandleFunc("GET "+prefix+"/threads/{thread_id}/state", s.handleThreadStateGet)
	mux.HandleFunc("PUT "+prefix+"/threads/{thread_id}/state", s.handleThreadStatePost)
	mux.HandleFunc("POST "+prefix+"/threads/{thread_id}/state", s.handleThreadStatePost)
	mux.HandleFunc("PATCH "+prefix+"/threads/{thread_id}/state", s.handleThreadStatePatch)
	mux.HandleFunc("GET "+prefix+"/threads/{thread_id}/history", s.handleThreadHistory)
	mux.HandleFunc("POST "+prefix+"/threads/{thread_id}/history", s.handleThreadHistory)
	mux.HandleFunc("GET "+prefix+"/threads/{thread_id}/runs", s.handleThreadRunsList)
	mux.HandleFunc("GET "+prefix+"/threads/{thread_id}/runs/{run_id}", s.handleThreadRunGet)
	mux.HandleFunc("POST "+prefix+"/threads/{thread_id}/runs", s.handleThreadRunsCreate)
	mux.HandleFunc("POST "+prefix+"/threads/{thread_id}/runs/stream", s.handleThreadRunsStream)
	mux.HandleFunc("GET "+prefix+"/threads/{thread_id}/runs/{run_id}/stream", s.handleThreadRunStream)
	mux.HandleFunc("POST "+prefix+"/threads/{thread_id}/runs/{run_id}/cancel", s.handleThreadRunCancel)
	mux.HandleFunc("GET "+prefix+"/threads/{thread_id}/stream", s.handleThreadJoinStream)
	mux.HandleFunc("GET "+prefix+"/threads/{thread_id}/clarifications", s.handleThreadClarificationList)
	mux.HandleFunc("POST "+prefix+"/threads/{thread_id}/clarifications", s.handleThreadClarificationCreate)
	mux.HandleFunc("GET "+prefix+"/threads/{thread_id}/clarifications/{id}", s.handleThreadClarificationGet)
	mux.HandleFunc("POST "+prefix+"/threads/{thread_id}/clarifications/{id}/resolve", s.handleThreadClarificationResolve)
}

func (s *Server) Start() error {
	s.logger.Printf("LangGraph-compatible server starting on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Handler returns the underlying HTTP handler, useful for embedding into a
// combined mux without starting a separate listener.
func (s *Server) Handler() http.Handler {
	if s == nil || s.httpServer == nil {
		return http.NotFoundHandler()
	}
	return s.httpServer.Handler
}

func (s *Server) Shutdown(ctx context.Context) error {
	var shutdownErr error
	if s.httpServer != nil {
		shutdownErr = s.httpServer.Shutdown(ctx)
	}
	s.waitForBackgroundTasks()
	if s.store != nil {
		s.store.Close()
	}
	if s.channelService != nil {
		s.channelService.stop()
	}
	if s.memoryStoreCloser != nil {
		s.memoryStoreCloser.Close()
	}
	s.closeGatewayMCPClients()
	if s.tracer != nil {
		s.tracer.Shutdown(ctx)
	}
	if s.sandbox != nil {
		if err := s.sandbox.Close(); err != nil && shutdownErr == nil {
			shutdownErr = err
		}
	}
	return shutdownErr
}

func (s *Server) waitForBackgroundTasks() {
	if s == nil {
		return
	}
	s.backgroundTasks.Wait()
}

func (s *Server) healthStatus(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Status:     "ok",
		Components: map[string]string{},
		Uptime:     time.Since(s.startedAt).Round(time.Second),
	}

	componentCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	status.Components["llm"] = s.checkLLMProvider(componentCtx)
	status.Components["database"] = s.checkDatabase(componentCtx)
	status.Components["sandbox"] = s.checkSandbox(componentCtx)

	overall := "ok"
	for name, componentStatus := range status.Components {
		switch componentStatus {
		case "down":
			if name == "llm" {
				overall = "down"
			} else if overall == "ok" {
				overall = "degraded"
			}
		case "disabled":
			if overall == "ok" {
				overall = "degraded"
			}
		}
	}
	status.Status = overall
	return status
}

func (s *Server) checkLLMProvider(ctx context.Context) string {
	if s == nil || s.llmProvider == nil {
		return "down"
	}
	if _, ok := s.llmProvider.(*llm.UnavailableProvider); ok {
		return "down"
	}
	stream, err := s.llmProvider.Stream(ctx, llm.ChatRequest{})
	if err == nil {
		for chunk := range stream {
			if chunk.Err != nil && !errors.Is(chunk.Err, context.DeadlineExceeded) {
				if errors.Is(chunk.Err, context.Canceled) || chunk.Err.Error() == "model is required" || chunk.Err.Error() == "messages are required" {
					return "ok"
				}
				return "down"
			}
		}
		return "ok"
	}
	if err.Error() == "model is required" || err.Error() == "messages are required" {
		return "ok"
	}
	return "down"
}

func (s *Server) checkDatabase(ctx context.Context) string {
	if s == nil || s.store == nil {
		return "disabled"
	}
	if err := s.store.Ping(ctx); err != nil {
		s.logger.Printf("health check: database down: %v", err)
		return "down"
	}
	return "ok"
}

func (s *Server) checkSandbox(ctx context.Context) string {
	if s == nil {
		return "disabled"
	}

	sb, err := s.getOrCreateSandbox()
	if err != nil {
		s.logger.Printf("health check: sandbox init failed: %v", err)
		return "down"
	}
	result, err := sb.Exec(ctx, "printf sandbox-ok", 2*time.Second)
	if err != nil {
		s.logger.Printf("health check: sandbox down: %v", err)
		return "down"
	}
	if result == nil || result.ExitCode() != 0 {
		return "down"
	}
	if result.Stdout() != "sandbox-ok" {
		s.logger.Printf("health check: sandbox unexpected output: %q", result.Stdout())
		return "down"
	}
	return "ok"
}

// SetStudioDocumentInjection registers a callback that prepends library document text to the
// latest human turn when the client sends studio_document_ids in run context (project notebook chat).
func (s *Server) SetStudioDocumentInjection(fn func(ctx context.Context, userID, conversationID int64, docIDs []int64) string) {
	if s == nil {
		return
	}
	s.studioDocInject = fn
}

// startCleanupTicker 启动定期清理过期会话和运行记录的goroutine
func (s *Server) startCleanupTicker() {
	ticker := time.NewTicker(1 * time.Hour) // 每小时清理一次
	defer ticker.Stop()
	for range ticker.C {
		s.cleanupExpiredData()
	}
}

// cleanupExpiredData 清理过期的会话、运行记录和流
func (s *Server) cleanupExpiredData() {
	now := time.Now()
	expireDuration := 24 * time.Hour // 24小时过期

	// 清理过期会话
	s.sessionsMu.Lock()
	for id, session := range s.sessions {
		if now.Sub(session.UpdatedAt) > expireDuration {
			delete(s.sessions, id)
		}
	}
	sessionsCount := len(s.sessions)
	s.sessionsMu.Unlock()

	// 清理过期运行记录
	s.runsMu.Lock()
	for id, run := range s.runs {
		if now.Sub(run.CreatedAt) > expireDuration {
			delete(s.runs, id)
			// 同时清理对应的流
			delete(s.runStreams, id)
		}
	}
	runsCount := len(s.runs)
	s.runsMu.Unlock()

	s.logger.Printf("langgraphcompat: cleaned up expired data - remaining sessions: %d, remaining runs: %d", sessionsCount, runsCount)
}
