package langgraphcompat

import (
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/easyspace-ai/minote/pkg/llm"
	"gopkg.in/yaml.v3"
)

// loadGatewayConfig reads and parses the config.yaml file.
// Returns nil (not an error) when no config file exists.
func loadGatewayConfig() *gatewayConfigFile {
	configPath := gatewayModelCatalogConfigPath()
	if configPath == "" {
		return nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}
	// Resolve environment variable references ($VAR or ${VAR}) before parsing
	resolved := resolveConfigEnvVars(string(data))

	var cfg gatewayConfigFile
	if err := yaml.Unmarshal([]byte(resolved), &cfg); err != nil {
		return nil
	}
	return &cfg
}

// envVarPattern matches $VAR_NAME and ${VAR_NAME} patterns in config values.
// Supports uppercase, lowercase, and mixed-case environment variable names.
var envVarPattern = regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?`)

// resolveConfigEnvVars replaces $VAR and ${VAR} references with environment variable values.
func resolveConfigEnvVars(input string) string {
	return envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name
		name := match
		name = strings.TrimPrefix(name, "${")
		name = strings.TrimPrefix(name, "$")
		name = strings.TrimSuffix(name, "}")

		if val, ok := os.LookupEnv(name); ok {
			return val
		}
		return match // Keep original if env var not set
	})
}

// configToolEnabled checks if a tool is configured as enabled in config.yaml
func configToolEnabled(cfg *gatewayConfigFile, toolName string) bool {
	if cfg == nil {
		return false
	}
	for _, tool := range cfg.Tools {
		if tool.Name == toolName {
			return true
		}
	}
	return false
}

// configToolSetting returns a tool config by name from the config file
func configToolSetting(cfg *gatewayConfigFile, toolName string) *gatewayConfigTool {
	if cfg == nil {
		return nil
	}
	for i, tool := range cfg.Tools {
		if tool.Name == toolName {
			return &cfg.Tools[i]
		}
	}
	return nil
}

// configTokenUsageEnabled checks if token usage tracking is enabled
func configTokenUsageEnabled(cfg *gatewayConfigFile) bool {
	if cfg == nil || cfg.TokenUsage == nil {
		return false
	}
	return cfg.TokenUsage.Enabled
}

// applyLogLevel sets the log output level based on config.yaml log_level field.
// Supported values: debug, info, warn, error, off.
func applyLogLevel(cfg *gatewayConfigFile, logger *log.Logger) {
	level := ""
	if cfg != nil {
		level = strings.ToLower(strings.TrimSpace(cfg.LogLevel))
	}
	// Env var overrides config file
	if env := strings.TrimSpace(os.Getenv("DEERFLOW_LOG_LEVEL")); env != "" {
		level = strings.ToLower(env)
	}
	switch level {
	case "off", "none":
		logger.SetOutput(io.Discard)
	case "error":
		logger.SetPrefix("[ERROR] ")
	case "warn", "warning":
		logger.SetPrefix("[WARN] ")
	case "info":
		logger.SetPrefix("[INFO] ")
	case "debug":
		logger.SetPrefix("[DEBUG] ")
		logger.SetFlags(log.LstdFlags | log.Lshortfile)
	}
}

// configToolSearchEnabled checks if tool search is enabled in config.
func configToolSearchEnabled(cfg *gatewayConfigFile) bool {
	if cfg == nil || cfg.ToolSearch == nil {
		return false
	}
	return cfg.ToolSearch.Enabled
}

// configToolGroups returns the configured tool groups from config.yaml.
func configToolGroups(cfg *gatewayConfigFile) []string {
	if cfg == nil {
		return nil
	}
	groups := make([]string, 0, len(cfg.ToolGroups))
	for _, tg := range cfg.ToolGroups {
		if name := strings.TrimSpace(tg.Name); name != "" {
			groups = append(groups, name)
		}
	}
	return groups
}

// configSummarization returns the summarization config or nil.
func configSummarization(cfg *gatewayConfigFile) *gatewaySummarize {
	if cfg == nil {
		return nil
	}
	return cfg.Summarization
}

// configMemory returns the memory config or nil.
func configMemory(cfg *gatewayConfigFile) *gatewayMemoryCfg {
	if cfg == nil {
		return nil
	}
	return cfg.Memory
}

// configUploads returns the uploads config or nil.
func configUploads(cfg *gatewayConfigFile) *gatewayUploads {
	if cfg == nil {
		return nil
	}
	return cfg.Uploads
}

// configCheckpointer returns the checkpointer config or nil.
func configCheckpointer(cfg *gatewayConfigFile) *gatewayCheckpointer {
	if cfg == nil {
		return nil
	}
	return cfg.Checkpointer
}

// resolveMemoryTimeout returns the memory update timeout, checking YAML config then env vars.
func resolveMemoryTimeout(memoryCfg *gatewayMemoryCfg) time.Duration {
	timeout := 30 * time.Second
	if memoryCfg != nil && memoryCfg.DebounceSeconds > 0 {
		timeout = time.Duration(memoryCfg.DebounceSeconds) * time.Second
	}
	if raw := strings.TrimSpace(os.Getenv("MEMORY_UPDATE_TIMEOUT")); raw != "" {
		if parsed, parseErr := time.ParseDuration(raw); parseErr == nil && parsed > 0 {
			timeout = parsed
		}
	}
	return timeout
}

// resolveMemoryModel returns the LLM model name for memory operations.
func resolveMemoryModel(memoryCfg *gatewayMemoryCfg, defaultModel string) string {
	model := defaultModel
	if memoryCfg != nil && memoryCfg.ModelName != "" {
		model = memoryCfg.ModelName
	}
	if env := strings.TrimSpace(os.Getenv("MEMORY_LLM_MODEL")); env != "" {
		model = env
	}
	return model
}

// buildSummarizationConfig converts the YAML summarization config into
// the llm.SummarizationConfig used by the ConversationSummarizer.
func buildSummarizationConfig(cfg *gatewaySummarize, defaultModel string) llm.SummarizationConfig {
	sc := llm.SummarizationConfig{
		Enabled:       cfg.Enabled,
		Model:         defaultModel,
		SummaryPrompt: cfg.SummaryPrompt,
	}
	if cfg.ModelName != "" {
		sc.Model = cfg.ModelName
	}
	if cfg.TrimTokensToSummarize != nil {
		sc.TrimTokensToSummarize = *cfg.TrimTokensToSummarize
	}
	// Parse trigger rules
	for _, trigger := range cfg.Trigger {
		switch trigger.Type {
		case "token_count":
			if int(trigger.Value) > 0 {
				sc.MaxTokenThreshold = int(trigger.Value)
			}
		case "message_count":
			if int(trigger.Value) > 0 {
				sc.MaxMessageCount = int(trigger.Value)
			}
		}
	}
	// Parse keep rules
	if cfg.Keep != nil {
		switch cfg.Keep.Type {
		case "last_n_rounds":
			if cfg.Keep.Value > 0 {
				sc.KeepRounds = cfg.Keep.Value
			}
		}
	}
	return sc
}
