package notex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type rowScanner interface {
	Scan(dest ...any) error
}

type rows interface {
	Close()
	Err() error
	Next() bool
	Scan(dest ...any) error
}

type sqlDB interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...any) (rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) rowScanner
}

type pgxPoolDB struct {
	pool *pgxpool.Pool
}

func (p *pgxPoolDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, sql, arguments...)
}

func (p *pgxPoolDB) Query(ctx context.Context, sql string, arguments ...any) (rows, error) {
	return p.pool.Query(ctx, sql, arguments...)
}

func (p *pgxPoolDB) QueryRow(ctx context.Context, sql string, arguments ...any) rowScanner {
	return p.pool.QueryRow(ctx, sql, arguments...)
}

type Store struct {
	pool  *pgxpool.Pool
	db    sqlDB
	close func()
}

func scanTimestampRFC3339(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339)
}

func decodeInt64Slice(raw []byte) ([]int64, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out []int64
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("database url is required")
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	store := &Store{
		pool:  pool,
		db:    &pgxPoolDB{pool: pool},
		close: pool.Close,
	}
	if err := store.AutoMigrate(ctx); err != nil {
		store.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() {
	if s != nil && s.close != nil {
		s.close()
	}
}

func (s *Store) AutoMigrate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("notex store is not initialized")
	}
	if _, err := s.db.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("migrate notex schema: %w", err)
	}
	return nil
}

const schemaSQL = `
create table if not exists notex_users (
	id bigserial primary key,
	email text not null unique,
	password_hash text not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists notex_agents (
	id bigserial primary key,
	user_id bigint not null references notex_users(id) on delete cascade,
	name text not null,
	description text not null default '',
	prompt text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_notex_agents_user_id on notex_agents(user_id);

create table if not exists notex_libraries (
	id bigserial primary key,
	user_id bigint not null references notex_users(id) on delete cascade,
	name text not null,
	chunk_size integer not null default 800,
	chunk_overlap integer not null default 200,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_notex_libraries_user_id on notex_libraries(user_id);

create table if not exists notex_documents (
	id bigserial primary key,
	library_id bigint not null references notex_libraries(id) on delete cascade,
	original_name text not null,
	base64_data text not null default '',
	file_size integer not null default 0,
	mime_type text not null default 'application/octet-stream',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_notex_documents_library_id on notex_documents(library_id);

alter table notex_documents add column if not exists starred boolean not null default false;

alter table notex_documents add column if not exists file_path text not null default '';
alter table notex_documents add column if not exists extracted_text text not null default '';
alter table notex_documents add column if not exists extraction_status text not null default 'pending';
alter table notex_documents add column if not exists extraction_error text not null default '';

create index if not exists idx_notex_documents_extraction on notex_documents (extraction_status);

create table if not exists notex_projects (
	id bigserial primary key,
	user_id bigint not null references notex_users(id) on delete cascade,
	library_id bigint not null references notex_libraries(id),
	name text not null,
	description text not null default '',
	category text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_notex_projects_user_id on notex_projects(user_id);

alter table notex_projects add column if not exists starred boolean not null default false;
alter table notex_projects add column if not exists archived boolean not null default false;
alter table notex_projects add column if not exists icon_index integer not null default -1;
alter table notex_projects add column if not exists accent_hex text not null default '';
alter table notex_projects add column if not exists studio_scope jsonb not null default '{}'::jsonb;

create table if not exists notex_materials (
	id bigserial primary key,
	project_id bigint not null references notex_projects(id) on delete cascade,
	kind text not null,
	title text not null,
	status text not null default 'pending',
	subtitle text not null default '',
	payload jsonb not null default '{}'::jsonb,
	file_path text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_notex_materials_project_id on notex_materials(project_id);

create table if not exists notex_conversations (
	id bigserial primary key,
	user_id bigint not null references notex_users(id) on delete cascade,
	agent_id bigint not null references notex_agents(id),
	name text not null,
	last_message text not null default '',
	library_ids jsonb not null default '[]'::jsonb,
	chat_mode text not null default 'chat',
	thread_id text not null default '',
	studio_only boolean not null default false,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

alter table notex_conversations add column if not exists studio_only boolean not null default false;

create index if not exists idx_notex_conversations_user_id on notex_conversations(user_id);
create index if not exists idx_notex_conversations_thread_id on notex_conversations(thread_id);

create table if not exists notex_messages (
	id bigserial primary key,
	conversation_id bigint not null references notex_conversations(id) on delete cascade,
	role text not null,
	content text not null default '',
	status text not null default 'done',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_notex_messages_conversation_id on notex_messages(conversation_id, created_at);

-- 新增优化索引
create index if not exists idx_notex_documents_library_id_created on notex_documents(library_id, created_at DESC);
create index if not exists idx_notex_projects_user_id_updated on notex_projects(user_id, updated_at DESC);
create index if not exists idx_notex_conversations_user_id_updated on notex_conversations(user_id, updated_at DESC);
create index if not exists idx_notex_materials_project_id_created on notex_materials(project_id, created_at DESC);
`
