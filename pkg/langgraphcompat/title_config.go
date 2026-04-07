package langgraphcompat

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	titleEnabledEnv  = "DEERFLOW_TITLE_ENABLED"
	titleMaxWordsEnv = "DEERFLOW_TITLE_MAX_WORDS"
	titleMaxCharsEnv = "DEERFLOW_TITLE_MAX_CHARS"
	titleModelEnv    = "DEERFLOW_TITLE_MODEL"

	defaultTitleMaxWords = 6
	defaultTitleMaxChars = 60
	minTitleMaxWords     = 1
	maxTitleMaxWords     = 20
	minTitleMaxChars     = 10
	maxTitleMaxChars     = 200
)

type titleConfig struct {
	Enabled  bool
	MaxWords int
	MaxChars int
	Model    string
}

func loadTitleConfig() titleConfig {
	cfg := titleConfig{
		Enabled:  true,
		MaxWords: defaultTitleMaxWords,
		MaxChars: defaultTitleMaxChars,
	}

	// Load from config.yaml first
	if fileCfg := loadTitleConfigFromFile(); fileCfg != nil {
		if fileCfg.Enabled != nil {
			cfg.Enabled = *fileCfg.Enabled
		}
		if fileCfg.MaxWords > 0 {
			cfg.MaxWords = fileCfg.MaxWords
		}
		if fileCfg.MaxChars > 0 {
			cfg.MaxChars = fileCfg.MaxChars
		}
		if fileCfg.ModelName != "" {
			cfg.Model = fileCfg.ModelName
		}
	}

	// Env vars override file config
	if raw := strings.TrimSpace(os.Getenv(titleModelEnv)); raw != "" {
		cfg.Model = raw
	}
	if raw := strings.TrimSpace(os.Getenv(titleEnabledEnv)); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			cfg.Enabled = parsed
		}
	}
	if parsed, ok := parseTitleBound(titleMaxWordsEnv, minTitleMaxWords, maxTitleMaxWords); ok {
		cfg.MaxWords = parsed
	}
	if parsed, ok := parseTitleBound(titleMaxCharsEnv, minTitleMaxChars, maxTitleMaxChars); ok {
		cfg.MaxChars = parsed
	}

	return cfg
}

type titleFileConfig struct {
	Enabled   *bool  `yaml:"enabled"`
	MaxWords  int    `yaml:"max_words"`
	MaxChars  int    `yaml:"max_chars"`
	ModelName string `yaml:"model_name"`
}

func loadTitleConfigFromFile() *titleFileConfig {
	path, ok := resolveGatewayConfigPath()
	if !ok {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw struct {
		Title titleFileConfig `yaml:"title"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}
	return &raw.Title
}

func parseTitleBound(env string, min, max int) (int, bool) {
	raw := strings.TrimSpace(os.Getenv(env))
	if raw == "" {
		return 0, false
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < min || value > max {
		return 0, false
	}
	return value, true
}
