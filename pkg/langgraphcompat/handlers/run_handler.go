package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/easyspace-ai/minote/pkg/langgraphcompat/transform"
)

// RunInfo represents a simplified run for API responses.
type RunInfo struct {
	RunID       string    `json:"run_id"`
	ThreadID    string    `json:"thread_id"`
	AssistantID string    `json:"assistant_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Error       string    `json:"error,omitempty"`
}

// RunStore defines the interface for run storage and execution.
type RunStore interface {
	// CRUD operations
	List(threadID string) ([]RunInfo, error)
	Get(runID string) (*RunInfo, error)
	GetByThreadAndRun(threadID, runID string) (*RunInfo, error)

	// Lifecycle operations
	Create(req RunCreateRequest) (*RunInfo, error)
	Cancel(runID string) (*RunInfo, error)
	CancelByThread(threadID, runID string) (*RunInfo, error)

	// Streaming support
	Subscribe(runID string) (*RunInfo, chan transform.StreamEvent, error)
	Unsubscribe(runID string, ch chan transform.StreamEvent)
}

// RunCreateRequest represents a request to create a run.
type RunCreateRequest struct {
	AssistantID      string         `json:"assistant_id"`
	ThreadID         string         `json:"thread_id,omitempty"`
	Input            map[string]any `json:"input,omitempty"`
	Config           map[string]any `json:"config,omitempty"`
	Context          map[string]any `json:"context,omitempty"`
	AutoAcceptedPlan *bool          `json:"auto_accepted_plan,omitempty"`
	Feedback         string         `json:"feedback,omitempty"`
}

// RunHandler handles run-related requests.
type RunHandler struct {
	store RunStore
}

// NewRunHandler creates a new RunHandler.
func NewRunHandler(store RunStore) *RunHandler {
	return &RunHandler{store: store}
}

// RunListResponse represents the response for listing runs.
type RunListResponse struct {
	Runs []RunInfo `json:"runs"`
}

// HandleList handles GET /api/threads/{thread_id}/runs.
func (h *RunHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	threadID := r.PathValue("thread_id")
	if err := validateThreadID(threadID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	runs, err := h.store.List(threadID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}

	writeJSON(w, http.StatusOK, RunListResponse{Runs: runs})
}

// HandleGet handles GET /api/threads/{thread_id}/runs/{run_id}.
func (h *RunHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	threadID := r.PathValue("thread_id")
	if err := validateThreadID(threadID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	runID := r.PathValue("run_id")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "run_id is required")
		return
	}

	run, err := h.store.GetByThreadAndRun(threadID, runID)
	if err != nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	writeJSON(w, http.StatusOK, run)
}

// HandleGetByID handles GET /api/runs/{run_id} (without thread_id).
func (h *RunHandler) HandleGetByID(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("run_id")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "run_id is required")
		return
	}

	run, err := h.store.Get(runID)
	if err != nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	writeJSON(w, http.StatusOK, run)
}

// HandleCreate handles POST /api/threads/{thread_id}/runs.
func (h *RunHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	threadID := r.PathValue("thread_id")
	if err := validateThreadID(threadID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req RunCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Use thread_id from path if not provided in body
	if req.ThreadID == "" {
		req.ThreadID = threadID
	}

	run, err := h.store.Create(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create run")
		return
	}

	w.Header().Set("Content-Location", fmt.Sprintf("/threads/%s/runs/%s", run.ThreadID, run.RunID))
	writeJSON(w, http.StatusOK, run)
}

// HandleCancel handles POST /api/threads/{thread_id}/runs/{run_id}/cancel.
func (h *RunHandler) HandleCancel(w http.ResponseWriter, r *http.Request) {
	threadID := r.PathValue("thread_id")
	if err := validateThreadID(threadID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	runID := r.PathValue("run_id")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "run_id is required")
		return
	}

	run, err := h.store.CancelByThread(threadID, runID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "run not found" {
			status = http.StatusNotFound
		} else if err.Error() == "run is not active" {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, run)
}

// HandleCancelByID handles POST /api/runs/{run_id}/cancel (without thread_id).
func (h *RunHandler) HandleCancelByID(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("run_id")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "run_id is required")
		return
	}

	run, err := h.store.Cancel(runID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "run not found" {
			status = http.StatusNotFound
		} else if err.Error() == "run is not active" {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, run)
}

// HandleStream handles POST /api/threads/{thread_id}/runs/stream.
// Note: This is a simplified version. Full streaming implementation
// requires integration with the agent execution system.
func (h *RunHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	// Validate thread
	threadID := r.PathValue("thread_id")
	if err := validateThreadID(threadID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Parse request
	var req RunCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Use thread_id from path
	if req.ThreadID == "" {
		req.ThreadID = threadID
	}

	// Create run
	run, err := h.store.Create(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create run")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Content-Location", fmt.Sprintf("/threads/%s/runs/%s", run.ThreadID, run.RunID))

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Subscribe to run events
	_, stream, err := h.store.Subscribe(run.RunID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to subscribe to run")
		return
	}
	defer h.store.Unsubscribe(run.RunID, stream)

	// Stream events
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-stream:
			if !ok {
				return
			}
			sendSSEEvent(w, flusher, event)
			if event.Event == "end" || event.Event == "error" {
				return
			}
		}
	}
}

// HandleRunStream handles GET /api/threads/{thread_id}/runs/{run_id}/stream.
func (h *RunHandler) HandleRunStream(w http.ResponseWriter, r *http.Request) {
	threadID := r.PathValue("thread_id")
	if err := validateThreadID(threadID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	runID := r.PathValue("run_id")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "run_id is required")
		return
	}

	// Verify run exists and belongs to thread
	run, err := h.store.GetByThreadAndRun(threadID, runID)
	if err != nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Content-Location", fmt.Sprintf("/threads/%s/runs/%s", run.ThreadID, run.RunID))

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Subscribe to run events
	_, stream, err := h.store.Subscribe(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to subscribe to run")
		return
	}
	defer h.store.Unsubscribe(runID, stream)

	// Stream events
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-stream:
			if !ok {
				return
			}
			sendSSEEvent(w, flusher, event)
			if event.Event == "end" || event.Event == "error" {
				return
			}
		}
	}
}

// sendSSEEvent sends a Server-Sent Event.
func sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, event transform.StreamEvent) {
	data, err := json.Marshal(event.Data)
	if err != nil {
		return
	}

	if event.ID != "" {
		fmt.Fprintf(w, "id: %s\n", event.ID)
	}
	fmt.Fprintf(w, "event: %s\n", event.Event)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}
