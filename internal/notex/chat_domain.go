package notex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var studioPreferredSkillNames = []string{
	"ppt-agent-workflow-san",
	"PPT-as-code",
	"ppt-agent",
	"frontend-slides",
}

func (s *Server) listConversations(ctx context.Context, userID int64, agentID int64) ([]*Conversation, error) {
	if s.store != nil {
		return s.store.ListConversationsByUser(ctx, userID, agentID)
	}
	s.conversationMu.RLock()
	defer s.conversationMu.RUnlock()
	list := make([]*Conversation, 0)
	for _, c := range s.conversationsByUser[userID] {
		if c.StudioOnly {
			continue
		}
		if agentID > 0 && c.AgentID != agentID {
			continue
		}
		list = append(list, c)
	}
	return list, nil
}

func (s *Server) createConversation(ctx context.Context, userID int64, agentID int64, name string, libraryIDs []int64, chatMode string, studioOnly bool) (*Conversation, error) {
	if s.store != nil {
		return s.store.CreateConversation(ctx, userID, agentID, name, libraryIDs, chatMode, studioOnly)
	}
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()
	c := &Conversation{ID: s.nextConversationID, AgentID: agentID, Name: name, LibraryIDs: append([]int64(nil), libraryIDs...), ChatMode: chatMode, StudioOnly: studioOnly}
	s.nextConversationID++
	s.conversationsByUser[userID] = append(s.conversationsByUser[userID], c)
	return c, nil
}

func libraryIDsEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *Server) ensureStudioConversation(ctx context.Context, userID int64, agentID int64, libraryIDs []int64, chatMode string) (*Conversation, error) {
	if s.store != nil {
		return s.store.EnsureStudioConversation(ctx, userID, agentID, libraryIDs, chatMode)
	}
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()
	for _, c := range s.conversationsByUser[userID] {
		if c.AgentID != agentID || !c.StudioOnly {
			continue
		}
		if libraryIDsEqual(c.LibraryIDs, libraryIDs) {
			return c, nil
		}
	}
	c := &Conversation{
		ID:         s.nextConversationID,
		AgentID:    agentID,
		Name:       "Studio",
		LibraryIDs: append([]int64(nil), libraryIDs...),
		ChatMode:   strings.TrimSpace(chatMode),
		StudioOnly: true,
	}
	if c.ChatMode == "" {
		c.ChatMode = "chat"
	}
	s.nextConversationID++
	s.conversationsByUser[userID] = append(s.conversationsByUser[userID], c)
	return c, nil
}

func (s *Server) getConversation(ctx context.Context, userID int64, conversationID int64) (*Conversation, error) {
	if s.store != nil {
		return s.store.GetConversationByID(ctx, userID, conversationID)
	}
	s.conversationMu.RLock()
	defer s.conversationMu.RUnlock()
	for _, c := range s.conversationsByUser[userID] {
		if c.ID == conversationID {
			return c, nil
		}
	}
	return nil, nil
}

func (s *Server) listMessages(ctx context.Context, conversationID int64) ([]*Message, error) {
	if s.store != nil {
		return s.store.ListMessagesByConversation(ctx, conversationID)
	}
	s.messageMu.RLock()
	defer s.messageMu.RUnlock()
	return append([]*Message(nil), s.messagesByConv[conversationID]...), nil
}

func (s *Server) createMessage(ctx context.Context, conversationID int64, role string, content string, status string) (*Message, error) {
	if s.store != nil {
		return s.store.CreateMessage(ctx, conversationID, role, content, status)
	}
	s.messageMu.Lock()
	defer s.messageMu.Unlock()
	msg := &Message{ID: s.nextMessageID, ConversationID: conversationID, Role: role, Content: content, Status: status}
	s.nextMessageID++
	s.messagesByConv[conversationID] = append(s.messagesByConv[conversationID], msg)
	return msg, nil
}

func (s *Server) setConversationThreadID(ctx context.Context, userID int64, conversationID int64, threadID string) error {
	if s.store != nil {
		return s.store.SetConversationThreadID(ctx, userID, conversationID, threadID)
	}
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()
	s.convThreadIDs[conversationID] = threadID
	for _, list := range s.conversationsByUser {
		for _, c := range list {
			if c.ID == conversationID {
				c.ThreadID = threadID
			}
		}
	}
	return nil
}

func (s *Server) updateConversationLastMessage(ctx context.Context, userID int64, conversationID int64, lastMessage string) error {
	if s.store != nil {
		return s.store.UpdateConversationLastMessage(ctx, userID, conversationID, lastMessage)
	}
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()
	for _, list := range s.conversationsByUser {
		for _, c := range list {
			if c.ID == conversationID {
				c.LastMessage = lastMessage
			}
		}
	}
	return nil
}

func (s *Server) handleConversationsList(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	agentID, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("agent_id")), 10, 64)
	list, err := s.listConversations(r.Context(), uid, agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleConversationsCreate(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	var body struct {
		AgentID    int64   `json:"agent_id"`
		Name       string  `json:"name"`
		LibraryIDs []int64 `json:"library_ids"`
		ChatMode   string  `json:"chat_mode"`
		StudioOnly bool    `json:"studio_only"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = "New Conversation"
	}
	chatMode := strings.TrimSpace(body.ChatMode)
	if chatMode == "" {
		chatMode = "chat"
	}
	conv, err := s.createConversation(r.Context(), uid, body.AgentID, name, body.LibraryIDs, chatMode, body.StudioOnly)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, conv)
}

func (s *Server) handleConversationsEnsureStudio(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	var body struct {
		AgentID    int64   `json:"agent_id"`
		LibraryIDs []int64 `json:"library_ids"`
		ChatMode   string  `json:"chat_mode"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if body.AgentID <= 0 || len(body.LibraryIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "agent_id_and_library_ids_required"})
		return
	}
	chatMode := strings.TrimSpace(body.ChatMode)
	if chatMode == "" {
		chatMode = "chat"
	}
	conv, err := s.ensureStudioConversation(r.Context(), uid, body.AgentID, body.LibraryIDs, chatMode)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, conv)
}

func (s *Server) patchConversationName(ctx context.Context, userID int64, conversationID int64, name string) (*Conversation, error) {
	if s.store != nil {
		return s.store.PatchConversationName(ctx, userID, conversationID, name)
	}
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()
	for _, c := range s.conversationsByUser[userID] {
		if c.ID == conversationID {
			c.Name = name
			return c, nil
		}
	}
	return nil, nil
}

func (s *Server) deleteConversation(ctx context.Context, userID int64, conversationID int64) (bool, error) {
	if s.store != nil {
		return s.store.DeleteConversationForUser(ctx, userID, conversationID)
	}
	s.conversationMu.Lock()
	s.messageMu.Lock()
	defer s.conversationMu.Unlock()
	defer s.messageMu.Unlock()
	list := s.conversationsByUser[userID]
	for i, c := range list {
		if c.ID == conversationID {
			s.conversationsByUser[userID] = append(list[:i], list[i+1:]...)
			delete(s.messagesByConv, conversationID)
			delete(s.convThreadIDs, conversationID)
			return true, nil
		}
	}
	return false, nil
}

func (s *Server) handleConversationsPatch(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, pathOK := pathInt64(r, "id")
	if !pathOK || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_name"})
		return
	}
	conv, err := s.patchConversationName(r.Context(), uid, id, name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if conv == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, conv)
}

func (s *Server) handleConversationsDelete(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, pathOK := pathInt64(r, "id")
	if !pathOK || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	deleted, err := s.deleteConversation(r.Context(), uid, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !deleted {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleConversationsMessages(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireUserID(w, r); !ok {
		return
	}
	id, ok := pathInt64(r, "id")
	if !ok || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	msgs, err := s.listMessages(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (s *Server) handleConversationsEnsureThread(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, pathOK := pathInt64(r, "id")
	if !pathOK || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	conversation, err := s.getConversation(r.Context(), uid, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if conversation == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation_not_found"})
		return
	}
	tid, err := s.ensureLangGraphThread(r.Context(), uid, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"thread_id": tid})
}

// handleConversationsStudioSlidesArtifactStatus reports whether the LangGraph thread already exposes a skill .pptx
// (strict Agent+skill path); the web UI polls this after chat stream ends before POST .../slides-pptx.
func (s *Server) handleConversationsStudioSlidesArtifactStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	id, pathOK := pathInt64(r, "id")
	if !pathOK || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}
	conversation, err := s.getConversation(r.Context(), uid, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if conversation == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation_not_found"})
		return
	}
	ready, artifactPath := s.studioSlidesSkillPPTXProbe(r.Context(), uid, id)
	resp := map[string]any{"ready": ready}
	if artifactPath != "" {
		resp["artifact_path"] = artifactPath
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatMessages(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	var body struct {
		ConversationID int64  `json:"conversation_id"`
		Content        string `json:"content"`
		TabID          string `json:"tab_id"`
		RequestID      string `json:"request_id"`
		// Must match web: useChatSend sends studio_document_ids (snake_case like conversation_id).
		StudioDocumentIds []int64 `json:"studio_document_ids"`
		// Optional alias for the same injection path (aligned with LangGraph chat_document_ids).
		ChatDocumentIds []int64 `json:"chat_document_ids"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if body.ConversationID <= 0 || strings.TrimSpace(body.Content) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conversation_id_and_content_required"})
		return
	}
	requestID := strings.TrimSpace(body.RequestID)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	s.logger.Printf(
		"[chat] request conversation_id=%d tab_id=%q stream=%t request_id=%s body_bytes=%d doc_ids=%v",
		body.ConversationID,
		strings.TrimSpace(body.TabID),
		r.URL.Query().Get("stream") == "1" || strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/event-stream"),
		requestID,
		len(body.Content),
		mergeInt64Dedupe(body.StudioDocumentIds, body.ChatDocumentIds),
	)
	conversation, err := s.getConversation(r.Context(), uid, body.ConversationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if conversation == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversation_not_found"})
		return
	}
	userContent := strings.TrimSpace(body.Content)

	docIDs := mergeInt64Dedupe(body.StudioDocumentIds, body.ChatDocumentIds)

	// Studio / 资料勾选：与 LangGraph 运行路径共用注入逻辑（校验会话绑定库与用户）
	if prefix := strings.TrimSpace(s.StudioInjectionPrefixForLangGraph(r.Context(), uid, body.ConversationID, docIDs)); prefix != "" {
		userContent = prefix + "\n\n---\n\n" + userContent
	}

	if _, err := s.createMessage(r.Context(), body.ConversationID, "user", userContent, "done"); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	isStream := r.URL.Query().Get("stream") == "1" || strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/event-stream")
	if s.aiHandler == nil {
		s.chatFallback(r.Context(), w, uid, body.ConversationID, requestID, userContent, isStream)
		return
	}
	threadID, err := s.ensureLangGraphThread(r.Context(), uid, body.ConversationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !isStream {
		writeJSON(w, http.StatusOK, map[string]any{"request_id": requestID})
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming_not_supported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	s.streamFromLangGraph(r.Context(), w, flusher, uid, threadID, body.ConversationID, requestID, userContent, strings.TrimSpace(body.TabID))
}

func (s *Server) ensureLangGraphThread(ctx context.Context, userID int64, convID int64) (string, error) {
	conversation, err := s.getConversation(ctx, userID, convID)
	if err != nil {
		return "", err
	}
	if conversation == nil {
		return "", fmt.Errorf("conversation_not_found")
	}
	if strings.TrimSpace(conversation.ThreadID) != "" {
		return conversation.ThreadID, nil
	}
	tid := uuid.NewString()
	if s.aiHandler != nil {
		body, _ := json.Marshal(map[string]any{"metadata": map[string]any{"source": "notex", "conversation_id": convID, "user_id": userID}})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/threads", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addInternalAuth(req)
		s.aiHandler.ServeHTTP(rec, req)
		var result map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err == nil {
			if v, _ := result["thread_id"].(string); v != "" {
				tid = v
			}
		}
	}
	if err := s.setConversationThreadID(ctx, userID, convID, tid); err != nil {
		return "", err
	}
	return tid, nil
}

func addInternalAuth(r *http.Request) {
	if tok := strings.TrimSpace(os.Getenv("DEERFLOW_AUTH_TOKEN")); tok != "" {
		r.Header.Set("Authorization", "Bearer "+tok)
	}
}

type pipeResponseWriter struct {
	pw     *io.PipeWriter
	header http.Header
	code   int
}

func (p *pipeResponseWriter) Header() http.Header         { return p.header }
func (p *pipeResponseWriter) WriteHeader(code int)        { p.code = code }
func (p *pipeResponseWriter) Write(b []byte) (int, error) { return p.pw.Write(b) }
func (p *pipeResponseWriter) Flush()                      {}

func (s *Server) streamFromLangGraph(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, userID int64, threadID string, convID int64, requestID string, content string, tabID string) {
	writeSSE := func(event string, payload any) {
		b, _ := json.Marshal(payload)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
		flusher.Flush()
	}
	runRequest := map[string]any{
		"assistant_id": "",
		"input": map[string]any{
			"messages": []map[string]any{{"role": "user", "content": content}},
		},
	}
	// Studio messages are generation-oriented; hint the runtime to expose slide skills first.
	if tabID == "web-ui-studio" {
		runRequest["context"] = map[string]any{
			"skill_names": studioPreferredSkillNames,
		}
	}
	if tabID == "web-ui-studio" {
		s.logger.Printf(
			"[studio-run] start request_id=%s conversation_id=%d thread_id=%s skill_names=%v prompt_shape=%s",
			requestID,
			convID,
			threadID,
			studioPreferredSkillNames,
			summarizeMarkdownShape(content),
		)
	}
	runBody, _ := json.Marshal(runRequest)
	pr, pw := io.Pipe()
	aiWriter := &pipeResponseWriter{pw: pw, header: make(http.Header)}
	go func() {
		defer pw.Close()
		aiReq := httptest.NewRequest(http.MethodPost, "/threads/"+threadID+"/runs/stream", bytes.NewReader(runBody))
		aiReq = aiReq.WithContext(ctx)
		aiReq.Header.Set("Content-Type", "application/json")
		aiReq.Header.Set("Accept", "text/event-stream")
		addInternalAuth(aiReq)
		s.aiHandler.ServeHTTP(aiWriter, aiReq)
	}()
	acc := ""
	assistantContent := ""
	startSent := false
	scanner := bufio.NewScanner(pr)
	scanBuf := make([]byte, 0, 256*1024)
	scanner.Buffer(scanBuf, 8*1024*1024)
	var evName, evData string
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event:"):
			evName = strings.TrimSpace(line[6:])
		case strings.HasPrefix(line, "data:"):
			evData = strings.TrimSpace(line[5:])
		case line == "":
			if evName == "" && evData == "" {
				continue
			}
			switch evName {
			case "metadata":
				if tabID == "web-ui-studio" {
					s.logger.Printf("[studio-run] metadata request_id=%s raw=%s", requestID, truncateForLog(evData, 400))
				}
				if !startSent {
					writeSSE("start", map[string]any{"request_id": requestID})
					startSent = true
				}
			case "chunk":
				var payload map[string]any
				_ = json.Unmarshal([]byte(evData), &payload)
				delta, _ := payload["delta"].(string)
				if delta == "" {
					delta, _ = payload["content"].(string)
				}
				if delta != "" {
					acc += delta
					assistantContent = acc
					if tabID == "web-ui-studio" {
						s.logger.Printf("[studio-run] chunk request_id=%s acc_chars=%d", requestID, len(acc))
					}
					writeSSE("chunk", map[string]any{"content": acc})
				}
			case "messages":
				if tabID == "web-ui-studio" {
					if tc := toolCallsFromMessagesEvent(evData); len(tc) > 0 {
						s.logger.Printf("[studio-run] tool_calls request_id=%s names=%v", requestID, tc)
					}
				}
				// LangGraph SDK / deerflow: final assistant text is often only in "messages" tuples,
				// while some providers never emit per-token "chunk" deltas (only Done → AgentEventEnd).
				if txt := assistantTextFromLangGraphMessagesEvent(evData); txt != "" && len(txt) > len(assistantContent) {
					assistantContent = txt
					acc = txt
					if tabID == "web-ui-studio" {
						s.logger.Printf("[studio-run] assistant_snapshot request_id=%s chars=%d shape=%s", requestID, len(acc), summarizeMarkdownShape(acc))
					}
					if !startSent {
						writeSSE("start", map[string]any{"request_id": requestID})
						startSent = true
					}
					writeSSE("chunk", map[string]any{"content": acc})
				}
			case "error":
				var payload map[string]any
				_ = json.Unmarshal([]byte(evData), &payload)
				msg, _ := payload["message"].(string)
				if msg == "" {
					msg = "ai_error"
				}
				if tabID == "web-ui-studio" {
					s.logger.Printf("[studio-run] error request_id=%s message=%q raw=%s", requestID, msg, truncateForLog(evData, 400))
				}
				writeSSE("error", map[string]any{"message": msg})
				return
			}
			evName, evData = "", ""
		}
	}
	if err := scanner.Err(); err != nil {
		if !startSent {
			writeSSE("start", map[string]any{"request_id": requestID})
		}
		writeSSE("error", map[string]any{"message": fmt.Sprintf("读取模型流失败: %v", err)})
		return
	}
	if assistantContent == "" {
		// Nothing came through: either LangGraph returned a non-SSE HTTP error
		// (config/startup failure) or the model returned an empty response.
		if !startSent {
			writeSSE("start", map[string]any{"request_id": requestID})
		}
		writeSSE("error", map[string]any{"message": "模型未返回内容，请检查模型配置或重试。"})
		return
	}
	if tabID == "web-ui-studio" {
		s.logger.Printf("[studio-run] done request_id=%s chars=%d shape=%s", requestID, len(assistantContent), summarizeMarkdownShape(assistantContent))
	}
	if !startSent {
		writeSSE("start", map[string]any{"request_id": requestID})
	}
	writeSSE("done", map[string]any{"request_id": requestID})
	if assistantContent != "" {
		if _, err := s.createMessage(ctx, convID, "assistant", assistantContent, "done"); err == nil {
			_ = s.updateConversationLastMessage(ctx, userID, convID, assistantContent)
		}
	}
}

func toolCallsFromMessagesEvent(evData string) []string {
	evData = strings.TrimSpace(evData)
	if evData == "" {
		return nil
	}
	var tuple []json.RawMessage
	if err := json.Unmarshal([]byte(evData), &tuple); err != nil || len(tuple) < 1 {
		return nil
	}
	var msg map[string]any
	if err := json.Unmarshal(tuple[0], &msg); err != nil {
		return nil
	}
	raw, ok := msg["tool_calls"].([]any)
	if !ok || len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		if strings.TrimSpace(name) != "" {
			out = append(out, strings.TrimSpace(name))
		}
	}
	return out
}

func truncateForLog(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// assistantTextFromLangGraphMessagesEvent parses SSE data for event "messages": [serializedMessage, metadata].
func assistantTextFromLangGraphMessagesEvent(evData string) string {
	evData = strings.TrimSpace(evData)
	if evData == "" {
		return ""
	}
	var tuple []json.RawMessage
	if err := json.Unmarshal([]byte(evData), &tuple); err != nil || len(tuple) < 1 {
		return ""
	}
	var msg map[string]any
	if err := json.Unmarshal(tuple[0], &msg); err != nil {
		return ""
	}
	role, _ := msg["type"].(string)
	if role == "" {
		role, _ = msg["role"].(string)
	}
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "ai" && role != "assistant" {
		return ""
	}
	return assistantTextFromLangGraphContent(msg["content"])
}

func assistantTextFromLangGraphContent(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		var b strings.Builder
		for _, part := range x {
			pm, ok := part.(map[string]any)
			if !ok {
				continue
			}
			pt, _ := pm["type"].(string)
			if strings.EqualFold(pt, "text") {
				if t, _ := pm["text"].(string); t != "" {
					b.WriteString(t)
				}
			}
		}
		return b.String()
	default:
		return ""
	}
}

func (s *Server) chatFallback(ctx context.Context, w http.ResponseWriter, userID int64, convID int64, requestID string, userContent string, stream bool) {
	reply := "Echo: " + userContent
	assistantMsg, err := s.createMessage(ctx, convID, "assistant", reply, "done")
	if err == nil {
		_ = s.updateConversationLastMessage(ctx, userID, convID, reply)
	}
	if !stream {
		response := map[string]any{"request_id": requestID}
		if assistantMsg != nil {
			response["message_id"] = assistantMsg.ID
		}
		writeJSON(w, http.StatusOK, response)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming_not_supported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	writeSSE := func(event string, payload any) {
		b, _ := json.Marshal(payload)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
		flusher.Flush()
	}
	writeSSE("start", map[string]any{"request_id": requestID})
	acc := ""
	for _, ch := range []rune(reply) {
		acc += string(ch)
		writeSSE("chunk", map[string]any{"content": acc})
		time.Sleep(8 * time.Millisecond)
	}
	writeSSE("done", map[string]any{"request_id": requestID})
}
