package checkpoint

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/easyspace-ai/minote/pkg/models"
)

// MemoryStore is an in-memory checkpoint store for development and testing.
// Data is lost when the process exits.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*models.Session
	messages map[string][]models.Message // keyed by session_id
}

// NewMemoryStore creates a new in-memory checkpoint store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*models.Session),
		messages: make(map[string][]models.Message),
	}
}

func (m *MemoryStore) AutoMigrate(_ context.Context) error {
	return nil
}

func (m *MemoryStore) Close() {}

func (m *MemoryStore) CreateSession(ctx context.Context, session *models.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.sessions[session.ID]; exists {
		return errors.New("session already exists")
	}
	now := time.Now().UTC()
	session.CreatedAt = now
	session.UpdatedAt = now
	copied := *session
	m.sessions[session.ID] = &copied
	return nil
}

func (m *MemoryStore) GetSession(_ context.Context, id string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	copied := *session
	return &copied, nil
}

func (m *MemoryStore) UpdateSession(_ context.Context, session *models.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[session.ID]; !ok {
		return ErrNotFound
	}
	session.UpdatedAt = time.Now().UTC()
	copied := *session
	m.sessions[session.ID] = &copied
	return nil
}

func (m *MemoryStore) DeleteSession(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	delete(m.messages, id)
	return nil
}

func (m *MemoryStore) ListSessions(_ context.Context, userID string) ([]*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Session
	for _, session := range m.sessions {
		if userID != "" && session.UserID != userID {
			continue
		}
		copied := *session
		result = append(result, &copied)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result, nil
}

func (m *MemoryStore) AddMessage(_ context.Context, sessionID string, msg *models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[sessionID] = append(m.messages[sessionID], *msg)
	if session, ok := m.sessions[sessionID]; ok {
		session.UpdatedAt = time.Now().UTC()
	}
	return nil
}

func (m *MemoryStore) GetMessages(_ context.Context, sessionID string) ([]models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	msgs := m.messages[sessionID]
	result := make([]models.Message, len(msgs))
	copy(result, msgs)
	return result, nil
}

func (m *MemoryStore) DeleteMessages(_ context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.messages, sessionID)
	return nil
}

func (m *MemoryStore) ReplaceMessages(_ context.Context, sessionID string, msgs []models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]models.Message, len(msgs))
	copy(copied, msgs)
	m.messages[sessionID] = copied
	return nil
}
