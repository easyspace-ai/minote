// Package redisutil provides optional connectivity checks for Redis (docker compose).
package redisutil

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Status returns a short status for health endpoints: "ok", "skipped", or "error: ...".
func Status(ctx context.Context) string {
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		return "skipped"
	}
	db := 0
	if s := strings.TrimSpace(os.Getenv("REDIS_DB")); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			db = n
		}
	}
	pass := strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))
	user := strings.TrimSpace(os.Getenv("REDIS_USERNAME"))

	opt := &redis.Options{
		Addr:     addr,
		Password: pass,
		DB:       db,
	}
	if user != "" {
		opt.Username = user
	}
	c := redis.NewClient(opt)
	defer func() { _ = c.Close() }()

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := c.Ping(pingCtx).Err(); err != nil {
		return "error: " + err.Error()
	}
	return "ok"
}
