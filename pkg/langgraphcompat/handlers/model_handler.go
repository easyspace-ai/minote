package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/easyspace-ai/minote/pkg/agent"
	"github.com/easyspace-ai/minote/pkg/langgraphcompat/types"
	"gopkg.in/yaml.v3"
)

// ModelHandler handles model-related requests.
type ModelHandler struct {
	defaultModel string
}

// NewModelHandler creates a new ModelHandler.
func NewModelHandler(defaultModel string) *ModelHandler {
	return &ModelHandler{defaultModel: defaultModel}
}

// SetDefaultModel updates the default model.
func (h *ModelHandler) SetDefaultModel(model string) {
	h.defaultModel = model
}

// HandleList handles GET /api/models.
func (h *ModelHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	models := h.configuredModels()
	writeJSON(w, http.StatusOK, map[string]any{"models": models})
}

// HandleGet handles GET /api/models/{model_name...}.
func (h *ModelHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	modelName := strings.TrimSpace(r.PathValue("model_name"))
	model, ok := h.findModel(modelName)
	if modelName == "" || !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"detail": fmt.Sprintf("Model '%s' not found", modelName),
		})
		return
	}
	writeJSON(w, http.StatusOK, model)
}

// configuredModels returns all configured models.
func (h *ModelHandler) configuredModels() []types.GatewayModel {
	if models := h.modelsFromJSON(); len(models) > 0 {
		return models
	}
	if models := h.modelsFromEnvList(); len(models) > 0 {
		return models
	}
	if models := h.modelsFromConfig(); len(models) > 0 {
		return models
	}
	return []types.GatewayModel{h.defaultModelConfig()}
}

// modelsFromJSON loads models from DEERFLOW_MODELS_JSON env var.
func (h *ModelHandler) modelsFromJSON() []types.GatewayModel {
	raw := strings.TrimSpace(os.Getenv("DEERFLOW_MODELS_JSON"))
	if raw == "" {
		return nil
	}

	var parsed []types.RawGatewayModel
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil
	}

	models := make([]types.GatewayModel, 0, len(parsed))
	seen := map[string]struct{}{}
	for _, item := range parsed {
		model := h.normalizeModel(item)
		if model.Name == "" {
			continue
		}
		if _, exists := seen[model.Name]; exists {
			continue
		}
		seen[model.Name] = struct{}{}
		models = append(models, model)
	}
	return models
}

// modelsFromEnvList loads models from DEERFLOW_MODELS env var.
func (h *ModelHandler) modelsFromEnvList() []types.GatewayModel {
	raw := strings.TrimSpace(os.Getenv("DEERFLOW_MODELS"))
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	models := make([]types.GatewayModel, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		entry := strings.TrimSpace(part)
		if entry == "" {
			continue
		}

		name := entry
		modelID := entry
		if left, right, ok := strings.Cut(entry, "="); ok {
			name = strings.TrimSpace(left)
			modelID = strings.TrimSpace(right)
		}

		model := h.normalizeModel(types.RawGatewayModel{
			Name:  name,
			Model: modelID,
		})
		if model.Name == "" {
			continue
		}
		if _, exists := seen[model.Name]; exists {
			continue
		}
		seen[model.Name] = struct{}{}
		models = append(models, model)
	}
	return models
}

// modelsFromConfig loads models from config file.
func (h *ModelHandler) modelsFromConfig() []types.GatewayModel {
	configPath := h.modelCatalogConfigPath()
	if configPath == "" {
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var cfg struct {
		Version string                    `yaml:"version"`
		Models  []types.RawGatewayModel   `yaml:"models"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	models := make([]types.GatewayModel, 0, len(cfg.Models))
	seen := map[string]struct{}{}
	for _, item := range cfg.Models {
		model := h.normalizeModel(item)
		if model.Name == "" {
			continue
		}
		if _, exists := seen[model.Name]; exists {
			continue
		}
		seen[model.Name] = struct{}{}
		if item.MaxTokens > 0 {
			model.MaxTokens = item.MaxTokens
		}
		if item.Temperature != nil {
			model.Temperature = item.Temperature
		}
		models = append(models, model)
	}
	return models
}

// modelCatalogConfigPath returns the path to the model catalog config.
func (h *ModelHandler) modelCatalogConfigPath() string {
	for _, key := range []string{"DEERFLOW_CONFIG_PATH", "DEER_FLOW_CONFIG_PATH"} {
		if path := strings.TrimSpace(os.Getenv(key)); path != "" {
			return path
		}
	}

	const defaultConfigPath = "config.yaml"
	if _, err := os.Stat(defaultConfigPath); err == nil {
		return defaultConfigPath
	}
	return ""
}

// defaultModelConfig returns the default model configuration.
func (h *ModelHandler) defaultModelConfig() types.GatewayModel {
	name := strings.TrimSpace(h.defaultModel)
	if name == "" {
		name = "qwen/Qwen3.5-9B"
	}
	thinking, reasoning := h.inferCapabilities(name)
	return types.GatewayModel{
		ID:                      "default",
		Name:                    name,
		Model:                   name,
		DisplayName:             name,
		Description:             "Default model configured by deerflow-go",
		SupportsThinking:        thinking,
		SupportsReasoningEffort: reasoning,
		SupportsVision:          h.inferVisionSupport(name),
	}
}

// normalizeModel normalizes a raw model configuration.
func (h *ModelHandler) normalizeModel(raw types.RawGatewayModel) types.GatewayModel {
	name := strings.TrimSpace(raw.Name)
	modelID := strings.TrimSpace(raw.Model)
	switch {
	case name == "" && modelID != "":
		name = modelID
	case modelID == "" && name != "":
		modelID = name
	case name == "" && modelID == "":
		fallback := h.defaultModelConfig()
		name = fallback.Name
		modelID = fallback.Model
	}

	id := strings.TrimSpace(raw.ID)
	if id == "" {
		id = name
	}
	displayName := strings.TrimSpace(raw.DisplayName)
	if displayName == "" {
		displayName = name
	}

	thinking, reasoning := h.inferCapabilities(firstNonEmpty(modelID, name))
	if raw.SupportsThinking != nil {
		thinking = *raw.SupportsThinking
	}
	if raw.SupportsReasoningEffort != nil {
		reasoning = *raw.SupportsReasoningEffort
	}
	vision := h.inferVisionSupport(firstNonEmpty(modelID, name))
	if raw.SupportsVision != nil {
		vision = *raw.SupportsVision
	}

	return types.GatewayModel{
		ID:                      id,
		Name:                    name,
		Model:                   modelID,
		DisplayName:             displayName,
		Description:             strings.TrimSpace(raw.Description),
		SupportsThinking:        thinking,
		SupportsReasoningEffort: reasoning,
		SupportsVision:          vision,
	}
}

// inferVisionSupport infers vision support from model name.
func (h *ModelHandler) inferVisionSupport(name string) bool {
	return agent.ModelLikelySupportsVision(name)
}

// inferCapabilities infers thinking/reasoning capabilities from model name.
func (h *ModelHandler) inferCapabilities(name string) (supportsThinking bool, supportsReasoningEffort bool) {
	model := strings.ToLower(strings.TrimSpace(name))
	if model == "" {
		return false, false
	}

	for _, token := range []string{
		"gpt-5", "o1", "o3", "o4", "qwen3", "qwq", "deepseek-r1",
		"gemini-2.5", "claude-3.7", "claude-3-7", "claude-sonnet-4",
		"reasoner", "thinking",
	} {
		if strings.Contains(model, token) {
			return true, true
		}
	}
	return false, false
}

// findModel finds a model by name.
func (h *ModelHandler) findModel(modelName string) (types.GatewayModel, bool) {
	target := strings.TrimSpace(modelName)
	if target == "" {
		return types.GatewayModel{}, false
	}
	for _, model := range h.configuredModels() {
		if model.Name == target {
			return model, true
		}
	}
	return types.GatewayModel{}, false
}

// FindModelByNameOrID finds a model by name or ID (case-insensitive).
func (h *ModelHandler) FindModelByNameOrID(modelName string) (types.GatewayModel, bool) {
	target := strings.TrimSpace(modelName)
	if target == "" {
		return types.GatewayModel{}, false
	}
	for _, model := range h.configuredModels() {
		if strings.EqualFold(model.Name, target) || strings.EqualFold(model.Model, target) {
			return model, true
		}
	}
	return types.GatewayModel{}, false
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
