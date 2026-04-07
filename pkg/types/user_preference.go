package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TonePreference 用户语气偏好
type TonePreference string

const (
	ToneFormal    TonePreference = "formal"    // 正式、专业
	ToneCasual    TonePreference = "casual"    // 随意、友好
	ToneTechnical TonePreference = "technical" // 技术化、详细
	ToneConcise   TonePreference = "concise"   // 简洁、直接
)

// OutputFormat 默认输出格式
type OutputFormat string

const (
	OutputMarkdown OutputFormat = "markdown"
	OutputHTML     OutputFormat = "html"
	OutputPPT      OutputFormat = "ppt"
	OutputPDF      OutputFormat = "pdf"
	OutputPlain    OutputFormat = "plain"
)

// ClarificationType 澄清场景类型
type ClarificationType string

const (
	ClarifyMissingInfo          ClarificationType = "missing_info"
	ClarifyAmbiguousRequirement ClarificationType = "ambiguous_requirement"
	ClarifyApproachChoice       ClarificationType = "approach_choice"
	ClarifyRiskConfirmation     ClarificationType = "risk_confirmation"
	ClarifySuggestion           ClarificationType = "suggestion"
)

// UITheme UI 主题偏好
type UITheme string

const (
	ThemeLight    UITheme = "light"
	ThemeDark     UITheme = "dark"
	ThemeSystem   UITheme = "system"
	ThemeHighContrast UITheme = "high-contrast"
)

// UserPreference 用户偏好配置
// 存储用户级提示词覆盖和个性化设置
type UserPreference struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// 核心偏好
	TonePreference        TonePreference `json:"tone_preference" db:"tone_preference"`
	DefaultOutputFormat   OutputFormat   `json:"default_output_format" db:"default_output_format"`
	PreferredLanguage     string         `json:"preferred_language" db:"preferred_language"` // zh-CN, en-US, etc.
	UITheme               UITheme        `json:"ui_theme" db:"ui_theme"`

	// 澄清场景配置
	AutoSkipClarification []ClarificationType `json:"auto_skip_clarification" db:"auto_skip_clarification"`
	ClarificationThreshold float64             `json:"clarification_threshold" db:"clarification_threshold"` // 0.0-1.0, 低于此阈值自动澄清

	// Studio 默认配置
	StudioDefaults StudioDefaults `json:"studio_defaults" db:"studio_defaults"`

	// 自定义提示词覆盖
	CustomPrompts CustomPromptOverrides `json:"custom_prompts" db:"custom_prompts"`

	// 通知和交互偏好
	Notifications NotificationPrefs `json:"notifications" db:"notifications"`

	// 隐私设置
	Privacy PrivacySettings `json:"privacy" db:"privacy"`
}

// StudioDefaults Studio 模块默认配置
type StudioDefaults struct {
	HTMLTheme        string   `json:"html_theme,omitempty"`        // modern, minimal, colorful, corporate
	PPTTheme         string   `json:"ppt_theme,omitempty"`         // professional, minimal, vibrant, tech, education
	AudioVoice       string   `json:"audio_voice,omitempty"`       // 默认音色
	AudioSpeed       float64  `json:"audio_speed,omitempty"`       // 默认语速 0.5-2.0
	InfographicTheme string   `json:"infographic_theme,omitempty"` // modern, minimal, colorful, corporate
	QuizDifficulty   string   `json:"quiz_difficulty,omitempty"`   // easy, medium, hard, mixed
	TableStyle       string   `json:"table_style,omitempty"`       // simple, card, striped, bordered
	TableExportFormats []string `json:"table_export_formats,omitempty"`
}

// CustomPromptOverrides 自定义提示词覆盖
type CustomPromptOverrides struct {
	// 系统级覆盖
	SystemPromptPrefix string `json:"system_prompt_prefix,omitempty"` // 添加到所有系统提示词前
	SystemPromptSuffix string `json:"system_prompt_suffix,omitempty"` // 添加到所有系统提示词后

	// 特定场景覆盖
	ToneInstruction      string `json:"tone_instruction,omitempty"`      // 语气指令
	ResponseFormatRule   string `json:"response_format_rule,omitempty"`  // 响应格式规则
	CodeStylePreference  string `json:"code_style_preference,omitempty"` // 代码风格偏好
	AnalysisDepth        string `json:"analysis_depth,omitempty"`        // 分析深度要求
}

// NotificationPrefs 通知偏好
type NotificationPrefs struct {
	EmailNotifications    bool     `json:"email_notifications"`
	TaskCompletionSound   bool     `json:"task_completion_sound"`
	AutoShowJobStatus     bool     `json:"auto_show_job_status"`     // 任务完成后自动显示状态
	NotifyOnLongTasks     bool     `json:"notify_on_long_tasks"`     // 长任务完成后通知
	LongTaskThresholdSec  int      `json:"long_task_threshold_sec"`  // 超过此秒数视为长任务
	MutedClarificationTypes []string `json:"muted_clarification_types"` // 不通知的澄清类型
}

// PrivacySettings 隐私设置
type PrivacySettings struct {
	StoreConversationHistory bool `json:"store_conversation_history"` // 是否存储对话历史
	AllowMemoryLearning      bool `json:"allow_memory_learning"`      // 是否允许记忆学习
	ShareUsageAnalytics      bool `json:"share_usage_analytics"`      // 是否共享使用数据
	RetainFilesDays          int  `json:"retain_files_days"`          // 文件保留天数，0 表示永久
}

// DefaultUserPreference 返回默认用户偏好
func DefaultUserPreference(userID string) *UserPreference {
	now := time.Now().UTC()
	return &UserPreference{
		ID:                    GeneratePreferenceID(),
		UserID:                userID,
		CreatedAt:             now,
		UpdatedAt:             now,
		TonePreference:        ToneCasual,
		DefaultOutputFormat:   OutputMarkdown,
		PreferredLanguage:     "zh-CN",
		UITheme:               ThemeSystem,
		AutoSkipClarification: []ClarificationType{}, // 默认不跳过任何澄清
		ClarificationThreshold: 0.7,
		StudioDefaults: StudioDefaults{
			HTMLTheme:          "modern",
			PPTTheme:           "professional",
			AudioVoice:         "zh-female-1",
			AudioSpeed:         1.0,
			InfographicTheme:   "modern",
			QuizDifficulty:     "mixed",
			TableStyle:         "card",
			TableExportFormats: []string{"html", "csv"},
		},
		CustomPrompts: CustomPromptOverrides{},
		Notifications: NotificationPrefs{
			EmailNotifications:     true,
			TaskCompletionSound:    true,
			AutoShowJobStatus:      true,
			NotifyOnLongTasks:      true,
			LongTaskThresholdSec:   60,
			MutedClarificationTypes: []string{},
		},
		Privacy: PrivacySettings{
			StoreConversationHistory: true,
			AllowMemoryLearning:      true,
			ShareUsageAnalytics:      false,
			RetainFilesDays:          30,
		},
	}
}

// GeneratePreferenceID 生成偏好设置 ID
func GeneratePreferenceID() string {
	return fmt.Sprintf("pref_%d", time.Now().UnixNano())
}

// ShouldSkipClarification 检查是否应该跳过某种澄清
func (p *UserPreference) ShouldSkipClarification(ct ClarificationType) bool {
	for _, skip := range p.AutoSkipClarification {
		if skip == ct {
			return true
		}
	}
	return false
}

// GetToneInstruction 获取语气指令
func (p *UserPreference) GetToneInstruction() string {
	if p.CustomPrompts.ToneInstruction != "" {
		return p.CustomPrompts.ToneInstruction
	}

	switch p.TonePreference {
	case ToneFormal:
		return "Use a formal and professional tone. Be precise and avoid colloquialisms."
	case ToneCasual:
		return "Use a friendly and conversational tone. Be approachable while remaining helpful."
	case ToneTechnical:
		return "Use technical terminology appropriately. Provide detailed explanations and cite specific implementation details."
	case ToneConcise:
		return "Be brief and direct. Focus on essential information only."
	default:
		return ""
	}
}

// GetLanguageInstruction 获取语言指令
func (p *UserPreference) GetLanguageInstruction() string {
	if p.PreferredLanguage == "" {
		return ""
	}

	langNames := map[string]string{
		"zh-CN": "Simplified Chinese",
		"zh-TW": "Traditional Chinese",
		"en-US": "English (US)",
		"en-GB": "English (UK)",
		"ja-JP": "Japanese",
		"ko-KR": "Korean",
		"fr-FR": "French",
		"de-DE": "German",
		"es-ES": "Spanish",
	}

	langName := langNames[p.PreferredLanguage]
	if langName == "" {
		langName = p.PreferredLanguage
	}

	return fmt.Sprintf("Always respond in %s unless explicitly requested otherwise.", langName)
}

// BuildSystemPromptOverlay 构建系统提示词覆盖
func (p *UserPreference) BuildSystemPromptOverlay() string {
	var parts []string

	if prefix := strings.TrimSpace(p.CustomPrompts.SystemPromptPrefix); prefix != "" {
		parts = append(parts, prefix)
	}

	if tone := p.GetToneInstruction(); tone != "" {
		parts = append(parts, "Tone: "+tone)
	}

	if lang := p.GetLanguageInstruction(); lang != "" {
		parts = append(parts, lang)
	}

	if respFormat := strings.TrimSpace(p.CustomPrompts.ResponseFormatRule); respFormat != "" {
		parts = append(parts, "Response Format: "+respFormat)
	}

	if codeStyle := strings.TrimSpace(p.CustomPrompts.CodeStylePreference); codeStyle != "" {
		parts = append(parts, "Code Style: "+codeStyle)
	}

	if analysis := strings.TrimSpace(p.CustomPrompts.AnalysisDepth); analysis != "" {
		parts = append(parts, "Analysis Depth: "+analysis)
	}

	if suffix := strings.TrimSpace(p.CustomPrompts.SystemPromptSuffix); suffix != "" {
		parts = append(parts, suffix)
	}

	return strings.Join(parts, "\n\n")
}

// StudioDefaultsForType 获取特定 Studio 类型的默认配置
func (p *UserPreference) StudioDefaultsForType(skillType string) map[string]interface{} {
	switch strings.ToLower(skillType) {
	case "html":
		return map[string]interface{}{
			"theme": p.StudioDefaults.HTMLTheme,
		}
	case "ppt":
		return map[string]interface{}{
			"theme": p.StudioDefaults.PPTTheme,
		}
	case "audio":
		return map[string]interface{}{
			"voice": p.StudioDefaults.AudioVoice,
			"speed": p.StudioDefaults.AudioSpeed,
		}
	case "infographic":
		return map[string]interface{}{
			"theme": p.StudioDefaults.InfographicTheme,
		}
	case "quiz":
		return map[string]interface{}{
			"difficulty": p.StudioDefaults.QuizDifficulty,
		}
	case "data-table":
		return map[string]interface{}{
			"style":         p.StudioDefaults.TableStyle,
			"export_formats": p.StudioDefaults.TableExportFormats,
		}
	default:
		return map[string]interface{}{}
	}
}

// Validate 验证偏好设置
func (p *UserPreference) Validate() error {
	if p.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	if p.ClarificationThreshold < 0 || p.ClarificationThreshold > 1 {
		return fmt.Errorf("clarification_threshold must be between 0 and 1")
	}

	if p.StudioDefaults.AudioSpeed < 0.5 || p.StudioDefaults.AudioSpeed > 2.0 {
		return fmt.Errorf("audio_speed must be between 0.5 and 2.0")
	}

	return nil
}

// ToJSON 转换为 JSON 字符串
func (p *UserPreference) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON 从 JSON 字符串解析
func (p *UserPreference) FromJSON(data string) error {
	return json.Unmarshal([]byte(data), p)
}

// MergeWithDefaults 与默认值合并（非零值优先）
func (p *UserPreference) MergeWithDefaults() *UserPreference {
	defaults := DefaultUserPreference(p.UserID)

	if p.TonePreference == "" {
		p.TonePreference = defaults.TonePreference
	}
	if p.DefaultOutputFormat == "" {
		p.DefaultOutputFormat = defaults.DefaultOutputFormat
	}
	if p.PreferredLanguage == "" {
		p.PreferredLanguage = defaults.PreferredLanguage
	}
	if p.UITheme == "" {
		p.UITheme = defaults.UITheme
	}
	if p.ClarificationThreshold == 0 {
		p.ClarificationThreshold = defaults.ClarificationThreshold
	}

	return p
}

// IsClarificationMuted 检查澄清类型是否被静音
func (p *UserPreference) IsClarificationMuted(ct ClarificationType) bool {
	for _, muted := range p.Notifications.MutedClarificationTypes {
		if muted == string(ct) {
			return true
		}
	}
	return false
}

// PreferenceStore 偏好存储接口
type PreferenceStore interface {
	GetByUserID(userID string) (*UserPreference, error)
	Save(pref *UserPreference) error
	Update(pref *UserPreference) error
	Delete(userID string) error
}
