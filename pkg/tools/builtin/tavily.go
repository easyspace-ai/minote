package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/easyspace-ai/minote/pkg/models"
)

const (
	tavilySearchURL        = "https://api.tavily.com/search"
	defaultTavilyMaxResults = 5
)

// TavilySearchHandler performs web search using the Tavily API.
// Requires TAVILY_API_KEY environment variable to be set.
func TavilySearchHandler(ctx context.Context, call models.ToolCall) (models.ToolResult, error) {
	query, ok := call.Arguments["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return models.ToolResult{CallID: call.ID, ToolName: call.Name}, fmt.Errorf("query is required")
	}
	query = strings.TrimSpace(query)

	maxResults := defaultTavilyMaxResults
	if raw, ok := call.Arguments["max_results"].(float64); ok && raw > 0 {
		maxResults = int(raw)
	}
	if maxResults <= 0 {
		maxResults = defaultTavilyMaxResults
	}
	if maxResults > 20 {
		maxResults = 20
	}

	apiKey := strings.TrimSpace(os.Getenv("TAVILY_API_KEY"))
	if apiKey == "" {
		return models.ToolResult{CallID: call.ID, ToolName: call.Name},
			fmt.Errorf("TAVILY_API_KEY environment variable is not set")
	}

	results, err := searchTavily(ctx, apiKey, query, maxResults)
	if err != nil {
		return models.ToolResult{CallID: call.ID, ToolName: call.Name},
			fmt.Errorf("tavily search failed: %w", err)
	}

	body, err := json.Marshal(webSearchResponse{
		Query:        query,
		TotalResults: len(results),
		Results:      results,
	})
	if err != nil {
		return models.ToolResult{CallID: call.ID, ToolName: call.Name},
			fmt.Errorf("encode search results: %w", err)
	}

	return models.ToolResult{
		CallID:   call.ID,
		ToolName: call.Name,
		Status:   models.CallStatusCompleted,
		Content:  string(body),
	}, nil
}

type tavilySearchRequest struct {
	APIKey     string `json:"api_key"`
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
}

type tavilySearchResponse struct {
	Query   string               `json:"query"`
	Results []tavilySearchResult `json:"results"`
}

type tavilySearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

func searchTavily(ctx context.Context, apiKey, query string, maxResults int) ([]webSearchResult, error) {
	return searchTavilyWithURL(ctx, tavilySearchURL, apiKey, query, maxResults)
}

func searchTavilyWithURL(ctx context.Context, apiURL, apiKey, query string, maxResults int) ([]webSearchResult, error) {
	reqBody, err := json.Marshal(tavilySearchRequest{
		APIKey:     apiKey,
		Query:      query,
		MaxResults: maxResults,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tavily request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("tavily API error (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var tavilyResp tavilySearchResponse
	if err := json.Unmarshal(body, &tavilyResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	results := make([]webSearchResult, 0, len(tavilyResp.Results))
	for _, r := range tavilyResp.Results {
		results = append(results, webSearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Content: r.Content,
		})
	}

	return results, nil
}

// TavilySearchTool returns the Tavily web search tool definition.
// The tool is only functional when TAVILY_API_KEY is set.
func TavilySearchTool() models.Tool {
	return models.Tool{
		Name:        "tavily_search",
		Description: "Search the web using Tavily API for high-quality, relevant results. Requires TAVILY_API_KEY.",
		Groups:      []string{"builtin", "web"},
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":       map[string]any{"type": "string", "description": "Search query"},
				"max_results": map[string]any{"type": "number", "description": "Maximum number of results (default 5, max 20)"},
			},
			"required": []any{"query"},
		},
		Handler: TavilySearchHandler,
	}
}

// TavilyAvailable returns true if the Tavily API key is configured.
func TavilyAvailable() bool {
	return strings.TrimSpace(os.Getenv("TAVILY_API_KEY")) != ""
}
