package notex

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// minimalPPTXBytesForTests is enough for isZipSignedPPTX (PK…) and material file write; not a real deck.
func minimalPPTXBytesForTests() []byte {
	return []byte("PK\x03\x04" + strings.Repeat("\x00", 24))
}

// newMockLangGraphPPTXHandler fakes the LangGraph compat endpoints that Notex calls via internal httptest routing.
// See studio_skill_pptx.go: GET /api/threads/{id}/files and GET artifact with ?download=true.
func newMockLangGraphPPTXHandler(t *testing.T, threadID string, pptx []byte) http.Handler {
	t.Helper()
	artifactPath := "mnt/user-data/outputs/studio-automation-test.pptx"
	artifactURL := "/api/threads/" + threadID + "/artifacts/" + artifactPath
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case r.Method == http.MethodPost && p == "/threads":
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]string{"thread_id": threadID}); err != nil {
				t.Errorf("encode thread response: %v", err)
			}
		case r.Method == http.MethodGet && p == "/api/threads/"+threadID+"/files":
			w.Header().Set("Content-Type", "application/json")
			files := []map[string]any{{
				"path":          artifactPath,
				"artifact_url":  artifactURL,
				"created_at":    time.Now().UTC().Format(time.RFC3339Nano),
			}}
			if err := json.NewEncoder(w).Encode(map[string]any{"files": files}); err != nil {
				t.Errorf("encode files response: %v", err)
			}
		case r.Method == http.MethodGet && strings.HasPrefix(p, "/api/threads/"+threadID+"/artifacts/") && r.URL.Query().Get("download") == "true":
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
			_, _ = w.Write(pptx)
		default:
			t.Logf("mock langgraph unhandled: %s %s", r.Method, p)
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func authMeUserID(t *testing.T, authed *testClient) int64 {
	t.Helper()
	resp, data := authed.request(http.MethodGet, "/api/v1/auth/me", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("auth/me status=%d body=%s", resp.StatusCode, string(data))
	}
	body := decodeJSON[map[string]any](t, data)
	u, _ := body["user"].(map[string]any)
	id, _ := u["id"].(float64)
	return int64(id)
}

// TestStudioSlidesArtifactAndPptxWithMockLangGraph covers:
// - GET .../studio/slides-artifact-status when thread lists a .pptx
// - POST .../materials/slides-pptx copying bytes from the mock artifact
//
// Run: go test ./internal/notex -run TestStudioSlidesArtifactAndPptxWithMockLangGraph -count=1
func TestStudioSlidesArtifactAndPptxWithMockLangGraph(t *testing.T) {
	server, httpServer := setupNotexTestServer(t)
	threadID := "automation-lg-thread-pptx"
	pptx := minimalPPTXBytesForTests()
	server.SetAIHandler(newMockLangGraphPPTXHandler(t, threadID, pptx))

	client := newTestClient(t, httpServer.URL)
	token := registerUser(t, client, "pptx-automation@example.com")
	authed := client.withToken(token)
	uid := authMeUserID(t, authed)

	resp, data := authed.request(http.MethodGet, "/api/v1/agents", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("agents list status=%d body=%s", resp.StatusCode, string(data))
	}
	agents := decodeJSON[[]Agent](t, data)

	resp, data = authed.request(http.MethodPost, "/api/v1/projects", map[string]string{"name": "PPTX Automation"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project status=%d body=%s", resp.StatusCode, string(data))
	}
	project := decodeJSON[Project](t, data)

	resp, data = authed.request(http.MethodPost, "/api/v1/conversations", map[string]any{
		"agent_id":    agents[0].ID,
		"name":        "slides-automation",
		"library_ids": []int64{project.LibraryID},
		"chat_mode":   "chat",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create conversation status=%d body=%s", resp.StatusCode, string(data))
	}
	conversation := decodeJSON[Conversation](t, data)

	ctx := context.Background()
	if err := server.setConversationThreadID(ctx, uid, conversation.ID, threadID); err != nil {
		t.Fatalf("setConversationThreadID: %v", err)
	}

	resp, data = authed.request(http.MethodGet, "/api/v1/conversations/"+jsonNumber(conversation.ID)+"/studio/slides-artifact-status", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("slides-artifact-status status=%d body=%s", resp.StatusCode, string(data))
	}
	st := decodeJSON[map[string]any](t, data)
	ready, _ := st["ready"].(bool)
	if !ready {
		t.Fatalf("expected ready=true, got %v", st)
	}
	path, _ := st["artifact_path"].(string)
	if !strings.Contains(path, "user-data/outputs/") {
		t.Fatalf("artifact_path should prefer outputs path, got %q", path)
	}

	resp, data = authed.request(http.MethodPost, "/api/v1/projects/"+project.ID+"/materials/slides-pptx", map[string]any{
		"title":             "Automation Deck",
		"markdown":          "# Placeholder\n",
		"conversation_id":   conversation.ID,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("slides-pptx want 201 got %d %s", resp.StatusCode, string(data))
	}
	mat := decodeJSON[Material](t, data)
	if mat.Kind != "slides" {
		t.Fatalf("material kind=%q want slides", mat.Kind)
	}
	if mat.Status != "ready" {
		t.Fatalf("material status=%q want ready", mat.Status)
	}
}
