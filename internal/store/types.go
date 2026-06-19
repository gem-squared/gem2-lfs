package store

import "encoding/json"

// Task represents a work item.
type Task struct {
	TaskID        string `json:"task_id"`
	Role          string `json:"role"`
	Title         string `json:"title"`
	Description   string `json:"description,omitempty"`
	Status        string `json:"status"`
	Priority      string `json:"priority,omitempty"`
	ProjectSlug   string `json:"project_slug,omitempty"`
	PlanDoc       string `json:"plan_doc,omitempty"`
	ResultSummary string `json:"result_summary,omitempty"`
	ParentTaskID  string `json:"parent_task_id,omitempty"`
	Tags          Tags   `json:"tags,omitempty"`
	MetadataJSON  string `json:"metadata_json,omitempty"`
	State         string `json:"state,omitempty"`
	StartedAt     string `json:"started_at,omitempty"`
	CompletedAt   string `json:"completed_at,omitempty"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// Message represents a protocol message.
type Message struct {
	MsgID         string `json:"msg_id"`
	MsgType       string `json:"msg_type,omitempty"`
	FromRole      string `json:"from_role"`
	ToRole        string `json:"to_role"`
	MessageText   string `json:"message"`
	Content       string `json:"content,omitempty"`
	ProjectSlug   string `json:"project_slug,omitempty"`
	GServiceID    string `json:"g_service_id,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
	TerminalID    string `json:"terminal_id,omitempty"`
	TerminalTitle string `json:"terminal_title,omitempty"`
	MetadataJSON  string `json:"metadata_json,omitempty"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// Decision represents an architectural decision.
type Decision struct {
	DecisionID   string `json:"decision_id"`
	Role         string `json:"role"`
	Category     string `json:"category"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	Status       string `json:"status"`
	SupersededBy string `json:"superseded_by,omitempty"`
	ProjectSlug  string `json:"project_slug,omitempty"`
	Tags         Tags   `json:"tags,omitempty"`
	MetadataJSON string `json:"metadata_json,omitempty"`
	DecidedAt    string `json:"decided_at,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// Knowledge represents a knowledge document.
type Knowledge struct {
	KnowledgeID    string         `json:"knowledge_id"`
	EntityType     string         `json:"entity_type"`
	Title          string         `json:"title"`
	ContentRaw     string         `json:"content_raw,omitempty"`
	ContentNL      string         `json:"content_nl,omitempty"`
	ProjectSlug    string         `json:"project_slug,omitempty"`
	Tags           Tags           `json:"tags,omitempty"`
	FilePath       string         `json:"file_path,omitempty"`
	LineCount      int            `json:"line_count,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	EmbeddingModel string         `json:"embedding_model,omitempty"`
	EmbeddingDim   int            `json:"embedding_dim,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

// KnowledgeChunk represents a chunk of a knowledge document.
type KnowledgeChunk struct {
	ChunkID     string `json:"chunk_id"`
	KnowledgeID string `json:"knowledge_id"`
	ChunkIndex  int    `json:"chunk_index"`
	ContentNL   string `json:"content_nl,omitempty"`
	Tags        Tags   `json:"tags,omitempty"`
	Heading     string `json:"heading,omitempty"`
	Text        string `json:"text,omitempty"`
}

// Edge represents a provenance link between knowledge documents.
type Edge struct {
	EdgeID       string `json:"edge_id"`
	SourceID     string `json:"source_id"`
	TargetID     string `json:"target_id"`
	ProjectSlug  string `json:"project_slug,omitempty"`
	Tags         Tags   `json:"tags,omitempty"`
	MetadataJSON string `json:"metadata_json,omitempty"`
	CreatedAt    string `json:"created_at"`
}

// Skill represents a skill (grounding prompt or formal contract).
type Skill struct {
	SkillID    string `json:"skill_id"`
	UserID     string `json:"user_id"`
	SkillType  string `json:"skill_type"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Tags       Tags   `json:"tags,omitempty"`
	SourceTool string `json:"source_tool,omitempty"`
	Version    int    `json:"version"`
	Metadata   string `json:"metadata,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// Project represents a registered project.
type Project struct {
	ProjectSlug  string `json:"project_slug"`
	Name         string `json:"name"`
	RootPath     string `json:"root_path,omitempty"`
	Status       string `json:"status,omitempty"`
	MetadataJSON string `json:"metadata_json,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// SemanticResult is a single result from semantic search.
type SemanticResult struct {
	Chunk     KnowledgeChunk `json:"chunk"`
	Knowledge Knowledge      `json:"knowledge"`
	Score     float64        `json:"score"`
}

// EmbeddingModel describes a registered embedding model.
type EmbeddingModel struct {
	ModelName  string `json:"model_name"`
	Provider   string `json:"provider"`
	Dimensions int    `json:"dimensions"`
	Status     string `json:"status"`
}

// Tags is a JSON-encoded string slice for SQLite storage.
type Tags []string

func (t Tags) String() string {
	b, _ := json.Marshal(t)
	return string(b)
}

// ParseTags parses a JSON-encoded tags string.
func ParseTags(s string) Tags {
	if s == "" || s == "null" {
		return nil
	}
	var tags Tags
	json.Unmarshal([]byte(s), &tags)
	return tags
}

// Pagination controls result paging.
type Pagination struct {
	PageSize  int    `json:"page_size,omitempty"`
	PageToken string `json:"page_token,omitempty"`
}
