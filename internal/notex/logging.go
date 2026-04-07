package notex

import (
	"context"
	"os"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

var appLogger = newNotexLogger()

func newNotexLogger() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(os.Stdout)
	l.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
		DisableQuote:    true,
	})
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug":
		l.SetLevel(logrus.DebugLevel)
	case "warn", "warning":
		l.SetLevel(logrus.WarnLevel)
	case "error":
		l.SetLevel(logrus.ErrorLevel)
	default:
		l.SetLevel(logrus.InfoLevel)
	}
	return l
}

func (s *Server) logInfo(ctx context.Context, msg string, fields map[string]any) {
	notexLogEntry(ctx, fields).Info(msg)
}

func (s *Server) logWarn(ctx context.Context, msg string, fields map[string]any) {
	notexLogEntry(ctx, fields).Warn(msg)
}

func (s *Server) logError(ctx context.Context, msg string, fields map[string]any) {
	notexLogEntry(ctx, fields).Error(msg)
}

func notexLogEntry(ctx context.Context, fields map[string]any) *logrus.Entry {
	entry := logrus.NewEntry(appLogger)
	if rid := requestIDFromContext(ctx); rid != "" {
		entry = entry.WithField("request_id", rid)
	}
	// 自动添加user_id字段
	if uid, ok := ctx.Value(userIDKey{}).(int64); ok && uid > 0 {
		entry = entry.WithField("user_id", uid)
	}
	// stable field order for readability in dev terminals
	if len(fields) > 0 {
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			entry = entry.WithField(k, fields[k])
		}
	}
	return entry
}

