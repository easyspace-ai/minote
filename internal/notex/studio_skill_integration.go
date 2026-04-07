package notex

// StudioSkillIntegration - 后端主导的 Studio 生成架构
// 前端只传：type + content + options + project_id
// 后端负责：路由 → 构建提示词 → 调用 LLM → 生成文件 → 返回结果

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ==================== Skill 定义结构 ====================

// StudioSkillDefinition 解析后的 YAML skill 定义
type StudioSkillDefinition struct {
	Name            string                 `yaml:"name"`
	Version         string                 `yaml:"version"`
	Description     string                 `yaml:"description"`
	TriggerKeywords []string               `yaml:"trigger_keywords"`
	Tools           []string               `yaml:"tools"`
	InputSchema     StudioInputSchema      `yaml:"input_schema"`
	OutputSchema    StudioOutputSchema     `yaml:"output_schema"`
	ThemeOptions    map[string]interface{} `yaml:"theme_options,omitempty"`
	PageTypeOptions map[string]interface{} `yaml:"page_type_options,omitempty"`
	VoiceOptions    map[string]interface{} `yaml:"voice_options,omitempty"`
	Limits          map[string]interface{} `yaml:"limits,omitempty"`
	RawPrompt       string                 `yaml:"-"` // --- 分隔符后的内容
}

// StudioInputSchema 输入参数定义
type StudioInputSchema struct {
	Required []string `yaml:"required"`
	Optional []string `yaml:"optional,omitempty"`
}

// StudioOutputSchema 输出字段定义
type StudioOutputSchema struct {
	Fields []string `yaml:"fields"`
}

// StudioType 常量
const (
	StudioTypeAudio  = "audio"
	StudioTypeHTML   = "html"
	StudioTypePPT    = "ppt"
	StudioTypeSlides = "slides" // 兼容旧命名
)

// ==================== Skill 加载与解析 ====================

// ParseStudioSkillYAML 解析单个 skill YAML 文件
func ParseStudioSkillYAML(path string) (*StudioSkillDefinition, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skill file: %w", err)
	}

	var def StudioSkillDefinition
	if err := yaml.Unmarshal(b, &def); err != nil {
		return nil, fmt.Errorf("parse skill YAML: %w", err)
	}

	// 提取 --- 分隔符后的 prompt 内容
	content := string(b)
	if idx := strings.Index(content, "\n---"); idx > 0 {
		rest := content[idx+4:]
		def.RawPrompt = strings.TrimSpace(rest)
	}

	if def.Version == "" {
		def.Version = "1.0"
	}

	return &def, nil
}

// LoadStudioSkills 加载所有 studio skills
func (s *Server) LoadStudioSkills() (map[string]*StudioSkillDefinition, error) {
	defs := make(map[string]*StudioSkillDefinition)

	roots := s.skillsRoots()
	for _, root := range roots {
		studioDir := filepath.Join(root, "studio")
		entries, err := os.ReadDir(studioDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(strings.ToLower(name), ".yaml") {
				continue
			}

			path := filepath.Join(studioDir, name)
			def, err := ParseStudioSkillYAML(path)
			if err != nil {
				s.logger.Printf("[studio-skill] failed to parse %s: %v", path, err)
				continue
			}

			skillType := extractSkillType(def.Name, name)
			defs[skillType] = def
			s.logger.Printf("[studio-skill] loaded %s from %s", skillType, path)
		}
	}

	return defs, nil
}

func extractSkillType(name, filename string) string {
	name = strings.ToLower(name)
	filename = strings.ToLower(filename)

	if strings.Contains(name, "audio") || strings.Contains(filename, "audio") {
		return StudioTypeAudio
	}
	if strings.Contains(name, "html") || strings.Contains(filename, "html") {
		return StudioTypeHTML
	}
	if strings.Contains(name, "ppt") || strings.Contains(filename, "ppt") {
		return StudioTypePPT
	}

	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	base = strings.TrimPrefix(base, "studio-")
	return base
}

// GetStudioSkill 获取指定类型的 skill 定义
func (s *Server) GetStudioSkill(skillType string) (*StudioSkillDefinition, bool) {
	skills, err := s.LoadStudioSkills()
	if err != nil {
		return nil, false
	}

	// 兼容旧命名 slides -> ppt
	if skillType == StudioTypeSlides {
		skillType = StudioTypePPT
	}

	def, ok := skills[skillType]
	return def, ok
}

// ==================== 输入验证与提示词构建 ====================

// ValidateStudioInput 验证输入参数
func (def *StudioSkillDefinition) ValidateStudioInput(input map[string]interface{}) error {
	for _, field := range def.InputSchema.Required {
		if _, ok := input[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	return nil
}

// BuildGenerationPrompt 根据 skill 模板和用户输入构建提示词
func (def *StudioSkillDefinition) BuildGenerationPrompt(content string, options map[string]interface{}) (string, error) {
	if def.RawPrompt == "" {
		return "", fmt.Errorf("no prompt template defined in skill %s", def.Name)
	}

	// 构建变量替换映射
	vars := make(map[string]string)
	vars["content"] = content
	vars["input"] = content // 别名

	// 添加 options 中的变量
	for key, val := range options {
		vars[key] = fmt.Sprintf("%v", val)
	}

	// 默认值处理
	if _, ok := vars["language"]; !ok {
		vars["language"] = "zh-CN"
	}
	if _, ok := vars["theme"]; !ok {
		// 根据 skill 类型设置默认主题
		switch {
		case strings.Contains(def.Name, "ppt"):
			vars["theme"] = "professional"
		case strings.Contains(def.Name, "html"):
			vars["theme"] = "light"
		}
	}

	// 执行模板替换
	prompt := def.RawPrompt
	for key, val := range vars {
		// 支持 {{variable}} 和 ${variable} 两种格式
		placeholder1 := fmt.Sprintf("{{%s}}", key)
		placeholder2 := fmt.Sprintf("${%s}", key)
		prompt = strings.ReplaceAll(prompt, placeholder1, val)
		prompt = strings.ReplaceAll(prompt, placeholder2, val)
	}

	return prompt, nil
}

// ==================== API 请求/响应结构 ====================

// StudioGenerateRequest 前端调用的新接口（简化版）
type StudioGenerateRequest struct {
	Type    string                 `json:"type"`              // audio | html | ppt
	Content string                 `json:"content"`           // 要转换的内容（文本/Markdown）
	Title   string                 `json:"title"`             // 生成结果的标题
	Options map[string]interface{} `json:"options,omitempty"` // 可选参数：theme, language, voice 等
}

// StudioGenerateResponse 统一响应结构
type StudioGenerateResponse struct {
	Success    bool                   `json:"success"`
	MaterialID int64                  `json:"material_id,omitempty"`
	Status     string                 `json:"status"` // pending | processing | done | failed
	Error      string                 `json:"error,omitempty"`
	Result     map[string]interface{} `json:"result,omitempty"`
}

// ==================== 核心路由与处理 ====================

// HandleStudioGenerateV2 新的统一生成入口
// 这是前端应该调用的主要接口
func (s *Server) HandleStudioGenerateV2(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	uid, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	var req StudioGenerateRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json: " + err.Error()})
		return
	}

	// 验证必需字段
	if req.Type == "" || req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type and content are required"})
		return
	}

	s.logger.Printf("[studio-generate] type=%s title=%q content_bytes=%d options=%v",
		req.Type, req.Title, len(req.Content), req.Options)

	// 根据类型路由到对应处理器
	ctx := r.Context()
	var resp StudioGenerateResponse

	// 标准化类型名称
	switch strings.ToLower(req.Type) {
	case StudioTypeAudio:
		resp = s.generateAudioFromSkill(ctx, uid, req)
	case StudioTypeHTML:
		resp = s.generateHTMLFromSkill(ctx, uid, req)
	case StudioTypePPT, StudioTypeSlides:
		resp = s.generatePPTFromSkill(ctx, uid, req)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported type: " + req.Type})
		return
	}

	if resp.Success {
		writeJSON(w, http.StatusOK, resp)
	} else {
		status := http.StatusInternalServerError
		if strings.HasPrefix(resp.Error, "ppt_skill_required") {
			status = http.StatusUnprocessableEntity
		}
		writeJSON(w, status, resp)
	}
}

// generateAudioFromSkill 根据 skill 配置生成音频
func (s *Server) generateAudioFromSkill(ctx context.Context, uid int64, req StudioGenerateRequest) StudioGenerateResponse {
	def, ok := s.GetStudioSkill(StudioTypeAudio)
	if !ok {
		return StudioGenerateResponse{Success: false, Error: "audio skill not found"}
	}

	// 验证输入
	input := map[string]interface{}{
		"text": req.Content,
	}
	for k, v := range req.Options {
		input[k] = v
	}

	if err := def.ValidateStudioInput(input); err != nil {
		return StudioGenerateResponse{Success: false, Error: err.Error()}
	}

	// 构建生成提示词（用于日志或后续处理）
	prompt, err := def.BuildGenerationPrompt(req.Content, req.Options)
	if err != nil {
		s.logger.Printf("[studio-audio] build prompt warning: %v", err)
	}

	s.logger.Printf("[studio-audio] prompt built, length=%d", len(prompt))

	// TODO: 调用 TTS 服务生成音频
	// 这里应该调用现有的 TTS 逻辑或新的 LLM 生成
	// 暂时返回成功状态，实际实现需要集成 TTS

	return StudioGenerateResponse{
		Success: true,
		Status:  "processing",
		Result: map[string]interface{}{
			"skill":   def.Name,
			"version": def.Version,
			"message": "音频生成任务已创建",
		},
	}
}

// generateHTMLFromSkill 根据 skill 配置生成 HTML
func (s *Server) generateHTMLFromSkill(ctx context.Context, uid int64, req StudioGenerateRequest) StudioGenerateResponse {
	def, ok := s.GetStudioSkill(StudioTypeHTML)
	if !ok {
		return StudioGenerateResponse{Success: false, Error: "html skill not found"}
	}

	// 验证输入
	input := map[string]interface{}{
		"content": req.Content,
	}
	for k, v := range req.Options {
		input[k] = v
	}

	if err := def.ValidateStudioInput(input); err != nil {
		return StudioGenerateResponse{Success: false, Error: err.Error()}
	}

	// 构建提示词
	prompt, err := def.BuildGenerationPrompt(req.Content, req.Options)
	if err != nil {
		return StudioGenerateResponse{Success: false, Error: "build prompt failed: " + err.Error()}
	}

	s.logger.Printf("[studio-html] prompt built, length=%d", len(prompt))

	// TODO: 调用 LLM 生成 HTML
	// 这里应该：
	// 1. 调用 LLM 生成 HTML 代码
	// 2. 验证 HTML 语法
	// 3. 保存文件
	// 4. 创建 material

	return StudioGenerateResponse{
		Success: true,
		Status:  "processing",
		Result: map[string]interface{}{
			"skill":       def.Name,
			"version":     def.Version,
			"page_type":   req.Options["page_type"],
			"theme":       req.Options["theme"],
			"prompt_hash": fmt.Sprintf("%x", len(prompt)),
			"message":     "HTML生成任务已创建，请轮询状态",
		},
	}
}

// generatePPTFromSkill 不提供服务端生成：PPT 必须由 Agent 在会话线程上跑 presentation skill 并产出 .pptx，
// 再通过 POST /api/v1/projects/{id}/materials/slides-pptx（conversation_id）落库。
func (s *Server) generatePPTFromSkill(ctx context.Context, uid int64, req StudioGenerateRequest) StudioGenerateResponse {
	_ = ctx
	_ = uid
	_ = req
	return StudioGenerateResponse{
		Success: false,
		Error: "ppt_skill_required: PPT must be produced by the agent presentation skill as a .pptx on the LangGraph thread, " +
			"then use POST /api/v1/projects/{id}/materials/slides-pptx with conversation_id. POST /api/v1/studio/generate type=ppt is not supported.",
	}
}

// ==================== Skill 查询 API ====================

// StudioSkillInfoDTO 返回给前端的 skill 信息
type StudioSkillInfoDTO struct {
	Type            string                 `json:"type"`
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	Description     string                 `json:"description"`
	TriggerKeywords []string               `json:"trigger_keywords"`
	InputSchema     StudioInputSchema      `json:"input_schema"`
	OutputSchema    StudioOutputSchema     `json:"output_schema"`
	ThemeOptions    map[string]interface{} `json:"theme_options,omitempty"`
	VoiceOptions    map[string]interface{} `json:"voice_options,omitempty"`
	Limits          map[string]interface{} `json:"limits,omitempty"`
}

// HandleStudioSkillsList 返回可用的 studio skills 列表
func (s *Server) HandleStudioSkillsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if _, ok := s.requireUserID(w, r); !ok {
		return
	}

	skills, err := s.LoadStudioSkills()
	if err != nil {
		s.logger.Printf("[studio-skill] failed to list skills: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load skills"})
		return
	}

	var result []StudioSkillInfoDTO
	for skillType, def := range skills {
		result = append(result, StudioSkillInfoDTO{
			Type:            skillType,
			Name:            def.Name,
			Version:         def.Version,
			Description:     def.Description,
			TriggerKeywords: def.TriggerKeywords,
			InputSchema:     def.InputSchema,
			OutputSchema:    def.OutputSchema,
			ThemeOptions:    def.ThemeOptions,
			VoiceOptions:    def.VoiceOptions,
			Limits:          def.Limits,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

// HandleStudioSkillDetail 返回单个 skill 的详细信息（包含完整 prompt）
func (s *Server) HandleStudioSkillDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if _, ok := s.requireUserID(w, r); !ok {
		return
	}

	skillType := strings.TrimSpace(r.PathValue("type"))
	if skillType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type is required"})
		return
	}

	def, ok := s.GetStudioSkill(skillType)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "skill not found: " + skillType})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"type":         skillType,
		"name":         def.Name,
		"version":      def.Version,
		"description":  def.Description,
		"input_schema": def.InputSchema,
		"output_schema": def.OutputSchema,
		"limits":       def.Limits,
		// 不包含 RawPrompt，避免泄露内部逻辑
	})
}

// MatchSkillByKeywords 根据用户输入匹配 skill（用于智能路由）
func (s *Server) MatchSkillByKeywords(userInput string) (string, *StudioSkillDefinition, bool) {
	userInput = strings.ToLower(userInput)

	skills, err := s.LoadStudioSkills()
	if err != nil {
		return "", nil, false
	}

	for skillType, def := range skills {
		for _, keyword := range def.TriggerKeywords {
			if strings.Contains(userInput, strings.ToLower(keyword)) {
				return skillType, def, true
			}
		}
	}

	return "", nil, false
}
