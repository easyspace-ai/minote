package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/easyspace-ai/minote/pkg/langgraphcompat/types"
)

// MemoryStore defines the interface for memory storage.
type MemoryStore interface {
	Get() (*types.MemoryResponse, error)
	Put(memory *types.MemoryResponse) error
	Clear() error
	DeleteFact(factID string) error
	Reload() error
}

// MemoryHandler handles memory-related requests.
type MemoryHandler struct {
	store MemoryStore
}

// NewMemoryHandler creates a new MemoryHandler.
func NewMemoryHandler(store MemoryStore) *MemoryHandler {
	return &MemoryHandler{store: store}
}

// HandleGet handles GET /api/memory.
func (h *MemoryHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	memory, err := h.store.Get()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get memory")
		return
	}

	writeJSON(w, http.StatusOK, memory)
}

// HandlePut handles PUT /api/memory.
func (h *MemoryHandler) HandlePut(w http.ResponseWriter, r *http.Request) {
	var req types.MemoryResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Update timestamp
	req.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	if req.Version == "" {
		req.Version = "1.0"
	}

	if err := h.store.Put(&req); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update memory")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// HandleReload handles POST /api/memory/reload.
func (h *MemoryHandler) HandleReload(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Reload(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reload memory")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// HandleClear handles DELETE /api/memory.
func (h *MemoryHandler) HandleClear(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Clear(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to clear memory")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// HandleFactDelete handles DELETE /api/memory/facts/{fact_id}.
func (h *MemoryHandler) HandleFactDelete(w http.ResponseWriter, r *http.Request) {
	factID := r.PathValue("fact_id")
	if factID == "" {
		writeError(w, http.StatusBadRequest, "fact_id is required")
		return
	}

	if err := h.store.DeleteFact(factID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete fact")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// HandleConfigGet handles GET /api/memory/config.
func (h *MemoryHandler) HandleConfigGet(w http.ResponseWriter, r *http.Request) {
	// Return default memory configuration
	config := map[string]any{
		"enabled":             true,
		"auto_update":         true,
		"max_facts":           1000,
		"summary_interval":    "1h",
	}
	writeJSON(w, http.StatusOK, config)
}

// HandleStatusGet handles GET /api/memory/status.
func (h *MemoryHandler) HandleStatusGet(w http.ResponseWriter, r *http.Request) {
	memory, err := h.store.Get()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "error",
			"error":   err.Error(),
			"enabled": false,
		})
		return
	}

	status := map[string]any{
		"status":       "ok",
		"enabled":      true,
		"version":      memory.Version,
		"last_updated": memory.LastUpdated,
		"facts_count":  len(memory.Facts),
	}
	writeJSON(w, http.StatusOK, status)
}
