package notex

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (s *Store) HasUsers(ctx context.Context) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("notex store is not initialized")
	}
	var count int
	if err := s.db.QueryRow(ctx, `select count(1) from notex_users`).Scan(&count); err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}
	return count > 0, nil
}

func (s *Store) CreateUser(ctx context.Context, email string, passwordHash string) (*User, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("notex store is not initialized")
	}
	now := time.Now().UTC()
	var id int64
	var createdAt time.Time
	if err := s.db.QueryRow(ctx, `
		insert into notex_users (email, password_hash, created_at, updated_at)
		values ($1, $2, $3, $3)
		returning id, created_at
	`, strings.TrimSpace(email), passwordHash, now).Scan(&id, &createdAt); err != nil {
		return nil, fmt.Errorf("create user %q: %w", email, err)
	}
	return &User{ID: id, Email: strings.TrimSpace(email), PasswordHash: passwordHash, CreatedAt: scanTimestampRFC3339(createdAt)}, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("notex store is not initialized")
	}
	var (
		user      User
		createdAt time.Time
	)
	err := s.db.QueryRow(ctx, `
		select id, email, password_hash, created_at
		from notex_users
		where email = $1
	`, strings.TrimSpace(email)).Scan(&user.ID, &user.Email, &user.PasswordHash, &createdAt)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by email %q: %w", email, err)
	}
	user.CreatedAt = scanTimestampRFC3339(createdAt)
	return &user, nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("notex store is not initialized")
	}
	var (
		user      User
		createdAt time.Time
	)
	err := s.db.QueryRow(ctx, `
		select id, email, password_hash, created_at
		from notex_users
		where id = $1
	`, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &createdAt)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id %d: %w", id, err)
	}
	user.CreatedAt = scanTimestampRFC3339(createdAt)
	return &user, nil
}

func (s *Store) ListAgentsByUser(ctx context.Context, userID int64) ([]*Agent, error) {
	rows, err := s.db.Query(ctx, `
		select id, name, description, prompt, created_at, updated_at
		from notex_agents
		where user_id = $1
		order by id asc
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var out []*Agent
	for rows.Next() {
		var (
			agent             Agent
			createdAt, updatedAt time.Time
		)
		if err := rows.Scan(&agent.ID, &agent.Name, &agent.Description, &agent.Prompt, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		agent.CreatedAt = scanTimestampRFC3339(createdAt)
		agent.UpdatedAt = scanTimestampRFC3339(updatedAt)
		out = append(out, &agent)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agents: %w", err)
	}
	return out, nil
}

func (s *Store) CreateAgent(ctx context.Context, userID int64, name string, description string, prompt string) (*Agent, error) {
	now := time.Now().UTC()
	var (
		id                int64
		createdAt, updatedAt time.Time
	)
	err := s.db.QueryRow(ctx, `
		insert into notex_agents (user_id, name, description, prompt, created_at, updated_at)
		values ($1, $2, $3, $4, $5, $5)
		returning id, created_at, updated_at
	`, userID, name, description, prompt, now).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}
	return &Agent{ID: id, Name: name, Description: description, Prompt: prompt, CreatedAt: scanTimestampRFC3339(createdAt), UpdatedAt: scanTimestampRFC3339(updatedAt)}, nil
}

func (s *Store) GetAgentByID(ctx context.Context, userID int64, agentID int64) (*Agent, error) {
	var (
		agent             Agent
		createdAt, updatedAt time.Time
	)
	err := s.db.QueryRow(ctx, `
		select id, name, description, prompt, created_at, updated_at
		from notex_agents
		where user_id = $1 and id = $2
	`, userID, agentID).Scan(&agent.ID, &agent.Name, &agent.Description, &agent.Prompt, &createdAt, &updatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent: %w", err)
	}
	agent.CreatedAt = scanTimestampRFC3339(createdAt)
	agent.UpdatedAt = scanTimestampRFC3339(updatedAt)
	return &agent, nil
}

func (s *Store) UpdateAgent(ctx context.Context, userID int64, agentID int64, name string, description string, prompt string) (*Agent, error) {
	var updatedAt time.Time
	err := s.db.QueryRow(ctx, `
		update notex_agents
		set name = $3, description = $4, prompt = $5, updated_at = now()
		where user_id = $1 and id = $2
		returning updated_at
	`, userID, agentID, name, description, prompt).Scan(&updatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("update agent: %w", err)
	}
	agent, err := s.GetAgentByID(ctx, userID, agentID)
	if err != nil || agent == nil {
		return agent, err
	}
	agent.UpdatedAt = scanTimestampRFC3339(updatedAt)
	return agent, nil
}

func (s *Store) DeleteAgent(ctx context.Context, userID int64, agentID int64) error {
	if _, err := s.db.Exec(ctx, `delete from notex_agents where user_id = $1 and id = $2`, userID, agentID); err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	return nil
}

func (s *Store) ListLibrariesByUser(ctx context.Context, userID int64) ([]*Library, error) {
	rows, err := s.db.Query(ctx, `
		select id, name, chunk_size, chunk_overlap
		from notex_libraries
		where user_id = $1
		order by id asc
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list libraries: %w", err)
	}
	defer rows.Close()

	var out []*Library
	for rows.Next() {
		var lib Library
		if err := rows.Scan(&lib.ID, &lib.Name, &lib.ChunkSize, &lib.ChunkOverlap); err != nil {
			return nil, fmt.Errorf("scan library: %w", err)
		}
		out = append(out, &lib)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate libraries: %w", err)
	}
	return out, nil
}

func (s *Store) CreateLibrary(ctx context.Context, userID int64, name string, chunkSize int, chunkOverlap int) (*Library, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		insert into notex_libraries (user_id, name, chunk_size, chunk_overlap, created_at, updated_at)
		values ($1, $2, $3, $4, now(), now())
		returning id
	`, userID, name, chunkSize, chunkOverlap).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create library: %w", err)
	}
	return &Library{ID: id, Name: name, ChunkSize: chunkSize, ChunkOverlap: chunkOverlap}, nil
}

func (s *Store) ListConversationsByUser(ctx context.Context, userID int64, agentID int64) ([]*Conversation, error) {
	query := `
		select id, agent_id, name, last_message, library_ids, chat_mode, thread_id, studio_only
		from notex_conversations
		where user_id = $1 and not studio_only`
	args := []any{userID}
	if agentID > 0 {
		query += ` and agent_id = $2`
		args = append(args, agentID)
	}
	query += ` order by id asc`
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	var out []*Conversation
	for rows.Next() {
		var (
			conv       Conversation
			libraryIDs []byte
		)
		if err := rows.Scan(&conv.ID, &conv.AgentID, &conv.Name, &conv.LastMessage, &libraryIDs, &conv.ChatMode, &conv.ThreadID, &conv.StudioOnly); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		ids, err := decodeInt64Slice(libraryIDs)
		if err != nil {
			return nil, fmt.Errorf("decode conversation library ids: %w", err)
		}
		conv.LibraryIDs = ids
		out = append(out, &conv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversations: %w", err)
	}
	return out, nil
}

func (s *Store) CreateConversation(ctx context.Context, userID int64, agentID int64, name string, libraryIDs []int64, chatMode string, studioOnly bool) (*Conversation, error) {
	encoded, err := json.Marshal(libraryIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal library ids: %w", err)
	}
	var conv Conversation
	err = s.db.QueryRow(ctx, `
		insert into notex_conversations (user_id, agent_id, name, library_ids, chat_mode, thread_id, studio_only, created_at, updated_at)
		values ($1, $2, $3, $4, $5, '', $6, now(), now())
		returning id, agent_id, name, last_message, chat_mode, thread_id, studio_only
	`, userID, agentID, name, encoded, chatMode, studioOnly).Scan(&conv.ID, &conv.AgentID, &conv.Name, &conv.LastMessage, &conv.ChatMode, &conv.ThreadID, &conv.StudioOnly)
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}
	conv.LibraryIDs = append([]int64(nil), libraryIDs...)
	return &conv, nil
}

// EnsureStudioConversation returns an existing studio_only row for the same user, agent, and library_ids, or creates one.
func (s *Store) EnsureStudioConversation(ctx context.Context, userID int64, agentID int64, libraryIDs []int64, chatMode string) (*Conversation, error) {
	encoded, err := json.Marshal(libraryIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal library ids: %w", err)
	}
	mode := strings.TrimSpace(chatMode)
	if mode == "" {
		mode = "chat"
	}
	var (
		conv       Conversation
		libraryRaw []byte
	)
	err = s.db.QueryRow(ctx, `
		select id, agent_id, name, last_message, library_ids, chat_mode, thread_id, studio_only
		from notex_conversations
		where user_id = $1 and agent_id = $2 and studio_only = true and library_ids = $3::jsonb
		order by id asc
		limit 1
	`, userID, agentID, encoded).Scan(&conv.ID, &conv.AgentID, &conv.Name, &conv.LastMessage, &libraryRaw, &conv.ChatMode, &conv.ThreadID, &conv.StudioOnly)
	if err == nil {
		ids, decErr := decodeInt64Slice(libraryRaw)
		if decErr != nil {
			return nil, fmt.Errorf("decode conversation library ids: %w", decErr)
		}
		conv.LibraryIDs = ids
		return &conv, nil
	}
	if !isNoRows(err) {
		return nil, fmt.Errorf("ensure studio conversation: %w", err)
	}
	return s.CreateConversation(ctx, userID, agentID, "Studio", libraryIDs, mode, true)
}

// GetConversationWithOwner loads a conversation by id and returns its owner user_id (internal: LangGraph studio inject).
func (s *Store) GetConversationWithOwner(ctx context.Context, conversationID int64) (userID int64, conv *Conversation, err error) {
	var (
		c          Conversation
		libraryIDs []byte
		owner      int64
	)
	err = s.db.QueryRow(ctx, `
		select user_id, id, agent_id, name, last_message, library_ids, chat_mode, thread_id, studio_only
		from notex_conversations
		where id = $1
	`, conversationID).Scan(&owner, &c.ID, &c.AgentID, &c.Name, &c.LastMessage, &libraryIDs, &c.ChatMode, &c.ThreadID, &c.StudioOnly)
	if err != nil {
		if isNoRows(err) {
			return 0, nil, nil
		}
		return 0, nil, fmt.Errorf("get conversation with owner: %w", err)
	}
	ids, err := decodeInt64Slice(libraryIDs)
	if err != nil {
		return 0, nil, fmt.Errorf("decode conversation library ids: %w", err)
	}
	c.LibraryIDs = ids
	return owner, &c, nil
}

func (s *Store) GetConversationByID(ctx context.Context, userID int64, conversationID int64) (*Conversation, error) {
	var (
		conv       Conversation
		libraryIDs []byte
	)
	err := s.db.QueryRow(ctx, `
		select id, agent_id, name, last_message, library_ids, chat_mode, thread_id, studio_only
		from notex_conversations
		where user_id = $1 and id = $2
	`, userID, conversationID).Scan(&conv.ID, &conv.AgentID, &conv.Name, &conv.LastMessage, &libraryIDs, &conv.ChatMode, &conv.ThreadID, &conv.StudioOnly)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	ids, err := decodeInt64Slice(libraryIDs)
	if err != nil {
		return nil, fmt.Errorf("decode conversation library ids: %w", err)
	}
	conv.LibraryIDs = ids
	return &conv, nil
}

func (s *Store) SetConversationThreadID(ctx context.Context, userID int64, conversationID int64, threadID string) error {
	if _, err := s.db.Exec(ctx, `
		update notex_conversations
		set thread_id = $3, updated_at = now()
		where user_id = $1 and id = $2
	`, userID, conversationID, strings.TrimSpace(threadID)); err != nil {
		return fmt.Errorf("set conversation thread id: %w", err)
	}
	return nil
}

func (s *Store) UpdateConversationLastMessage(ctx context.Context, userID int64, conversationID int64, lastMessage string) error {
	if _, err := s.db.Exec(ctx, `
		update notex_conversations
		set last_message = $3, updated_at = now()
		where user_id = $1 and id = $2
	`, userID, conversationID, lastMessage); err != nil {
		return fmt.Errorf("update conversation last message: %w", err)
	}
	return nil
}

// PatchConversationName updates the display name; returns nil conversation if not found or not owned.
func (s *Store) PatchConversationName(ctx context.Context, userID int64, conversationID int64, name string) (*Conversation, error) {
	var (
		conv       Conversation
		libraryIDs []byte
	)
	err := s.db.QueryRow(ctx, `
		update notex_conversations
		set name = $3, updated_at = now()
		where user_id = $1 and id = $2
		returning id, agent_id, name, last_message, library_ids, chat_mode, thread_id, studio_only
	`, userID, conversationID, name).Scan(&conv.ID, &conv.AgentID, &conv.Name, &conv.LastMessage, &libraryIDs, &conv.ChatMode, &conv.ThreadID, &conv.StudioOnly)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("patch conversation name: %w", err)
	}
	ids, err := decodeInt64Slice(libraryIDs)
	if err != nil {
		return nil, fmt.Errorf("decode conversation library ids: %w", err)
	}
	conv.LibraryIDs = ids
	return &conv, nil
}

// DeleteConversationForUser deletes the row (messages cascade). Returns false if not found or not owned.
func (s *Store) DeleteConversationForUser(ctx context.Context, userID int64, conversationID int64) (bool, error) {
	var gone int64
	err := s.db.QueryRow(ctx, `
		delete from notex_conversations where user_id = $1 and id = $2 returning id
	`, userID, conversationID).Scan(&gone)
	if err != nil {
		if isNoRows(err) {
			return false, nil
		}
		return false, fmt.Errorf("delete conversation: %w", err)
	}
	return true, nil
}

func (s *Store) ListMessagesByConversation(ctx context.Context, conversationID int64) ([]*Message, error) {
	rows, err := s.db.Query(ctx, `
		select id, conversation_id, role, content, status
		from notex_messages
		where conversation_id = $1
		order by id asc
	`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var out []*Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.Status); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		out = append(out, &msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}
	return out, nil
}

func (s *Store) CreateMessage(ctx context.Context, conversationID int64, role string, content string, status string) (*Message, error) {
	var msg Message
	err := s.db.QueryRow(ctx, `
		insert into notex_messages (conversation_id, role, content, status, created_at, updated_at)
		values ($1, $2, $3, $4, now(), now())
		returning id, conversation_id, role, content, status
	`, conversationID, role, content, status).Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.Status)
	if err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}
	return &msg, nil
}