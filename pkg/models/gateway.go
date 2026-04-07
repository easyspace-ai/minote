package models

import "time"

// GatewaySkill represents a skill in the gateway
type GatewaySkill struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	License     string    `json:"license"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GatewayAgent represents an agent profile in the gateway
type GatewayAgent struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Model       *string   `json:"model"`
	ToolGroups  []string  `json:"tool_groups"`
	Soul        string    `json:"soul,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
