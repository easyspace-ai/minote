// Package types defines shared types for LangGraph compatibility layer.
package types

import (
	"time"

	"github.com/easyspace-ai/minote/pkg/memory"
	"github.com/easyspace-ai/minote/pkg/models"
)

// GatewayModel represents a model configuration for the gateway.
type GatewayModel struct {
	ID                      string   `json:"id"`
	Name                    string   `json:"name"`
	Model                   string   `json:"model"`
	DisplayName             string   `json:"display_name"`
	Description             string   `json:"description,omitempty"`
	SupportsThinking        bool     `json:"supports_thinking,omitempty"`
	SupportsReasoningEffort bool     `json:"supports_reasoning_effort,omitempty"`
	SupportsVision          bool     `json:"supports_vision,omitempty"`
	MaxTokens               int      `json:"max_tokens,omitempty"`
	Temperature             *float64 `json:"temperature,omitempty"`
}

// GatewaySkill is an alias for models.GatewaySkill.
type GatewaySkill = models.GatewaySkill

// GatewayAgent is an alias for models.GatewayAgent.
type GatewayAgent = models.GatewayAgent

// MCPServerConfig represents MCP server configuration.
type MCPServerConfig struct {
	Type        string            `json:"type,omitempty"`
	Enabled     bool              `json:"enabled"`
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	OAuth       *MCPOAuthConfig   `json:"oauth,omitempty"`
	Description string            `json:"description"`
}

// MCPOAuthConfig represents OAuth configuration for MCP.
type MCPOAuthConfig struct {
	Enabled            bool              `json:"enabled"`
	TokenURL           string            `json:"token_url,omitempty"`
	GrantType          string            `json:"grant_type,omitempty"`
	ClientID           string            `json:"client_id,omitempty"`
	ClientSecret       string            `json:"client_secret,omitempty"`
	RefreshToken       string            `json:"refresh_token,omitempty"`
	Scope              string            `json:"scope,omitempty"`
	Audience           string            `json:"audience,omitempty"`
	TokenField         string            `json:"token_field,omitempty"`
	TokenTypeField     string            `json:"token_type_field,omitempty"`
	ExpiresInField     string            `json:"expires_in_field,omitempty"`
	DefaultTokenType   string            `json:"default_token_type,omitempty"`
	RefreshSkewSeconds int               `json:"refresh_skew_seconds,omitempty"`
	ExtraTokenParams   map[string]string `json:"extra_token_params,omitempty"`
}

// MCPConfig represents the overall MCP configuration.
type MCPConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcp_servers"`
}

// ChannelsConfig represents channel configuration.
type ChannelsConfig struct {
	Channels []ChannelInfo `json:"channels,omitempty"`
}

// ChannelInfo represents a single channel's information.
type ChannelInfo struct {
	Name          string    `json:"name"`
	Status        string    `json:"status"`
	LastMessageAt time.Time `json:"last_message_at,omitempty"`
	Error         string    `json:"error,omitempty"`
}

// MemorySection represents a section of memory.
type MemorySection struct {
	Summary   string `json:"summary"`
	UpdatedAt string `json:"updatedAt"`
}

// MemoryUser represents user-related memory.
type MemoryUser struct {
	WorkContext     MemorySection `json:"workContext"`
	PersonalContext MemorySection `json:"personalContext"`
	TopOfMind       MemorySection `json:"topOfMind"`
}

// MemoryHistory represents historical memory.
type MemoryHistory struct {
	RecentMonths       MemorySection `json:"recentMonths"`
	EarlierContext     MemorySection `json:"earlierContext"`
	LongTermBackground MemorySection `json:"longTermBackground"`
}

// MemoryFact represents a single memory fact.
type MemoryFact struct {
	ID         string  `json:"id"`
	Content    string  `json:"content"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	CreatedAt  string  `json:"createdAt"`
	Source     string  `json:"source"`
}

// MemoryResponse represents the memory response structure.
type MemoryResponse struct {
	Version     string        `json:"version"`
	LastUpdated string        `json:"lastUpdated"`
	User        MemoryUser    `json:"user"`
	History     MemoryHistory `json:"history"`
	Facts       []MemoryFact  `json:"facts"`
}

// PersistedState represents the persisted gateway state.
type PersistedState struct {
	Skills       map[string]GatewaySkill `json:"skills"`
	MCPConfig    MCPConfig               `json:"mcp_config"`
	Channels     ChannelsConfig          `json:"channels,omitempty"`
	Agents       map[string]GatewayAgent `json:"agents,omitempty"`
	UserProfile  string                  `json:"user_profile,omitempty"`
	MemoryThread string                  `json:"memory_thread,omitempty"`
	Memory       MemoryResponse          `json:"memory"`
}

// SuggestionContext provides context for generating suggestions.
type SuggestionContext struct {
	Language        string
	CurrentSubject  string
	PreviousSummary string
}

// ThreadMessage represents a simplified message structure for threads.
type ThreadMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
}

// ThreadRun represents a run execution on a thread.
type ThreadRun struct {
	ID        string   `json:"id"`
	ThreadID  string   `json:"thread_id"`
	Status    string   `json:"status"`
	Model     string   `json:"model"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
	Messages  []models.Message `json:"messages,omitempty"`
}

// Constants for skill categories.
const (
	SkillCategoryPublic = "public"
	SkillCategoryCustom = "custom"
)

// Constants for archive processing.
const (
	MaxSkillArchiveSize int64 = 512 << 20 // 512MB
)

// RawGatewayModel represents raw model configuration from JSON.
type RawGatewayModel struct {
	ID                      string   `json:"id"`
	Name                    string   `json:"name"`
	Model                   string   `json:"model"`
	DisplayName             string   `json:"display_name"`
	Description             string   `json:"description"`
	SupportsThinking        *bool    `json:"supports_thinking,omitempty"`
	SupportsReasoningEffort *bool    `json:"supports_reasoning_effort,omitempty"`
	SupportsVision          *bool    `json:"supports_vision,omitempty"`
	MaxTokens               int      `json:"max_tokens,omitempty"`
	Temperature             *float64 `json:"temperature,omitempty"`
}

// ModelCatalogConfig represents the model catalog configuration file structure.
type ModelCatalogConfig struct {
	Version string            `json:"version"`
	Models  []RawGatewayModel `json:"models"`
}

// MemoryDocument is an alias for memory.Document.
type MemoryDocument = memory.Document
