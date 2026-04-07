package handlers

import (
	"net/http"
)

// ServerAdapter wraps the new handlers and provides integration with the existing
// langgraphcompat.Server. This allows gradual migration from legacy handlers.
type ServerAdapter struct {
	adapter *Adapter
}

// NewServerAdapter creates a new adapter that bridges legacy Server and new handlers.
func NewServerAdapter(defaultModel string) *ServerAdapter {
	return &ServerAdapter{
		adapter: NewAdapter(defaultModel),
	}
}

// SetThreadStore sets the thread store for the adapter.
func (s *ServerAdapter) SetThreadStore(store ThreadStore) {
	s.adapter.SetThreadHandler(store)
}

// SetMemoryStore sets the memory store for the adapter.
func (s *ServerAdapter) SetMemoryStore(store MemoryStore) {
	s.adapter.SetMemoryHandler(store)
}

// SetAgentStore sets the agent store for the adapter.
func (s *ServerAdapter) SetAgentStore(store AgentStore) {
	s.adapter.SetAgentHandler(store)
}

// SetRunStore sets the run store for the adapter.
func (s *ServerAdapter) SetRunStore(store RunStore) {
	s.adapter.SetRunHandler(store)
}

// RegisterMigrationRoutes registers new handlers alongside legacy ones.
// This allows A/B testing or gradual migration.
func (s *ServerAdapter) RegisterMigrationRoutes(mux *http.ServeMux, useNewHandlers map[string]bool) {
	// Models - new handlers
	if useNewHandlers["models"] {
		mux.HandleFunc("GET /api/models", s.adapter.HandleModelsList)
		mux.HandleFunc("GET /api/models/{model_name...}", s.adapter.HandleModelGet)
	}

	// Threads - new handlers
	if useNewHandlers["threads"] {
		mux.HandleFunc("GET /api/threads", s.adapter.HandleThreadsList)
		mux.HandleFunc("POST /api/threads/search", s.adapter.HandleThreadSearch)
		mux.HandleFunc("POST /api/threads", s.adapter.HandleThreadCreate)
		mux.HandleFunc("GET /api/threads/{thread_id}", s.adapter.HandleThreadGet)
		mux.HandleFunc("PUT /api/threads/{thread_id}", s.adapter.HandleThreadUpdate)
		mux.HandleFunc("DELETE /api/threads/{thread_id}", s.adapter.HandleThreadDelete)
	}

	// Memory - new handlers
	if useNewHandlers["memory"] {
		mux.HandleFunc("GET /api/memory", s.adapter.HandleMemoryGet)
		mux.HandleFunc("PUT /api/memory", s.adapter.HandleMemoryPut)
		mux.HandleFunc("DELETE /api/memory", s.adapter.HandleMemoryClear)
		mux.HandleFunc("DELETE /api/memory/facts/{fact_id}", s.adapter.HandleMemoryFactDelete)
		mux.HandleFunc("GET /api/memory/config", s.adapter.HandleMemoryConfigGet)
		mux.HandleFunc("GET /api/memory/status", s.adapter.HandleMemoryStatusGet)
		mux.HandleFunc("POST /api/memory/reload", s.adapter.HandleMemoryReload)
	}

	// Agents - new handlers
	if useNewHandlers["agents"] {
		mux.HandleFunc("GET /api/agents", s.adapter.HandleAgentsList)
		mux.HandleFunc("GET /api/agents/check", s.adapter.HandleAgentCheck)
		mux.HandleFunc("GET /api/agents/{name}", s.adapter.HandleAgentGet)
		mux.HandleFunc("POST /api/agents", s.adapter.HandleAgentCreate)
		mux.HandleFunc("PUT /api/agents/{name}", s.adapter.HandleAgentUpdate)
		mux.HandleFunc("DELETE /api/agents/{name}", s.adapter.HandleAgentDelete)
	}

	// Runs - new handlers
	if useNewHandlers["runs"] {
		mux.HandleFunc("GET /api/threads/{thread_id}/runs", s.adapter.HandleRunsList)
		mux.HandleFunc("POST /api/threads/{thread_id}/runs", s.adapter.HandleRunsCreate)
		mux.HandleFunc("GET /api/threads/{thread_id}/runs/{run_id}", s.adapter.HandleRunGet)
		mux.HandleFunc("POST /api/threads/{thread_id}/runs/{run_id}/cancel", s.adapter.HandleRunCancel)
		mux.HandleFunc("POST /api/threads/{thread_id}/runs/stream", s.adapter.HandleRunsStream)
		mux.HandleFunc("GET /api/threads/{thread_id}/runs/{run_id}/stream", s.adapter.HandleRunStream)
	}
}

// RegisterAllNewRoutes registers all new handlers (full replacement mode).
func (s *ServerAdapter) RegisterAllNewRoutes(mux *http.ServeMux) {
	s.adapter.RegisterAllRoutes(mux)
}

// GetAdapter returns the underlying Adapter for direct access.
func (s *ServerAdapter) GetAdapter() *Adapter {
	return s.adapter
}

// LegacyHandler wraps a legacy handler function for compatibility.
func LegacyHandler(fn http.HandlerFunc) http.HandlerFunc {
	return fn
}
