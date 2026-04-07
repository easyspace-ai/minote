package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/easyspace-ai/minote/internal/notexapp"
	"github.com/easyspace-ai/minote/pkg/dotenv"
)

type config struct {
	Port              string `env:"GATEWAY_PORT" envDefault:":8080"`
	DatabaseURL       string `env:"DATABASE_URL" envDefault:""`
	OpenAIAPIKey      string `env:"OPENAI_API_KEY"`
	AnthropicAPIKey   string `env:"ANTHROPIC_API_KEY"`
	SiliconFlowAPIKey string `env:"SILICONFLOW_API_KEY"`
	DefaultModel      string `env:"DEFAULT_MODEL" envDefault:"openai/gpt-4o"`
}

func main() {
	dotenv.Load()
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}

	applyEnv(cfg)

	logger := log.New(os.Stderr, "gateway ", log.LstdFlags)
	logger.Printf("cmd/gateway now boots the unified notex server")

	app, err := notexapp.New(notexapp.Config{
		Addr:         firstNonEmpty(strings.TrimSpace(os.Getenv("NOTEX_PORT")), cfg.Port),
		DataRoot:     firstNonEmpty(strings.TrimSpace(os.Getenv("NOTEX_DATA_ROOT")), "./data/notex"),
		AuthRequired: resolveRequireAuth(),
		DefaultModel: modelNameForNotex(cfg.DefaultModel),
		DatabaseURL:  firstNonEmpty(strings.TrimSpace(os.Getenv("POSTGRES_URL")), cfg.DatabaseURL),
		Logger:       logger,
		SkillsPaths:  strings.TrimSpace(os.Getenv("NOTEX_SKILLS_PATH")),
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil {
			log.Fatal(err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := app.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			log.Fatal(err)
		}
	}
}

func applyEnv(cfg config) {
	if cfg.OpenAIAPIKey != "" {
		_ = os.Setenv("OPENAI_API_KEY", cfg.OpenAIAPIKey)
	}
	if cfg.AnthropicAPIKey != "" {
		_ = os.Setenv("ANTHROPIC_API_KEY", cfg.AnthropicAPIKey)
	}
	if cfg.SiliconFlowAPIKey != "" {
		_ = os.Setenv("SILICONFLOW_API_KEY", cfg.SiliconFlowAPIKey)
	}

	_, modelName := splitConfiguredModel(cfg.DefaultModel)
	if modelName != "" {
		_ = os.Setenv("DEFAULT_LLM_MODEL", modelName)
	}
}

func normalizeAddr(addr string) string {
	return notexapp.NormalizeAddr(addr)
}

func splitConfiguredModel(modelRef string) (string, string) {
	modelRef = strings.TrimSpace(modelRef)
	parts := strings.SplitN(modelRef, "/", 2)
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "openai", modelRef
}

func modelNameForNotex(modelRef string) string {
	_, modelName := splitConfiguredModel(modelRef)
	return strings.TrimSpace(modelName)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func resolveRequireAuth() bool {
	raw := strings.TrimSpace(os.Getenv("NOTEX_REQUIRE_AUTH"))
	if raw == "" {
		return true
	}
	switch strings.ToLower(raw) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}
