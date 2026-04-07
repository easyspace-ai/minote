# LangGraph 兼容性层重构集成指南

## 概述

新的模块化 handlers 已创建完成，现演示如何集成到 `cmd/notex` 主程序中。

## 新 Handler 架构

```
pkg/langgraphcompat/handlers/
├── adapter.go           # 统一适配器
├── agent_handler.go     # 智能体管理
├── memory_handler.go    # 内存管理
├── model_handler.go     # 模型配置
├── run_handler.go       # 运行执行
├── thread_handler.go    # 线程管理
├── integration.go       # Server 集成适配器
└── handlers_test.go     # 完整测试套件
```

## 集成方式

### 方式一：渐进式迁移（推荐）

逐步替换原有 handler，保持 API 兼容。

```go
// internal/notexapp/app.go

import (
    "github.com/easyspace-ai/minote/pkg/langgraphcompat/handlers"
)

func New(cfg Config) (*App, error) {
    // ... 原有代码 ...

    aiSrv, err := langgraphcompat.NewServer(...)
    // ...

    // 创建新的 handlers adapter
    handlerAdapter := handlers.NewServerAdapter(cfg.DefaultModel)
    
    // 设置存储实现（可以是数据库、内存或混合）
    handlerAdapter.SetThreadStore(newThreadStore(databaseURL))
    handlerAdapter.SetMemoryStore(newMemoryStore(databaseURL))
    handlerAdapter.SetAgentStore(newAgentStore(dataRoot))
    handlerAdapter.SetRunStore(newRunStore(aiSrv)) // 需要包装原有 Server 的 run 管理

    // 获取 mux 并注册路由
    mux := aiSrv.Handler().(*http.ServeMux)
    
    // 渐进式迁移：逐个模块启用新 handlers
    useNewHandlers := map[string]bool{
        "models":  true,   // 使用新 handler
        "threads": false,  // 仍用旧 handler
        "memory":  false,
        "agents":  false,
        "runs":    false,
    }
    
    handlerAdapter.RegisterMigrationRoutes(mux, useNewHandlers)
    
    // ... 其余代码 ...
}
```

### 方式二：完整替换

完全使用新的 handlers，需要实现所有存储接口。

```go
// 创建完整的存储实现
stores := handlers.HandlerStores{
    ThreadStore: NewThreadStore(db),
    MemoryStore: NewMemoryStore(db),
    AgentStore:  NewAgentStore(fs),
    RunStore:    NewRunStore(executor),
}

adapter := handlers.NewAdapterWithStores(defaultModel, stores)

// 注册所有路由
mux := http.NewServeMux()
adapter.RegisterAllRoutes(mux)
```

## 存储接口实现示例

### Thread Store（数据库实现）

```go
type dbThreadStore struct {
    db *sql.DB
}

func (s *dbThreadStore) List(offset, limit int) ([]types.Thread, error) {
    rows, err := s.db.Query("SELECT id, agent_name, title, created_at, updated_at FROM threads LIMIT $1 OFFSET $2", limit, offset)
    // ... 实现 ...
}

func (s *dbThreadStore) Get(threadID string) (*types.Thread, error) {
    var t types.Thread
    err := s.db.QueryRow("SELECT ... FROM threads WHERE id = $1", threadID).Scan(...)
    return &t, err
}

// ... 其他方法 ...
```

### Agent Store（文件系统实现）

```go
type fsAgentStore struct {
    dataRoot string
}

func (s *fsAgentStore) List() ([]models.GatewayAgent, error) {
    // 从文件系统读取 agents 目录
}

func (s *fsAgentStore) Create(agent *models.GatewayAgent) error {
    // 保存到文件系统
}

// ... 其他方法 ...
```

## 与现有 Server 集成要点

### 1. 保留原有 Server 的生命周期管理

```go
// aiSrv 仍然负责：
// - Agent 执行引擎
// - SSE 流管理
// - 工具注册
// - 状态管理

// 新 handlers 负责：
// - HTTP API 处理
// - 数据持久化
// - 请求验证
```

### 2. Run Handler 的特殊处理

Run Handler 需要与原有的执行引擎集成：

```go
type serverRunStore struct {
    server *langgraphcompat.Server
}

func (s *serverRunStore) Create(req handlers.RunCreateRequest) (*handlers.RunInfo, error) {
    // 调用原有 Server 的 run 创建逻辑
    // 但使用新的 API 格式
}

func (s *serverRunStore) Subscribe(runID string) (*handlers.RunInfo, chan transform.StreamEvent, error) {
    // 订阅原有 Server 的 SSE 流
}
```

### 3. 配置迁移

```go
// 原有配置
aiSrv, err := langgraphcompat.NewServer(
    "",
    databaseURL,
    defaultModel,
    aiOpts...,
)

// 新配置（可选）
stores := handlers.HandlerStores{
    ThreadStore: NewThreadStore(databaseURL),
    // ...
}
adapter := handlers.NewAdapterWithStores(defaultModel, stores)
```

## 测试验证

```bash
# 运行 handlers 测试
go test ./pkg/langgraphcompat/handlers/... -v

# 运行集成测试
go test ./internal/notexapp/... -v

# 完整构建验证
go build ./cmd/notex
```

## 迁移检查清单

- [ ] 实现 ThreadStore 接口
- [ ] 实现 MemoryStore 接口
- [ ] 实现 AgentStore 接口
- [ ] 实现 RunStore 接口（最复杂，需集成执行引擎）
- [ ] 在 app.go 中配置 adapter
- [ ] 逐个模块启用新 handlers 并测试
- [ ] 前端 API 兼容性测试
- [ ] 性能测试

## 下一步建议

1. **短期**：先迁移 Model 和 Thread handlers（相对独立）
2. **中期**：迁移 Agent 和 Memory handlers
3. **长期**：迁移 Run handlers（需要重构执行引擎接口）
