package llm

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// TokenUsageTracker tracks LLM token usage across model calls.
// Enabled via config.yaml token_usage.enabled or DEERFLOW_TOKEN_USAGE_ENABLED env var.
type TokenUsageTracker struct {
	enabled bool
	logger  *log.Logger
	mu      sync.Mutex

	// Cumulative counters
	totalInputTokens  atomic.Int64
	totalOutputTokens atomic.Int64
	totalCalls        atomic.Int64
}

// NewTokenUsageTracker creates a new token usage tracker.
// Pass enabled=true to activate logging, or it reads from env.
func NewTokenUsageTracker(enabled bool) *TokenUsageTracker {
	if !enabled {
		envVal := strings.TrimSpace(os.Getenv("DEERFLOW_TOKEN_USAGE_ENABLED"))
		enabled = envVal == "true" || envVal == "1"
	}
	return &TokenUsageTracker{
		enabled: enabled,
		logger:  log.New(os.Stderr, "[token-usage] ", log.LstdFlags),
	}
}

// Track records token usage from a completed LLM call.
func (t *TokenUsageTracker) Track(model string, usage Usage, duration time.Duration) {
	if t == nil || !t.enabled {
		return
	}

	t.totalCalls.Add(1)
	t.totalInputTokens.Add(int64(usage.InputTokens))
	t.totalOutputTokens.Add(int64(usage.OutputTokens))

	t.logger.Printf("model=%s input_tokens=%d output_tokens=%d total_tokens=%d reasoning_tokens=%d cached_input=%d duration=%s",
		model,
		usage.InputTokens,
		usage.OutputTokens,
		usage.TotalTokens,
		usage.ReasoningTokens,
		usage.CachedInputTokens,
		duration.Round(time.Millisecond),
	)
}

// Stats returns cumulative usage statistics.
func (t *TokenUsageTracker) Stats() TokenUsageStats {
	if t == nil {
		return TokenUsageStats{}
	}
	return TokenUsageStats{
		TotalCalls:        t.totalCalls.Load(),
		TotalInputTokens:  t.totalInputTokens.Load(),
		TotalOutputTokens: t.totalOutputTokens.Load(),
	}
}

// TokenUsageStats holds cumulative token usage data.
type TokenUsageStats struct {
	TotalCalls        int64 `json:"total_calls"`
	TotalInputTokens  int64 `json:"total_input_tokens"`
	TotalOutputTokens int64 `json:"total_output_tokens"`
}

// TrackingProvider wraps an LLMProvider to add token usage tracking.
type TrackingProvider struct {
	inner   LLMProvider
	tracker *TokenUsageTracker
}

// NewTrackingProvider wraps a provider with token usage tracking.
func NewTrackingProvider(inner LLMProvider, tracker *TokenUsageTracker) LLMProvider {
	if tracker == nil || !tracker.enabled {
		return inner
	}
	return &TrackingProvider{inner: inner, tracker: tracker}
}

func (p *TrackingProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	start := time.Now()
	resp, err := p.inner.Chat(ctx, req)
	if err == nil {
		p.tracker.Track(req.Model, resp.Usage, time.Since(start))
	}
	return resp, err
}

func (p *TrackingProvider) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	start := time.Now()
	ch, err := p.inner.Stream(ctx, req)
	if err != nil {
		return ch, err
	}

	// Wrap the channel to intercept the final chunk with usage data
	wrappedCh := make(chan StreamChunk)
	go func() {
		defer close(wrappedCh)
		for chunk := range ch {
			if chunk.Done && chunk.Usage != nil {
				p.tracker.Track(req.Model, *chunk.Usage, time.Since(start))
			}
			wrappedCh <- chunk
		}
	}()

	return wrappedCh, nil
}
