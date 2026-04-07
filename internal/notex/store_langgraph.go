package notex

// LangGraphStoreSchema 为 LangGraph 兼容性层提供数据库表结构
// 这些表独立于 notex 核心表，专门用于 AI 会话管理
const LangGraphStoreSchema = `
-- LangGraph 线程表
CREATE TABLE IF NOT EXISTS lg_threads (
    id BIGSERIAL PRIMARY KEY,
    thread_id TEXT NOT NULL UNIQUE,
    agent_name TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    values JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'idle',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lg_threads_agent ON lg_threads(agent_name);
CREATE INDEX IF NOT EXISTS idx_lg_threads_status ON lg_threads(status);
CREATE INDEX IF NOT EXISTS idx_lg_threads_updated ON lg_threads(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_lg_threads_metadata ON lg_threads USING GIN(metadata);

-- LangGraph 运行记录表
CREATE TABLE IF NOT EXISTS lg_runs (
    id BIGSERIAL PRIMARY KEY,
    run_id TEXT NOT NULL UNIQUE,
    thread_id TEXT NOT NULL REFERENCES lg_threads(thread_id) ON DELETE CASCADE,
    assistant_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    input JSONB NOT NULL DEFAULT '{}'::jsonb,
    output JSONB NOT NULL DEFAULT '{}'::jsonb,
    error TEXT NOT NULL DEFAULT '',
    usage_tokens JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lg_runs_thread ON lg_runs(thread_id);
CREATE INDEX IF NOT EXISTS idx_lg_runs_status ON lg_runs(status);
CREATE INDEX IF NOT EXISTS idx_lg_runs_created ON lg_runs(created_at DESC);

-- LangGraph 内存/事实表
CREATE TABLE IF NOT EXISTS lg_memory_facts (
    id BIGSERIAL PRIMARY KEY,
    session_id TEXT NOT NULL,
    fact_id TEXT NOT NULL,
    content TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'general',
    confidence FLOAT NOT NULL DEFAULT 1.0,
    source TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, fact_id)
);

CREATE INDEX IF NOT EXISTS idx_lg_memory_session ON lg_memory_facts(session_id);
CREATE INDEX IF NOT EXISTS idx_lg_memory_category ON lg_memory_facts(category);

-- LangGraph 内存摘要表
CREATE TABLE IF NOT EXISTS lg_memory_summaries (
    id BIGSERIAL PRIMARY KEY,
    session_id TEXT NOT NULL UNIQUE,
    version TEXT NOT NULL DEFAULT '1.0',
    user_context JSONB NOT NULL DEFAULT '{}'::jsonb,
    history_context JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lg_memory_summaries_session ON lg_memory_summaries(session_id);

-- LangGraph 事件流表（用于 SSE 持久化）
CREATE TABLE IF NOT EXISTS lg_run_events (
    id BIGSERIAL PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES lg_runs(run_id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    event_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    sequence_num INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lg_events_run ON lg_run_events(run_id, sequence_num);

-- 更新触发器
CREATE OR REPLACE FUNCTION lg_update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS lg_threads_update ON lg_threads;
CREATE TRIGGER lg_threads_update
    BEFORE UPDATE ON lg_threads
    FOR EACH ROW EXECUTE FUNCTION lg_update_timestamp();

DROP TRIGGER IF EXISTS lg_runs_update ON lg_runs;
CREATE TRIGGER lg_runs_update
    BEFORE UPDATE ON lg_runs
    FOR EACH ROW EXECUTE FUNCTION lg_update_timestamp();
`
