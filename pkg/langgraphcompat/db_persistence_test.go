package langgraphcompat

import (
	"log"
	"testing"
	"time"
)

// TestPersistSkillToDBNoopWhenNilStore verifies that DB persistence helpers
// do not panic when the persistence store is nil (the common case in tests
// and when running without a database).
func TestPersistSkillToDBNoopWhenNilStore(t *testing.T) {
	s := &Server{
		logger:      log.Default(),
		persistence: nil,
		skills:      defaultGatewaySkills(),
	}

	// Should be a no-op without panicking
	s.persistSkillToDB("public:deep-research", GatewaySkill{
		ID:       "public:deep-research",
		Name:     "deep-research",
		Category: "public",
		Enabled:  true,
	})

	s.deleteSkillFromDB("public:deep-research")
}

func TestPersistAgentToDBNoopWhenNilStore(t *testing.T) {
	s := &Server{
		logger:      log.Default(),
		persistence: nil,
		agents:      map[string]GatewayAgent{},
	}

	s.persistAgentToDB(GatewayAgent{
		ID:   "test-agent",
		Name: "test-agent",
	})

	s.deleteAgentFromDB("test-agent")
}

func TestPersistChannelToDBNoopWhenNilStore(t *testing.T) {
	s := &Server{
		logger:      log.Default(),
		persistence: nil,
	}

	s.persistChannelToDB("feishu", true, map[string]any{"token": "abc"})
}

func TestLoadPersistedEntitiesNoopWhenNilStore(t *testing.T) {
	s := &Server{
		logger:      log.Default(),
		persistence: nil,
		skills:      defaultGatewaySkills(),
		agents:      map[string]GatewayAgent{},
		channelConfig: gatewayChannelsConfig{
			Channels: make(map[string]map[string]any),
		},
	}

	// These should all be no-ops without panicking
	s.loadPersistedSkills()
	s.loadPersistedAgents()
	s.loadPersistedChannels()
	s.seedSkillsFromDefaults()

	// Verify skills unchanged
	if len(s.skills) != len(defaultGatewaySkills()) {
		t.Fatalf("skills count changed: got %d, want %d", len(s.skills), len(defaultGatewaySkills()))
	}
}

func TestEnsureSkillID(t *testing.T) {
	skill := GatewaySkill{
		Name:     "deep-research",
		Category: "public",
	}
	ensureSkillID("public:deep-research", &skill)
	if skill.ID != "public:deep-research" {
		t.Fatalf("skill ID=%q want public:deep-research", skill.ID)
	}

	// Should not overwrite existing ID
	skill.ID = "custom-id"
	ensureSkillID("other-key", &skill)
	if skill.ID != "custom-id" {
		t.Fatalf("skill ID=%q want custom-id (should not overwrite)", skill.ID)
	}
}

func TestNewGatewayEntityID(t *testing.T) {
	id1 := newGatewayEntityID()
	id2 := newGatewayEntityID()
	if id1 == "" || id2 == "" {
		t.Fatal("generated ID should not be empty")
	}
	if id1 == id2 {
		t.Fatal("generated IDs should be unique")
	}
}

func TestSyncAllEntitiesToDBNoopWhenNilStore(t *testing.T) {
	s := &Server{
		logger:      log.Default(),
		persistence: nil,
	}

	state := gatewayPersistedState{
		Skills: map[string]GatewaySkill{
			"public:deep-research": {
				Name:      "deep-research",
				Category:  "public",
				Enabled:   true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		Agents: map[string]GatewayAgent{
			"writer-bot": {
				Name:        "writer-bot",
				Description: "Test agent",
			},
		},
		Channels: gatewayChannelsConfig{
			Channels: map[string]map[string]any{
				"feishu": {"enabled": true},
			},
		},
	}

	// Should be a no-op without panicking
	s.syncAllEntitiesToDB(state)
}
