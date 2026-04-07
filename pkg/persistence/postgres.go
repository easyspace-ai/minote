package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/easyspace-ai/minote/pkg/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("record not found")

// DB interface abstracting database operations for transaction support
type db interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) rowScanner
	Begin(ctx context.Context) (tx, error)
}

type tx interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) rowScanner
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type rows interface {
	Close()
	Err() error
	Next() bool
	Scan(dest ...any) error
}

type rowScanner interface {
	Scan(dest ...any) error
}

type pgxPoolDB struct {
	pool *pgxpool.Pool
}

func (p *pgxPoolDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, sql, arguments...)
}

func (p *pgxPoolDB) Query(ctx context.Context, sql string, args ...any) (rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

func (p *pgxPoolDB) QueryRow(ctx context.Context, sql string, args ...any) rowScanner {
	return p.pool.QueryRow(ctx, sql, args...)
}

func (p *pgxPoolDB) Begin(ctx context.Context) (tx, error) {
	raw, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxTxDB{tx: raw}, nil
}

type pgxTxDB struct {
	tx pgx.Tx
}

func (t *pgxTxDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, arguments...)
}

func (t *pgxTxDB) Query(ctx context.Context, sql string, args ...any) (rows, error) {
	return t.tx.Query(ctx, sql, args...)
}

func (t *pgxTxDB) QueryRow(ctx context.Context, sql string, args ...any) rowScanner {
	return t.tx.QueryRow(ctx, sql, args...)
}

func (t *pgxTxDB) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *pgxTxDB) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// PostgresStore provides durable persistence for gateway entities
type PostgresStore struct {
	db    db
	close func()
}

// NewPostgresStore creates a new persistence store
func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	if databaseURL == "" {
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

	store := &PostgresStore{
		db:    &pgxPoolDB{pool: pool},
		close: pool.Close,
	}
	if err := store.AutoMigrate(ctx); err != nil {
		store.Close()
		return nil, err
	}

	return store, nil
}

// NewPostgresStoreFromPool creates a store from an existing pool
func NewPostgresStoreFromPool(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{
		db:    &pgxPoolDB{pool: pool},
		close: func() {}, // don't close the shared pool
	}
}

func (s *PostgresStore) Close() {
	if s != nil && s.close != nil {
		s.close()
	}
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("postgres store is not initialized")
	}
	if _, err := s.db.Exec(ctx, "select 1"); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

// AutoMigrate creates/updates the database schema
func (s *PostgresStore) AutoMigrate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("postgres store is not initialized")
	}
	_, err := s.db.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("migrate persistence schema: %w", err)
	}
	return nil
}

const schemaSQL = `
-- Skills: persisted skill enable/disable state
create table if not exists gateway_skills (
	id text primary key,
	name text not null,
	description text not null,
	category text not null,
	license text not null,
	enabled boolean not null default true,
	created_at timestamptz not null,
	updated_at timestamptz not null
);

-- Agents: stored agent configurations
create table if not exists gateway_agents (
	id text primary key,
	name text not null,
	description text not null,
	model text,
	tool_groups jsonb not null default '[]'::jsonb,
	soul text,
	created_at timestamptz not null,
	updated_at timestamptz not null
);

-- Channels: persisted channel configuration
create table if not exists gateway_channels (
	id text primary key,
	enabled boolean not null default false,
	config jsonb not null default '{}'::jsonb,
	created_at timestamptz not null,
	updated_at timestamptz not null
);

-- Users: basic user table for authentication
create table if not exists users (
	id text primary key,
	email text unique not null,
	password_hash text not null,
	name text,
	created_at timestamptz not null,
	updated_at timestamptz not null,
	last_login_at timestamptz
);

create index if not exists idx_users_email on users(email);
`

// ========== Skills ==========

// SaveSkill persists a gateway skill to the database
func (s *PostgresStore) SaveSkill(ctx context.Context, skill models.GatewaySkill) error {
	now := time.Now().UTC()
	if skill.CreatedAt.IsZero() {
		skill.CreatedAt = now
	}
	if skill.UpdatedAt.IsZero() {
		skill.UpdatedAt = now
	}

	_, err := s.db.Exec(ctx, `
		insert into gateway_skills (id, name, description, category, license, enabled, created_at, updated_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
		on conflict (id) do update
		set name = excluded.name,
			description = excluded.description,
			category = excluded.category,
			license = excluded.license,
			enabled = excluded.enabled,
			updated_at = excluded.updated_at
	`, skill.ID, skill.Name, skill.Description, skill.Category, skill.License, skill.Enabled, skill.CreatedAt, skill.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save skill %q: %w", skill.ID, err)
	}
	return nil
}

// GetSkill retrieves a skill by ID
func (s *PostgresStore) GetSkill(ctx context.Context, id string) (models.GatewaySkill, error) {
	row := s.db.QueryRow(ctx, `
		select id, name, description, category, license, enabled, created_at, updated_at
		from gateway_skills
		where id = $1
	`, id)

	var skill models.GatewaySkill
	err := row.Scan(&skill.ID, &skill.Name, &skill.Description, &skill.Category, &skill.License, &skill.Enabled, &skill.CreatedAt, &skill.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.GatewaySkill{}, ErrNotFound
		}
		return models.GatewaySkill{}, fmt.Errorf("get skill %q: %w", id, err)
	}
	return skill, nil
}

// ListSkills lists all persisted skills
func (s *PostgresStore) ListSkills(ctx context.Context) ([]models.GatewaySkill, error) {
	rows, err := s.db.Query(ctx, `
		select id, name, description, category, license, enabled, created_at, updated_at
		from gateway_skills
		order by created_at asc
	`)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()

	skills := make([]models.GatewaySkill, 0)
	for rows.Next() {
		var skill models.GatewaySkill
		err := rows.Scan(&skill.ID, &skill.Name, &skill.Description, &skill.Category, &skill.License, &skill.Enabled, &skill.CreatedAt, &skill.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		skills = append(skills, skill)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list skills rows: %w", err)
	}
	return skills, nil
}

// DeleteSkill deletes a skill from the database
func (s *PostgresStore) DeleteSkill(ctx context.Context, id string) error {
	tag, err := s.db.Exec(ctx, `delete from gateway_skills where id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete skill %q: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateSkillEnabled updates the enabled state of a skill
func (s *PostgresStore) UpdateSkillEnabled(ctx context.Context, id string, enabled bool) error {
	tag, err := s.db.Exec(ctx, `
		update gateway_skills
		set enabled = $2, updated_at = $3
		where id = $1
	`, id, enabled, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("update skill %q enabled: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ========== Agents ==========

// SaveAgent persists a gateway agent to the database
func (s *PostgresStore) SaveAgent(ctx context.Context, agent models.GatewayAgent) error {
	now := time.Now().UTC()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	if agent.UpdatedAt.IsZero() {
		agent.UpdatedAt = now
	}

	toolGroups, err := json.Marshal(agent.ToolGroups)
	if err != nil {
		return fmt.Errorf("marshal agent tool groups: %w", err)
	}

	_, err = s.db.Exec(ctx, `
		insert into gateway_agents (id, name, description, model, tool_groups, soul, created_at, updated_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
		on conflict (id) do update
		set name = excluded.name,
			description = excluded.description,
			model = excluded.model,
			tool_groups = excluded.tool_groups,
			soul = excluded.soul,
			updated_at = excluded.updated_at
	`, agent.ID, agent.Name, agent.Description, agent.Model, toolGroups, agent.Soul, agent.CreatedAt, agent.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save agent %q: %w", agent.ID, err)
	}
	return nil
}

// GetAgent retrieves an agent by ID
func (s *PostgresStore) GetAgent(ctx context.Context, id string) (models.GatewayAgent, error) {
	row := s.db.QueryRow(ctx, `
		select id, name, description, model, tool_groups, soul, created_at, updated_at
		from gateway_agents
		where id = $1
	`, id)

	var agent models.GatewayAgent
	var toolGroupsRaw []byte
	err := row.Scan(&agent.ID, &agent.Name, &agent.Description, &agent.Model, &toolGroupsRaw, &agent.Soul, &agent.CreatedAt, &agent.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.GatewayAgent{}, ErrNotFound
		}
		return models.GatewayAgent{}, fmt.Errorf("get agent %q: %w", id, err)
	}

	if len(toolGroupsRaw) > 0 {
		var toolGroups []string
		if err := json.Unmarshal(toolGroupsRaw, &toolGroups); err != nil {
			return models.GatewayAgent{}, fmt.Errorf("unmarshal agent tool groups: %w", err)
		}
		agent.ToolGroups = toolGroups
	}
	if agent.ToolGroups == nil {
		agent.ToolGroups = []string{}
	}

	return agent, nil
}

// ListAgents lists all persisted agents
func (s *PostgresStore) ListAgents(ctx context.Context) ([]models.GatewayAgent, error) {
	rows, err := s.db.Query(ctx, `
		select id, name, description, model, tool_groups, soul, created_at, updated_at
		from gateway_agents
		order by created_at asc
	`)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	agents := make([]models.GatewayAgent, 0)
	for rows.Next() {
		var agent models.GatewayAgent
		var toolGroupsRaw []byte
		err := rows.Scan(&agent.ID, &agent.Name, &agent.Description, &agent.Model, &toolGroupsRaw, &agent.Soul, &agent.CreatedAt, &agent.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		if len(toolGroupsRaw) > 0 {
			var toolGroups []string
			if err := json.Unmarshal(toolGroupsRaw, &toolGroups); err != nil {
				return nil, fmt.Errorf("unmarshal agent tool groups: %w", err)
			}
			agent.ToolGroups = toolGroups
		}
		if agent.ToolGroups == nil {
			agent.ToolGroups = []string{}
		}
		agents = append(agents, agent)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list agents rows: %w", err)
	}
	return agents, nil
}

// DeleteAgent deletes an agent from the database
func (s *PostgresStore) DeleteAgent(ctx context.Context, id string) error {
	tag, err := s.db.Exec(ctx, `delete from gateway_agents where id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete agent %q: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ========== Channels ==========

// SaveChannel persists a channel configuration to the database
func (s *PostgresStore) SaveChannel(ctx context.Context, id string, enabled bool, config map[string]any) error {
	now := time.Now().UTC()

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal channel config: %w", err)
	}

	_, err = s.db.Exec(ctx, `
		insert into gateway_channels (id, enabled, config, created_at, updated_at)
		values ($1, $2, $3, $4, $5)
		on conflict (id) do update
		set enabled = excluded.enabled,
			config = excluded.config,
			updated_at = excluded.updated_at
	`, id, enabled, configJSON, now, now)
	if err != nil {
		return fmt.Errorf("save channel %q: %w", id, err)
	}
	return nil
}

// GetChannel retrieves a channel by ID
func (s *PostgresStore) GetChannel(ctx context.Context, id string) (enabled bool, config map[string]any, err error) {
	row := s.db.QueryRow(ctx, `
		select enabled, config
		from gateway_channels
		where id = $1
	`, id)

	var configRaw []byte
	err = row.Scan(&enabled, &configRaw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil, ErrNotFound
		}
		return false, nil, fmt.Errorf("get channel %q: %w", id, err)
	}

	if len(configRaw) > 0 {
		if err := json.Unmarshal(configRaw, &config); err != nil {
			return false, nil, fmt.Errorf("unmarshal channel config: %w", err)
		}
	}
	if config == nil {
		config = make(map[string]any)
	}
	return enabled, config, nil
}

// ListChannels lists all persisted channels
func (s *PostgresStore) ListChannels(ctx context.Context) ([]struct {
	ID        string
	Enabled   bool
	Config    map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}, error) {
	rows, err := s.db.Query(ctx, `
		select id, enabled, config, created_at, updated_at
		from gateway_channels
		order by created_at asc
	`)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	defer rows.Close()

	channels := make([]struct {
		ID        string
		Enabled   bool
		Config    map[string]any
		CreatedAt time.Time
		UpdatedAt time.Time
	}, 0)

	for rows.Next() {
		var c struct {
			ID        string
			Enabled   bool
			ConfigRaw []byte
			CreatedAt time.Time
			UpdatedAt time.Time
		}
		err := rows.Scan(&c.ID, &c.Enabled, &c.ConfigRaw, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan channel: %w", err)
		}

		var config map[string]any
		if len(c.ConfigRaw) > 0 {
			if err := json.Unmarshal(c.ConfigRaw, &config); err != nil {
				return nil, fmt.Errorf("unmarshal channel config: %w", err)
			}
		}
		if config == nil {
			config = make(map[string]any)
		}

		channels = append(channels, struct {
			ID        string
			Enabled   bool
			Config    map[string]any
			CreatedAt time.Time
			UpdatedAt time.Time
		}{
			ID:        c.ID,
			Enabled:   c.Enabled,
			Config:    config,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list channels rows: %w", err)
	}
	return channels, nil
}

// DeleteChannel deletes a channel from the database
func (s *PostgresStore) DeleteChannel(ctx context.Context, id string) error {
	tag, err := s.db.Exec(ctx, `delete from gateway_channels where id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete channel %q: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateChannelEnabled updates the enabled state of a channel
func (s *PostgresStore) UpdateChannelEnabled(ctx context.Context, id string, enabled bool) error {
	tag, err := s.db.Exec(ctx, `
		update gateway_channels
		set enabled = $2, updated_at = $3
		where id = $1
	`, id, enabled, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("update channel %q enabled: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
