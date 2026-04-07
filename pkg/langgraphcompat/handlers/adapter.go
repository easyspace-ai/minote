package handlers

import (
	"net/http"

	"github.com/easyspace-ai/minote/pkg/langgraphcompat/types"
)

// Adapter provides helper methods to integrate handlers with the existing Server.
// It wraps all individual handlers and provides a unified interface for registration.
type Adapter struct {
	modelHandler  *ModelHandler
	threadHandler *ThreadHandler
	memoryHandler *MemoryHandler
	agentHandler  *AgentHandler
	runHandler    *RunHandler
}

// NewAdapter creates a new handler adapter with all handlers initialized.
func NewAdapter(defaultModel string) *Adapter {
	return &Adapter{
		modelHandler: NewModelHandler(defaultModel),
	}
}

// NewAdapterWithStores creates a new adapter with custom stores.
// This allows for dependency injection of different storage implementations.
func NewAdapterWithStores(defaultModel string, stores HandlerStores) *Adapter {
	a := &Adapter{
		modelHandler: NewModelHandler(defaultModel),
	}

	if stores.ThreadStore != nil {
		a.threadHandler = NewThreadHandler(stores.ThreadStore)
	}
	if stores.MemoryStore != nil {
		a.memoryHandler = NewMemoryHandler(stores.MemoryStore)
	}
	if stores.AgentStore != nil {
		a.agentHandler = NewAgentHandler(stores.AgentStore)
	}
	if stores.RunStore != nil {
		a.runHandler = NewRunHandler(stores.RunStore)
	}

	return a
}

// HandlerStores holds all store implementations for injection.
type HandlerStores struct {
	ThreadStore ThreadStore
	MemoryStore MemoryStore
	AgentStore  AgentStore
	RunStore    RunStore
}

// SetDefaultModel updates the default model for all handlers.
func (a *Adapter) SetDefaultModel(model string) {
	a.modelHandler.SetDefaultModel(model)
}

// ModelHandler returns the model handler.
func (a *Adapter) ModelHandler() *ModelHandler {
	return a.modelHandler
}

// ThreadHandler returns the thread handler.
func (a *Adapter) ThreadHandler() *ThreadHandler {
	return a.threadHandler
}

// MemoryHandler returns the memory handler.
func (a *Adapter) MemoryHandler() *MemoryHandler {
	return a.memoryHandler
}

// AgentHandler returns the agent handler.
func (a *Adapter) AgentHandler() *AgentHandler {
	return a.agentHandler
}

// RunHandler returns the run handler.
func (a *Adapter) RunHandler() *RunHandler {
	return a.runHandler
}

// SetThreadHandler sets the thread handler.
func (a *Adapter) SetThreadHandler(store ThreadStore) {
	a.threadHandler = NewThreadHandler(store)
}

// SetMemoryHandler sets the memory handler.
func (a *Adapter) SetMemoryHandler(store MemoryStore) {
	a.memoryHandler = NewMemoryHandler(store)
}

// SetAgentHandler sets the agent handler.
func (a *Adapter) SetAgentHandler(store AgentStore) {
	a.agentHandler = NewAgentHandler(store)
}

// SetRunHandler sets the run handler.
func (a *Adapter) SetRunHandler(store RunStore) {
	a.runHandler = NewRunHandler(store)
}

// ==================== Model Routes ====================

// HandleModelsList wraps ModelHandler.HandleList for compatibility.
func (a *Adapter) HandleModelsList(w http.ResponseWriter, r *http.Request) {
	a.modelHandler.HandleList(w, r)
}

// HandleModelGet wraps ModelHandler.HandleGet for compatibility.
func (a *Adapter) HandleModelGet(w http.ResponseWriter, r *http.Request) {
	a.modelHandler.HandleGet(w, r)
}

// FindModelByNameOrID is a convenience method to find a model.
func (a *Adapter) FindModelByNameOrID(modelName string) (types.GatewayModel, bool) {
	return a.modelHandler.FindModelByNameOrID(modelName)
}

// ==================== Thread Routes ====================

// HandleThreadsList wraps ThreadHandler.HandleList.
func (a *Adapter) HandleThreadsList(w http.ResponseWriter, r *http.Request) {
	if a.threadHandler == nil {
		writeError(w, http.StatusNotImplemented, "thread handler not initialized")
		return
	}
	a.threadHandler.HandleList(w, r)
}

// HandleThreadSearch wraps ThreadHandler.HandleSearch.
func (a *Adapter) HandleThreadSearch(w http.ResponseWriter, r *http.Request) {
	if a.threadHandler == nil {
		writeError(w, http.StatusNotImplemented, "thread handler not initialized")
		return
	}
	a.threadHandler.HandleSearch(w, r)
}

// HandleThreadCreate wraps ThreadHandler.HandleCreate.
func (a *Adapter) HandleThreadCreate(w http.ResponseWriter, r *http.Request) {
	if a.threadHandler == nil {
		writeError(w, http.StatusNotImplemented, "thread handler not initialized")
		return
	}
	a.threadHandler.HandleCreate(w, r)
}

// HandleThreadGet wraps ThreadHandler.HandleGet.
func (a *Adapter) HandleThreadGet(w http.ResponseWriter, r *http.Request) {
	if a.threadHandler == nil {
		writeError(w, http.StatusNotImplemented, "thread handler not initialized")
		return
	}
	a.threadHandler.HandleGet(w, r)
}

// HandleThreadUpdate wraps ThreadHandler.HandleUpdate.
func (a *Adapter) HandleThreadUpdate(w http.ResponseWriter, r *http.Request) {
	if a.threadHandler == nil {
		writeError(w, http.StatusNotImplemented, "thread handler not initialized")
		return
	}
	a.threadHandler.HandleUpdate(w, r)
}

// HandleThreadDelete wraps ThreadHandler.HandleDelete.
func (a *Adapter) HandleThreadDelete(w http.ResponseWriter, r *http.Request) {
	if a.threadHandler == nil {
		writeError(w, http.StatusNotImplemented, "thread handler not initialized")
		return
	}
	a.threadHandler.HandleDelete(w, r)
}

// ==================== Memory Routes ====================

// HandleMemoryGet wraps MemoryHandler.HandleGet.
func (a *Adapter) HandleMemoryGet(w http.ResponseWriter, r *http.Request) {
	if a.memoryHandler == nil {
		writeError(w, http.StatusNotImplemented, "memory handler not initialized")
		return
	}
	a.memoryHandler.HandleGet(w, r)
}

// HandleMemoryPut wraps MemoryHandler.HandlePut.
func (a *Adapter) HandleMemoryPut(w http.ResponseWriter, r *http.Request) {
	if a.memoryHandler == nil {
		writeError(w, http.StatusNotImplemented, "memory handler not initialized")
		return
	}
	a.memoryHandler.HandlePut(w, r)
}

// HandleMemoryClear wraps MemoryHandler.HandleClear.
func (a *Adapter) HandleMemoryClear(w http.ResponseWriter, r *http.Request) {
	if a.memoryHandler == nil {
		writeError(w, http.StatusNotImplemented, "memory handler not initialized")
		return
	}
	a.memoryHandler.HandleClear(w, r)
}

// HandleMemoryFactDelete wraps MemoryHandler.HandleFactDelete.
func (a *Adapter) HandleMemoryFactDelete(w http.ResponseWriter, r *http.Request) {
	if a.memoryHandler == nil {
		writeError(w, http.StatusNotImplemented, "memory handler not initialized")
		return
	}
	a.memoryHandler.HandleFactDelete(w, r)
}

// HandleMemoryConfigGet wraps MemoryHandler.HandleConfigGet.
func (a *Adapter) HandleMemoryConfigGet(w http.ResponseWriter, r *http.Request) {
	if a.memoryHandler == nil {
		writeError(w, http.StatusNotImplemented, "memory handler not initialized")
		return
	}
	a.memoryHandler.HandleConfigGet(w, r)
}

// HandleMemoryStatusGet wraps MemoryHandler.HandleStatusGet.
func (a *Adapter) HandleMemoryStatusGet(w http.ResponseWriter, r *http.Request) {
	if a.memoryHandler == nil {
		writeError(w, http.StatusNotImplemented, "memory handler not initialized")
		return
	}
	a.memoryHandler.HandleStatusGet(w, r)
}

// HandleMemoryReload wraps MemoryHandler.HandleReload.
func (a *Adapter) HandleMemoryReload(w http.ResponseWriter, r *http.Request) {
	if a.memoryHandler == nil {
		writeError(w, http.StatusNotImplemented, "memory handler not initialized")
		return
	}
	a.memoryHandler.HandleReload(w, r)
}

// ==================== Agent Routes ====================

// HandleAgentsList wraps AgentHandler.HandleList.
func (a *Adapter) HandleAgentsList(w http.ResponseWriter, r *http.Request) {
	if a.agentHandler == nil {
		writeError(w, http.StatusNotImplemented, "agent handler not initialized")
		return
	}
	a.agentHandler.HandleList(w, r)
}

// HandleAgentCheck wraps AgentHandler.HandleCheck.
func (a *Adapter) HandleAgentCheck(w http.ResponseWriter, r *http.Request) {
	if a.agentHandler == nil {
		writeError(w, http.StatusNotImplemented, "agent handler not initialized")
		return
	}
	a.agentHandler.HandleCheck(w, r)
}

// HandleAgentGet wraps AgentHandler.HandleGet.
func (a *Adapter) HandleAgentGet(w http.ResponseWriter, r *http.Request) {
	if a.agentHandler == nil {
		writeError(w, http.StatusNotImplemented, "agent handler not initialized")
		return
	}
	a.agentHandler.HandleGet(w, r)
}

// HandleAgentCreate wraps AgentHandler.HandleCreate.
func (a *Adapter) HandleAgentCreate(w http.ResponseWriter, r *http.Request) {
	if a.agentHandler == nil {
		writeError(w, http.StatusNotImplemented, "agent handler not initialized")
		return
	}
	a.agentHandler.HandleCreate(w, r)
}

// HandleAgentUpdate wraps AgentHandler.HandleUpdate.
func (a *Adapter) HandleAgentUpdate(w http.ResponseWriter, r *http.Request) {
	if a.agentHandler == nil {
		writeError(w, http.StatusNotImplemented, "agent handler not initialized")
		return
	}
	a.agentHandler.HandleUpdate(w, r)
}

// HandleAgentDelete wraps AgentHandler.HandleDelete.
func (a *Adapter) HandleAgentDelete(w http.ResponseWriter, r *http.Request) {
	if a.agentHandler == nil {
		writeError(w, http.StatusNotImplemented, "agent handler not initialized")
		return
	}
	a.agentHandler.HandleDelete(w, r)
}

// ==================== Run Routes ====================

// HandleRunsList wraps RunHandler.HandleList.
func (a *Adapter) HandleRunsList(w http.ResponseWriter, r *http.Request) {
	if a.runHandler == nil {
		writeError(w, http.StatusNotImplemented, "run handler not initialized")
		return
	}
	a.runHandler.HandleList(w, r)
}

// HandleRunGet wraps RunHandler.HandleGet.
func (a *Adapter) HandleRunGet(w http.ResponseWriter, r *http.Request) {
	if a.runHandler == nil {
		writeError(w, http.StatusNotImplemented, "run handler not initialized")
		return
	}
	a.runHandler.HandleGet(w, r)
}

// HandleRunGetByID wraps RunHandler.HandleGetByID.
func (a *Adapter) HandleRunGetByID(w http.ResponseWriter, r *http.Request) {
	if a.runHandler == nil {
		writeError(w, http.StatusNotImplemented, "run handler not initialized")
		return
	}
	a.runHandler.HandleGetByID(w, r)
}

// HandleRunsCreate wraps RunHandler.HandleCreate.
func (a *Adapter) HandleRunsCreate(w http.ResponseWriter, r *http.Request) {
	if a.runHandler == nil {
		writeError(w, http.StatusNotImplemented, "run handler not initialized")
		return
	}
	a.runHandler.HandleCreate(w, r)
}

// HandleRunCancel wraps RunHandler.HandleCancel.
func (a *Adapter) HandleRunCancel(w http.ResponseWriter, r *http.Request) {
	if a.runHandler == nil {
		writeError(w, http.StatusNotImplemented, "run handler not initialized")
		return
	}
	a.runHandler.HandleCancel(w, r)
}

// HandleRunCancelByID wraps RunHandler.HandleCancelByID.
func (a *Adapter) HandleRunCancelByID(w http.ResponseWriter, r *http.Request) {
	if a.runHandler == nil {
		writeError(w, http.StatusNotImplemented, "run handler not initialized")
		return
	}
	a.runHandler.HandleCancelByID(w, r)
}

// HandleRunsStream wraps RunHandler.HandleStream.
func (a *Adapter) HandleRunsStream(w http.ResponseWriter, r *http.Request) {
	if a.runHandler == nil {
		writeError(w, http.StatusNotImplemented, "run handler not initialized")
		return
	}
	a.runHandler.HandleStream(w, r)
}

// HandleRunStream wraps RunHandler.HandleRunStream.
func (a *Adapter) HandleRunStream(w http.ResponseWriter, r *http.Request) {
	if a.runHandler == nil {
		writeError(w, http.StatusNotImplemented, "run handler not initialized")
		return
	}
	a.runHandler.HandleRunStream(w, r)
}

// ==================== Route Registration ====================

// RegisterMigrationRoutes registers new handlers alongside legacy ones.
// This allows A/B testing or gradual migration.
// useNewHandlers map controls which handlers to use (key: "models", "threads", "memory", "agents", "runs")
func (a *Adapter) RegisterMigrationRoutes(mux *http.ServeMux, useNewHandlers map[string]bool) {
	// Models - new handlers
	if useNewHandlers["models"] {
		mux.HandleFunc("GET /api/models", a.HandleModelsList)
		mux.HandleFunc("GET /api/models/{model_name...}", a.HandleModelGet)
	}

	// Threads - new handlers
	if useNewHandlers["threads"] {
		mux.HandleFunc("GET /api/threads", a.HandleThreadsList)
		mux.HandleFunc("POST /api/threads/search", a.HandleThreadSearch)
		mux.HandleFunc("POST /api/threads", a.HandleThreadCreate)
		mux.HandleFunc("GET /api/threads/{thread_id}", a.HandleThreadGet)
		mux.HandleFunc("PUT /api/threads/{thread_id}", a.HandleThreadUpdate)
		mux.HandleFunc("DELETE /api/threads/{thread_id}", a.HandleThreadDelete)
	}

	// Memory - new handlers
	if useNewHandlers["memory"] {
		mux.HandleFunc("GET /api/memory", a.HandleMemoryGet)
		mux.HandleFunc("PUT /api/memory", a.HandleMemoryPut)
		mux.HandleFunc("DELETE /api/memory", a.HandleMemoryClear)
		mux.HandleFunc("DELETE /api/memory/facts/{fact_id}", a.HandleMemoryFactDelete)
		mux.HandleFunc("GET /api/memory/config", a.HandleMemoryConfigGet)
		mux.HandleFunc("GET /api/memory/status", a.HandleMemoryStatusGet)
		mux.HandleFunc("POST /api/memory/reload", a.HandleMemoryReload)
	}

	// Agents - new handlers
	if useNewHandlers["agents"] {
		mux.HandleFunc("GET /api/agents", a.HandleAgentsList)
		mux.HandleFunc("GET /api/agents/check", a.HandleAgentCheck)
		mux.HandleFunc("GET /api/agents/{name}", a.HandleAgentGet)
		mux.HandleFunc("POST /api/agents", a.HandleAgentCreate)
		mux.HandleFunc("PUT /api/agents/{name}", a.HandleAgentUpdate)
		mux.HandleFunc("DELETE /api/agents/{name}", a.HandleAgentDelete)
	}

	// Runs - new handlers
	if useNewHandlers["runs"] {
		mux.HandleFunc("GET /api/threads/{thread_id}/runs", a.HandleRunsList)
		mux.HandleFunc("POST /api/threads/{thread_id}/runs", a.HandleRunsCreate)
		mux.HandleFunc("GET /api/threads/{thread_id}/runs/{run_id}", a.HandleRunGet)
		mux.HandleFunc("POST /api/threads/{thread_id}/runs/{run_id}/cancel", a.HandleRunCancel)
		mux.HandleFunc("POST /api/threads/{thread_id}/runs/stream", a.HandleRunsStream)
		mux.HandleFunc("GET /api/threads/{thread_id}/runs/{run_id}/stream", a.HandleRunStream)
	}
}

// RegisterAllRoutes registers all handler routes with the provided mux.
// This is a convenience method for quick setup.
func (a *Adapter) RegisterAllRoutes(mux *http.ServeMux) {
	// Models
	mux.HandleFunc("GET /api/models", a.HandleModelsList)
	mux.HandleFunc("GET /api/models/{model_name...}", a.HandleModelGet)

	// Threads
	mux.HandleFunc("GET /api/threads", a.HandleThreadsList)
	mux.HandleFunc("POST /api/threads/search", a.HandleThreadSearch)
	mux.HandleFunc("POST /api/threads", a.HandleThreadCreate)
	mux.HandleFunc("GET /api/threads/{thread_id}", a.HandleThreadGet)
	mux.HandleFunc("PUT /api/threads/{thread_id}", a.HandleThreadUpdate)
	mux.HandleFunc("DELETE /api/threads/{thread_id}", a.HandleThreadDelete)

	// Memory
	mux.HandleFunc("GET /api/memory", a.HandleMemoryGet)
	mux.HandleFunc("PUT /api/memory", a.HandleMemoryPut)
	mux.HandleFunc("DELETE /api/memory", a.HandleMemoryClear)
	mux.HandleFunc("DELETE /api/memory/facts/{fact_id}", a.HandleMemoryFactDelete)
	mux.HandleFunc("GET /api/memory/config", a.HandleMemoryConfigGet)
	mux.HandleFunc("GET /api/memory/status", a.HandleMemoryStatusGet)
	mux.HandleFunc("POST /api/memory/reload", a.HandleMemoryReload)

	// Agents
	mux.HandleFunc("GET /api/agents", a.HandleAgentsList)
	mux.HandleFunc("GET /api/agents/check", a.HandleAgentCheck)
	mux.HandleFunc("GET /api/agents/{name}", a.HandleAgentGet)
	mux.HandleFunc("POST /api/agents", a.HandleAgentCreate)
	mux.HandleFunc("PUT /api/agents/{name}", a.HandleAgentUpdate)
	mux.HandleFunc("DELETE /api/agents/{name}", a.HandleAgentDelete)

	// Runs (thread-scoped)
	mux.HandleFunc("GET /api/threads/{thread_id}/runs", a.HandleRunsList)
	mux.HandleFunc("POST /api/threads/{thread_id}/runs", a.HandleRunsCreate)
	mux.HandleFunc("GET /api/threads/{thread_id}/runs/{run_id}", a.HandleRunGet)
	mux.HandleFunc("POST /api/threads/{thread_id}/runs/{run_id}/cancel", a.HandleRunCancel)
	mux.HandleFunc("POST /api/threads/{thread_id}/runs/stream", a.HandleRunsStream)
	mux.HandleFunc("GET /api/threads/{thread_id}/runs/{run_id}/stream", a.HandleRunStream)
}
