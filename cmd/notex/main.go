package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/easyspace-ai/minote/internal/notexapp"
	"github.com/easyspace-ai/minote/pkg/dotenv"
)

type config struct {
	Port         string `env:"NOTEX_PORT" envDefault:":8787"`
	DataRoot     string `env:"NOTEX_DATA_ROOT" envDefault:"./data/notex"`
	RequireAuth  bool   `env:"NOTEX_REQUIRE_AUTH" envDefault:"true"`
	DefaultModel string `env:"DEFAULT_LLM_MODEL" envDefault:""`
	DatabaseURL  string `env:"POSTGRES_URL" envDefault:""`
}

func main() {
	dotenv.Load()
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}

	app, err := notexapp.New(notexapp.Config{
		Addr:         cfg.Port,
		DataRoot:     cfg.DataRoot,
		AuthRequired: cfg.RequireAuth,
		DefaultModel: cfg.DefaultModel,
		DatabaseURL:  cfg.DatabaseURL,
		Logger:       log.New(os.Stderr, "notex ", log.LstdFlags),
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
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := app.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}
}

