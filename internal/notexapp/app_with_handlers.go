// +build handlers_refactor

package notexapp

// 这是一个示例文件，展示了如何集成新的 handlers 到 notexapp
// 将此代码合并到 app.go 中以启用新的 handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/easyspace-ai/minote/pkg/langgraphcompat"
	"github.com/easyspace-ai/minote/pkg/langgraphcompat/handlers"
	"github.com/easyspace-ai/minote/pkg/langgraphcompat/transform"
	"github.com/easyspace-ai/minote/pkg/langgraphcompat/types"
	"github.com/easyspace-ai/minote/pkg/models"
)

// ConfigWithHandlers 扩展配置以支持新的 handlers
type ConfigWithHandlers struct {
	Config
	// 控制使用哪些新 handlers
	UseNewHandlers map[string]bool
}

// NewWithHandlers 创建支持新 handlers 的 App
func NewWithHandlers(cfg ConfigWithHandlers) (*App, error) {
	addr := NormalizeAddr(cfg.Addr)
	logger := cfg.Logger
	if logger == nil {
		logger = log.New(os.Stderr, "notex ", log.LstdFlags)
	}

	databaseURL := strings.TrimSpace(cfg.DatabaseURL)
	defaultModel := strings.TrimSpace(cfg.DefaultModel)

	// ... 原有数据库和 webroot 初始化代码 ...

	// 创建 legacy server
	aiSrv, err := langgraphcompat.NewServer("", databaseURL, defaultModel, aiOpts...)
	if err != nil {
		return nil, err
	}

	// 创建新的 handlers adapter
	handlerAdapter := handlers.NewServerAdapter(defaultModel)

	// 初始化存储（示例使用内存存储，生产环境应使用数据库）
	if cfg.UseNewHandlers["threads"] {
		handlerAdapter.SetThreadStore(newInMemoryThreadStore())
	}
	if cfg.UseNewHandlers["memory"] {
		handlerAdapter.SetMemoryStore(newInMemoryMemoryStore())
	}
	if cfg.UseNewHandlers["agents"] {
		handlerAdapter.SetMemoryStore(newInMemoryAgentStore())
	}
	if cfg.UseNewHandlers["runs"] {
		// Run store 需要与 legacy server 集成
		handlerAdapter.SetRunStore(newServerRunStore(aiSrv))
	}

	// 创建组合路由
	combinedMux := http.NewServeMux()
	combinedMux.Handle("/api/v1/", notexSrv.Handler())
	combinedMux.Handle("/health", notexSrv.Handler())

	// 注册新 handlers 到 mux
	if len(cfg.UseNewHandlers) > 0 {
		// 获取或创建 mux
		mux := http.NewServeMux()
		handlerAdapter.RegisterMigrationRoutes(mux, cfg.UseNewHandlers)

		// 将新 handlers 挂载到 /api/ 路径
		combinedMux.Handle("/api/", mux)
	}

	// 其余路由交给 legacy server
	combinedMux.Handle("/", aiSrv.Handler())

	// ... 其余代码保持不变 ...
}

// newInMemoryThreadStore 返回内存线程存储（示例）
func newInMemoryThreadStore() handlers.ThreadStore {
	return &inMemoryThreadStore{
		threads: make(map[string]*types.Thread),
	}
}

type inMemoryThreadStore struct {
	threads map[string]*types.Thread
	mu      sync.RWMutex
}

func (s *inMemoryThreadStore) List(offset, limit int) ([]types.Thread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]types.Thread, 0, len(s.threads))
	for _, t := range s.threads {
		result = append(result, *t)
	}
	return result, nil
}

func (s *inMemoryThreadStore) Get(threadID string) (*types.Thread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if t, ok := s.threads[threadID]; ok {
		return t, nil
	}
	return nil, errors.New("thread not found")
}

func (s *inMemoryThreadStore) Create(thread *types.Thread) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.threads[thread.ID] = thread
	return nil
}

func (s *inMemoryThreadStore) Update(thread *types.Thread) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.threads[thread.ID] = thread
	return nil
}

func (s *inMemoryThreadStore) Delete(threadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.threads, threadID)
	return nil
}

func (s *inMemoryThreadStore) Search(query string, limit int) ([]types.Thread, error) {
	// 简单实现
	return s.List(0, limit)
}

// newInMemoryMemoryStore 返回内存内存存储（示例）
func newInMemoryMemoryStore() handlers.MemoryStore {
	return &inMemoryMemoryStore{
		memory: &types.MemoryResponse{
			Version:     "1.0",
			LastUpdated: time.Now().UTC().Format(time.RFC3339),
			Facts:       []types.MemoryFact{},
		},
	}
}

type inMemoryMemoryStore struct {
	memory *types.MemoryResponse
	mu     sync.RWMutex
}

func (s *inMemoryMemoryStore) Get() (*types.MemoryResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.memory, nil
}

func (s *inMemoryMemoryStore) Put(memory *types.MemoryResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memory = memory
	return nil
}

func (s *inMemoryMemoryStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memory.Facts = []types.MemoryFact{}
	return nil
}

func (s *inMemoryMemoryStore) DeleteFact(factID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	filtered := make([]types.MemoryFact, 0, len(s.memory.Facts))
	for _, f := range s.memory.Facts {
		if f.ID != factID {
			filtered = append(filtered, f)
		}
	}
	s.memory.Facts = filtered
	return nil
}

func (s *inMemoryMemoryStore) Reload() error {
	return nil
}

// newInMemoryAgentStore 返回内存智能体存储（示例）
func newInMemoryAgentStore() handlers.AgentStore {
	return &inMemoryAgentStore{
		agents: make(map[string]*models.GatewayAgent),
	}
}

type inMemoryAgentStore struct {
	agents map[string]*models.GatewayAgent
	mu     sync.RWMutex
}

func (s *inMemoryAgentStore) List() ([]models.GatewayAgent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]models.GatewayAgent, 0, len(s.agents))
	for _, a := range s.agents {
		result = append(result, *a)
	}
	return result, nil
}

func (s *inMemoryAgentStore) Get(name string) (*models.GatewayAgent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if a, ok := s.agents[name]; ok {
		return a, nil
	}
	return nil, errors.New("agent not found")
}

func (s *inMemoryAgentStore) Create(agent *models.GatewayAgent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents[agent.Name] = agent
	return nil
}

func (s *inMemoryAgentStore) Update(name string, updates map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.agents[name]; ok {
		if desc, ok := updates["description"].(string); ok {
			a.Description = desc
		}
		if model, ok := updates["model"].(*string); ok {
			a.Model = model
		}
		if groups, ok := updates["tool_groups"].([]string); ok {
			a.ToolGroups = groups
		}
		if soul, ok := updates["soul"].(string); ok {
			a.Soul = soul
		}
		if updatedAt, ok := updates["updated_at"].(time.Time); ok {
			a.UpdatedAt = updatedAt
		}
	}
	return nil
}

func (s *inMemoryAgentStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.agents, name)
	return nil
}

func (s *inMemoryAgentStore) Exists(name string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.agents[name]
	return ok, nil
}

// newServerRunStore 包装 legacy server 作为 RunStore
type serverRunStore struct {
	server *langgraphcompat.Server
}

func newServerRunStore(server *langgraphcompat.Server) handlers.RunStore {
	return &serverRunStore{server: server}
}

func (s *serverRunStore) List(threadID string) ([]handlers.RunInfo, error) {
	// 从 legacy server 获取 runs
	return nil, errors.New("not implemented")
}

func (s *serverRunStore) Get(runID string) (*handlers.RunInfo, error) {
	return nil, errors.New("not implemented")
}

func (s *serverRunStore) GetByThreadAndRun(threadID, runID string) (*handlers.RunInfo, error) {
	return nil, errors.New("not implemented")
}

func (s *serverRunStore) Create(req handlers.RunCreateRequest) (*handlers.RunInfo, error) {
	// 调用 legacy server 的创建逻辑
	return nil, errors.New("not implemented")
}

func (s *serverRunStore) Cancel(runID string) (*handlers.RunInfo, error) {
	return nil, errors.New("not implemented")
}

func (s *serverRunStore) CancelByThread(threadID, runID string) (*handlers.RunInfo, error) {
	return nil, errors.New("not implemented")
}

func (s *serverRunStore) Subscribe(runID string) (*handlers.RunInfo, chan transform.StreamEvent, error) {
	return nil, nil, errors.New("not implemented")
}

func (s *serverRunStore) Unsubscribe(runID string, ch chan transform.StreamEvent) {
	close(ch)
}
