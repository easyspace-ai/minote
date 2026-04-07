package langgraphcompat

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigEnvVars(t *testing.T) {
	t.Setenv("TEST_API_KEY", "sk-test-123")
	t.Setenv("TEST_BASE_URL", "https://api.example.com/v1")

	input := `
models:
  - name: gpt-4
    api_key: $TEST_API_KEY
    api_base: ${TEST_BASE_URL}
    model: gpt-4
`
	result := resolveConfigEnvVars(input)

	if !contains(result, "sk-test-123") {
		t.Errorf("expected resolved $TEST_API_KEY, got: %s", result)
	}
	if !contains(result, "https://api.example.com/v1") {
		t.Errorf("expected resolved ${TEST_BASE_URL}, got: %s", result)
	}
}

func TestResolveConfigEnvVarsUnset(t *testing.T) {
	input := `api_key: $NONEXISTENT_VAR_12345`
	result := resolveConfigEnvVars(input)
	if result != input {
		t.Errorf("expected unset var to remain, got: %s", result)
	}
}

func TestResolveConfigEnvVarsMixedCase(t *testing.T) {
	t.Setenv("my_api_key", "lowercase-key")
	t.Setenv("Mixed_Case_Key", "mixed-key")

	input := `api_key: $my_api_key
other: ${Mixed_Case_Key}`
	result := resolveConfigEnvVars(input)

	if !contains(result, "lowercase-key") {
		t.Errorf("expected lowercase var resolved, got: %s", result)
	}
	if !contains(result, "mixed-key") {
		t.Errorf("expected mixed-case var resolved, got: %s", result)
	}
}

func TestLoadGatewayConfigFull(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	configContent := `
config_version: 5
log_level: info

token_usage:
  enabled: true

models:
  - name: gpt-4
    model: gpt-4
    display_name: GPT-4
    supports_vision: true
  - name: claude-3.5
    model: claude-3-5-sonnet
    api_key: sk-test
    api_base: https://api.anthropic.com/v1
    supports_thinking: true

tools:
  - name: web_search
    group: web
    max_results: 5
  - name: tavily_search
    group: web
    api_key: tvly-test

sandbox:
  allow_host_bash: false

subagents:
  timeout_seconds: 900

tracing:
  enabled: false
  provider: langsmith
`

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("DEERFLOW_CONFIG_PATH", configPath)

	cfg := loadGatewayConfig()
	if cfg == nil {
		t.Fatal("expected config to be loaded")
	}

	// Check version
	if cfg.ConfigVersion != 5 {
		t.Errorf("expected config_version 5, got %d", cfg.ConfigVersion)
	}

	// Check models
	if len(cfg.Models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(cfg.Models))
	}
	if cfg.Models[0].Name != "gpt-4" {
		t.Errorf("expected first model name gpt-4, got %s", cfg.Models[0].Name)
	}
	if cfg.Models[1].APIKey != "sk-test" {
		t.Errorf("expected second model api_key sk-test, got %s", cfg.Models[1].APIKey)
	}
	if cfg.Models[1].APIBase != "https://api.anthropic.com/v1" {
		t.Errorf("expected second model api_base, got %s", cfg.Models[1].APIBase)
	}

	// Check tools
	if len(cfg.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(cfg.Tools))
	}
	if cfg.Tools[0].Name != "web_search" {
		t.Errorf("expected first tool name web_search, got %s", cfg.Tools[0].Name)
	}
	if cfg.Tools[0].MaxResults == nil || *cfg.Tools[0].MaxResults != 5 {
		t.Errorf("expected max_results 5")
	}

	// Check token usage
	if !configTokenUsageEnabled(cfg) {
		t.Error("expected token_usage.enabled to be true")
	}

	// Check tool helpers
	if !configToolEnabled(cfg, "web_search") {
		t.Error("expected web_search tool to be enabled")
	}
	if configToolEnabled(cfg, "nonexistent") {
		t.Error("expected nonexistent tool to not be enabled")
	}

	tool := configToolSetting(cfg, "tavily_search")
	if tool == nil {
		t.Fatal("expected tavily_search tool config")
	}
	if tool.APIKey != "tvly-test" {
		t.Errorf("expected tavily api_key, got %s", tool.APIKey)
	}

	// Check sandbox
	if cfg.Sandbox == nil {
		t.Fatal("expected sandbox config")
	}
	if cfg.Sandbox.AllowHostBash {
		t.Error("expected allow_host_bash to be false")
	}

	// Check subagents
	if cfg.Subagents == nil {
		t.Fatal("expected subagents config")
	}
	if cfg.Subagents.TimeoutSeconds != 900 {
		t.Errorf("expected timeout 900, got %d", cfg.Subagents.TimeoutSeconds)
	}

	// Check tracing
	if cfg.Tracing == nil {
		t.Fatal("expected tracing config")
	}
	if cfg.Tracing.Enabled {
		t.Error("expected tracing to be disabled")
	}
	if cfg.Tracing.Provider != "langsmith" {
		t.Errorf("expected provider langsmith, got %s", cfg.Tracing.Provider)
	}
}

func TestLoadGatewayConfigNoFile(t *testing.T) {
	t.Setenv("DEERFLOW_CONFIG_PATH", "/nonexistent/config.yaml")
	cfg := loadGatewayConfig()
	if cfg != nil {
		t.Error("expected nil config when file doesn't exist")
	}
}

func TestConfigTokenUsageDisabledByDefault(t *testing.T) {
	if configTokenUsageEnabled(nil) {
		t.Error("expected token usage disabled for nil config")
	}
	cfg := &gatewayConfigFile{}
	if configTokenUsageEnabled(cfg) {
		t.Error("expected token usage disabled when not set")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
