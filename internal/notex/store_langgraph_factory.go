package notex

import (
	"context"
	"fmt"

	"github.com/easyspace-ai/minote/pkg/langgraphcompat/handlers"
)

// NewLangGraphStores 创建所有 LangGraph 存储实例
// 这是集成到 notexapp 的便捷入口
func NewLangGraphStores(store *Store) (*LangGraphStore, error) {
	if store == nil {
		return nil, fmt.Errorf("store is nil")
	}

	lgStore := NewLangGraphStore(store)

	// 自动迁移表结构
	if err := lgStore.InitLangGraphTables(); err != nil {
		return nil, fmt.Errorf("init langgraph tables: %w", err)
	}

	return lgStore, nil
}

// CreateHandlerStores 创建 handlers.HandlerStores 实例
// 用于直接传递给 handlers.NewAdapterWithStores
func CreateHandlerStores(lgStore *LangGraphStore) *HandlerStores {
	if lgStore == nil {
		return nil
	}

	return &HandlerStores{
		ThreadStore: NewLangGraphThreadStore(lgStore),
		AgentStore:  NewLangGraphAgentStore(lgStore),
		MemoryStore: NewLangGraphMemoryStore(lgStore),
		// RunStore 需要特殊处理，暂时为 nil
		RunStore: nil,
	}
}

// HandlerStores 适配器（避免循环导入）
type HandlerStores struct {
	ThreadStore handlers.ThreadStore
	AgentStore  handlers.AgentStore
	MemoryStore handlers.MemoryStore
	RunStore    handlers.RunStore
}

// ToHandlerStores 转换为 handlers 包的 HandlerStores
func (s *HandlerStores) ToHandlerStores() interface{} {
	// 返回一个可以被 handlers 包使用的结构
	return s
}

// LangGraphStoreStats 返回存储统计信息
func (s *LangGraphStore) Stats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	// 线程数量
	var threadCount int64
	err := s.store.db.QueryRow(ctx, `SELECT COUNT(1) FROM lg_threads`).Scan(&threadCount)
	if err != nil {
		return nil, fmt.Errorf("count threads: %w", err)
	}
	stats["threads"] = threadCount

	// 运行数量
	var runCount int64
	err = s.store.db.QueryRow(ctx, `SELECT COUNT(1) FROM lg_runs`).Scan(&runCount)
	if err != nil {
		return nil, fmt.Errorf("count runs: %w", err)
	}
	stats["runs"] = runCount

	// 事实数量
	var factCount int64
	err = s.store.db.QueryRow(ctx, `SELECT COUNT(1) FROM lg_memory_facts`).Scan(&factCount)
	if err != nil {
		return nil, fmt.Errorf("count facts: %w", err)
	}
	stats["memory_facts"] = factCount

	// Agent 数量
	var agentCount int64
	err = s.store.db.QueryRow(ctx, `SELECT COUNT(1) FROM lg_agents`).Scan(&agentCount)
	if err != nil {
		return nil, fmt.Errorf("count agents: %w", err)
	}
	stats["agents"] = agentCount

	return stats, nil
}

// CleanupOldData 清理过期数据
// days: 删除多少天前的数据
func (s *LangGraphStore) CleanupOldData(ctx context.Context, days int) error {
	// 删除旧的已完成运行记录
	_, err := s.store.db.Exec(ctx, `
		DELETE FROM lg_runs
		WHERE status IN ('success', 'error', 'cancelled')
		AND updated_at < NOW() - INTERVAL '1 day' * $1
	`, days)
	if err != nil {
		return fmt.Errorf("cleanup old runs: %w", err)
	}

	// 删除没有关联运行的孤立线程
	_, err = s.store.db.Exec(ctx, `
		DELETE FROM lg_threads
		WHERE updated_at < NOW() - INTERVAL '1 day' * $1
		AND thread_id NOT IN (SELECT DISTINCT thread_id FROM lg_runs)
	`, days)
	if err != nil {
		return fmt.Errorf("cleanup orphaned threads: %w", err)
	}

	return nil
}
