package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/easyspace-ai/minote/pkg/types"
)

// UserPreferenceStore 用户偏好内存存储（可替换为数据库实现）
type UserPreferenceStore struct {
	mu    sync.RWMutex
	prefs map[string]*types.UserPreference // userID -> preference
}

// NewUserPreferenceStore 创建新的偏好存储
func NewUserPreferenceStore() *UserPreferenceStore {
	return &UserPreferenceStore{
		prefs: make(map[string]*types.UserPreference),
	}
}

// GetByUserID 根据用户 ID 获取偏好设置
func (s *UserPreferenceStore) GetByUserID(userID string) (*types.UserPreference, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	pref, exists := s.prefs[userID]
	if !exists {
		// 返回默认偏好
		return types.DefaultUserPreference(userID), nil
	}

	return pref, nil
}

// Save 保存新的偏好设置
func (s *UserPreferenceStore) Save(pref *types.UserPreference) error {
	if pref == nil {
		return fmt.Errorf("preference is nil")
	}

	if err := pref.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pref.CreatedAt = time.Now().UTC()
	pref.UpdatedAt = pref.CreatedAt
	s.prefs[pref.UserID] = pref

	return nil
}

// Update 更新偏好设置
func (s *UserPreferenceStore) Update(pref *types.UserPreference) error {
	if pref == nil {
		return fmt.Errorf("preference is nil")
	}

	if err := pref.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.prefs[pref.UserID]
	if !exists {
		return fmt.Errorf("preference not found for user %s", pref.UserID)
	}

	pref.CreatedAt = existing.CreatedAt
	pref.UpdatedAt = time.Now().UTC()
	s.prefs[pref.UserID] = pref

	return nil
}

// Delete 删除偏好设置
func (s *UserPreferenceStore) Delete(userID string) error {
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.prefs, userID)
	return nil
}

// GetOrCreate 获取或创建默认偏好
func (s *UserPreferenceStore) GetOrCreate(userID string) *types.UserPreference {
	pref, err := s.GetByUserID(userID)
	if err != nil {
		return types.DefaultUserPreference(userID)
	}
	return pref
}

// ListAll 列出所有偏好（仅用于管理）
func (s *UserPreferenceStore) ListAll() []*types.UserPreference {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*types.UserPreference, 0, len(s.prefs))
	for _, pref := range s.prefs {
		result = append(result, pref)
	}
	return result
}

// PreferenceContextKey 用于在 context 中存储偏好的 key
type PreferenceContextKey struct{}

// WithUserPreference 将用户偏好添加到 context
func WithUserPreference(ctx context.Context, pref *types.UserPreference) context.Context {
	return context.WithValue(ctx, PreferenceContextKey{}, pref)
}

// UserPreferenceFromContext 从 context 获取用户偏好
func UserPreferenceFromContext(ctx context.Context) *types.UserPreference {
	if ctx == nil {
		return nil
	}
	if pref, ok := ctx.Value(PreferenceContextKey{}).(*types.UserPreference); ok {
		return pref
	}
	return nil
}

// ApplyPreferenceToPrompt 将用户偏好应用到系统提示词
func ApplyPreferenceToPrompt(basePrompt string, pref *types.UserPreference) string {
	if pref == nil {
		return basePrompt
	}

	overlay := pref.BuildSystemPromptOverlay()
	if overlay == "" {
		return basePrompt
	}

	return overlay + "\n\n" + basePrompt
}

// PreferenceMiddleware 偏中间件（用于 HTTP 处理链）
type PreferenceMiddleware struct {
	store *UserPreferenceStore
}

// NewPreferenceMiddleware 创建偏好中间件
func NewPreferenceMiddleware(store *UserPreferenceStore) *PreferenceMiddleware {
	return &PreferenceMiddleware{store: store}
}

// GetStore 获取存储实例
func (m *PreferenceMiddleware) GetStore() *UserPreferenceStore {
	return m.store
}

// GlobalPreferenceStore 全局偏好存储实例（用于快速集成）
var GlobalPreferenceStore = NewUserPreferenceStore()

// InitGlobalPreferenceStore 初始化全局存储（可传入数据库实现）
func InitGlobalPreferenceStore(store *UserPreferenceStore) {
	GlobalPreferenceStore = store
}
