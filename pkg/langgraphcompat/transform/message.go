package transform

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/easyspace-ai/minote/pkg/models"
)

// LangChainToInternalRole converts LangChain role to internal role.
func LangChainToInternalRole(langchainRole string) models.Role {
	switch strings.ToLower(langchainRole) {
	case "human", "user":
		return models.RoleHuman
	case "ai", "assistant":
		return models.RoleAI
	case "system":
		return models.RoleSystem
	case "tool":
		return models.RoleTool
	default:
		return models.RoleHuman
	}
}

// InternalToLangChainRole converts internal role to LangChain role.
func InternalToLangChainRole(role models.Role) string {
	switch role {
	case models.RoleHuman:
		return "human"
	case models.RoleAI:
		return "ai"
	case models.RoleSystem:
		return "system"
	case models.RoleTool:
		return "tool"
	default:
		return "human"
	}
}

// ExtractMessageContent extracts text content from various message formats.
func ExtractMessageContent(raw any) (string, []map[string]any) {
	switch v := raw.(type) {
	case string:
		return v, nil
	case []any:
		parts := make([]string, 0, len(v))
		multiContent := make([]map[string]any, 0, len(v))
		for _, item := range v {
			part, ok := item.(map[string]any)
			if !ok {
				continue
			}
			partType, _ := part["type"].(string)
			switch partType {
			case "text":
				if text, _ := part["text"].(string); text != "" {
					parts = append(parts, text)
					multiContent = append(multiContent, map[string]any{
						"type": "text",
						"text": text,
					})
				}
			case "image_url":
				imageURL, _ := part["image_url"].(map[string]any)
				url := stringFromAny(imageURL["url"])
				if strings.TrimSpace(url) == "" {
					continue
				}
				multiContent = append(multiContent, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": url,
					},
				})
			}
		}
		return strings.Join(parts, "\n"), multiContent
	default:
		return "", nil
	}
}

// BuildMessageMetadata builds metadata map from additional_kwargs and multi_content.
func BuildMessageMetadata(additionalKwargs map[string]any, multiContent []map[string]any) map[string]string {
	metadata := make(map[string]string)

	if len(additionalKwargs) > 0 {
		if raw, err := json.Marshal(additionalKwargs); err == nil {
			metadata["additional_kwargs"] = string(raw)
		}
	}

	if len(multiContent) > 0 {
		if raw, err := json.Marshal(multiContent); err == nil {
			metadata["multi_content"] = string(raw)
		}
	}

	return metadata
}

// DecodeMultiContent decodes multi_content from metadata.
func DecodeMultiContent(metadata map[string]string) []map[string]any {
	raw, ok := metadata["multi_content"]
	if !ok || raw == "" {
		return nil
	}

	var result []map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil
	}
	return result
}

// DecodeAdditionalKwargs decodes additional_kwargs from metadata.
func DecodeAdditionalKwargs(metadata map[string]string) map[string]any {
	raw, ok := metadata["additional_kwargs"]
	if !ok || raw == "" {
		return nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil
	}
	return result
}

// stringFromAny converts any value to string.
func stringFromAny(v any) string {
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

// StreamEvent represents an SSE event for LangGraph streaming.
type StreamEvent struct {
	ID       string `json:"id,omitempty"`
	Event    string `json:"event"`
	Data     any    `json:"data"`
	RunID    string `json:"run_id,omitempty"`
	ThreadID string `json:"thread_id,omitempty"`
}

// FormatSSE formats an event for Server-Sent Events.
func FormatSSE(event StreamEvent) string {
	var parts []string
	if event.ID != "" {
		parts = append(parts, fmt.Sprintf("id: %s", event.ID))
	}
	parts = append(parts, fmt.Sprintf("event: %s", event.Event))

	data, _ := json.Marshal(event.Data)
	parts = append(parts, fmt.Sprintf("data: %s", data))

	return strings.Join(parts, "\n") + "\n\n"
}

// MessageForAPI converts internal message to API format.
type MessageForAPI struct {
	Type             string         `json:"type"`
	ID               string         `json:"id"`
	Role             string         `json:"role"`
	Content          string         `json:"content"`
	ToolCalls        []ToolCallAPI  `json:"tool_calls,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
	AdditionalKwargs map[string]any `json:"additional_kwargs,omitempty"`
	UsageMetadata    map[string]int `json:"usage_metadata,omitempty"`
}

// ToolCallAPI represents a tool call in API format.
type ToolCallAPI struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// ConvertInternalMessageToAPI converts an internal message to API format.
func ConvertInternalMessageToAPI(msg models.Message) MessageForAPI {
	apiMsg := MessageForAPI{
		Type:    InternalToLangChainRole(msg.Role),
		ID:      msg.ID,
		Role:    InternalToLangChainRole(msg.Role),
		Content: msg.Content,
	}

	if len(msg.ToolCalls) > 0 {
		apiMsg.ToolCalls = make([]ToolCallAPI, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			apiMsg.ToolCalls[i] = ToolCallAPI{
				ID:   tc.ID,
				Name: tc.Name,
				Args: tc.Arguments,
			}
		}
	}

	if msg.ToolResult != nil {
		apiMsg.ToolCallID = msg.ToolResult.CallID
	}

	// Decode metadata
	if len(msg.Metadata) > 0 {
		apiMsg.AdditionalKwargs = DecodeAdditionalKwargs(msg.Metadata)
	}

	return apiMsg
}
