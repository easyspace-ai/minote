package llm

import (
	"context"
	"testing"
	"time"

	"github.com/easyspace-ai/minote/pkg/models"
)

func TestTokenUsageTrackerDisabled(t *testing.T) {
	tracker := NewTokenUsageTracker(false)
	// Should not panic when tracking is disabled
	tracker.Track("gpt-4", Usage{InputTokens: 100, OutputTokens: 50}, time.Second)

	stats := tracker.Stats()
	if stats.TotalCalls != 0 {
		t.Errorf("expected 0 calls when disabled, got %d", stats.TotalCalls)
	}
}

func TestTokenUsageTrackerEnabled(t *testing.T) {
	tracker := NewTokenUsageTracker(true)
	tracker.Track("gpt-4", Usage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150}, time.Second)
	tracker.Track("gpt-4", Usage{InputTokens: 200, OutputTokens: 100, TotalTokens: 300}, 2*time.Second)

	stats := tracker.Stats()
	if stats.TotalCalls != 2 {
		t.Errorf("expected 2 calls, got %d", stats.TotalCalls)
	}
	if stats.TotalInputTokens != 300 {
		t.Errorf("expected 300 input tokens, got %d", stats.TotalInputTokens)
	}
	if stats.TotalOutputTokens != 150 {
		t.Errorf("expected 150 output tokens, got %d", stats.TotalOutputTokens)
	}
}

func TestTokenUsageTrackerNil(t *testing.T) {
	var tracker *TokenUsageTracker
	// Should not panic
	tracker.Track("gpt-4", Usage{InputTokens: 100}, time.Second)
	stats := tracker.Stats()
	if stats.TotalCalls != 0 {
		t.Errorf("expected 0 calls for nil tracker, got %d", stats.TotalCalls)
	}
}

type mockProvider struct {
	chatResp   ChatResponse
	chatErr    error
	streamResp []StreamChunk
}

func (m *mockProvider) Chat(_ context.Context, _ ChatRequest) (ChatResponse, error) {
	return m.chatResp, m.chatErr
}

func (m *mockProvider) Stream(_ context.Context, _ ChatRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, len(m.streamResp))
	for _, c := range m.streamResp {
		ch <- c
	}
	close(ch)
	return ch, nil
}

func TestTrackingProviderChat(t *testing.T) {
	tracker := NewTokenUsageTracker(true)
	inner := &mockProvider{
		chatResp: ChatResponse{
			Model: "gpt-4",
			Usage: Usage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150},
		},
	}

	provider := NewTrackingProvider(inner, tracker)
	req := ChatRequest{Model: "gpt-4", Messages: []models.Message{{Role: "user", Content: "hello"}}}
	_, err := provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stats := tracker.Stats()
	if stats.TotalCalls != 1 {
		t.Errorf("expected 1 call tracked, got %d", stats.TotalCalls)
	}
	if stats.TotalInputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", stats.TotalInputTokens)
	}
}

func TestTrackingProviderStream(t *testing.T) {
	tracker := NewTokenUsageTracker(true)
	usage := Usage{InputTokens: 200, OutputTokens: 100, TotalTokens: 300}
	inner := &mockProvider{
		streamResp: []StreamChunk{
			{Delta: "Hello"},
			{Delta: " world", Done: true, Usage: &usage},
		},
	}

	provider := NewTrackingProvider(inner, tracker)
	req := ChatRequest{Model: "gpt-4", Messages: []models.Message{{Role: "user", Content: "hello"}}}
	ch, err := provider.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Drain channel
	for range ch {
	}

	stats := tracker.Stats()
	if stats.TotalCalls != 1 {
		t.Errorf("expected 1 call tracked, got %d", stats.TotalCalls)
	}
	if stats.TotalInputTokens != 200 {
		t.Errorf("expected 200 input tokens, got %d", stats.TotalInputTokens)
	}
}

func TestNewTrackingProviderNilTracker(t *testing.T) {
	inner := &mockProvider{}
	provider := NewTrackingProvider(inner, nil)
	// Should return inner directly when tracker is nil
	if provider != inner {
		t.Error("expected inner provider to be returned when tracker is nil")
	}
}
