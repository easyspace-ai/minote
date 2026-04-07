package notex

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/easyspace-ai/minote/pkg/docreaderclient"
	"github.com/easyspace-ai/minote/pkg/milvusutil"
	"github.com/easyspace-ai/minote/pkg/redisutil"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON[T any](r *http.Request, out *T) error {
	if r.Body == nil {
		return io.EOF
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(out)
}

func pathInt64(r *http.Request, name string) (int64, bool) {
	value := strings.TrimSpace(r.PathValue(name))
	if value == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// sanitizeTextForPostgres strips NUL bytes. Postgres text (UTF8) rejects U+0000 and returns SQLSTATE 22021.
func sanitizeTextForPostgres(s string) string {
	return strings.ReplaceAll(s, "\x00", "")
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func hashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func ensureExtension(name string, ext string) string {
	if strings.HasSuffix(strings.ToLower(name), strings.ToLower(ext)) {
		return name
	}
	return name + ext
}

func (s *Server) ensureDefaultLibrary(uid int64) *Library {
	libs := s.librariesByUser[uid]
	if len(libs) > 0 {
		return libs[0]
	}
	lib := &Library{ID: s.nextLibraryID, Name: "Default Library", ChunkSize: 800, ChunkOverlap: 200}
	s.nextLibraryID++
	s.librariesByUser[uid] = []*Library{lib}
	return lib
}

func (s *Server) ensureDefaultAgent(uid int64) *Agent {
	agents := s.agentsByUser[uid]
	if len(agents) > 0 {
		return agents[0]
	}
	now := nowRFC3339()
	agent := &Agent{ID: s.nextAgentID, Name: "General Assistant", Description: "Default assistant for Notex", Prompt: defaultAgentPrompt, CreatedAt: now, UpdatedAt: now}
	s.nextAgentID++
	s.agentsByUser[uid] = []*Agent{agent}
	return agent
}

func (s *Server) materialFileDir(projectID int64) string {
	return filepath.Join(s.cfg.DataRoot, "projects", fmt.Sprintf("%d", projectID), "materials")
}

func (s *Server) writeMaterialFile(projectID int64, filename string, body []byte) (string, error) {
	dir := s.materialFileDir(projectID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func decodeBase64Length(data string) int {
	if idx := strings.Index(data, ","); idx >= 0 {
		data = data[idx+1:]
	}
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return 0
	}
	return len(b)
}

// mergeInt64Dedupe merges two id lists (e.g. studio_document_ids + chat_document_ids) without duplicates.
func mergeInt64Dedupe(a, b []int64) []int64 {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	seen := make(map[int64]struct{}, len(a)+len(b))
	for _, x := range a {
		if x > 0 {
			seen[x] = struct{}{}
		}
	}
	out := append([]int64(nil), a...)
	for _, x := range b {
		if x <= 0 {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r != nil {
		ctx = r.Context()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"mode":   "notex",
		"components": map[string]string{
			"redis":     redisutil.Status(ctx),
			"milvus":    milvusutil.HealthStatus(ctx),
			"docreader": docreaderclient.ConnectionStatus(ctx),
		},
	})
}