package notex

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/easyspace-ai/minote/pkg/cache"
)

// Server hosts business endpoints for project notex under /api/v1.
type Server struct {
	cfg    Config
	http   *http.Server
	logger *log.Logger
	store  *Store
	cache  cache.Cache // 缓存实例
	aiHandler http.Handler

	// 细粒度锁
	userMu         sync.RWMutex // 用户相关锁
	tokenMu        sync.RWMutex // Token相关锁
	libraryMu      sync.RWMutex // 文库相关锁
	documentMu     sync.RWMutex // 文档相关锁
	projectMu      sync.RWMutex // 项目相关锁
	materialMu     sync.RWMutex // 资料相关锁
	conversationMu sync.RWMutex // 会话相关锁
	messageMu      sync.RWMutex // 消息相关锁
	agentMu        sync.RWMutex // 代理相关锁

	nextUserID         int64
	nextLibraryID      int64
	nextDocumentID     int64
	nextProjectID      int64
	nextMaterialID     int64
	nextConversationID int64
	nextMessageID      int64
	nextAgentID        int64

	usersByEmail map[string]*User
	usersByID    map[int64]*User
	tokens       map[string]*TokenInfo // token -> token info

	librariesByUser map[int64][]*Library
	documentsByID   map[int64]*Document

	projectsByUser map[int64][]*Project
	materialsByID  map[int64]*Material

	conversationsByUser map[int64][]*Conversation
	messagesByConv      map[int64][]*Message
	agentsByUser        map[int64][]*Agent
	convThreadIDs       map[int64]string

	docExtractOnce sync.Once
	docExtractCh   chan int64
}

// TokenInfo 包含token的元信息
type TokenInfo struct {
	UserID    int64
	ExpiresAt time.Time
}

type contextKey string

const requestIDContextKey contextKey = "request_id"

var requestSeq uint64

const maxBodySize = 10 * 1024

func NewServer(cfg Config) (*Server, error) {
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		addr = ":8787"
	}
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}
	dataRoot := strings.TrimSpace(cfg.DataRoot)
	if dataRoot == "" {
		dataRoot = filepath.Join(".", "data", "notex")
	}
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create data root: %w", err)
	}
	if cfg.Logger == nil {
		cfg.Logger = log.New(os.Stderr, "notex ", log.LstdFlags)
	}
	cfg.Addr = addr
	cfg.DataRoot = dataRoot
	s := &Server{
		cfg:                 cfg,
		logger:              cfg.Logger,
		store:               cfg.Store,
		nextUserID:          1,
		nextLibraryID:       1,
		nextDocumentID:      1,
		nextProjectID:       1,
		nextMaterialID:      1,
		nextConversationID:  1,
		nextMessageID:       1,
		nextAgentID:         1,
		usersByEmail:        map[string]*User{},
		usersByID:           map[int64]*User{},
		tokens:              map[string]*TokenInfo{},
		librariesByUser:     map[int64][]*Library{},
		documentsByID:       map[int64]*Document{},
		projectsByUser:      map[int64][]*Project{},
		materialsByID:       map[int64]*Material{},
		conversationsByUser: map[int64][]*Conversation{},
		messagesByConv:      map[int64][]*Message{},
		agentsByUser:        map[int64][]*Agent{},
		convThreadIDs:       map[int64]string{},
	}
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	s.http = &http.Server{
		Addr:              cfg.Addr,
		Handler:           s.withMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	if s.store != nil {
		s.startDocumentExtractWorkers()
	}

	// 初始化缓存
	if s.cfg.CacheEnabled && s.cfg.RedisAddr != "" {
		var err error
		s.cache, err = cache.NewRedisCache(s.cfg.RedisAddr, s.cfg.RedisPassword, s.cfg.RedisDB)
		if err != nil {
			s.logger.Printf("warning: failed to initialize redis cache: %v, cache disabled", err)
			s.cache = nil
		} else {
			s.logger.Println("redis cache initialized successfully")
		}
	}

	// 启动定期清理过期token的goroutine
	go s.startTokenCleanupTicker()
	return s, nil
}

func (s *Server) Start() error {
	s.logger.Printf("notex server listening on %s", s.cfg.Addr)
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownErr := s.http.Shutdown(ctx)
	if s.store != nil {
		s.store.Close()
	}
	return shutdownErr
}

func (s *Server) Handler() http.Handler {
	return s.http.Handler
}

func (s *Server) SetAIHandler(h http.Handler) {
	s.aiHandler = h
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/meta", s.handleMeta)
	mux.HandleFunc("POST /api/v1/auth/register", s.handleAuthRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleAuthLogin)
	mux.HandleFunc("GET /api/v1/auth/me", s.handleAuthMe)
	mux.HandleFunc("GET /api/v1/agents", s.handleAgentsList)
	mux.HandleFunc("POST /api/v1/agents", s.handleAgentsCreate)
	mux.HandleFunc("GET /api/v1/agents/default-prompt", s.handleAgentsDefaultPrompt)
	mux.HandleFunc("GET /api/v1/agents/{id}", s.handleAgentsGet)
	mux.HandleFunc("PATCH /api/v1/agents/{id}", s.handleAgentsPatch)
	mux.HandleFunc("DELETE /api/v1/agents/{id}", s.handleAgentsDelete)
	mux.HandleFunc("GET /api/v1/libraries", s.handleLibrariesList)
	mux.HandleFunc("POST /api/v1/libraries", s.handleLibrariesCreate)
	mux.HandleFunc("POST /api/v1/libraries/{id}/documents/upload-browser", s.handleDocumentsUploadBrowser)
	mux.HandleFunc("POST /api/v1/libraries/{id}/documents/import-url", s.handleDocumentsImportURL)
	mux.HandleFunc("POST /api/v1/libraries/{id}/documents/query", s.handleDocumentsQuery)
	mux.HandleFunc("GET /api/v1/documents/{id}/vector-inspect", s.handleDocumentsVectorInspect)
	mux.HandleFunc("GET /api/v1/documents/{id}/chat-attachment", s.handleDocumentsChatAttachment)
	mux.HandleFunc("PATCH /api/v1/documents/{id}", s.handleDocumentsPatch)
	mux.HandleFunc("DELETE /api/v1/documents/{id}", s.handleDocumentsDelete)
	mux.HandleFunc("GET /api/v1/projects", s.handleProjectsList)
	mux.HandleFunc("POST /api/v1/projects", s.handleProjectsCreate)
	mux.HandleFunc("GET /api/v1/projects/{id}", s.handleProjectsGet)
	mux.HandleFunc("PATCH /api/v1/projects/{id}", s.handleProjectsPatch)
	mux.HandleFunc("DELETE /api/v1/projects/{id}", s.handleProjectsDelete)
	mux.HandleFunc("GET /api/v1/projects/{id}/materials", s.handleProjectMaterialsList)
	mux.HandleFunc("POST /api/v1/projects/{id}/materials", s.handleProjectMaterialsCreate)
	mux.HandleFunc("GET /api/v1/projects/{id}/materials/{materialId}", s.handleProjectMaterialsGet)
	mux.HandleFunc("PATCH /api/v1/projects/{id}/materials/{materialId}", s.handleProjectMaterialsPatch)
	// Studio materials creation - V2 (后端主导，简化参数)
	mux.HandleFunc("POST /api/v1/projects/{id}/studio/create", s.HandleProjectStudioCreate)

	// 保留旧接口以兼容现有前端
	mux.HandleFunc("POST /api/v1/projects/{id}/materials/slides-pptx", s.handleProjectMaterialsSlidesPPTX)
	mux.HandleFunc("POST /api/v1/projects/{id}/materials/studio-html", s.handleProjectMaterialsStudioHTML)
	mux.HandleFunc("POST /api/v1/projects/{id}/materials/studio-mindmap", s.handleProjectMaterialsStudioMindmap)
	mux.HandleFunc("POST /api/v1/projects/{id}/materials/studio-audio", s.handleProjectMaterialsStudioAudio)
	mux.HandleFunc("GET /api/v1/projects/{id}/materials/{materialId}/pptx", s.handleProjectMaterialPPTXDownload)
	mux.HandleFunc("GET /api/v1/projects/{id}/materials/{materialId}/studio-file", s.handleProjectMaterialStudioFileDownload)
	mux.HandleFunc("GET /api/v1/conversations", s.handleConversationsList)
	mux.HandleFunc("POST /api/v1/conversations", s.handleConversationsCreate)
	mux.HandleFunc("POST /api/v1/conversations/ensure-studio", s.handleConversationsEnsureStudio)
	mux.HandleFunc("PATCH /api/v1/conversations/{id}", s.handleConversationsPatch)
	mux.HandleFunc("DELETE /api/v1/conversations/{id}", s.handleConversationsDelete)
	mux.HandleFunc("GET /api/v1/conversations/{id}/messages", s.handleConversationsMessages)
	mux.HandleFunc("POST /api/v1/conversations/{id}/ensure-thread", s.handleConversationsEnsureThread)
	mux.HandleFunc("POST /api/v1/chat/messages", s.handleChatMessages)

	mux.HandleFunc("GET /api/v1/skills/installed", s.handleSkillsInstalled)
	mux.HandleFunc("GET /api/v1/skills/workspace", s.handleSkillsWorkspace)
	mux.HandleFunc("POST /api/v1/skills/refresh", s.handleSkillsRefresh)
	mux.HandleFunc("POST /api/v1/skills/install", s.handleSkillsInstall)
	mux.HandleFunc("POST /api/v1/skills/{slug}/enable", s.handleSkillsEnable)
	mux.HandleFunc("POST /api/v1/skills/{slug}/disable", s.handleSkillsDisable)
	mux.HandleFunc("DELETE /api/v1/skills/{slug}", s.handleSkillsUninstall)

	// Studio skill API endpoints (V2 - 后端主导架构)
	mux.HandleFunc("GET /api/v1/studio/skills", s.HandleStudioSkillsList)
	mux.HandleFunc("GET /api/v1/studio/skills/{type}", s.HandleStudioSkillDetail)
	mux.HandleFunc("POST /api/v1/studio/generate", s.HandleStudioGenerateV2)
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withSecurityHeaders 添加安全响应头
func (s *Server) withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 防止XSS
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		// 防止Clickjacking
		w.Header().Set("X-Frame-Options", "DENY")
		// 防止MIME类型混淆
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// CSP策略（根据实际情况调整）
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return s.withRecover(s.withLogging(s.withSecurityHeaders(s.withCORS(s.withAuth(s.withRequestID(next))))))
}

func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-Id", requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestBody := readRequestBodyForLog(r)
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK, body: &bytes.Buffer{}}
		next.ServeHTTP(rw, r)
		responseBody := truncateAndSanitizeBody(rw.body.String())
		s.logInfo(r.Context(), "http access", map[string]any{
			"client_ip":     clientIPFromRequest(r),
			"duration":      time.Since(started).Round(time.Millisecond).String(),
			"method":        r.Method,
			"path":          fullPathWithQuery(r),
			"request_body":  requestBody,
			"response_body": responseBody,
			"size":          rw.size,
			"status":        rw.status,
		})
	})
}

func (s *Server) withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logError(r.Context(), "panic recovered", map[string]any{
					"method": r.Method,
					"path":   r.URL.Path,
					"error":  recovered,
				})
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if !s.cfg.AuthRequired || isAuthExemptPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		tok := bearerToken(r)
		if tok == "" {
			writeAPIError(w, http.StatusUnauthorized, NewAPIError(ErrCodeUnauthorized, "unauthorized"))
			return
		}
		s.tokenMu.RLock()
		tokenInfo, ok := s.tokens[tok]
		s.tokenMu.RUnlock()
		if !ok || tokenInfo.UserID <= 0 {
			writeAPIError(w, http.StatusUnauthorized, NewAPIError(ErrCodeUnauthorized, "unauthorized"))
			return
		}
		// 验证token是否过期
		if time.Now().After(tokenInfo.ExpiresAt) {
			// 删除过期token
			s.tokenMu.Lock()
			delete(s.tokens, tok)
			s.tokenMu.Unlock()
			writeAPIError(w, http.StatusUnauthorized, NewAPIError(ErrCodeTokenExpired, "token expired"))
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey{}, tokenInfo.UserID)))
	})
}

type userIDKey struct{}

func (s *Server) requireUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	if uid, ok := r.Context().Value(userIDKey{}).(int64); ok && uid > 0 {
		return uid, true
	}
	if !s.cfg.AuthRequired {
		return 1, true
	}
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	return 0, false
}

func isAuthExemptPath(path string) bool {
	switch path {
	case "/health", "/api/v1/meta", "/api/v1/auth/register", "/api/v1/auth/login":
		return true
	default:
		return false
	}
}

func bearerToken(r *http.Request) string {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
	body   *bytes.Buffer
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.size += len(b)
	if r.body != nil && r.body.Len() < maxBodySize {
		remaining := maxBodySize - r.body.Len()
		if len(b) <= remaining {
			r.body.Write(b)
		} else {
			r.body.Write(b[:remaining])
		}
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	return requestID
}

func newRequestID() string {
	var random [6]byte
	_, _ = rand.Read(random[:])
	return fmt.Sprintf("%d-%s", atomic.AddUint64(&requestSeq, 1), hex.EncodeToString(random[:]))
}

func fullPathWithQuery(r *http.Request) string {
	if strings.TrimSpace(r.URL.RawQuery) == "" {
		return r.URL.Path
	}
	return r.URL.Path + "?" + r.URL.RawQuery
}

func clientIPFromRequest(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		return xrip
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func readRequestBodyForLog(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if !(strings.Contains(ct, "application/json") || strings.Contains(ct, "application/x-www-form-urlencoded") || strings.Contains(ct, "text/")) {
		return "[non-text body skipped]"
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return "[failed to read request body]"
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return truncateAndSanitizeBody(string(bodyBytes))
}

func truncateAndSanitizeBody(body string) string {
	if strings.TrimSpace(body) == "" {
		return ""
	}
	body = sanitizeBody(body)
	if len(body) <= maxBodySize {
		return body
	}
	return body[:maxBodySize] + "... [truncated]"
}

func sanitizeBody(body string) string {
	patterns := []struct {
		re          *regexp.Regexp
		replacement string
	}{
		{regexp.MustCompile(`"password"\s*:\s*"[^"]*"`), `"password":"***"`},
		{regexp.MustCompile(`"token"\s*:\s*"[^"]*"`), `"token":"***"`},
		{regexp.MustCompile(`"access_token"\s*:\s*"[^"]*"`), `"access_token":"***"`},
		{regexp.MustCompile(`"refresh_token"\s*:\s*"[^"]*"`), `"refresh_token":"***"`},
		{regexp.MustCompile(`"authorization"\s*:\s*"[^"]*"`), `"authorization":"***"`},
		{regexp.MustCompile(`"api_key"\s*:\s*"[^"]*"`), `"api_key":"***"`},
		{regexp.MustCompile(`"secret"\s*:\s*"[^"]*"`), `"secret":"***"`},
	}
	out := body
	for _, p := range patterns {
		out = p.re.ReplaceAllString(out, p.replacement)
	}
	return out
}

// startTokenCleanupTicker 定期清理过期token
func (s *Server) startTokenCleanupTicker() {
	ticker := time.NewTicker(1 * time.Hour) // 每小时清理一次
	defer ticker.Stop()
	for range ticker.C {
		s.cleanupExpiredTokens()
	}
}

// cleanupExpiredTokens 清理所有过期的token
func (s *Server) cleanupExpiredTokens() {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	now := time.Now()
	for token, info := range s.tokens {
		if now.After(info.ExpiresAt) {
			delete(s.tokens, token)
		}
	}
	s.logger.Printf("cleaned up expired tokens, remaining: %d", len(s.tokens))
}