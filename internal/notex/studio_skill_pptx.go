package notex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"time"
)

// Max bytes we'll accept when copying a skill-produced PPTX from the LangGraph thread (fail open → outline fallback).
const maxStudioSkillPPTXBytes = 48 << 20

type threadFilesListEntry struct {
	Path        string    `json:"path"`
	ArtifactURL string    `json:"artifact_url"`
	CreatedAt   time.Time `json:"created_at"`
}

// preferredStudioPPTXFromConversation loads the conversation's LangGraph thread (if any) and returns the newest
// suitable .zip-based PPTX emitted under outputs/ or workspace/. Used for Studio "skill path" before Markdown outline fallback.
func (s *Server) preferredStudioPPTXFromConversation(ctx context.Context, uid, conversationID int64) ([]byte, string, error) {
	if conversationID <= 0 || s.aiHandler == nil {
		if conversationID <= 0 {
			s.logger.Printf("[studio-pptx] skip skill lookup: conversation_id=%d invalid request_id=%s", conversationID, requestIDFromContext(ctx))
		} else {
			s.logger.Printf("[studio-pptx] skip skill lookup: ai handler not set request_id=%s", requestIDFromContext(ctx))
		}
		return nil, "", nil
	}
	conv, err := s.getConversation(ctx, uid, conversationID)
	if err != nil || conv == nil {
		if err != nil {
			s.logger.Printf("[studio-pptx] get conversation failed: conversation_id=%d request_id=%s err=%v", conversationID, requestIDFromContext(ctx), err)
		} else {
			s.logger.Printf("[studio-pptx] conversation not found: conversation_id=%d request_id=%s", conversationID, requestIDFromContext(ctx))
		}
		return nil, "", nil
	}
	tid := strings.TrimSpace(conv.ThreadID)
	if tid == "" {
		s.logger.Printf("[studio-pptx] conversation has empty thread_id: conversation_id=%d request_id=%s", conversationID, requestIDFromContext(ctx))
		return nil, "", nil
	}
	s.logger.Printf("[studio-pptx] lookup thread artifacts: conversation_id=%d thread_id=%s request_id=%s", conversationID, tid, requestIDFromContext(ctx))
	return s.fetchNewestThreadSkillPPTX(tid)
}

// listSortedThreadPPTXFileCandidates returns .pptx rows from the thread file list that have a non-empty artifact_url,
// sorted by the same path preference + recency rules as fetchNewestThreadSkillPPTX (ZIP validation happens on download).
func (s *Server) listSortedThreadPPTXFileCandidates(threadID string) []threadFilesListEntry {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" || s.aiHandler == nil {
		if threadID == "" {
			s.logger.Printf("[studio-pptx] fetch skipped: empty thread_id")
		}
		return nil
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/threads/"+threadID+"/files", nil)
	addInternalAuth(req)
	s.aiHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		s.logger.Printf("[studio-pptx] thread files request failed: thread_id=%s status=%d", threadID, rec.Code)
		return nil
	}
	var parsed struct {
		Files []threadFilesListEntry `json:"files"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil || len(parsed.Files) == 0 {
		if err != nil {
			s.logger.Printf("[studio-pptx] parse thread files failed: thread_id=%s err=%v", threadID, err)
		} else {
			s.logger.Printf("[studio-pptx] thread files empty: thread_id=%s", threadID)
		}
		return nil
	}
	s.logger.Printf("[studio-pptx] thread files fetched: thread_id=%s total=%d", threadID, len(parsed.Files))
	candidates := make([]threadFilesListEntry, 0, len(parsed.Files))
	for _, f := range parsed.Files {
		p := strings.TrimSpace(f.Path)
		if p == "" || !strings.HasSuffix(strings.ToLower(p), ".pptx") {
			continue
		}
		if strings.TrimSpace(f.ArtifactURL) == "" {
			continue
		}
		candidates = append(candidates, f)
	}
	if len(candidates) == 0 {
		s.logger.Printf("[studio-pptx] no pptx candidates: thread_id=%s", threadID)
		return nil
	}
	s.logger.Printf("[studio-pptx] pptx candidates: thread_id=%s %s", threadID, previewPPTXCandidates(candidates))
	sort.SliceStable(candidates, func(i, j int) bool {
		si, sj := pptxPathPreference(candidates[i].Path), pptxPathPreference(candidates[j].Path)
		if si != sj {
			return si > sj
		}
		return candidates[i].CreatedAt.After(candidates[j].CreatedAt)
	})
	return candidates
}

func (s *Server) fetchNewestThreadSkillPPTX(threadID string) ([]byte, string, error) {
	candidates := s.listSortedThreadPPTXFileCandidates(threadID)
	if len(candidates) == 0 {
		return nil, "", nil
	}
	chosen := candidates[0]
	rawURL := strings.TrimSpace(chosen.ArtifactURL)
	if !strings.HasPrefix(rawURL, "/") {
		s.logger.Printf("[studio-pptx] chosen artifact url is not relative: thread_id=%s path=%s url=%s", threadID, chosen.Path, rawURL)
		return nil, "", nil
	}
	downloadPath := rawURL + "?download=true"
	arec := httptest.NewRecorder()
	areq := httptest.NewRequest(http.MethodGet, downloadPath, nil)
	addInternalAuth(areq)
	s.aiHandler.ServeHTTP(arec, areq)
	if arec.Code != http.StatusOK {
		s.logger.Printf("[studio-pptx] artifact download failed: thread_id=%s path=%s status=%d", threadID, chosen.Path, arec.Code)
		return nil, "", nil
	}
	data, err := io.ReadAll(arec.Body)
	if err != nil {
		s.logger.Printf("[studio-pptx] artifact read failed: thread_id=%s path=%s err=%v", threadID, chosen.Path, err)
		return nil, "", nil
	}
	if len(data) == 0 || len(data) > maxStudioSkillPPTXBytes {
		s.logger.Printf("[studio-pptx] artifact size rejected: thread_id=%s path=%s bytes=%d", threadID, chosen.Path, len(data))
		return nil, "", nil
	}
	if !isZipSignedPPTX(data) {
		s.logger.Printf("[studio-pptx] artifact not zip/pptx signature: thread_id=%s path=%s bytes=%d", threadID, chosen.Path, len(data))
		return nil, "", nil
	}
	s.logger.Printf(
		"[studio-pptx] selected skill artifact: thread_id=%s path=%s skill_hint=%q bytes=%d",
		threadID,
		chosen.Path,
		skillHintFromArtifactPath(chosen.Path),
		len(data),
	)
	return data, chosen.Path, nil
}

func previewPPTXCandidates(cands []threadFilesListEntry) string {
	const max = 6
	parts := make([]string, 0, minInt(len(cands), max))
	for i := 0; i < len(cands) && i < max; i++ {
		f := cands[i]
		parts = append(parts, fmt.Sprintf("{path=%q created_at=%s}", f.Path, f.CreatedAt.Format(time.RFC3339)))
	}
	if len(cands) > max {
		parts = append(parts, fmt.Sprintf("... +%d more", len(cands)-max))
	}
	return strings.Join(parts, " ")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func skillHintFromArtifactPath(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return ""
	}
	parts := strings.Split(strings.ReplaceAll(p, "\\", "/"), "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "skills" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	if strings.Contains(strings.ToLower(p), "/user-data/outputs/") {
		return "user-data-outputs"
	}
	if strings.Contains(strings.ToLower(p), "/workspace/") {
		return "workspace-artifact"
	}
	return ""
}

func pptxPathPreference(p string) int {
	pl := strings.ToLower(p)
	switch {
	case strings.Contains(pl, "/user-data/outputs/"):
		return 3
	case strings.Contains(pl, "/user-data/workspace/"):
		return 2
	default:
		return 1
	}
}

func isZipSignedPPTX(b []byte) bool {
	// OOXML pptx is a ZIP container; reject obvious HTML/text mistaken as pptx.
	if len(b) < 4 {
		return false
	}
	if b[0] == 'P' && b[1] == 'K' {
		return true
	}
	return false
}

// studioSlidesSkillPPTXProbe is true when the conversation's LangGraph thread lists at least one .pptx with a relative
// artifact URL (same selection order as slides-pptx). Does not download bytes — used for client polling after agent stream ends.
func (s *Server) studioSlidesSkillPPTXProbe(ctx context.Context, uid, conversationID int64) (ready bool, artifactPath string) {
	if conversationID <= 0 || s.aiHandler == nil {
		return false, ""
	}
	conv, err := s.getConversation(ctx, uid, conversationID)
	if err != nil || conv == nil {
		return false, ""
	}
	tid := strings.TrimSpace(conv.ThreadID)
	if tid == "" {
		return false, ""
	}
	candidates := s.listSortedThreadPPTXFileCandidates(tid)
	if len(candidates) == 0 {
		return false, ""
	}
	chosen := candidates[0]
	rawURL := strings.TrimSpace(chosen.ArtifactURL)
	if !strings.HasPrefix(rawURL, "/") {
		s.logger.Printf("[studio-pptx] probe: chosen artifact url is not relative: thread_id=%s path=%s", tid, chosen.Path)
		return false, ""
	}
	return true, strings.TrimSpace(chosen.Path)
}
