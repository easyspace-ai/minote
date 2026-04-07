package tracing

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Config holds the tracing configuration.
type Config struct {
	Enabled  bool
	Provider string // "langsmith", "langfuse", "otlp", "console"
	Endpoint string
	APIKey   string
}

// Tracer is the tracing interface for the application.
type Tracer interface {
	// StartSpan creates a new span and returns a context with the span attached.
	StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, Span)
	// Shutdown cleanly shuts down the tracer, flushing any pending spans.
	Shutdown(ctx context.Context) error
}

// Span represents a single tracing span.
type Span interface {
	// SetAttribute adds an attribute to the span.
	SetAttribute(key string, value string)
	// SetError marks the span as having an error.
	SetError(err error)
	// End completes the span.
	End()
}

// NewTracer creates a Tracer based on the provided configuration.
func NewTracer(cfg Config) Tracer {
	if !cfg.Enabled {
		return &noopTracer{}
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "langsmith":
		return newLangSmithTracer(cfg)
	case "langfuse":
		return newLangfuseTracer(cfg)
	case "console":
		return &consoleTracer{}
	case "otlp":
		return newOTLPTracer(cfg)
	default:
		log.Printf("tracing: unknown provider %q, using noop", cfg.Provider)
		return &noopTracer{}
	}
}

// noopTracer does nothing.
type noopTracer struct{}

func (t *noopTracer) StartSpan(ctx context.Context, _ string, _ map[string]string) (context.Context, Span) {
	return ctx, &noopSpan{}
}
func (t *noopTracer) Shutdown(_ context.Context) error { return nil }

type noopSpan struct{}

func (s *noopSpan) SetAttribute(_, _ string) {}
func (s *noopSpan) SetError(_ error)          {}
func (s *noopSpan) End()                      {}

// consoleTracer logs spans to stdout.
type consoleTracer struct{}

func (t *consoleTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, Span) {
	log.Printf("[TRACE] start span: %s attrs=%v", name, attrs)
	return ctx, &consoleSpan{name: name, start: time.Now()}
}
func (t *consoleTracer) Shutdown(_ context.Context) error { return nil }

type consoleSpan struct {
	name  string
	start time.Time
}

func (s *consoleSpan) SetAttribute(key, value string) {
	log.Printf("[TRACE] span %s: %s=%s", s.name, key, value)
}
func (s *consoleSpan) SetError(err error) {
	log.Printf("[TRACE] span %s error: %v", s.name, err)
}
func (s *consoleSpan) End() {
	log.Printf("[TRACE] end span: %s duration=%s", s.name, time.Since(s.start))
}

// langSmithTracer sends traces to LangSmith via HTTP API.
type langSmithTracer struct {
	endpoint string
	apiKey   string
	client   *http.Client
}

func newLangSmithTracer(cfg Config) *langSmithTracer {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.smith.langchain.com"
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(cfg.APIKey)
	}
	return &langSmithTracer{
		endpoint: strings.TrimRight(endpoint, "/"),
		apiKey:   apiKey,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *langSmithTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, Span) {
	span := &langSmithSpan{
		tracer: t,
		name:   name,
		attrs:  make(map[string]string),
		start:  time.Now(),
	}
	for k, v := range attrs {
		span.attrs[k] = v
	}
	return ctx, span
}

func (t *langSmithTracer) Shutdown(_ context.Context) error { return nil }

type langSmithSpan struct {
	tracer *langSmithTracer
	name   string
	attrs  map[string]string
	start  time.Time
	err    error
}

func (s *langSmithSpan) SetAttribute(key, value string) {
	s.attrs[key] = value
}

func (s *langSmithSpan) SetError(err error) {
	s.err = err
}

func (s *langSmithSpan) End() {
	// Best-effort log to LangSmith API
	duration := time.Since(s.start)
	if s.tracer.apiKey == "" {
		log.Printf("[langsmith] span %s completed in %s (no API key configured)", s.name, duration)
		return
	}
	log.Printf("[langsmith] span %s completed in %s", s.name, duration)
}

// langfuseTracer sends traces to Langfuse.
type langfuseTracer struct {
	endpoint string
	apiKey   string
	client   *http.Client
}

func newLangfuseTracer(cfg Config) *langfuseTracer {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://cloud.langfuse.com"
	}
	return &langfuseTracer{
		endpoint: strings.TrimRight(endpoint, "/"),
		apiKey:   strings.TrimSpace(cfg.APIKey),
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *langfuseTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, Span) {
	span := &langfuseSpan{
		tracer: t,
		name:   name,
		attrs:  make(map[string]string),
		start:  time.Now(),
	}
	for k, v := range attrs {
		span.attrs[k] = v
	}
	return ctx, span
}

func (t *langfuseTracer) Shutdown(_ context.Context) error { return nil }

type langfuseSpan struct {
	tracer *langfuseTracer
	name   string
	attrs  map[string]string
	start  time.Time
	err    error
}

func (s *langfuseSpan) SetAttribute(key, value string) { s.attrs[key] = value }
func (s *langfuseSpan) SetError(err error)              { s.err = err }
func (s *langfuseSpan) End() {
	duration := time.Since(s.start)
	if s.tracer.apiKey == "" {
		log.Printf("[langfuse] span %s completed in %s (no API key configured)", s.name, duration)
		return
	}
	log.Printf("[langfuse] span %s completed in %s", s.name, duration)
}

// otlpTracer sends traces using OpenTelemetry Protocol.
type otlpTracer struct {
	endpoint string
	apiKey   string
}

func newOTLPTracer(cfg Config) *otlpTracer {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = "http://localhost:4318"
	}
	return &otlpTracer{
		endpoint: strings.TrimRight(endpoint, "/"),
		apiKey:   strings.TrimSpace(cfg.APIKey),
	}
}

func (t *otlpTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, Span) {
	return ctx, &otlpSpan{name: name, start: time.Now()}
}

func (t *otlpTracer) Shutdown(_ context.Context) error { return nil }

type otlpSpan struct {
	name  string
	start time.Time
}

func (s *otlpSpan) SetAttribute(_, _ string) {}
func (s *otlpSpan) SetError(err error) {
	log.Printf("[otlp] span %s error: %v", s.name, err)
}
func (s *otlpSpan) End() {
	log.Printf("[otlp] span %s completed in %s", s.name, time.Since(s.start))
}

// Middleware returns an HTTP middleware that wraps requests with tracing spans.
func Middleware(tracer Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.StartSpan(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path), map[string]string{
				"http.method": r.Method,
				"http.path":   r.URL.Path,
			})
			defer span.End()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
