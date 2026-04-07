package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/easyspace-ai/minote/pkg/models"
)

// AgentStore defines the interface for agent storage.
type AgentStore interface {
	List() ([]models.GatewayAgent, error)
	Get(name string) (*models.GatewayAgent, error)
	Create(agent *models.GatewayAgent) error
	Update(name string, updates map[string]any) error
	Delete(name string) error
	Exists(name string) (bool, error)
}

// AgentHandler handles agent-related requests.
type AgentHandler struct {
	store AgentStore
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(store AgentStore) *AgentHandler {
	return &AgentHandler{store: store}
}

// AgentListResponse represents the response for listing agents.
type AgentListResponse struct {
	Agents []models.GatewayAgent `json:"agents"`
}

// AgentCheckResponse represents the response for checking agent availability.
type AgentCheckResponse struct {
	Available bool   `json:"available"`
	Name      string `json:"name"`
}

var agentNameRE = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

// HandleList handles GET /api/agents.
func (h *AgentHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	agents, err := h.store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list agents")
		return
	}

	// Sort by name
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})

	// Clear Soul field for list view (as in original implementation)
	for i := range agents {
		agents[i].Soul = ""
	}

	writeJSON(w, http.StatusOK, AgentListResponse{Agents: agents})
}

// HandleCheck handles GET /api/agents/check.
func (h *AgentHandler) HandleCheck(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if !agentNameRE.MatchString(name) {
		writeError(w, http.StatusUnprocessableEntity, "Invalid agent name")
		return
	}

	normalized := strings.ToLower(name)
	exists, err := h.store.Exists(normalized)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check agent")
		return
	}

	writeJSON(w, http.StatusOK, AgentCheckResponse{
		Available: !exists,
		Name:      normalized,
	})
}

// HandleGet handles GET /api/agents/{name}.
func (h *AgentHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	name, ok := normalizeAgentName(r.PathValue("name"))
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "Invalid agent name")
		return
	}

	agent, err := h.store.Get(name)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Agent '%s' not found", name))
		return
	}

	writeJSON(w, http.StatusOK, agent)
}

// HandleCreate handles POST /api/agents.
func (h *AgentHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req models.GatewayAgent
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	name, ok := normalizeAgentName(req.Name)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "Invalid agent name")
		return
	}

	exists, err := h.store.Exists(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check agent existence")
		return
	}
	if exists {
		writeError(w, http.StatusConflict, fmt.Sprintf("Agent '%s' already exists", name))
		return
	}

	// Set defaults
	req.Name = name
	if req.ID == "" {
		req.ID = name
	}
	now := time.Now().UTC()
	req.CreatedAt = now
	req.UpdatedAt = now

	if err := h.store.Create(&req); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create agent")
		return
	}

	writeJSON(w, http.StatusCreated, &req)
}

// HandleUpdate handles PUT /api/agents/{name}.
func (h *AgentHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	name, ok := normalizeAgentName(r.PathValue("name"))
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "Invalid agent name")
		return
	}

	var req struct {
		Description *string   `json:"description"`
		Model       **string  `json:"model"`
		ToolGroups  *[]string `json:"tool_groups"`
		Soul        *string   `json:"soul"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Check if agent exists
	_, err := h.store.Get(name)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Agent '%s' not found", name))
		return
	}

	// Build updates map
	updates := make(map[string]any)
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Model != nil {
		updates["model"] = *req.Model
	}
	if req.ToolGroups != nil {
		updates["tool_groups"] = *req.ToolGroups
	}
	if req.Soul != nil {
		updates["soul"] = *req.Soul
	}
	updates["updated_at"] = time.Now().UTC()

	if err := h.store.Update(name, updates); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update agent")
		return
	}

	// Return updated agent
	agent, _ := h.store.Get(name)
	writeJSON(w, http.StatusOK, agent)
}

// HandleDelete handles DELETE /api/agents/{name}.
func (h *AgentHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	name, ok := normalizeAgentName(r.PathValue("name"))
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "Invalid agent name")
		return
	}

	// Check if agent exists
	_, err := h.store.Get(name)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Agent '%s' not found", name))
		return
	}

	if err := h.store.Delete(name); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete agent")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// normalizeAgentName normalizes and validates an agent name.
func normalizeAgentName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	name = strings.ToLower(name)
	if !agentNameRE.MatchString(name) {
		return "", false
	}
	return name, true
}
