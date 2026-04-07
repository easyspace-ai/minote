// Package milvusutil provides lightweight Milvus standalone health checks via HTTP (no Milvus SDK).
package milvusutil

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"
)

// HealthStatus returns "ok", "skipped", or "error: ..." for use in health and vector-inspect metadata.
func HealthStatus(ctx context.Context) string {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("MILVUS_HEALTH_CHECK")), "false") {
		return "skipped"
	}
	url := strings.TrimSpace(os.Getenv("MILVUS_HTTP_HEALTH"))
	if url == "" && strings.TrimSpace(os.Getenv("MILVUS_ADDRESS")) == "" {
		return "skipped"
	}
	if url == "" {
		port := strings.TrimSpace(os.Getenv("MILVUS_HEALTH_PORT"))
		if port == "" {
			port = "9091"
		}
		url = "http://127.0.0.1:" + port + "/healthz"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "error: " + err.Error()
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "error: " + err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "error: http " + resp.Status
	}
	return "ok"
}

// AddressConfigured is true when MILVUS_ADDRESS is non-empty (for UI: vector path planned / wired later).
func AddressConfigured() bool {
	return strings.TrimSpace(os.Getenv("MILVUS_ADDRESS")) != ""
}
