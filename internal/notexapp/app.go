package notexapp

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/easyspace-ai/minote/internal/notex"
	"github.com/easyspace-ai/minote/pkg/langgraphcompat"
)

func resolveWebRoot() string {
	if v := strings.TrimSpace(os.Getenv("NOTEX_WEB_ROOT")); v != "" {
		return filepath.Clean(v)
	}
	if fi, err := os.Stat(filepath.Clean("bin/web")); err == nil && fi.IsDir() {
		return filepath.Clean("bin/web")
	}
	if exe, err := os.Executable(); err == nil {
		next := filepath.Join(filepath.Dir(exe), "web")
		if fi, err := os.Stat(next); err == nil && fi.IsDir() {
			return next
		}
	}
	return filepath.Clean("bin/web")
}

func parseCommaPaths(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

const defaultAddr = ":8787"

type Config struct {
	Addr         string
	DataRoot     string
	AuthRequired bool
	DefaultModel string
	DatabaseURL  string
	Logger       *log.Logger
	// SkillsPaths is a comma-separated list in NOTEX_SKILLS_PATH; passed to notex.Config.SkillsPaths.
	SkillsPaths string
}

type App struct {
	httpServer *http.Server
	logger     *log.Logger
	shutdown   func(context.Context) error
}

func New(cfg Config) (*App, error) {
	addr := NormalizeAddr(cfg.Addr)
	logger := cfg.Logger
	if logger == nil {
		logger = log.New(os.Stderr, "notex ", log.LstdFlags)
	}

	databaseURL := strings.TrimSpace(cfg.DatabaseURL)
	defaultModel := strings.TrimSpace(cfg.DefaultModel)

	var notexStore *notex.Store
	if databaseURL != "" {
		store, err := notex.NewStore(context.Background(), databaseURL)
		if err != nil {
			return nil, err
		}
		notexStore = store
		logger.Printf("notex database schema synchronized")
	}

	webRoot := resolveWebRoot()
	var aiOpts []langgraphcompat.ServerOption
	if fi, err := os.Stat(webRoot); err == nil && fi.IsDir() {
		aiOpts = append(aiOpts, langgraphcompat.WithFrontendFS(os.DirFS(webRoot)))
		logger.Printf("web UI: %s", webRoot)
	} else {
		logger.Printf("web UI skipped (missing %s; set NOTEX_WEB_ROOT or run: cd web && pnpm build)", webRoot)
	}

	aiSrv, err := langgraphcompat.NewServer("", databaseURL, defaultModel, aiOpts...)
	if err != nil {
		if notexStore != nil {
			notexStore.Close()
		}
		return nil, err
	}

	skillsPaths := parseCommaPaths(cfg.SkillsPaths)
	if len(skillsPaths) == 0 {
		skillsPaths = parseCommaPaths(os.Getenv("NOTEX_SKILLS_PATH"))
	}

	notexSrv, err := notex.NewServer(notex.Config{
		Addr:         addr,
		DataRoot:     strings.TrimSpace(cfg.DataRoot),
		AuthRequired: cfg.AuthRequired,
		Logger:       logger,
		Store:        notexStore,
		SkillsPaths:  skillsPaths,
	})
	if err != nil {
		if notexStore != nil {
			notexStore.Close()
		}
		_ = aiSrv.Shutdown(context.Background())
		return nil, err
	}
	notexSrv.SetAIHandler(aiSrv.Handler())
	aiSrv.SetStudioDocumentInjection(func(ctx context.Context, userID, conversationID int64, docIDs []int64) string {
		return notexSrv.StudioInjectionPrefixForLangGraph(ctx, userID, conversationID, docIDs)
	})

	combinedMux := http.NewServeMux()
	combinedMux.Handle("/api/v1/", notexSrv.Handler())
	combinedMux.Handle("/health", notexSrv.Handler())
	combinedMux.Handle("/", aiSrv.Handler())

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           combinedMux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	app := &App{
		httpServer: httpServer,
		logger:     logger,
	}
	app.shutdown = func(ctx context.Context) error {
		var shutdownErr error
		if err := httpServer.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownErr = err
		}
		if err := aiSrv.Shutdown(ctx); err != nil && shutdownErr == nil {
			shutdownErr = err
		}
		if notexStore != nil {
			notexStore.Close()
		}
		return shutdownErr
	}
	return app, nil
}

func (a *App) ListenAndServe() error {
	if a == nil || a.httpServer == nil {
		return errors.New("notex app is not initialized")
	}
	a.logger.Printf("notex unified server on %s (business + AI)", a.httpServer.Addr)
	err := a.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *App) Shutdown(ctx context.Context) error {
	if a == nil || a.shutdown == nil {
		return nil
	}
	return a.shutdown(ctx)
}

func NormalizeAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return defaultAddr
	}
	if strings.HasPrefix(addr, ":") {
		return addr
	}
	return ":" + addr
}
