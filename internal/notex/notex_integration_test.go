package notex

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type testClient struct {
	baseURL string
	t       *testing.T
	token   string
}

func newTestClient(t *testing.T, baseURL string) *testClient {
	t.Helper()
	return &testClient{t: t, baseURL: baseURL}
}

func (c *testClient) withToken(token string) *testClient {
	return &testClient{t: c.t, baseURL: c.baseURL, token: token}
}

func (c *testClient) request(method string, path string, body any) (*http.Response, []byte) {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatalf("read response body: %v", err)
	}
	return resp, data
}

func decodeJSON[T any](t *testing.T, data []byte) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("decode json: %v; body=%s", err, string(data))
	}
	return out
}

func setupNotexTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	dataRoot := t.TempDir()
	server, err := NewServer(Config{Addr: ":0", DataRoot: dataRoot, AuthRequired: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(func() {
		httpServer.Close()
		_ = server.Shutdown(context.Background())
	})
	return server, httpServer
}

func registerUser(t *testing.T, client *testClient, email string) string {
	t.Helper()
	resp, data := client.request(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": "secret123",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", resp.StatusCode, string(data))
	}
	payload := decodeJSON[map[string]any](t, data)
	token, _ := payload["token"].(string)
	if token == "" {
		t.Fatalf("missing token in register response: %s", string(data))
	}
	return token
}

func TestNotexMainFlow(t *testing.T) {
	_, httpServer := setupNotexTestServer(t)
	client := newTestClient(t, httpServer.URL)
	token := registerUser(t, client, "flow@example.com")
	authed := client.withToken(token)

	resp, data := authed.request(http.MethodGet, "/api/v1/agents", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("agents list status=%d body=%s", resp.StatusCode, string(data))
	}
	agents := decodeJSON[[]Agent](t, data)
	if len(agents) == 0 {
		t.Fatal("expected default agent")
	}

	resp, data = authed.request(http.MethodPost, "/api/v1/projects", map[string]string{
		"name":        "Demo Project",
		"description": "desc",
		"category":    "note",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project status=%d body=%s", resp.StatusCode, string(data))
	}
	project := decodeJSON[Project](t, data)

	resp, data = authed.request(http.MethodPost, "/api/v1/conversations", map[string]any{
		"agent_id":    agents[0].ID,
		"name":        "Demo Chat",
		"library_ids": []int64{project.LibraryID},
		"chat_mode":   "chat",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create conversation status=%d body=%s", resp.StatusCode, string(data))
	}
	conversation := decodeJSON[Conversation](t, data)

	resp, data = authed.request(http.MethodPost, "/api/v1/chat/messages", map[string]any{
		"conversation_id": conversation.ID,
		"content":         "hello notex",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("chat message status=%d body=%s", resp.StatusCode, string(data))
	}

	resp, data = authed.request(http.MethodGet, "/api/v1/conversations/"+jsonNumber(conversation.ID)+"/messages", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("messages status=%d body=%s", resp.StatusCode, string(data))
	}
	messages := decodeJSON[[]Message](t, data)
	if len(messages) != 2 {
		t.Fatalf("messages len=%d want=2 body=%s", len(messages), string(data))
	}
	if messages[0].Role != "user" || messages[1].Role != "assistant" {
		t.Fatalf("unexpected roles: %+v", messages)
	}
}

func TestNotexConversationPatchAndDelete(t *testing.T) {
	_, httpServer := setupNotexTestServer(t)
	client := newTestClient(t, httpServer.URL)
	token := registerUser(t, client, "convpatch@example.com")
	authed := client.withToken(token)

	resp, data := authed.request(http.MethodGet, "/api/v1/agents", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("agents list status=%d body=%s", resp.StatusCode, string(data))
	}
	agents := decodeJSON[[]Agent](t, data)

	resp, data = authed.request(http.MethodPost, "/api/v1/projects", map[string]string{"name": "Conv Patch Project"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project status=%d body=%s", resp.StatusCode, string(data))
	}
	project := decodeJSON[Project](t, data)

	resp, data = authed.request(http.MethodPost, "/api/v1/conversations", map[string]any{
		"agent_id":    agents[0].ID,
		"name":        "Before",
		"library_ids": []int64{project.LibraryID},
		"chat_mode":   "chat",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create conversation status=%d body=%s", resp.StatusCode, string(data))
	}
	conversation := decodeJSON[Conversation](t, data)

	resp, data = authed.request(http.MethodPatch, "/api/v1/conversations/"+jsonNumber(conversation.ID), map[string]any{
		"name": "After Rename",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch conversation status=%d body=%s", resp.StatusCode, string(data))
	}
	renamed := decodeJSON[Conversation](t, data)
	if renamed.Name != "After Rename" {
		t.Fatalf("patch name: got %q", renamed.Name)
	}

	resp, data = authed.request(http.MethodDelete, "/api/v1/conversations/"+jsonNumber(conversation.ID), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete conversation status=%d body=%s", resp.StatusCode, string(data))
	}
	okBody := decodeJSON[map[string]bool](t, data)
	if !okBody["ok"] {
		t.Fatalf("delete response: %v", okBody)
	}

	resp, data = authed.request(http.MethodPatch, "/api/v1/conversations/"+jsonNumber(conversation.ID), map[string]any{
		"name": "gone",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("patch after delete want 404 got %d body=%s", resp.StatusCode, string(data))
	}
}

func TestNotexStudioConversationHiddenFromList(t *testing.T) {
	_, httpServer := setupNotexTestServer(t)
	client := newTestClient(t, httpServer.URL)
	token := registerUser(t, client, "studioconv@example.com")
	authed := client.withToken(token)

	resp, data := authed.request(http.MethodGet, "/api/v1/agents", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("agents list status=%d body=%s", resp.StatusCode, string(data))
	}
	agents := decodeJSON[[]Agent](t, data)

	resp, data = authed.request(http.MethodPost, "/api/v1/projects", map[string]string{"name": "Studio Hidden Project"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project status=%d body=%s", resp.StatusCode, string(data))
	}
	project := decodeJSON[Project](t, data)
	libID := project.LibraryID

	resp, data = authed.request(http.MethodPost, "/api/v1/conversations/ensure-studio", map[string]any{
		"agent_id":    agents[0].ID,
		"library_ids": []int64{libID},
		"chat_mode":   "chat",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ensure-studio status=%d body=%s", resp.StatusCode, string(data))
	}
	studioConv := decodeJSON[Conversation](t, data)
	if !studioConv.StudioOnly {
		t.Fatalf("ensure-studio conversation should be studio_only: %+v", studioConv)
	}

	resp, data = authed.request(http.MethodGet, "/api/v1/conversations?agent_id="+jsonNumber(agents[0].ID)+"&agent_type=eino", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list conversations status=%d body=%s", resp.StatusCode, string(data))
	}
	list := decodeJSON[[]Conversation](t, data)
	for _, c := range list {
		if c.ID == studioConv.ID {
			t.Fatalf("studio_only conversation %d should not appear in list", studioConv.ID)
		}
	}
}

func TestNotexProjectStudioScope(t *testing.T) {
	_, httpServer := setupNotexTestServer(t)
	client := newTestClient(t, httpServer.URL)
	token := registerUser(t, client, "scope@example.com")
	authed := client.withToken(token)

	resp, data := authed.request(http.MethodPost, "/api/v1/projects", map[string]string{"name": "Scope Project"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project status=%d body=%s", resp.StatusCode, string(data))
	}
	project := decodeJSON[Project](t, data)

	resp, data = authed.request(http.MethodPatch, "/api/v1/projects/"+jsonNumber(project.ID), map[string]any{
		"studio_scope": map[string]any{
			"includeChat":     true,
			"chatSummaryOnly": true,
			"chatMaxMessages": "12",
			"customExtra":     "面向初学者",
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch studio_scope status=%d body=%s", resp.StatusCode, string(data))
	}
	updated := decodeJSON[Project](t, data)
	if !updated.StudioScope.IncludeChat || !updated.StudioScope.ChatSummaryOnly {
		t.Fatalf("studio_scope not applied: %+v", updated.StudioScope)
	}
	if updated.StudioScope.ChatMaxMessages != "12" || updated.StudioScope.CustomExtra != "面向初学者" {
		t.Fatalf("studio_scope fields: %+v", updated.StudioScope)
	}

	resp, data = authed.request(http.MethodGet, "/api/v1/projects/"+jsonNumber(project.ID), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get project status=%d body=%s", resp.StatusCode, string(data))
	}
	again := decodeJSON[Project](t, data)
	if again.StudioScope.ChatMaxMessages != "12" || again.StudioScope.CustomExtra != "面向初学者" {
		t.Fatalf("persisted scope lost: %+v", again.StudioScope)
	}
}

func TestNotexDocumentExtractAndChatDocumentIDs(t *testing.T) {
	_, httpServer := setupNotexTestServer(t)
	client := newTestClient(t, httpServer.URL)
	token := registerUser(t, client, "docextract@example.com")
	authed := client.withToken(token)

	resp, data := authed.request(http.MethodGet, "/api/v1/agents", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("agents list status=%d body=%s", resp.StatusCode, string(data))
	}
	agents := decodeJSON[[]Agent](t, data)

	resp, data = authed.request(http.MethodPost, "/api/v1/projects", map[string]string{
		"name":        "Doc Chat Project",
		"description": "d",
		"category":    "note",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project status=%d body=%s", resp.StatusCode, string(data))
	}
	project := decodeJSON[Project](t, data)
	libraryID := project.LibraryID
	if libraryID <= 0 {
		t.Fatal("project missing library_id")
	}

	marker := "UNIQUE_CHAT_INJECT_42"
	fileData := base64.StdEncoding.EncodeToString([]byte(marker))
	resp, data = authed.request(http.MethodPost, "/api/v1/libraries/"+jsonNumber(libraryID)+"/documents/upload-browser", map[string]any{
		"files": []map[string]any{{"file_name": "note.txt", "base64_data": fileData}},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload doc status=%d body=%s", resp.StatusCode, string(data))
	}
	documents := decodeJSON[[]Document](t, data)
	docID := documents[0].ID

	deadline := time.Now().Add(5 * time.Second)
	var st string
	for time.Now().Before(deadline) {
		resp, data = authed.request(http.MethodPost, "/api/v1/libraries/"+jsonNumber(libraryID)+"/documents/query", map[string]any{})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("query status=%d body=%s", resp.StatusCode, string(data))
		}
		list := decodeJSON[[]Document](t, data)
		for i := range list {
			if list[i].ID == docID {
				st = list[i].ExtractionStatus
				break
			}
		}
		if st == DocExtractionCompleted {
			break
		}
		time.Sleep(40 * time.Millisecond)
	}
	if st != DocExtractionCompleted {
		t.Fatalf("extraction did not complete in time: status=%q", st)
	}

	resp, data = authed.request(http.MethodPost, "/api/v1/conversations", map[string]any{
		"agent_id":    agents[0].ID,
		"name":        "Doc Chat",
		"library_ids": []int64{project.LibraryID},
		"chat_mode":   "chat",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create conversation status=%d body=%s", resp.StatusCode, string(data))
	}
	conversation := decodeJSON[Conversation](t, data)

	resp, data = authed.request(http.MethodPost, "/api/v1/chat/messages", map[string]any{
		"conversation_id":   conversation.ID,
		"content":           "ping",
		"chat_document_ids": []int64{docID},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("chat message status=%d body=%s", resp.StatusCode, string(data))
	}

	resp, data = authed.request(http.MethodGet, "/api/v1/conversations/"+jsonNumber(conversation.ID)+"/messages", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("messages status=%d body=%s", resp.StatusCode, string(data))
	}
	messages := decodeJSON[[]Message](t, data)
	if len(messages) < 1 {
		t.Fatalf("expected messages, got %s", string(data))
	}
	userContent := messages[0].Content
	if !strings.Contains(userContent, marker) {
		t.Fatalf("user message missing injected doc text: %q", userContent)
	}
	if !strings.Contains(userContent, "【知识库文档：") {
		t.Fatalf("user message missing doc header: %q", userContent)
	}
}

func TestNotexStudioPPTAndHTMLMaterials(t *testing.T) {
	_, httpServer := setupNotexTestServer(t)
	client := newTestClient(t, httpServer.URL)
	token := registerUser(t, client, "studioppt@example.com")
	authed := client.withToken(token)

	resp, data := authed.request(http.MethodPost, "/api/v1/projects", map[string]string{"name": "Studio Materials"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project status=%d body=%s", resp.StatusCode, string(data))
	}
	project := decodeJSON[Project](t, data)

	// Slides require a skill-produced .pptx on the LangGraph thread; no markdown fallback.
	resp, data = authed.request(http.MethodPost, "/api/v1/projects/"+jsonNumber(project.ID)+"/materials/slides-pptx", map[string]any{
		"title":    "Deck",
		"markdown": "# Title slide\n",
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("slides-pptx without thread artifact: want 422, got %d body=%s", resp.StatusCode, string(data))
	}
	var slideErr map[string]string
	if err := json.Unmarshal(data, &slideErr); err != nil {
		t.Fatalf("decode slides-pptx error body: %v raw=%s", err, string(data))
	}
	if !strings.Contains(slideErr["error"], "pptx_skill_artifact_missing") {
		t.Fatalf("expected pptx_skill_artifact_missing in error, got %q", slideErr["error"])
	}

	resp, data = authed.request(http.MethodPost, "/api/v1/projects/"+jsonNumber(project.ID)+"/materials/studio-html", map[string]any{
		"title":    "Page",
		"markdown": "<p>Hello studio html</p>",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("studio-html status=%d body=%s", resp.StatusCode, string(data))
	}
	htmlMat := decodeJSON[Material](t, data)
	if htmlMat.Kind != "html" {
		t.Fatalf("kind=%q want html", htmlMat.Kind)
	}
}

func TestNotexDocumentPatch(t *testing.T) {
	_, httpServer := setupNotexTestServer(t)
	client := newTestClient(t, httpServer.URL)
	token := registerUser(t, client, "docpatch@example.com")
	authed := client.withToken(token)

	resp, data := authed.request(http.MethodGet, "/api/v1/libraries", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("libraries status=%d body=%s", resp.StatusCode, string(data))
	}
	libraries := decodeJSON[[]Library](t, data)
	libraryID := libraries[0].ID

	fileData := base64.StdEncoding.EncodeToString([]byte("hello"))
	resp, data = authed.request(http.MethodPost, "/api/v1/libraries/"+jsonNumber(libraryID)+"/documents/upload-browser", map[string]any{
		"files": []map[string]any{{"file_name": "a.txt", "base64_data": fileData}},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload doc status=%d body=%s", resp.StatusCode, string(data))
	}
	documents := decodeJSON[[]Document](t, data)
	docID := documents[0].ID

	resp, data = authed.request(http.MethodPatch, "/api/v1/documents/"+jsonNumber(docID), map[string]any{
		"original_name": "renamed.txt",
		"starred":       true,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", resp.StatusCode, string(data))
	}
	updated := decodeJSON[Document](t, data)
	if updated.OriginalName != "renamed.txt" || !updated.Starred {
		t.Fatalf("patch result: %+v", updated)
	}

	resp, data = authed.request(http.MethodPost, "/api/v1/libraries/"+jsonNumber(libraryID)+"/documents/query", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("query status=%d body=%s", resp.StatusCode, string(data))
	}
	list := decodeJSON[[]Document](t, data)
	var found *Document
	for i := range list {
		if list[i].ID == docID {
			found = &list[i]
			break
		}
	}
	if found == nil {
		t.Fatal("patched document missing from query")
	}
	if found.OriginalName != "renamed.txt" || !found.Starred {
		t.Fatalf("query row: %+v", found)
	}
}

func TestNotexOwnershipBoundaries(t *testing.T) {
	_, httpServer := setupNotexTestServer(t)
	client := newTestClient(t, httpServer.URL)
	ownerToken := registerUser(t, client, "owner@example.com")
	otherToken := registerUser(t, client, "other@example.com")
	owner := client.withToken(ownerToken)
	other := client.withToken(otherToken)

	resp, data := owner.request(http.MethodGet, "/api/v1/libraries", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("libraries status=%d body=%s", resp.StatusCode, string(data))
	}
	libraries := decodeJSON[[]Library](t, data)
	if len(libraries) == 0 {
		t.Fatal("expected default library")
	}
	libraryID := libraries[0].ID

	resp, data = owner.request(http.MethodPost, "/api/v1/projects", map[string]string{"name": "Private Project"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project status=%d body=%s", resp.StatusCode, string(data))
	}
	project := decodeJSON[Project](t, data)

	fileData := base64.StdEncoding.EncodeToString([]byte("top secret"))
	resp, data = owner.request(http.MethodPost, "/api/v1/libraries/"+jsonNumber(libraryID)+"/documents/upload-browser", map[string]any{
		"files": []map[string]any{{"file_name": "secret.txt", "base64_data": fileData}},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload doc status=%d body=%s", resp.StatusCode, string(data))
	}
	documents := decodeJSON[[]Document](t, data)
	if len(documents) != 1 {
		t.Fatalf("documents len=%d want=1", len(documents))
	}
	documentID := documents[0].ID

	resp, data = owner.request(http.MethodPost, "/api/v1/projects/"+jsonNumber(project.ID)+"/materials", map[string]any{
		"kind":   "report",
		"title":  "Private Report",
		"status": "ready",
		"payload": map[string]any{
			"summary": "confidential",
		},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create material status=%d body=%s", resp.StatusCode, string(data))
	}
	material := decodeJSON[Material](t, data)

	cases := []struct {
		name string
		path string
	}{
		{name: "project get", path: "/api/v1/projects/" + jsonNumber(project.ID)},
		{name: "documents query", path: "/api/v1/libraries/" + jsonNumber(libraryID) + "/documents/query"},
		{name: "document attachment", path: "/api/v1/documents/" + jsonNumber(documentID) + "/chat-attachment"},
		{name: "material get", path: "/api/v1/projects/" + jsonNumber(project.ID) + "/materials/" + jsonNumber(material.ID)},
		{name: "material download", path: "/api/v1/projects/" + jsonNumber(project.ID) + "/materials/" + jsonNumber(material.ID) + "/studio-file"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			method := http.MethodGet
			var body any
			if filepath.Base(tc.path) == "query" {
				method = http.MethodPost
				body = map[string]any{}
			}
			resp, data := other.request(method, tc.path, body)
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("status=%d want=404 body=%s", resp.StatusCode, string(data))
			}
		})
	}
}

func jsonNumber(v int64) string {
	return strconv.FormatInt(v, 10)
}

func TestNotexSkillsInstalled(t *testing.T) {
	dataRoot := t.TempDir()
	skillsRoot := filepath.Join(dataRoot, "skills-scan")
	if err := os.MkdirAll(filepath.Join(skillsRoot, "demo", "try"), 0o755); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(skillsRoot, "demo", "try", "SKILL.md")
	content := "---\nname: demo-skill\ndescription: integration test\nversion: \"1.0.0\"\n---\n\n# Demo\n"
	if err := os.WriteFile(skillPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	server, err := NewServer(Config{
		Addr:         ":0",
		DataRoot:     dataRoot,
		AuthRequired: true,
		SkillsPaths:  []string{skillsRoot},
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(func() {
		httpServer.Close()
		_ = server.Shutdown(context.Background())
	})

	client := newTestClient(t, httpServer.URL)
	token := registerUser(t, client, "skills@example.com")
	authed := client.withToken(token)

	resp, data := authed.request(http.MethodGet, "/api/v1/skills/installed", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("installed status=%d body=%s", resp.StatusCode, string(data))
	}
	list := decodeJSON[[]map[string]any](t, data)
	if len(list) != 1 {
		t.Fatalf("want 1 skill got %d: %s", len(list), string(data))
	}
	if list[0]["slug"] != "demo-try" {
		t.Fatalf("slug=%v", list[0]["slug"])
	}
	if list[0]["name"] != "demo-skill" {
		t.Fatalf("name=%v", list[0]["name"])
	}

	resp, data = authed.request(http.MethodGet, "/api/v1/skills/workspace", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("workspace status=%d body=%s", resp.StatusCode, string(data))
	}
	ws := decodeJSON[map[string]string](t, data)
	if ws["skills_dir"] != skillsRoot {
		t.Fatalf("skills_dir=%q want %q", ws["skills_dir"], skillsRoot)
	}

	resp, data = authed.request(http.MethodPost, "/api/v1/skills/demo-try/disable", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("disable status=%d body=%s", resp.StatusCode, string(data))
	}
	resp, data = authed.request(http.MethodGet, "/api/v1/skills/installed", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("installed2 status=%d body=%s", resp.StatusCode, string(data))
	}
	list = decodeJSON[[]map[string]any](t, data)
	if len(list) != 1 || list[0]["enabled"] != false {
		t.Fatalf("expected disabled: %s", string(data))
	}
}
