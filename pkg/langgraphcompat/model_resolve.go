package langgraphcompat

import "strings"

// effectiveChatModelID maps thread/run config to a real API model id.
// The UI sometimes sends only a provider slug (e.g. "openai") which is not a valid
// OpenAI / Ark `model` parameter — use the server default (DEFAULT_LLM_MODEL / -default-model) instead.
func effectiveChatModelID(requested, serverDefault string) string {
	requested = strings.TrimSpace(requested)
	serverDefault = strings.TrimSpace(serverDefault)
	if requested == "" {
		return serverDefault
	}
	switch strings.ToLower(requested) {
	case "openai", "anthropic", "siliconflow", "azure", "google", "ollama", "cohere", "mistral", "deepseek":
		if serverDefault != "" {
			return serverDefault
		}
	}
	return requested
}
