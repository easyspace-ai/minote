package notex

import "log"

// Config defines runtime options for Notex business server.
type Config struct {
	Addr         string
	DataRoot     string
	AuthRequired bool
	Logger       *log.Logger
	Store        *Store
	// SkillsPaths lists directories to scan for SKILL.md (see skills_domain). Empty = default: DataRoot/skills + cwd/skills.
	SkillsPaths []string
	// Redis配置
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	CacheEnabled  bool
}

type User struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"created_at"`
}

type Library struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	ChunkSize    int    `json:"chunk_size"`
	ChunkOverlap int    `json:"chunk_overlap"`
}

type UploadFileInput struct {
	FileName   string `json:"file_name"`
	Base64Data string `json:"base64_data"`
}

type Document struct {
	ID           int64  `json:"id"`
	LibraryID    int64  `json:"library_id"`
	OriginalName string `json:"original_name"`
	Base64Data   string `json:"-"`
	FileSize     int    `json:"file_size"`
	MimeType     string `json:"mime_type"`
	CreatedAt    string `json:"created_at"`
	Starred      bool   `json:"starred"`
	// FilePath is relative to Server DataRoot (e.g. documents/1/42_report.pdf).
	FilePath         string `json:"-"`
	ExtractedText    string `json:"-"`
	ExtractionStatus string `json:"extraction_status,omitempty"`
	ExtractionError  string `json:"extraction_error,omitempty"`
}

// Document extraction lifecycle (three/notex-style async parse → Source.Content equivalent).
const (
	DocExtractionPending    = "pending"
	DocExtractionProcessing = "processing"
	DocExtractionCompleted  = "completed"
	DocExtractionError      = "error"
)

// StudioScopeSettings is persisted per project (Studio「生成范围与自定义」).
type StudioScopeSettings struct {
	IncludeChat     bool   `json:"includeChat"`
	ChatSummaryOnly bool   `json:"chatSummaryOnly"`
	ChatMaxMessages string `json:"chatMaxMessages"`
	CustomExtra     string `json:"customExtra"`
}

type Project struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	LibraryID   int64  `json:"library_id"`
	Starred     bool   `json:"starred"`
	Archived    bool   `json:"archived"`
	IconIndex   int    `json:"icon_index"`  // 0–n for picker; -1 = derive from id
	AccentHex   string `json:"accent_hex"` // optional #RRGGBB for icon tile
	StudioScope StudioScopeSettings `json:"studio_scope"`
}

type Material struct {
	ID        int64          `json:"id"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
	ProjectID string         `json:"project_id"`
	Kind      string         `json:"kind"`
	Title     string         `json:"title"`
	Status    string         `json:"status"`
	Subtitle  string         `json:"subtitle"`
	Payload   map[string]any `json:"payload"`
	FilePath  string         `json:"-"`
}

type Conversation struct {
	ID          int64   `json:"id"`
	AgentID     int64   `json:"agent_id"`
	Name        string  `json:"name"`
	LastMessage string  `json:"last_message"`
	LibraryIDs  []int64 `json:"library_ids"`
	ChatMode    string  `json:"chat_mode"`
	ThreadID    string  `json:"thread_id,omitempty"`
	// StudioOnly marks a hidden backend conversation used only for Studio generations
	// (not listed in GET /api/v1/conversations).
	StudioOnly bool `json:"studio_only"`
}

type Message struct {
	ID             int64  `json:"id"`
	ConversationID int64  `json:"conversation_id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	Status         string `json:"status"`
}

type Agent struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}