package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/easyspace-ai/minote/pkg/models"
)

func TestTavilySearchHandlerMissingQuery(t *testing.T) {
	call := models.ToolCall{
		ID:        "test-1",
		Name:      "tavily_search",
		Arguments: map[string]any{},
	}
	_, err := TavilySearchHandler(context.Background(), call)
	if err == nil {
		t.Fatal("expected error for missing query")
	}
	if !contains(err.Error(), "query is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTavilySearchHandlerMissingAPIKey(t *testing.T) {
	t.Setenv("TAVILY_API_KEY", "")

	call := models.ToolCall{
		ID:        "test-2",
		Name:      "tavily_search",
		Arguments: map[string]any{"query": "test query"},
	}
	_, err := TavilySearchHandler(context.Background(), call)
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
	if !contains(err.Error(), "TAVILY_API_KEY") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSearchTavilyWithMockServer(t *testing.T) {
	mockResp := tavilySearchResponse{
		Query: "golang best practices",
		Results: []tavilySearchResult{
			{
				Title:   "Go Best Practices",
				URL:     "https://example.com/go-best-practices",
				Content: "Here are the best practices for Go programming...",
				Score:   0.95,
			},
			{
				Title:   "Effective Go",
				URL:     "https://go.dev/doc/effective_go",
				Content: "Go is a new language...",
				Score:   0.90,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var req tavilySearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.APIKey != "test-api-key" {
			t.Errorf("expected api_key test-api-key, got %s", req.APIKey)
		}
		if req.Query != "golang best practices" {
			t.Errorf("expected query 'golang best practices', got %s", req.Query)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	// Override the Tavily URL for testing
	oldURL := tavilySearchURL
	// We can't easily override the const, so test via searchTavily directly
	_ = oldURL

	results, err := searchTavilyWithURL(context.Background(), server.URL, "test-api-key", "golang best practices", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Title != "Go Best Practices" {
		t.Errorf("expected title 'Go Best Practices', got %s", results[0].Title)
	}
	if results[1].URL != "https://go.dev/doc/effective_go" {
		t.Errorf("expected URL https://go.dev/doc/effective_go, got %s", results[1].URL)
	}
}

func TestTavilySearchErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid API key"}`))
	}))
	defer server.Close()

	_, err := searchTavilyWithURL(context.Background(), server.URL, "bad-key", "test", 5)
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
	if !contains(err.Error(), "401") {
		t.Errorf("expected status 401 in error, got: %v", err)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
