package notex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Server) hasUsers(ctx context.Context) bool {
	if s.store != nil {
		hasUsers, err := s.store.HasUsers(ctx)
		if err == nil {
			return hasUsers
		}
		s.logger.Printf("notex: store HasUsers failed, fallback to memory: %v", err)
	}
	s.userMu.RLock()
	defer s.userMu.RUnlock()
	return len(s.usersByID) > 0
}

func (s *Server) getUserByEmail(ctx context.Context, email string) (*User, error) {
	if s.store != nil {
		return s.store.GetUserByEmail(ctx, email)
	}
	s.userMu.RLock()
	defer s.userMu.RUnlock()
	return s.usersByEmail[email], nil
}

func (s *Server) getUserByID(ctx context.Context, userID int64) (*User, error) {
	// 先查缓存
	if s.cache != nil {
		key := fmt.Sprintf("user:id:%d", userID)
		var user User
		if err := s.cache.Get(ctx, key, &user); err == nil {
			return &user, nil
		}
	}

	if s.store != nil {
		user, err := s.store.GetUserByID(ctx, userID)
		// 写入缓存
		if err == nil && user != nil && s.cache != nil {
			key := fmt.Sprintf("user:id:%d", userID)
			_ = s.cache.Set(ctx, key, user, 1*time.Hour) // 缓存1小时
		}
		return user, err
	}

	s.userMu.RLock()
	defer s.userMu.RUnlock()
	return s.usersByID[userID], nil
}

func (s *Server) createUser(ctx context.Context, email string, passwordHash string) (*User, error) {
	if s.store != nil {
		return s.store.CreateUser(ctx, email, passwordHash)
	}
	s.userMu.Lock()
	defer s.userMu.Unlock()
	if _, exists := s.usersByEmail[email]; exists {
		return nil, errors.New("email_already_exists")
	}
	u := &User{ID: s.nextUserID, Email: email, PasswordHash: passwordHash, CreatedAt: nowRFC3339()}
	s.nextUserID++
	s.usersByEmail[email] = u
	s.usersByID[u.ID] = u
	return u, nil
}

func (s *Server) ensureDefaultLibraryForUser(ctx context.Context, userID int64) (*Library, error) {
	if s.store != nil {
		libs, err := s.store.ListLibrariesByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if len(libs) > 0 {
			return libs[0], nil
		}
		return s.store.CreateLibrary(ctx, userID, "Default Library", 800, 200)
	}
	s.libraryMu.Lock()
	defer s.libraryMu.Unlock()
	return s.ensureDefaultLibrary(userID), nil
}

func (s *Server) ensureDefaultAgentForUser(ctx context.Context, userID int64) (*Agent, error) {
	if s.store != nil {
		agents, err := s.store.ListAgentsByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if len(agents) > 0 {
			return agents[0], nil
		}
		return s.store.CreateAgent(ctx, userID, "General Assistant", "Default assistant for Notex", defaultAgentPrompt)
	}
	s.agentMu.Lock()
	defer s.agentMu.Unlock()
	return s.ensureDefaultAgent(userID), nil
}

func (s *Server) listAgents(ctx context.Context, userID int64) ([]*Agent, error) {
	if s.store != nil {
		agents, err := s.store.ListAgentsByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if len(agents) == 0 {
			if _, err := s.ensureDefaultAgentForUser(ctx, userID); err != nil {
				return nil, err
			}
			return s.store.ListAgentsByUser(ctx, userID)
		}
		return agents, nil
	}
	s.agentMu.Lock()
	defer s.agentMu.Unlock()
	agents := s.agentsByUser[userID]
	if len(agents) == 0 {
		s.ensureDefaultAgent(userID)
		agents = s.agentsByUser[userID]
	}
	return append([]*Agent(nil), agents...), nil
}

func (s *Server) listLibraries(ctx context.Context, userID int64) ([]*Library, error) {
	if s.store != nil {
		libs, err := s.store.ListLibrariesByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if len(libs) == 0 {
			if _, err := s.ensureDefaultLibraryForUser(ctx, userID); err != nil {
				return nil, err
			}
			return s.store.ListLibrariesByUser(ctx, userID)
		}
		return libs, nil
	}
	s.libraryMu.Lock()
	defer s.libraryMu.Unlock()
	libs := s.librariesByUser[userID]
	if len(libs) == 0 {
		s.ensureDefaultLibrary(userID)
		libs = s.librariesByUser[userID]
	}
	return append([]*Library(nil), libs...), nil
}

const defaultAgentPrompt = "You are a helpful assistant for project Notex."

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"run_mode":      "notex",
		"auth_required": s.cfg.AuthRequired,
		"has_users":     s.hasUsers(r.Context()),
	})
}

func (s *Server) handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	pwd := strings.TrimSpace(body.Password)
	if email == "" || pwd == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_and_password_required"})
		return
	}
	if existing, err := s.getUserByEmail(r.Context(), email); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	} else if existing != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_already_exists"})
		return
	}
	u, err := s.createUser(r.Context(), email, hashPassword(pwd))
	if err != nil {
		if err.Error() == "email_already_exists" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_already_exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if _, err := s.ensureDefaultLibraryForUser(r.Context(), u.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if _, err := s.ensureDefaultAgentForUser(r.Context(), u.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	token := uuid.NewString()
	s.tokenMu.Lock()
	s.tokens[token] = &TokenInfo{
		UserID:    u.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // Token有效期7天
	}
	s.tokenMu.Unlock()
	writeJSON(w, http.StatusCreated, map[string]any{"token": token, "user": u})
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	pwd := strings.TrimSpace(body.Password)
	u, err := s.getUserByEmail(r.Context(), email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if u == nil || u.PasswordHash != hashPassword(pwd) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
		return
	}
	token := uuid.NewString()
	s.tokenMu.Lock()
	s.tokens[token] = &TokenInfo{
		UserID:    u.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // Token有效期7天
	}
	s.tokenMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": u})
}

func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	u, err := s.getUserByID(r.Context(), uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if u == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u})
}

func (s *Server) handleAgentsList(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	agents, err := s.listAgents(r.Context(), uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, agents)
}

func (s *Server) handleAgentsDefaultPrompt(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireUserID(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"prompt": defaultAgentPrompt})
}

func (s *Server) handleAgentsCreate(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Prompt      string `json:"prompt"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = "Untitled Agent"
	}
	if s.store != nil {
		agent, err := s.store.CreateAgent(r.Context(), uid, name, strings.TrimSpace(body.Description), strings.TrimSpace(body.Prompt))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, agent)
		return
	}
	now := nowRFC3339()
	s.agentMu.Lock()
	a := &Agent{ID: s.nextAgentID, Name: name, Description: strings.TrimSpace(body.Description), Prompt: strings.TrimSpace(body.Prompt), CreatedAt: now, UpdatedAt: now}
	s.nextAgentID++
	s.agentsByUser[uid] = append(s.agentsByUser[uid], a)
	s.agentMu.Unlock()
	writeJSON(w, http.StatusCreated, a)
}

func (s *Server) handleAgentsGet(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	if s.store != nil {
		agent, err := s.store.GetAgentByID(r.Context(), uid, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if agent == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "agent_not_found"})
			return
		}
		writeJSON(w, http.StatusOK, agent)
		return
	}
	s.agentMu.RLock()
	defer s.agentMu.RUnlock()
	for _, a := range s.agentsByUser[uid] {
		if a.ID == id {
			writeJSON(w, http.StatusOK, a)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "agent_not_found"})
}

func (s *Server) handleAgentsPatch(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Prompt      *string `json:"prompt"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if s.store != nil {
		agent, err := s.store.GetAgentByID(r.Context(), uid, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if agent == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "agent_not_found"})
			return
		}
		if body.Name != nil {
			agent.Name = strings.TrimSpace(*body.Name)
		}
		if body.Description != nil {
			agent.Description = strings.TrimSpace(*body.Description)
		}
		if body.Prompt != nil {
			agent.Prompt = strings.TrimSpace(*body.Prompt)
		}
		updated, err := s.store.UpdateAgent(r.Context(), uid, id, agent.Name, agent.Description, agent.Prompt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, updated)
		return
	}
	s.agentMu.Lock()
	defer s.agentMu.Unlock()
	for _, a := range s.agentsByUser[uid] {
		if a.ID != id {
			continue
		}
		if body.Name != nil {
			a.Name = strings.TrimSpace(*body.Name)
		}
		if body.Description != nil {
			a.Description = strings.TrimSpace(*body.Description)
		}
		if body.Prompt != nil {
			a.Prompt = strings.TrimSpace(*body.Prompt)
		}
		a.UpdatedAt = nowRFC3339()
		writeJSON(w, http.StatusOK, a)
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "agent_not_found"})
}

func (s *Server) handleAgentsDelete(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	if s.store != nil {
		if err := s.store.DeleteAgent(r.Context(), uid, id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	s.agentMu.Lock()
	list := s.agentsByUser[uid]
	filtered := list[:0]
	for _, a := range list {
		if a.ID != id {
			filtered = append(filtered, a)
		}
	}
	s.agentsByUser[uid] = filtered
	s.agentMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleLibrariesList(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	libs, err := s.listLibraries(r.Context(), uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, libs)
}

func (s *Server) handleLibrariesCreate(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = "Library"
	}
	if s.store != nil {
		lib, err := s.store.CreateLibrary(r.Context(), uid, name, 800, 200)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, lib)
		return
	}
	s.libraryMu.Lock()
	lib := &Library{ID: s.nextLibraryID, Name: name, ChunkSize: 800, ChunkOverlap: 200}
	s.nextLibraryID++
	s.librariesByUser[uid] = append(s.librariesByUser[uid], lib)
	s.libraryMu.Unlock()
	writeJSON(w, http.StatusCreated, lib)
}