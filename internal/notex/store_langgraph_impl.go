package notex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/easyspace-ai/minote/pkg/langgraphcompat/handlers"
	"github.com/easyspace-ai/minote/pkg/langgraphcompat/types"
	"github.com/easyspace-ai/minote/pkg/models"
)

// LangGraphStore 为 LangGraph 兼容性层提供优化的数据库存储
type LangGraphStore struct {
	store *Store
}

// NewLangGraphStore 创建新的 LangGraph 存储实例
func NewLangGraphStore(store *Store) *LangGraphStore {
	return &LangGraphStore{store: store}
}

// MigrateLangGraphSchema 执行 LangGraph 表结构迁移
func (s *LangGraphStore) MigrateLangGraphSchema(ctx context.Context) error {
	if s.store == nil || s.store.db == nil {
		return errors.New("store is not initialized")
	}
	if _, err := s.store.db.Exec(ctx, LangGraphStoreSchema); err != nil {
		return fmt.Errorf("migrate langgraph schema: %w", err)
	}
	return nil
}

// ==================== ThreadStore 实现 ====================

var _ handlers.ThreadStore = (*LangGraphThreadStore)(nil)

// LangGraphThreadStore 实现 ThreadStore 接口
type LangGraphThreadStore struct {
	lgStore *LangGraphStore
}

// NewLangGraphThreadStore 创建线程存储实例
func NewLangGraphThreadStore(lgStore *LangGraphStore) *LangGraphThreadStore {
	return &LangGraphThreadStore{lgStore: lgStore}
}

// List 获取线程列表（支持分页和排序）
func (s *LangGraphThreadStore) List(offset, limit int) ([]types.Thread, error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT thread_id, agent_name, title, metadata, created_at, updated_at
		FROM lg_threads
		ORDER BY updated_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := s.lgStore.store.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list threads: %w", err)
	}
	defer rows.Close()

	var result []types.Thread
	for rows.Next() {
		var t types.Thread
		var metadata []byte
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&t.ID, &t.AgentName, &t.Title, &metadata, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan thread: %w", err)
		}

		t.CreatedAt = createdAt.Unix()
		t.UpdatedAt = updatedAt.Unix()

		// 解析 metadata
		if len(metadata) > 0 {
			var meta map[string]string
			if err := json.Unmarshal(metadata, &meta); err == nil {
				t.Metadata = meta
			}
		}

		result = append(result, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate threads: %w", err)
	}
	return result, nil
}

// Get 根据 ID 获取线程
func (s *LangGraphThreadStore) Get(threadID string) (*types.Thread, error) {
	ctx := context.Background()
	var t types.Thread
	var metadata []byte
	var createdAt, updatedAt time.Time

	query := `
		SELECT thread_id, agent_name, title, metadata, created_at, updated_at
		FROM lg_threads
		WHERE thread_id = $1
	`
	err := s.lgStore.store.db.QueryRow(ctx, query, threadID).Scan(
		&t.ID, &t.AgentName, &t.Title, &metadata, &createdAt, &updatedAt,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, errors.New("thread not found")
		}
		return nil, fmt.Errorf("get thread: %w", err)
	}

	t.CreatedAt = createdAt.Unix()
	t.UpdatedAt = updatedAt.Unix()

	if len(metadata) > 0 {
		var meta map[string]string
		if err := json.Unmarshal(metadata, &meta); err == nil {
			t.Metadata = meta
		}
	}

	return &t, nil
}

// Create 创建新线程
func (s *LangGraphThreadStore) Create(thread *types.Thread) error {
	ctx := context.Background()

	// 验证 thread_id
	if strings.TrimSpace(thread.ID) == "" {
		return errors.New("thread_id is required")
	}

	// 序列化 metadata
	metadata, err := json.Marshal(thread.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	now := time.Now().UTC()
	createdAt := time.Unix(thread.CreatedAt, 0)
	updatedAt := time.Unix(thread.UpdatedAt, 0)

	// 如果时间为 0，使用当前时间
	if thread.CreatedAt == 0 {
		createdAt = now
	}
	if thread.UpdatedAt == 0 {
		updatedAt = now
	}

	query := `
		INSERT INTO lg_threads (thread_id, agent_name, title, metadata, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'idle', $5, $6)
		ON CONFLICT (thread_id) DO UPDATE SET
			agent_name = EXCLUDED.agent_name,
			title = EXCLUDED.title,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`
	_, err = s.lgStore.store.db.Exec(ctx, query,
		thread.ID, thread.AgentName, thread.Title, metadata, createdAt, updatedAt,
	)
	if err != nil {
		return fmt.Errorf("create thread: %w", err)
	}
	return nil
}

// Update 更新线程
func (s *LangGraphThreadStore) Update(thread *types.Thread) error {
	ctx := context.Background()

	metadata, err := json.Marshal(thread.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	updatedAt := time.Unix(thread.UpdatedAt, 0)
	if thread.UpdatedAt == 0 {
		updatedAt = time.Now().UTC()
	}

	query := `
		UPDATE lg_threads
		SET agent_name = $2,
		    title = $3,
		    metadata = $4,
		    updated_at = $5
		WHERE thread_id = $1
		RETURNING updated_at
	`
	var dbUpdatedAt time.Time
	err = s.lgStore.store.db.QueryRow(ctx, query,
		thread.ID, thread.AgentName, thread.Title, metadata, updatedAt,
	).Scan(&dbUpdatedAt)

	if err != nil {
		if isNoRows(err) {
			return errors.New("thread not found")
		}
		return fmt.Errorf("update thread: %w", err)
	}

	thread.UpdatedAt = dbUpdatedAt.Unix()
	return nil
}

// Delete 删除线程（级联删除相关 runs）
func (s *LangGraphThreadStore) Delete(threadID string) error {
	ctx := context.Background()
	_, err := s.lgStore.store.db.Exec(ctx, `DELETE FROM lg_threads WHERE thread_id = $1`, threadID)
	if err != nil {
		return fmt.Errorf("delete thread: %w", err)
	}
	return nil
}

// Search 搜索线程
func (s *LangGraphThreadStore) Search(query string, limit int) ([]types.Thread, error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 10
	}

	searchPattern := "%" + strings.ToLower(query) + "%"

	sql := `
		SELECT thread_id, agent_name, title, metadata, created_at, updated_at
		FROM lg_threads
		WHERE LOWER(thread_id) LIKE $1
		   OR LOWER(agent_name) LIKE $1
		   OR LOWER(title) LIKE $1
		ORDER BY updated_at DESC
		LIMIT $2
	`
	rows, err := s.lgStore.store.db.Query(ctx, sql, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search threads: %w", err)
	}
	defer rows.Close()

	var result []types.Thread
	for rows.Next() {
		var t types.Thread
		var metadata []byte
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&t.ID, &t.AgentName, &t.Title, &metadata, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan thread: %w", err)
		}

		t.CreatedAt = createdAt.Unix()
		t.UpdatedAt = updatedAt.Unix()

		if len(metadata) > 0 {
			var meta map[string]string
			if err := json.Unmarshal(metadata, &meta); err == nil {
				t.Metadata = meta
			}
		}

		result = append(result, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate threads: %w", err)
	}
	return result, nil
}

// ==================== AgentStore 实现 ====================

var _ handlers.AgentStore = (*LangGraphAgentStore)(nil)

// LangGraphAgentStore 实现 AgentStore 接口
type LangGraphAgentStore struct {
	lgStore *LangGraphStore
}

// NewLangGraphAgentStore 创建智能体存储实例
func NewLangGraphAgentStore(lgStore *LangGraphStore) *LangGraphAgentStore {
	return &LangGraphAgentStore{lgStore: lgStore}
}

// List 获取所有智能体
func (s *LangGraphAgentStore) List() ([]models.GatewayAgent, error) {
	ctx := context.Background()
	// 注意：这里复用 notex_agents 表，但添加 LangGraph 特有的字段
	query := `
		SELECT id, name, description, model, tool_groups, soul, created_at, updated_at
		FROM lg_agents
		ORDER BY name ASC
	`
	rows, err := s.lgStore.store.db.Query(ctx, query)
	if err != nil {
		// 如果表不存在，返回空列表
		return []models.GatewayAgent{}, nil
	}
	defer rows.Close()

	var result []models.GatewayAgent
	for rows.Next() {
		var a models.GatewayAgent
		var model, soul *string
		var toolGroups []byte
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &model, &toolGroups, &soul, &createdAt, &updatedAt); err != nil {
			continue
		}

		a.Model = model
		a.Soul = strings.TrimSpace(func() string {
			if soul == nil {
				return ""
			}
			return *soul
		}())

		if len(toolGroups) > 0 {
			json.Unmarshal(toolGroups, &a.ToolGroups)
		}

		a.CreatedAt = createdAt
		a.UpdatedAt = updatedAt
		result = append(result, a)
	}

	return result, nil
}

// Get 根据名称获取智能体
func (s *LangGraphAgentStore) Get(name string) (*models.GatewayAgent, error) {
	ctx := context.Background()
	var a models.GatewayAgent
	var model, soul *string
	var toolGroups []byte
	var createdAt, updatedAt time.Time

	query := `
		SELECT id, name, description, model, tool_groups, soul, created_at, updated_at
		FROM lg_agents
		WHERE name = $1
	`
	err := s.lgStore.store.db.QueryRow(ctx, query, name).Scan(
		&a.ID, &a.Name, &a.Description, &model, &toolGroups, &soul, &createdAt, &updatedAt,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, errors.New("agent not found")
		}
		return nil, fmt.Errorf("get agent: %w", err)
	}

	a.Model = model
	if soul != nil {
		a.Soul = *soul
	}
	if len(toolGroups) > 0 {
		json.Unmarshal(toolGroups, &a.ToolGroups)
	}
	a.CreatedAt = createdAt
	a.UpdatedAt = updatedAt

	return &a, nil
}

// Create 创建智能体
func (s *LangGraphAgentStore) Create(agent *models.GatewayAgent) error {
	ctx := context.Background()

	toolGroups, _ := json.Marshal(agent.ToolGroups)
	now := time.Now().UTC()

	query := `
		INSERT INTO lg_agents (name, description, model, tool_groups, soul, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		ON CONFLICT (name) DO UPDATE SET
			description = EXCLUDED.description,
			model = EXCLUDED.model,
			tool_groups = EXCLUDED.tool_groups,
			soul = EXCLUDED.soul,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`
	var id int64
	err := s.lgStore.store.db.QueryRow(ctx, query,
		agent.Name, agent.Description, agent.Model, toolGroups, agent.Soul, now,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	agent.ID = fmt.Sprintf("%d", id)
	agent.CreatedAt = now
	agent.UpdatedAt = now
	return nil
}

// Update 更新智能体
func (s *LangGraphAgentStore) Update(name string, updates map[string]any) error {
	ctx := context.Background()

	// 构建动态更新语句
	setClauses := []string{}
	args := []any{}
	argIndex := 1

	if desc, ok := updates["description"].(string); ok {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, desc)
		argIndex++
	}
	if model, ok := updates["model"].(*string); ok {
		setClauses = append(setClauses, fmt.Sprintf("model = $%d", argIndex))
		args = append(args, model)
		argIndex++
	}
	if groups, ok := updates["tool_groups"].([]string); ok {
		data, _ := json.Marshal(groups)
		setClauses = append(setClauses, fmt.Sprintf("tool_groups = $%d", argIndex))
		args = append(args, data)
		argIndex++
	}
	if soul, ok := updates["soul"].(string); ok {
		setClauses = append(setClauses, fmt.Sprintf("soul = $%d", argIndex))
		args = append(args, soul)
		argIndex++
	}
	if _, ok := updates["updated_at"].(time.Time); ok {
		setClauses = append(setClauses, "updated_at = NOW()")
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, name)
	query := fmt.Sprintf("UPDATE lg_agents SET %s WHERE name = $%d",
		strings.Join(setClauses, ", "), argIndex)

	_, err := s.lgStore.store.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update agent: %w", err)
	}
	return nil
}

// Delete 删除智能体
func (s *LangGraphAgentStore) Delete(name string) error {
	ctx := context.Background()
	_, err := s.lgStore.store.db.Exec(ctx, `DELETE FROM lg_agents WHERE name = $1`, name)
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	return nil
}

// Exists 检查智能体是否存在
func (s *LangGraphAgentStore) Exists(name string) (bool, error) {
	ctx := context.Background()
	var count int
	err := s.lgStore.store.db.QueryRow(ctx,
		`SELECT COUNT(1) FROM lg_agents WHERE name = $1`, name,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check agent exists: %w", err)
	}
	return count > 0, nil
}

// ==================== MemoryStore 实现 ====================

var _ handlers.MemoryStore = (*LangGraphMemoryStore)(nil)

// LangGraphMemoryStore 实现 MemoryStore 接口
type LangGraphMemoryStore struct {
	lgStore *LangGraphStore
}

// NewLangGraphMemoryStore 创建内存存储实例
func NewLangGraphMemoryStore(lgStore *LangGraphStore) *LangGraphMemoryStore {
	return &LangGraphMemoryStore{lgStore: lgStore}
}

// Get 获取内存数据
func (s *LangGraphMemoryStore) Get() (*types.MemoryResponse, error) {
	ctx := context.Background()
	// 获取默认会话的内存（或使用全局配置）
	sessionID := "default"

	// 获取摘要
	var summary types.MemoryResponse
	var userCtx, historyCtx []byte
	query := `
		SELECT version, user_context, history_context, last_updated
		FROM lg_memory_summaries
		WHERE session_id = $1
	`
	var lastUpdated time.Time
	err := s.lgStore.store.db.QueryRow(ctx, query, sessionID).Scan(
		&summary.Version, &userCtx, &historyCtx, &lastUpdated,
	)
	if err != nil && !isNoRows(err) {
		return nil, fmt.Errorf("get memory summary: %w", err)
	}

	if err == nil {
		summary.LastUpdated = lastUpdated.Format(time.RFC3339)
		if len(userCtx) > 0 {
			json.Unmarshal(userCtx, &summary.User)
		}
		if len(historyCtx) > 0 {
			json.Unmarshal(historyCtx, &summary.History)
		}
	}

	// 获取事实列表
	factsQuery := `
		SELECT fact_id, content, category, confidence, created_at, source
		FROM lg_memory_facts
		WHERE session_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.lgStore.store.db.Query(ctx, factsQuery, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get memory facts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var f types.MemoryFact
		var createdAt time.Time
		if err := rows.Scan(&f.ID, &f.Content, &f.Category, &f.Confidence, &createdAt, &f.Source); err != nil {
			continue
		}
		f.CreatedAt = createdAt.Format(time.RFC3339)
		summary.Facts = append(summary.Facts, f)
	}

	return &summary, nil
}

// Put 更新内存数据
func (s *LangGraphMemoryStore) Put(memory *types.MemoryResponse) error {
	ctx := context.Background()
	sessionID := "default"

	// 更新摘要
	userCtx, _ := json.Marshal(memory.User)
	historyCtx, _ := json.Marshal(memory.History)
	lastUpdated, _ := time.Parse(time.RFC3339, memory.LastUpdated)
	if lastUpdated.IsZero() {
		lastUpdated = time.Now().UTC()
	}

	summaryQuery := `
		INSERT INTO lg_memory_summaries (session_id, version, user_context, history_context, last_updated)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (session_id) DO UPDATE SET
			version = EXCLUDED.version,
			user_context = EXCLUDED.user_context,
			history_context = EXCLUDED.history_context,
			last_updated = EXCLUDED.last_updated
	`
	_, err := s.lgStore.store.db.Exec(ctx, summaryQuery,
		sessionID, memory.Version, userCtx, historyCtx, lastUpdated,
	)
	if err != nil {
		return fmt.Errorf("put memory summary: %w", err)
	}

	// 批量更新事实（简化实现：先删除再插入）
	_, err = s.lgStore.store.db.Exec(ctx, `DELETE FROM lg_memory_facts WHERE session_id = $1`, sessionID)
	if err != nil {
		return fmt.Errorf("clear memory facts: %w", err)
	}

	for _, fact := range memory.Facts {
		createdAt, _ := time.Parse(time.RFC3339, fact.CreatedAt)
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}

		_, err := s.lgStore.store.db.Exec(ctx, `
			INSERT INTO lg_memory_facts (session_id, fact_id, content, category, confidence, source, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, sessionID, fact.ID, fact.Content, fact.Category, fact.Confidence, fact.Source, createdAt)
		if err != nil {
			return fmt.Errorf("insert memory fact: %w", err)
		}
	}

	return nil
}

// Clear 清空内存
func (s *LangGraphMemoryStore) Clear() error {
	ctx := context.Background()
	sessionID := "default"

	_, err := s.lgStore.store.db.Exec(ctx, `DELETE FROM lg_memory_facts WHERE session_id = $1`, sessionID)
	if err != nil {
		return fmt.Errorf("clear memory facts: %w", err)
	}

	_, err = s.lgStore.store.db.Exec(ctx,
		`UPDATE lg_memory_summaries SET user_context = '{}', history_context = '{}', last_updated = NOW() WHERE session_id = $1`,
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("clear memory summary: %w", err)
	}

	return nil
}

// DeleteFact 删除单个事实
func (s *LangGraphMemoryStore) DeleteFact(factID string) error {
	ctx := context.Background()
	sessionID := "default"

	_, err := s.lgStore.store.db.Exec(ctx,
		`DELETE FROM lg_memory_facts WHERE session_id = $1 AND fact_id = $2`,
		sessionID, factID,
	)
	if err != nil {
		return fmt.Errorf("delete memory fact: %w", err)
	}
	return nil
}

// Reload 重新加载内存（从外部源刷新）
func (s *LangGraphMemoryStore) Reload() error {
	// 这里可以实现从外部向量数据库或文件重新加载的逻辑
	return nil
}

// ==================== 初始化表结构 ====================

// InitLangGraphTables 初始化 LangGraph 相关表
func (s *LangGraphStore) InitLangGraphTables() error {
	ctx := context.Background()

	// 检查 agents 表是否存在，不存在则创建
	agentTableSQL := `
	CREATE TABLE IF NOT EXISTS lg_agents (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT NOT NULL DEFAULT '',
		model TEXT,
		tool_groups JSONB NOT NULL DEFAULT '[]'::jsonb,
		soul TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	`
	if _, err := s.store.db.Exec(ctx, agentTableSQL); err != nil {
		return fmt.Errorf("create lg_agents table: %w", err)
	}

	return s.MigrateLangGraphSchema(ctx)
}
