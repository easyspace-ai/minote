package notex

// Studio Handlers V2 - 后端主导的生成接口
// 前端只需传：type + content + options，后端负责构建提示词和生成

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// StudioCreateRequest 简化的创建请求 - 前端只需要传这些
type StudioCreateRequest struct {
	Type       string                 `json:"type"`                  // audio | html | ppt | mindmap
	Content    string                 `json:"content"`               // 要转换的文本/Markdown
	Title      string                 `json:"title,omitempty"`       // 结果标题（可选）
	Options    map[string]interface{} `json:"options,omitempty"`     // 类型特定选项
	MaterialID int64                  `json:"material_id,omitempty"` // 用于更新现有 pending material
}

// StudioCreateResponse 创建响应
type StudioCreateResponse struct {
	Success    bool                   `json:"success"`
	MaterialID int64                  `json:"material_id,omitempty"`
	Status     string                 `json:"status"`
	Error      string                 `json:"error,omitempty"`
	JobID      string                 `json:"job_id,omitempty"`
	Result     map[string]interface{} `json:"result,omitempty"`
}

// HandleProjectStudioCreate POST /api/v1/projects/{id}/studio/create
// 新的统一创建入口，后端根据 type 路由并构建提示词
func (s *Server) HandleProjectStudioCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	projectID, ok := pathProjectID(r)
	if !ok || projectID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_id"})
		return
	}

	var req StudioCreateRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json: " + err.Error()})
		return
	}

	// 验证参数
	if req.Type == "" || req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type and content are required"})
		return
	}

	s.logger.Printf("[studio-create] project_id=%s type=%s title=%q content_bytes=%d options=%v",
		projectID, req.Type, req.Title, len(req.Content), req.Options)

	// 验证并转换 material_id
	if req.MaterialID > 0 {
		if !s.verifyStudioPendingReplace(w, r.Context(), uid, projectID, req.MaterialID, req.Type) {
			return
		}
	}

	// 根据类型路由到具体生成逻辑
	ctx := r.Context()
	switch strings.ToLower(req.Type) {
	case "audio":
		s.logger.Printf("[studio-create] routing to handleStudioCreateAudio, type=%s", req.Type)
		s.handleStudioCreateAudio(w, r, ctx, uid, projectID, req)
	case "html":
		s.logger.Printf("[studio-create] routing to handleStudioCreateHTML, type=%s", req.Type)
		s.handleStudioCreateHTML(w, r, ctx, uid, projectID, req)
	case "ppt", "slides":
		s.logger.Printf("[studio-create] routing to handleStudioCreatePPT, type=%s", req.Type)
		s.handleStudioCreatePPT(w, r, ctx, uid, projectID, req)
	case "mindmap":
		s.logger.Printf("[studio-create] routing to handleStudioCreateMindmap, type=%s", req.Type)
		s.handleStudioCreateMindmap(w, r, ctx, uid, projectID, req)
	default:
		s.logger.Printf("[studio-create] unsupported type: %s", req.Type)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported type: " + req.Type})
	}
}

// handleStudioCreateAudio 处理音频生成
func (s *Server) handleStudioCreateAudio(w http.ResponseWriter, r *http.Request, ctx context.Context, uid int64, projectID string, req StudioCreateRequest) {
	// 加载 skill 配置
	def, ok := s.GetStudioSkill("audio")
	if !ok {
		// 降级：使用默认逻辑
		s.logger.Printf("[studio-audio] skill not found, using default logic")
	}

	// 构建生成配置
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "AI 播客"
	}

	// 从 options 提取音频特定参数
	language := getStringOption(req.Options, "language", "zh-CN")
	voice := getStringOption(req.Options, "voice", "zh-female-1")
	speed := getFloatOption(req.Options, "speed", 1.0)

	s.logger.Printf("[studio-audio] generating: language=%s voice=%s speed=%.1f", language, voice, speed)

	// 构建提示词（用于 TTS 文本处理）
	var prompt string
	if def != nil {
		p, err := def.BuildGenerationPrompt(req.Content, req.Options)
		if err == nil {
			prompt = p
		}
	}

	// 创建 pending material
	payload := map[string]interface{}{
		"language": language,
		"voice":    voice,
		"speed":    speed,
		"prompt":   prompt,
	}

	material, err := s.createPendingMaterial(ctx, uid, projectID, req.MaterialID, "audio", title, payload)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, StudioCreateResponse{
			Success: false,
			Error:   "create material failed: " + err.Error(),
		})
		return
	}

	s.logger.Printf("[studio-audio] created pending material: id=%d", material.ID)

	// TODO: 启动异步任务调用 TTS 服务生成音频
	// 暂时返回 processing 状态

	resp := StudioCreateResponse{
		Success:    true,
		MaterialID: material.ID,
		Status:     material.Status,
		JobID:      fmt.Sprintf("audio-%s-%s", projectID, generateShortID()),
		Result: map[string]interface{}{
			"type":        "audio",
			"title":       title,
			"language":    language,
			"voice":       voice,
			"prompt_hash": fmt.Sprintf("%x", len(prompt)),
			"message":     "音频生成任务已提交，请轮询状态",
		},
	}

	writeJSON(w, http.StatusAccepted, resp)
}

// handleStudioCreateHTML 处理 HTML 生成
func (s *Server) handleStudioCreateHTML(w http.ResponseWriter, r *http.Request, ctx context.Context, uid int64, projectID string, req StudioCreateRequest) {
	def, ok := s.GetStudioSkill("html")
	if !ok {
		s.logger.Printf("[studio-html] skill not found, using default logic")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "网页内容"
	}

	// 提取选项
	pageType := getStringOption(req.Options, "page_type", "article")
	theme := getStringOption(req.Options, "theme", "light")
	interactive := getBoolOption(req.Options, "interactive", true)

	s.logger.Printf("[studio-html] generating: page_type=%s theme=%s interactive=%v", pageType, theme, interactive)

	// 构建提示词
	var prompt string
	if def != nil {
		p, err := def.BuildGenerationPrompt(req.Content, req.Options)
		if err == nil {
			prompt = p
			s.logger.Printf("[studio-html] prompt built from skill, length=%d", len(prompt))
		}
	}

	// 如果没有 skill 模板，使用默认提示词构建
	if prompt == "" {
		prompt = s.buildDefaultHTMLPrompt(req.Content, pageType, theme, interactive)
	}

	// 创建 pending material
	material, err := s.createPendingMaterial(ctx, uid, projectID, req.MaterialID, "html", title, map[string]interface{}{
		"page_type":   pageType,
		"theme":       theme,
		"interactive": interactive,
		"prompt":      prompt,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// TODO: 启动异步生成任务
	// 应该启动一个 goroutine 或提交到队列：
	// 1. 调用 LLM 生成 HTML
	// 2. 保存文件
	// 3. 更新 material 状态

	resp := StudioCreateResponse{
		Success:    true,
		MaterialID: material.ID,
		Status:     material.Status,
		JobID:      fmt.Sprintf("html-%s-%s", projectID, generateShortID()),
		Result: map[string]interface{}{
			"type":        "html",
			"title":       title,
			"page_type":   pageType,
			"theme":       theme,
			"prompt_hash": fmt.Sprintf("%x", len(prompt)),
			"message":     "HTML生成任务已提交，请轮询状态",
		},
	}

	writeJSON(w, http.StatusAccepted, resp)
}

// handleStudioCreatePPT refuses server-side themed PPTX: slides must come from the agent presentation skill
// (.pptx on the LangGraph thread), then POST .../materials/slides-pptx copies that file into project materials.
func (s *Server) handleStudioCreatePPT(w http.ResponseWriter, r *http.Request, ctx context.Context, uid int64, projectID string, req StudioCreateRequest) {
	_ = ctx
	_ = uid
	_ = projectID
	_ = req
	s.logger.Printf("[studio-create] ppt/slides rejected: must use skill thread + POST /projects/{id}/materials/slides-pptx")
	writeJSON(w, http.StatusUnprocessableEntity, StudioCreateResponse{
		Success: false,
		Error: "ppt_skill_required: PPT must be generated by the presentation skill and written as a .pptx on the conversation LangGraph thread; " +
			"then call POST /api/v1/projects/{id}/materials/slides-pptx with conversation_id. /studio/create type ppt|slides is not supported.",
	})
}

// createFinalMaterial 创建最终状态的 material
func (s *Server) createFinalMaterial(ctx context.Context, uid int64, projectID string, materialID int64, kind, title, status string, payload map[string]interface{}, filePath string) (*Material, error) {
	if payload == nil {
		payload = make(map[string]interface{})
	}

	if s.store != nil {
		if materialID > 0 {
			m, err := s.store.UpdateMaterialForUser(ctx, uid, projectID, materialID, kind, title, status, "", payload, filePath)
			if err != nil {
				return nil, err
			}
			return m, nil
		}
		m, err := s.store.CreateMaterialForUser(ctx, uid, projectID, kind, title, status, "", payload, filePath)
		return m, err
	}

	// 内存模式
	s.materialMu.Lock()
	defer s.materialMu.Unlock()

	now := nowRFC3339()
	if materialID > 0 {
		m := s.materialsByID[materialID]
		if m != nil && m.ProjectID == projectID {
			m.Kind = kind
			m.Title = title
			m.Status = status
			m.Subtitle = ""
			m.Payload = payload
			m.FilePath = filePath
			m.UpdatedAt = now
			return m, nil
		}
	}

	m := &Material{
		ID:        s.nextMaterialID,
		CreatedAt: now,
		UpdatedAt: now,
		ProjectID: projectID,
		Kind:      kind,
		Title:     title,
		Status:    status,
		Payload:   payload,
		FilePath:  filePath,
	}
	s.nextMaterialID++
	s.materialsByID[m.ID] = m
	return m, nil
}

// createFinalMaterialWithSubtitle 创建最终状态的 material（带自定义 subtitle）
func (s *Server) createFinalMaterialWithSubtitle(ctx context.Context, uid int64, projectID string, materialID int64, kind, title, subtitle, status string, payload map[string]interface{}, filePath string) (*Material, error) {
	if payload == nil {
		payload = make(map[string]interface{})
	}

	if s.store != nil {
		if materialID > 0 {
			m, err := s.store.UpdateMaterialForUser(ctx, uid, projectID, materialID, kind, title, status, subtitle, payload, filePath)
			if err != nil {
				return nil, err
			}
			return m, nil
		}
		m, err := s.store.CreateMaterialForUser(ctx, uid, projectID, kind, title, status, subtitle, payload, filePath)
		return m, err
	}

	// 内存模式
	s.materialMu.Lock()
	defer s.materialMu.Unlock()

	now := nowRFC3339()
	if materialID > 0 {
		m := s.materialsByID[materialID]
		if m != nil && m.ProjectID == projectID {
			m.Kind = kind
			m.Title = title
			m.Status = status
			m.Subtitle = subtitle
			m.Payload = payload
			m.FilePath = filePath
			m.UpdatedAt = now
			return m, nil
		}
	}

	m := &Material{
		ID:        s.nextMaterialID,
		CreatedAt: now,
		UpdatedAt: now,
		ProjectID: projectID,
		Kind:      kind,
		Title:     title,
		Status:    status,
		Subtitle:  subtitle,
		Payload:   payload,
		FilePath:  filePath,
	}
	s.nextMaterialID++
	s.materialsByID[m.ID] = m
	return m, nil
}

// handleStudioCreateMindmap 处理思维导图生成
func (s *Server) handleStudioCreateMindmap(w http.ResponseWriter, r *http.Request, ctx context.Context, uid int64, projectID string, req StudioCreateRequest) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "思维导图"
	}

	// 思维导图使用 Markdown 列表格式
	content := req.Content
	if !strings.Contains(content, "\n-") && !strings.Contains(content, "\n*") {
		// 如果不是列表格式，尝试转换
		content = s.convertToMindmapFormat(content)
	}

	// 创建 pending material
	material, err := s.createPendingMaterial(ctx, uid, projectID, req.MaterialID, "mindmap", title, map[string]interface{}{
		"content": content,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	resp := StudioCreateResponse{
		Success:    true,
		MaterialID: material.ID,
		Status:     material.Status,
		Result: map[string]interface{}{
			"type":    "mindmap",
			"title":   title,
			"message": "思维导图生成任务已提交",
		},
	}

	writeJSON(w, http.StatusAccepted, resp)
}

// ==================== 辅助函数 ====================

// createPendingMaterial 创建或更新 pending 状态的 material
func (s *Server) createPendingMaterial(ctx context.Context, uid int64, projectID string, materialID int64, kind, title string, payload map[string]interface{}) (*Material, error) {
	if payload == nil {
		payload = make(map[string]interface{})
	}

	if s.store != nil {
		if materialID > 0 {
			// 更新现有 material
			m, err := s.store.UpdateMaterialForUser(ctx, uid, projectID, materialID, kind, title, "processing", "生成中...", payload, "")
			if err != nil {
				return nil, err
			}
			return m, nil
		}
		// 创建新 material
		m, err := s.store.CreateMaterialForUser(ctx, uid, projectID, kind, title, "processing", "生成中...", payload, "")
		return m, err
	}

	// 内存模式
	s.materialMu.Lock()
	defer s.materialMu.Unlock()

	now := nowRFC3339()
	if materialID > 0 {
		m := s.materialsByID[materialID]
		if m != nil && m.ProjectID == projectID {
			m.Kind = kind
			m.Title = title
			m.Status = "processing"
			m.Subtitle = "生成中..."
			m.Payload = payload
			m.UpdatedAt = now
			return m, nil
		}
	}

	m := &Material{
		ID:        s.nextMaterialID,
		CreatedAt: now,
		UpdatedAt: now,
		ProjectID: projectID,
		Kind:      kind,
		Title:     title,
		Status:    "processing",
		Subtitle:  "生成中...",
		Payload:   payload,
	}
	s.nextMaterialID++
	s.materialsByID[m.ID] = m
	return m, nil
}

// buildDefaultHTMLPrompt 构建默认 HTML 生成提示词
func (s *Server) buildDefaultHTMLPrompt(content, pageType, theme string, interactive bool) string {
	return fmt.Sprintf(`请根据以下内容生成一个精美的 HTML 页面。

内容：
%s

要求：
- 页面类型：%s
- 主题风格：%s
- 交互功能：%v
- 输出完整的、独立的 HTML 文件（包含内联 CSS 和 JS）
- 响应式设计，支持移动端
- 使用现代 CSS 特性

请直接输出 HTML 代码，不需要解释。`, content, pageType, theme, interactive)
}

// buildDefaultPPTPrompt 构建默认 PPT 生成提示词
func (s *Server) buildDefaultPPTPrompt(content, title string, slideCount int, theme string) string {
	return fmt.Sprintf(`请根据以下内容生成演示文稿大纲。

标题：%s
内容：
%s

要求：
- 生成 %d 张幻灯片的大纲
- 主题风格：%s
- 包含：封面、目录、内容页、总结页
- 每页包含标题和要点（不超过6个要点）
- 输出 JSON 格式的幻灯片结构

JSON 格式：
{
  "title": "演示文稿标题",
  "slides": [
    {"type": "cover", "title": "...", "subtitle": "..."},
    {"type": "content", "title": "...", "bullets": ["...", "..."]}
  ]
}`, title, content, slideCount, theme)
}

// convertToMindmapFormat 将普通文本转换为思维导图格式
func (s *Server) convertToMindmapFormat(content string) string {
	// 简单转换：将每行转换为列表项
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "*") {
			line = "- " + line
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

// generateShortID 生成短 ID
func generateShortID() string {
	// 使用纳秒时间戳作为 ID
	return fmt.Sprintf("%x", nowUnixNano())
}

func nowUnixNano() int64 {
	return time.Now().UnixNano()
}

// getStringOption 安全获取字符串选项
func getStringOption(opts map[string]interface{}, key, defaultVal string) string {
	if opts == nil {
		return defaultVal
	}
	if v, ok := opts[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}

// getIntOption 安全获取整数选项
func getIntOption(opts map[string]interface{}, key string, defaultVal int) int {
	if opts == nil {
		return defaultVal
	}
	switch v := opts[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return defaultVal
}

// getFloatOption 安全获取浮点数选项
func getFloatOption(opts map[string]interface{}, key string, defaultVal float64) float64 {
	if opts == nil {
		return defaultVal
	}
	switch v := opts[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return defaultVal
}

// getBoolOption 安全获取布尔选项
func getBoolOption(opts map[string]interface{}, key string, defaultVal bool) bool {
	if opts == nil {
		return defaultVal
	}
	if v, ok := opts[key].(bool); ok {
		return v
	}
	return defaultVal
}
