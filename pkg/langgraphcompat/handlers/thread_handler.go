package handlers

import (
	"fmt"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/easyspace-ai/minote/pkg/langgraphcompat/types"
)

// ThreadStore defines the interface for thread storage.
type ThreadStore interface {
	List(offset, limit int) ([]types.Thread, error)
	Get(threadID string) (*types.Thread, error)
	Create(thread *types.Thread) error
	Update(thread *types.Thread) error
	Delete(threadID string) error
	Search(query string, limit int) ([]types.Thread, error)
}

// ThreadHandler handles thread-related requests.
type ThreadHandler struct {
	store ThreadStore
}

// NewThreadHandler creates a new ThreadHandler.
func NewThreadHandler(store ThreadStore) *ThreadHandler {
	return &ThreadHandler{store: store}
}

// ThreadCreateRequest represents a request to create a thread.
type ThreadCreateRequest struct {
	ThreadID string            `json:"thread_id,omitempty"`
	AgentName string           `json:"agent_name,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ThreadResponse represents a thread response.
type ThreadResponse struct {
	ID        string            `json:"id"`
	AgentName string            `json:"agent_name"`
	Title     string            `json:"title"`
	CreatedAt int64             `json:"created_at"`
	UpdatedAt int64             `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// HandleList handles GET /api/threads.
func (h *ThreadHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	// Parse pagination
	limit := 100
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := parseInt(o); err == nil {
			offset = parsed
		}
	}

	threads, err := h.store.List(offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list threads")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"threads": threads,
	})
}

// HandleSearch handles POST /api/threads/search.
func (h *ThreadHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	threads, err := h.store.Search(req.Query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to search threads")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"threads": threads,
	})
}

// HandleCreate handles POST /api/threads.
func (h *ThreadHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req ThreadCreateRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
	}

	// Validate or generate thread ID
	threadID := strings.TrimSpace(req.ThreadID)
	if threadID != "" {
		if err := validateThreadID(threadID); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}
	if threadID == "" {
		threadID = uuid.New().String()
	}

	now := time.Now().UTC().Unix()
	thread := &types.Thread{
		ID:        threadID,
		AgentName: req.AgentName,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  req.Metadata,
	}

	if err := h.store.Create(thread); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create thread")
		return
	}

	writeJSON(w, http.StatusCreated, threadToResponse(thread))
}

// HandleGet handles GET /api/threads/{thread_id}.
func (h *ThreadHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	threadID := strings.TrimSpace(r.PathValue("thread_id"))
	if err := validateThreadID(threadID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	thread, err := h.store.Get(threadID)
	if err != nil {
		writeError(w, http.StatusNotFound, "thread not found")
		return
	}

	writeJSON(w, http.StatusOK, threadToResponse(thread))
}

// HandleUpdate handles PUT/PATCH /api/threads/{thread_id}.
func (h *ThreadHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	threadID := strings.TrimSpace(r.PathValue("thread_id"))
	if err := validateThreadID(threadID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	var req struct {
		AgentName string            `json:"agent_name,omitempty"`
		Title     string            `json:"title,omitempty"`
		Metadata  map[string]string `json:"metadata,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	thread, err := h.store.Get(threadID)
	if err != nil {
		writeError(w, http.StatusNotFound, "thread not found")
		return
	}

	// Update fields
	if req.AgentName != "" {
		thread.AgentName = req.AgentName
	}
	if req.Title != "" {
		thread.Title = req.Title
	}
	if req.Metadata != nil {
		if thread.Metadata == nil {
			thread.Metadata = make(map[string]string)
		}
		for k, v := range req.Metadata {
			thread.Metadata[k] = v
		}
	}
	thread.UpdatedAt = time.Now().UTC().Unix()

	if err := h.store.Update(thread); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update thread")
		return
	}

	writeJSON(w, http.StatusOK, threadToResponse(thread))
}

// HandleDelete handles DELETE /api/threads/{thread_id}.
func (h *ThreadHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	threadID := strings.TrimSpace(r.PathValue("thread_id"))
	if err := validateThreadID(threadID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	if err := h.store.Delete(threadID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete thread")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// Helper functions

func threadToResponse(thread *types.Thread) ThreadResponse {
	return ThreadResponse{
		ID:        thread.ID,
		AgentName: thread.AgentName,
		Title:     thread.Title,
		CreatedAt: thread.CreatedAt,
		UpdatedAt: thread.UpdatedAt,
		Metadata:  thread.Metadata,
	}
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func validateThreadID(threadID string) error {
	if threadID == "" {
		return fmt.Errorf("thread_id is required")
	}
	// Add more validation as needed (e.g., format check)
	return nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

// Ensure we import fmt for validateThreadID
