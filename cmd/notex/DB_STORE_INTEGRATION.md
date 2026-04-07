# 数据库存储集成指南

## 概述

已创建优化的 PostgreSQL 存储实现，支持 LangGraph 兼容性层。

## 新建的数据库表

```sql
-- 线程表
lg_threads
  - thread_id (TEXT, UNIQUE)
  - agent_name (TEXT)
  - title (TEXT)
  - metadata (JSONB)
  - config (JSONB)
  - values (JSONB)
  - status (TEXT)
  - created_at, updated_at

-- 运行记录表
lg_runs
  - run_id (TEXT, UNIQUE)
  - thread_id (TEXT, FK)
  - assistant_id (TEXT)
  - status (TEXT)
  - input, output (JSONB)
  - error (TEXT)
  - usage_tokens (JSONB)

-- 内存事实表
lg_memory_facts
  - session_id (TEXT)
  - fact_id (TEXT)
  - content (TEXT)
  - category (TEXT)
  - confidence (FLOAT)
  - source (TEXT)

-- 内存摘要表
lg_memory_summaries
  - session_id (TEXT, UNIQUE)
  - version (TEXT)
  - user_context (JSONB)
  - history_context (JSONB)

-- 智能体表
lg_agents
  - name (TEXT, UNIQUE)
  - description (TEXT)
  - model (TEXT)
  - tool_groups (JSONB)
  - soul (TEXT)
```

## 快速集成

### 修改 internal/notexapp/app.go

```go
func New(cfg Config) (*App, error) {
    // ... 原有代码 ...

    // 创建 legacy server
    aiSrv, err := langgraphcompat.NewServer("", databaseURL, defaultModel, aiOpts...)
    if err != nil {
        return nil, err
    }

    // ========== 新的数据库存储集成 ==========
    var handlerAdapter *handlers.Adapter
    if databaseURL != "" {
        // 创建 LangGraph 存储
        lgStore, err := notex.NewLangGraphStores(notexStore)
        if err != nil {
            logger.Printf("warning: failed to create langgraph store: %v", err)
        } else {
            // 创建存储实例
            stores := notex.CreateHandlerStores(lgStore)
            
            // 创建适配器
            handlerAdapter = handlers.NewAdapterWithStores(defaultModel, handlers.HandlerStores{
                ThreadStore: stores.ThreadStore,
                AgentStore:  stores.AgentStore,
                MemoryStore: stores.MemoryStore,
                // RunStore 暂时使用 legacy 实现
            })
            
            logger.Printf("langgraph db stores initialized")
        }
    }
    
    // 如果没有创建 adapter，使用默认的
    if handlerAdapter == nil {
        handlerAdapter = handlers.NewAdapter(defaultModel)
    }
    // ========================================

    // 路由设置
    combinedMux := http.NewServeMux()
    combinedMux.Handle("/api/v1/", notexSrv.Handler())
    combinedMux.Handle("/health", notexSrv.Handler())
    
    // 注册新的 handlers（使用数据库存储）
    handlerAdapter.RegisterAllRoutes(combinedMux)
    
    // 其余路由交给 legacy server
    combinedMux.Handle("/", aiSrv.Handler())

    // ... 其余代码 ...
}
```

## 存储实现详情

### ThreadStore (数据库存储)

文件: `internal/notex/store_langgraph_impl.go`

```go
// 支持的操作:
- List(offset, limit) - 分页查询，按 updated_at 倒序
- Get(threadID) - 根据 ID 获取
- Create(thread) - 创建或更新 (UPSERT)
- Update(thread) - 更新字段
- Delete(threadID) - 删除（级联删除 runs）
- Search(query, limit) - 全文搜索

// 索引优化:
- idx_lg_threads_agent - 按 agent 查询
- idx_lg_threads_status - 按状态查询
- idx_lg_threads_updated - 排序优化
- idx_lg_threads_metadata - GIN 索引支持 metadata 查询
```

### AgentStore (数据库存储)

```go
// 支持的操作:
- List() - 按名称排序
- Get(name) - 根据名称获取
- Create(agent) - 创建（支持 UPSERT）
- Update(name, updates) - 动态更新字段
- Delete(name) - 删除
- Exists(name) - 检查存在

// 字段:
- model: *string (可为 nil)
- tool_groups: []string (JSONB 存储)
- soul: string (Markdown 文本)
```

### MemoryStore (数据库存储)

```go
// 支持的操作:
- Get() - 获取默认会话的内存
- Put(memory) - 更新内存摘要和事实
- Clear() - 清空所有事实和摘要
- DeleteFact(factID) - 删除单个事实
- Reload() - 从外部源刷新

// 数据模型:
- lg_memory_summaries: 存储用户和历史上下文
- lg_memory_facts: 存储结构化事实列表
```

## 性能优化

### 1. 数据库索引

```sql
-- 线程查询优化
CREATE INDEX idx_lg_threads_updated ON lg_threads(updated_at DESC);
CREATE INDEX idx_lg_threads_metadata ON lg_threads USING GIN(metadata);

-- 运行查询优化
CREATE INDEX idx_lg_runs_thread ON lg_runs(thread_id);
CREATE INDEX idx_lg_runs_created ON lg_runs(created_at DESC);

-- 内存查询优化
CREATE INDEX idx_lg_memory_session ON lg_memory_facts(session_id);
```

### 2. 批量操作

```go
// 批量插入事实
func (s *LangGraphMemoryStore) Put(memory *types.MemoryResponse) error {
    // 使用事务批量插入
    // 先 DELETE 再 INSERT，简化冲突处理
}
```

### 3. 连接池优化

```go
// 复用 notex.Store 的连接池
// 无需新建连接
```

## 数据清理

```go
// 清理 30 天前的数据
func (s *LangGraphStore) CleanupOldData(ctx context.Context, days int) error {
    // 删除旧的已完成运行
    // 删除孤立的线程
}

// 使用示例
lgStore, _ := notex.NewLangGraphStores(notexStore)
lgStore.CleanupOldData(context.Background(), 30)
```

## 统计信息

```go
// 获取存储统计
stats, err := lgStore.Stats(context.Background())
// stats["threads"] - 线程数量
// stats["runs"] - 运行数量
// stats["memory_facts"] - 内存事实数量
// stats["agents"] - 智能体数量
```

## 完整示例

```go
package main

import (
    "context"
    "log"
    
    "github.com/easyspace-ai/minote/internal/notex"
    "github.com/easyspace-ai/minote/pkg/langgraphcompat/handlers"
)

func main() {
    // 1. 创建 notex store
    notexStore, err := notex.NewStore(context.Background(), databaseURL)
    if err != nil {
        log.Fatal(err)
    }
    defer notexStore.Close()

    // 2. 创建 LangGraph 存储（自动迁移表结构）
    lgStore, err := notex.NewLangGraphStores(notexStore)
    if err != nil {
        log.Fatal(err)
    }

    // 3. 创建 handler stores
    stores := notex.CreateHandlerStores(lgStore)

    // 4. 创建适配器
    adapter := handlers.NewAdapterWithStores("gpt-4", handlers.HandlerStores{
        ThreadStore: stores.ThreadStore,
        AgentStore:  stores.AgentStore,
        MemoryStore: stores.MemoryStore,
        RunStore:    nil, // 使用 legacy 实现
    })

    // 5. 注册路由
    mux := http.NewServeMux()
    adapter.RegisterAllRoutes(mux)

    // 6. 启动服务
    http.ListenAndServe(":8787", mux)
}
```

## 迁移检查清单

- [x] 创建数据库表结构
- [x] 实现 ThreadStore 接口
- [x] 实现 AgentStore 接口
- [x] 实现 MemoryStore 接口
- [ ] 实现 RunStore 接口（需要与执行引擎深度集成）
- [ ] 数据迁移脚本（从文件系统到数据库）
- [ ] 性能基准测试
- [ ] 生产环境验证

## 注意事项

1. **RunStore 未实现**: Run 管理需要与执行引擎深度集成，暂时使用 legacy 实现
2. **会话隔离**: MemoryStore 使用 "default" 作为默认 session_id，可扩展为多用户
3. **数据备份**: 定期备份 lg_* 表数据
4. **监控**: 关注 lg_run_events 表大小，避免无限增长
