package langgraphcompat

// db_persistence.go adds durable PostgreSQL persistence for gateway entities
// (skills, agents, channels). This supplements the existing file-based gateway
// state (gateway_state.json) with database-backed storage. On startup, persisted
// records are merged into the in-memory state so that they survive restarts even
// when file storage is unavailable.

import (
	"context"
	"time"

	"github.com/easyspace-ai/minote/pkg/models"
	"github.com/easyspace-ai/minote/pkg/persistence"
	"github.com/google/uuid"
)

// persistSkillToDB saves a skill to the database store if available.
// The key is used as the skill ID when the skill has no explicit ID.
func (s *Server) persistSkillToDB(key string, skill GatewaySkill) {
	if s.persistence == nil {
		return
	}
	if skill.ID == "" {
		skill.ID = key
	}
	now := time.Now().UTC()
	if skill.CreatedAt.IsZero() {
		skill.CreatedAt = now
	}
	skill.UpdatedAt = now
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.persistence.SaveSkill(ctx, skill); err != nil {
		s.logger.Printf("Warning: failed to persist skill %q to database: %v", key, err)
	}
}

// deleteSkillFromDB removes a skill from the database store if available.
func (s *Server) deleteSkillFromDB(key string) {
	if s.persistence == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.persistence.DeleteSkill(ctx, key); err != nil && err != persistence.ErrNotFound {
		s.logger.Printf("Warning: failed to delete skill %q from database: %v", key, err)
	}
}

// persistAgentToDB saves an agent to the database store if available.
func (s *Server) persistAgentToDB(agent GatewayAgent) {
	if s.persistence == nil {
		return
	}
	if agent.ID == "" {
		agent.ID = agent.Name
	}
	now := time.Now().UTC()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	agent.UpdatedAt = now
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.persistence.SaveAgent(ctx, agent); err != nil {
		s.logger.Printf("Warning: failed to persist agent %q to database: %v", agent.Name, err)
	}
}

// deleteAgentFromDB removes an agent from the database store if available.
func (s *Server) deleteAgentFromDB(name string) {
	if s.persistence == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.persistence.DeleteAgent(ctx, name); err != nil && err != persistence.ErrNotFound {
		s.logger.Printf("Warning: failed to delete agent %q from database: %v", name, err)
	}
}

// persistChannelToDB saves a channel configuration to the database store if available.
func (s *Server) persistChannelToDB(name string, enabled bool, config map[string]any) {
	if s.persistence == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.persistence.SaveChannel(ctx, name, enabled, config); err != nil {
		s.logger.Printf("Warning: failed to persist channel %q to database: %v", name, err)
	}
}

// loadPersistedSkills loads skills from the database and merges them into the
// in-memory skills map. Database records take precedence for enabled/disabled
// state, while filesystem-discovered skills provide the base set.
func (s *Server) loadPersistedSkills() {
	if s.persistence == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	skills, err := s.persistence.ListSkills(ctx)
	if err != nil {
		s.logger.Printf("Warning: failed to load skills from database: %v", err)
		return
	}
	if len(skills) == 0 {
		return
	}
	// Merge DB skills into in-memory state. DB records override the enabled
	// flag but do not remove skills that only exist on the filesystem.
	for _, skill := range skills {
		key := skill.ID
		if key == "" {
			key = skillStorageKey(skill.Category, skill.Name)
		}
		if existing, ok := s.skills[key]; ok {
			existing.Enabled = skill.Enabled
			if skill.UpdatedAt.After(existing.UpdatedAt) {
				existing.UpdatedAt = skill.UpdatedAt
			}
			s.skills[key] = existing
		} else {
			s.skills[key] = skill
		}
	}
}

// loadPersistedAgents loads agents from the database and merges them into the
// in-memory agents map. Database records supplement filesystem-discovered agents.
// For agents that already exist in memory, database fields are merged if the
// database record is newer.
func (s *Server) loadPersistedAgents() {
	if s.persistence == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	agents, err := s.persistence.ListAgents(ctx)
	if err != nil {
		s.logger.Printf("Warning: failed to load agents from database: %v", err)
		return
	}
	if len(agents) == 0 {
		return
	}
	for _, agent := range agents {
		name := agent.Name
		if name == "" {
			name = agent.ID
		}
		existing, ok := s.agents[name]
		if !ok {
			// Agent only in DB, add to memory
			s.agents[name] = agent
		} else if agent.UpdatedAt.After(existing.UpdatedAt) {
			// DB is newer, merge fields
			if agent.Description != "" {
				existing.Description = agent.Description
			}
			if agent.Model != nil {
				existing.Model = agent.Model
			}
			if len(agent.ToolGroups) > 0 {
				existing.ToolGroups = agent.ToolGroups
			}
			if agent.Soul != "" {
				existing.Soul = agent.Soul
			}
			existing.UpdatedAt = agent.UpdatedAt
			s.agents[name] = existing
		}
	}
}

// loadPersistedChannels loads channel configurations from the database and
// merges them into the channel config.
func (s *Server) loadPersistedChannels() {
	if s.persistence == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	channels, err := s.persistence.ListChannels(ctx)
	if err != nil {
		s.logger.Printf("Warning: failed to load channels from database: %v", err)
		return
	}
	if len(channels) == 0 {
		return
	}
	for _, ch := range channels {
		if s.channelConfig.Channels == nil {
			s.channelConfig.Channels = make(map[string]map[string]any)
		}
		cfg, ok := s.channelConfig.Channels[ch.ID]
		if ok {
			cfg["enabled"] = ch.Enabled
			s.channelConfig.Channels[ch.ID] = cfg
		} else {
			s.channelConfig.Channels[ch.ID] = map[string]any{"enabled": ch.Enabled}
		}
	}
}

// seedSkillsFromDefaults writes the default skills into the database so that
// subsequent loads find them there. This is called once during startup after
// loadPersistedSkills populates the in-memory map.
func (s *Server) seedSkillsFromDefaults() {
	if s.persistence == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for key, skill := range s.skills {
		if skill.ID == "" {
			skill.ID = key
		}
		now := time.Now().UTC()
		if skill.CreatedAt.IsZero() {
			skill.CreatedAt = now
		}
		if skill.UpdatedAt.IsZero() {
			skill.UpdatedAt = now
		}
		if err := s.persistence.SaveSkill(ctx, skill); err != nil {
			s.logger.Printf("Warning: failed to seed skill %q to database: %v", key, err)
		}
	}
}

// newGatewayEntityID generates a unique identifier for a gateway entity.
func newGatewayEntityID() string {
	return uuid.New().String()
}

// ensureSkillID makes sure the skill has an ID; if empty, one is derived from
// the storage key (category:name).
func ensureSkillID(key string, skill *models.GatewaySkill) {
	if skill.ID == "" {
		skill.ID = key
	}
}
