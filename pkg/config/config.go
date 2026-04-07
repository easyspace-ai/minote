package config

import (
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	"gopkg.in/yaml.v3"
)

// Config 统一全局配置
type Config struct {
	// 服务通用配置
	Service struct {
		Port         string `yaml:"port" env:"PORT" envDefault:":8080"`
		Env          string `yaml:"env" env:"ENV" envDefault:"development"`
		LogLevel     string `yaml:"log_level" env:"LOG_LEVEL" envDefault:"info"`
		ShutdownWait int    `yaml:"shutdown_wait" env:"SHUTDOWN_WAIT" envDefault:"30"`
	} `yaml:"service"`

	// 数据库配置
	Database struct {
		URL             string `yaml:"url" env:"DATABASE_URL" envDefault:"postgres://postgres:postgres@localhost:5432/youmind?sslmode=disable"`
		MaxOpenConns    int    `yaml:"max_open_conns" env:"DB_MAX_OPEN_CONNS" envDefault:"50"`
		MaxIdleConns    int    `yaml:"max_idle_conns" env:"DB_MAX_IDLE_CONNS" envDefault:"10"`
		ConnMaxLifetime int    `yaml:"conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME" envDefault:"300"`
	} `yaml:"database"`

	// Redis配置
	Redis struct {
		Addr     string `yaml:"addr" env:"REDIS_ADDR" envDefault:"localhost:6379"`
		Password string `yaml:"password" env:"REDIS_PASSWORD" envDefault:""`
		DB       int    `yaml:"db" env:"REDIS_DB" envDefault:"0"`
		Enabled  bool   `yaml:"enabled" env:"REDIS_ENABLED" envDefault:"true"`
	} `yaml:"redis"`

	// LLM配置
	LLM struct {
		DefaultModel      string `yaml:"default_model" env:"DEFAULT_MODEL" envDefault:"openai/gpt-4o"`
		OpenAIAPIKey      string `yaml:"openai_api_key" env:"OPENAI_API_KEY"`
		AnthropicAPIKey   string `yaml:"anthropic_api_key" env:"ANTHROPIC_API_KEY"`
		SiliconFlowAPIKey string `yaml:"siliconflow_api_key" env:"SILICONFLOW_API_KEY"`
		BaseURL           string `yaml:"base_url" env:"LLM_BASE_URL"`
		MaxTokens         int    `yaml:"max_tokens" env:"LLM_MAX_TOKENS" envDefault:"4096"`
		Temperature       float64 `yaml:"temperature" env:"LLM_TEMPERATURE" envDefault:"0.7"`
	} `yaml:"llm"`

	// Notex服务配置
	Notex struct {
		DataRoot     string `yaml:"data_root" env:"NOTEX_DATA_ROOT" envDefault:"./data/notex"`
		AuthRequired bool   `yaml:"auth_required" env:"NOTEX_REQUIRE_AUTH" envDefault:"true"`
		SkillsPath   string `yaml:"skills_path" env:"NOTEX_SKILLS_PATH"`
	} `yaml:"notex"`

	// 安全配置
	Security struct {
		JWTSecret     string `yaml:"jwt_secret" env:"JWT_SECRET"`
		TokenExpiry   int    `yaml:"token_expiry_hours" env:"TOKEN_EXPIRY_HOURS" envDefault:"168"` // 7天
		CORSAllowOrigins string `yaml:"cors_allow_origins" env:"CORS_ALLOW_ORIGINS" envDefault:"*"`
	} `yaml:"security"`

	// 外部服务配置
	Services struct {
		MarkItDownURL string `yaml:"markitdown_url" env:"MARKITDOWN_URL" envDefault:"http://localhost:8787"`
		DocReaderAddr string `yaml:"docreader_addr" env:"DOCREADER_ADDR" envDefault:"localhost:50051"`
		MinIOEndpoint string `yaml:"minio_endpoint" env:"MINIO_ENDPOINT" envDefault:"localhost:9000"`
		MinIOAccessKey string `yaml:"minio_access_key" env:"MINIO_ACCESS_KEY" envDefault:"minioadmin"`
		MinIOSecretKey string `yaml:"minio_secret_key" env:"MINIO_SECRET_KEY" envDefault:"minioadmin"`
	} `yaml:"services"`
}

// Load 加载配置，优先从环境变量加载，其次从yaml配置文件加载
func Load(configPath ...string) (*Config, error) {
	cfg := &Config{}

	// 首先尝试从配置文件加载
	if len(configPath) > 0 && configPath[0] != "" {
		data, err := os.ReadFile(configPath[0])
		if err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
	}

	// 然后从环境变量加载，覆盖配置文件的值
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// IsDevelopment 判断是否是开发环境
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Service.Env) == "development" || strings.ToLower(c.Service.Env) == "dev"
}

// IsProduction 判断是否是生产环境
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Service.Env) == "production" || strings.ToLower(c.Service.Env) == "prod"
}
