package models

import (
	"time"
)

// 注意：User模型已简化，仅保留核心字段
// 这里保留业务相关的模型定义（Order, OperationLog等）

// Order 订单表
type Order struct {
	OrderID      string     `gorm:"primaryKey;column:order_id;size:32" json:"order_id"`
	UserID       uint       `gorm:"column:user_id;not null;index" json:"user_id"`
	PackageID    uint       `gorm:"column:package_id;not null" json:"package_id"`
	Amount       int        `gorm:"not null" json:"amount"`
	Status       string     `gorm:"default:PENDING;not null;index" json:"status"`
	PayChannel   string     `gorm:"column:pay_channel" json:"pay_channel"`
	PayTradeNo   string     `gorm:"column:pay_trade_no;size:64" json:"pay_trade_no"`
	CallbackData string     `gorm:"type:text;column:callback_data" json:"callback_data"`
	CreateTime   time.Time  `gorm:"column:create_time;not null;index" json:"create_time"`
	PayTime      *time.Time `gorm:"column:pay_time" json:"pay_time"`
	ExpireTime   *time.Time `gorm:"column:expire_time" json:"expire_time"`

	User User `gorm:"foreignKey:UserID"`
}

func (Order) TableName() string {
	return "order"
}

// TokenRecord Token记录表
type TokenRecord struct {
	RecordID      uint      `gorm:"primaryKey;column:record_id" json:"record_id"`
	UserID        uint      `gorm:"column:user_id;not null" json:"user_id"`
	OrderID       *string   `gorm:"column:order_id;size:32" json:"order_id"`
	Type          string    `gorm:"size:20;not null" json:"type"` // RECHARGE/DEDUCT
	Amount        int       `gorm:"not null" json:"amount"`
	BalanceBefore int       `gorm:"column:balance_before;not null" json:"balance_before"`
	BalanceAfter  int       `gorm:"column:balance_after;not null" json:"balance_after"`
	Remark        string    `gorm:"type:text" json:"remark"`
	CreateTime    time.Time `gorm:"column:create_time;not null" json:"create_time"`

	User User `gorm:"foreignKey:UserID"`
}

func (TokenRecord) TableName() string {
	return "token_record"
}

// ApiKey API密钥表
type ApiKey struct {
	KeyID      string     `gorm:"primaryKey;column:key_id;size:64" json:"key_id"`
	UserID     uint       `gorm:"column:user_id;not null" json:"user_id"`
	KeyName    string     `gorm:"size:100;not null" json:"key_name"`
	ApiKey     string     `gorm:"column:api_key;size:255;not null" json:"api_key"`
	IsActive   bool       `gorm:"default:true" json:"is_active"`
	LastUsed   *time.Time `gorm:"column:last_used" json:"last_used"`
	CreateTime time.Time  `gorm:"column:create_time;not null" json:"create_time"`

	User User `gorm:"foreignKey:UserID"`
}

func (ApiKey) TableName() string {
	return "api_key"
}

// OperationLog 操作日志表
type OperationLog struct {
	LogID         uint      `gorm:"primaryKey;column:log_id" json:"log_id"`
	UserID        uint      `gorm:"column:user_id;not null" json:"user_id"`
	OperationType string    `gorm:"column:operation_type;size:50;not null" json:"operation_type"`
	ResourceType  string    `gorm:"column:resource_type;size:50;not null" json:"resource_type"`
	ResourceID    string    `gorm:"column:resource_id;size:100" json:"resource_id"`
	Action        string    `gorm:"size:50;not null" json:"action"`
	Detail        string    `gorm:"type:text" json:"detail"`
	IPAddress     string    `gorm:"column:ip_address;size:50" json:"ip_address"`
	UserAgent     string    `gorm:"column:user_agent;size:500" json:"user_agent"`
	Status        string    `gorm:"default:success;size:20" json:"status"`
	ErrorMessage  string    `gorm:"type:text;column:error_message" json:"error_message"`
	CreateTime    time.Time `gorm:"column:create_time;not null" json:"create_time"`

	User User `gorm:"foreignKey:UserID"`
}

func (OperationLog) TableName() string {
	return "operation_log"
}

// ===== AI阅读相关模型 =====

// BookCategory 图书分类
type BookCategory struct {
	CategoryID  uint          `gorm:"primaryKey;column:category_id" json:"category_id"`
	Name        string        `gorm:"size:100;not null" json:"name"`
	Description string        `gorm:"type:text" json:"description"`
	ParentID    *uint         `gorm:"column:parent_id" json:"parent_id"`
	Parent      *BookCategory `gorm:"foreignKey:ParentID"`
	CreateTime  time.Time     `gorm:"column:create_time" json:"create_time"`
	UpdateTime  time.Time     `gorm:"column:update_time" json:"update_time"`

	Books []Book `gorm:"foreignKey:CategoryID"`
}

func (BookCategory) TableName() string {
	return "book_categories"
}

// Book 图书
type Book struct {
	BookID      uint          `gorm:"primaryKey;column:book_id" json:"book_id"`
	Title       string        `gorm:"size:200;not null" json:"title"`
	Author      string        `gorm:"size:100;not null" json:"author"`
	Description string        `gorm:"type:text" json:"description"`
	CategoryID  *uint         `gorm:"column:category_id" json:"category_id"`
	Category    *BookCategory `gorm:"foreignKey:CategoryID"`
	Format      string        `gorm:"size:10;not null" json:"format"`
	FilePath    string        `gorm:"size:500" json:"file_path"`
	FileSize    int64         `gorm:"not null;default:0" json:"file_size"`
	Pages       int           `gorm:"default:0" json:"pages"`
	Status      string        `gorm:"size:20;default:processing" json:"status"`
	Tags        string        `gorm:"type:json" json:"tags"`
	Metadata    string        `gorm:"type:json" json:"metadata"`
	UploadedBy  uint          `gorm:"column:uploaded_by;not null" json:"uploaded_by"`
	Uploader    User          `gorm:"foreignKey:UploadedBy"`
	CreateTime  time.Time     `gorm:"column:create_time" json:"create_time"`
	UpdateTime  time.Time     `gorm:"column:update_time" json:"update_time"`

	// 关系
	ReadingProgress []ReadingProgress `gorm:"foreignKey:BookID"`
	Bookmarks       []Bookmark        `gorm:"foreignKey:BookID"`
	ReadingSettings []ReadingSettings `gorm:"foreignKey:BookID"`
}

func (Book) TableName() string {
	return "books"
}

// ReadingProgress 阅读进度
type ReadingProgress struct {
	ProgressID         uint      `gorm:"primaryKey;column:progress_id" json:"progress_id"`
	UserID             uint      `gorm:"column:user_id;not null" json:"user_id"`
	User               User      `gorm:"foreignKey:UserID"`
	BookID             uint      `gorm:"column:book_id;not null" json:"book_id"`
	Book               Book      `gorm:"foreignKey:BookID"`
	CurrentPage        int       `gorm:"default:1" json:"current_page"`
	TotalPages         int       `gorm:"default:0" json:"total_pages"`
	ProgressPercentage float64   `gorm:"type:decimal(5,2);default:0.00" json:"progress_percentage"`
	LastReadAt         time.Time `gorm:"column:last_read_at;autoUpdateTime" json:"last_read_at"`
	CreateTime         time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime         time.Time `gorm:"column:update_time" json:"update_time"`
}

func (ReadingProgress) TableName() string {
	return "reading_progress"
}

// Bookmark 书签
type Bookmark struct {
	BookmarkID uint      `gorm:"primaryKey;column:bookmark_id" json:"bookmark_id"`
	UserID     uint      `gorm:"column:user_id;not null" json:"user_id"`
	User       User      `gorm:"foreignKey:UserID"`
	BookID     uint      `gorm:"column:book_id;not null" json:"book_id"`
	Book       Book      `gorm:"foreignKey:BookID"`
	Page       int       `gorm:"not null" json:"page"`
	Position   float64   `gorm:"type:decimal(5,2);default:0.00" json:"position"`
	Note       string    `gorm:"type:text" json:"note"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
}

func (Bookmark) TableName() string {
	return "bookmarks"
}

// ReadingSettings 阅读设置
type ReadingSettings struct {
	SettingsID uint      `gorm:"primaryKey;column:settings_id" json:"settings_id"`
	UserID     uint      `gorm:"column:user_id;not null" json:"user_id"`
	User       User      `gorm:"foreignKey:UserID"`
	BookID     uint      `gorm:"column:book_id;not null" json:"book_id"`
	Book       Book      `gorm:"foreignKey:BookID"`
	FontSize   string    `gorm:"size:10;default:medium" json:"font_size"`
	Theme      string    `gorm:"size:10;default:light" json:"theme"`
	LineHeight float64   `gorm:"type:decimal(3,2);default:1.50" json:"line_height"`
	Margin     int       `gorm:"default:20" json:"margin"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time" json:"update_time"`
}

func (ReadingSettings) TableName() string {
	return "reading_settings"
}

// KnowledgeBase 知识库
type KnowledgeBase struct {
	KnowledgeBaseID uint      `gorm:"primaryKey;column:knowledge_base_id" json:"knowledge_base_id"`
	Name            string    `gorm:"size:200;not null" json:"name"`
	Description     string    `gorm:"type:text" json:"description"`
	Config          string    `gorm:"type:json" json:"config"`
	OwnerID         uint      `gorm:"column:owner_id;not null" json:"owner_id"`
	Owner           User      `gorm:"foreignKey:OwnerID"`
	IsPublic        bool      `gorm:"default:false" json:"is_public"`
	Status          string    `gorm:"size:20;default:active" json:"status"`
	CreateTime      time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime      time.Time `gorm:"column:update_time" json:"update_time"`

	// 关系
	Documents []KnowledgeDocument `gorm:"foreignKey:KnowledgeBaseID"`
	Searches  []KnowledgeSearch   `gorm:"foreignKey:KnowledgeBaseID"`
}

func (KnowledgeBase) TableName() string {
	return "knowledge_bases"
}

// KnowledgeDocument 知识库文档
type KnowledgeDocument struct {
	DocumentID      uint          `gorm:"primaryKey;column:document_id" json:"document_id"`
	KnowledgeBaseID uint          `gorm:"column:knowledge_base_id;not null" json:"knowledge_base_id"`
	KnowledgeBase   KnowledgeBase `gorm:"foreignKey:KnowledgeBaseID"`
	Title           string        `gorm:"size:200;not null" json:"title"`
	Content         string        `gorm:"type:text;not null" json:"content"`
	Source          string        `gorm:"size:20;not null" json:"source"`
	SourceURL       string        `gorm:"size:500" json:"source_url"`
	FilePath        string        `gorm:"size:500" json:"file_path"`
	Metadata        string        `gorm:"type:json" json:"metadata"`
	Status          string        `gorm:"size:20;default:processing" json:"status"`
	VectorID        string        `gorm:"size:255" json:"vector_id"`
	TotalTokens     int           `gorm:"column:total_tokens;default:0" json:"total_tokens"` // 文档总token数
	ProcessingMode  string        `gorm:"column:processing_mode;size:20;default:fallback" json:"processing_mode"` // full_read | fallback
	CreateTime      time.Time     `gorm:"column:create_time" json:"create_time"`
	UpdateTime      time.Time     `gorm:"column:update_time" json:"update_time"`

	// 关系
	Chunks []KnowledgeChunk `gorm:"foreignKey:DocumentID"`
}

func (KnowledgeDocument) TableName() string {
	return "knowledge_documents"
}

// KnowledgeChunk 知识块
type KnowledgeChunk struct {
	ChunkID          uint              `gorm:"primaryKey;column:chunk_id" json:"chunk_id"`
	DocumentID       uint              `gorm:"column:document_id;not null;index" json:"document_id"`
	Document         KnowledgeDocument `gorm:"foreignKey:DocumentID"`
	Content          string            `gorm:"type:text;not null" json:"content"`
	ChunkIndex       int               `gorm:"not null;index" json:"chunk_index"`
	VectorID         string            `gorm:"size:255;not null" json:"vector_id"`
	Embedding        string            `gorm:"type:json" json:"embedding"`
	Metadata         string            `gorm:"type:json" json:"metadata"`
	TokenCount       int               `gorm:"column:token_count;default:0" json:"token_count"`                    // 当前块的token数
	PrevChunkID      *uint             `gorm:"column:prev_chunk_id;index" json:"prev_chunk_id"`                    // 前一个块的ID
	NextChunkID      *uint             `gorm:"column:next_chunk_id;index" json:"next_chunk_id"`                    // 下一个块的ID
	DocumentTotalTokens int            `gorm:"column:document_total_tokens;default:0" json:"document_total_tokens"` // 文档总token数（冗余字段，便于查询）
	ChunkPosition    int               `gorm:"column:chunk_position;default:0" json:"chunk_position"`                // 块在文档中的位置（0-based）
	RelatedChunkIDs  string            `gorm:"type:json;column:related_chunk_ids" json:"related_chunk_ids"`      // 关联块ID列表（JSON数组）
	CreateTime       time.Time         `gorm:"column:create_time" json:"create_time"`
}

func (KnowledgeChunk) TableName() string {
	return "knowledge_chunks"
}

// KnowledgeSearch 知识库搜索记录
type KnowledgeSearch struct {
	SearchID        uint          `gorm:"primaryKey;column:search_id" json:"search_id"`
	KnowledgeBaseID uint          `gorm:"column:knowledge_base_id;not null" json:"knowledge_base_id"`
	KnowledgeBase   KnowledgeBase `gorm:"foreignKey:KnowledgeBaseID"`
	UserID          uint          `gorm:"column:user_id;not null" json:"user_id"`
	User            User          `gorm:"foreignKey:UserID"`
	Query           string        `gorm:"type:text;not null" json:"query"`
	Results         string        `gorm:"type:json" json:"results"`
	CreateTime      time.Time     `gorm:"column:create_time" json:"create_time"`
}

func (KnowledgeSearch) TableName() string {
	return "knowledge_searches"
}

// ChatSession 聊天会话
type ChatSession struct {
	SessionID  uint      `gorm:"primaryKey;column:session_id" json:"session_id"`
	UserID     uint      `gorm:"column:user_id;not null" json:"user_id"`
	User       User      `gorm:"foreignKey:UserID"`
	Title      string    `gorm:"size:200;not null" json:"title"`
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time" json:"update_time"`

	// 关系
	Messages []ChatMessage `gorm:"foreignKey:SessionID"`
}

func (ChatSession) TableName() string {
	return "chat_sessions"
}

// ChatMessage 聊天消息
type ChatMessage struct {
	MessageID  uint         `gorm:"primaryKey;column:message_id" json:"message_id"`
	UserID     uint         `gorm:"column:user_id;not null" json:"user_id"`
	User       User         `gorm:"foreignKey:UserID"`
	SessionID  *uint        `gorm:"column:session_id" json:"session_id"`
	Session    *ChatSession `gorm:"foreignKey:SessionID"`
	Role       string       `gorm:"size:20;not null" json:"role"`
	Content    string       `gorm:"type:text;not null" json:"content"`
	Context    string       `gorm:"type:json" json:"context"`
	TokensUsed int          `gorm:"default:0" json:"tokens_used"`
	ModelUsed  string       `gorm:"size:100;default:gpt-4" json:"model_used"`
	CreateTime time.Time    `gorm:"column:create_time" json:"create_time"`
}

func (ChatMessage) TableName() string {
	return "chat_messages"
}

// SearchHistory 搜索历史
type SearchHistory struct {
	HistoryID  uint      `gorm:"primaryKey;column:history_id" json:"history_id"`
	UserID     uint      `gorm:"column:user_id;not null" json:"user_id"`
	User       User      `gorm:"foreignKey:UserID"`
	Query      string    `gorm:"type:text;not null" json:"query"`
	Results    string    `gorm:"type:json" json:"results"`
	Filters    string    `gorm:"type:json" json:"filters"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
}

func (SearchHistory) TableName() string {
	return "search_history"
}

// SearchSuggestion 搜索建议
type SearchSuggestion struct {
	SuggestionID uint      `gorm:"primaryKey;column:suggestion_id" json:"suggestion_id"`
	Query        string    `gorm:"size:200;not null" json:"query"`
	Suggestion   string    `gorm:"size:200;not null" json:"suggestion"`
	Frequency    int       `gorm:"default:1" json:"frequency"`
	CreateTime   time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime   time.Time `gorm:"column:update_time" json:"update_time"`
}

func (SearchSuggestion) TableName() string {
	return "search_suggestions"
}

// AssistantConfig 助手配置
type AssistantConfig struct {
	ConfigID      uint      `gorm:"primaryKey;column:config_id" json:"config_id"`
	UserID        uint      `gorm:"column:user_id;not null;unique" json:"user_id"`
	User          User      `gorm:"foreignKey:UserID"`
	Model         string    `gorm:"size:100;default:gpt-4" json:"model"`
	Temperature   float64   `gorm:"type:decimal(3,2);default:0.70" json:"temperature"`
	MaxTokens     int       `gorm:"default:2000" json:"max_tokens"`
	SystemPrompt  string    `gorm:"type:text;default:'你是一个有用的AI助手。'" json:"system_prompt"`
	ContextLength int       `gorm:"default:10" json:"context_length"`
	CreateTime    time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime    time.Time `gorm:"column:update_time" json:"update_time"`
}

func (AssistantConfig) TableName() string {
	return "assistant_configs"
}

// Workflow 工作流
type Workflow struct {
	WorkflowID  uint      `gorm:"primaryKey;column:workflow_id" json:"workflow_id"`
	Name        string    `gorm:"size:200;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Config      string    `gorm:"type:json" json:"config"`
	OwnerID     uint      `gorm:"column:owner_id;not null" json:"owner_id"`
	Owner       User      `gorm:"foreignKey:OwnerID"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreateTime  time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time" json:"update_time"`
}

func (Workflow) TableName() string {
	return "workflows"
}

// WorkflowExecution 工作流执行记录
type WorkflowExecution struct {
	ExecutionID uint       `gorm:"primaryKey;column:execution_id" json:"execution_id"`
	WorkflowID  uint       `gorm:"column:workflow_id;not null" json:"workflow_id"`
	Workflow    Workflow   `gorm:"foreignKey:WorkflowID"`
	Status      string     `gorm:"size:20;default:pending" json:"status"`      // pending, running, completed, failed, cancelled
	TriggerType string     `gorm:"size:20;default:manual" json:"trigger_type"` // manual, scheduled, webhook
	Result      string     `gorm:"type:json" json:"result"`
	Error       string     `gorm:"type:text" json:"error"`
	OperatorID  uint       `gorm:"column:operator_id" json:"operator_id"`
	Operator    User       `gorm:"foreignKey:OperatorID"`
	StartTime   time.Time  `gorm:"column:start_time" json:"start_time"`
	EndTime     *time.Time `gorm:"column:end_time" json:"end_time"`
	Duration    int64      `gorm:"column:duration" json:"duration"` // 毫秒
	Logs        string     `gorm:"type:json" json:"logs"`
	CreateTime  time.Time  `gorm:"column:create_time" json:"create_time"`
	UpdateTime  time.Time  `gorm:"column:update_time" json:"update_time"`
}

func (WorkflowExecution) TableName() string {
	return "workflow_executions"
}

// NodeExecution 节点执行记录
type NodeExecution struct {
	ExecutionID         uint              `gorm:"primaryKey;column:execution_id" json:"execution_id"`
	NodeID              string            `gorm:"size:100;not null" json:"node_id"`
	NodeType            string            `gorm:"size:50;not null" json:"node_type"`
	NodeLabel           string            `gorm:"size:200" json:"node_label"`
	Status              string            `gorm:"size:20;default:pending" json:"status"` // pending, running, completed, failed
	Input               string            `gorm:"type:json" json:"input"`
	Output              string            `gorm:"type:json" json:"output"`
	Error               string            `gorm:"type:text" json:"error"`
	Duration            int64             `gorm:"column:duration" json:"duration"` // 毫秒
	StartTime           time.Time         `gorm:"column:start_time" json:"start_time"`
	EndTime             *time.Time        `gorm:"column:end_time" json:"end_time"`
	WorkflowExecutionID uint              `gorm:"column:workflow_execution_id;not null" json:"workflow_execution_id"`
	WorkflowExecution   WorkflowExecution `gorm:"foreignKey:WorkflowExecutionID"`
	CreateTime          time.Time         `gorm:"column:create_time" json:"create_time"`
	UpdateTime          time.Time         `gorm:"column:update_time" json:"update_time"`
}

func (NodeExecution) TableName() string {
	return "node_executions"
}

// CrawlerTask 爬虫任务
type CrawlerTask struct {
	TaskID       uint       `gorm:"primaryKey;column:task_id" json:"task_id"`
	Name         string     `gorm:"size:200;not null" json:"name"`
	Description  string     `gorm:"type:text" json:"description"`
	URL          string     `gorm:"size:500;not null" json:"url"`
	Config       string     `gorm:"type:json" json:"config"`
	Status       string     `gorm:"size:20;default:pending" json:"status"`
	Result       string     `gorm:"type:json" json:"result"`
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	CreatedBy    uint       `gorm:"column:created_by;not null" json:"created_by"`
	Creator      User       `gorm:"foreignKey:CreatedBy"`
	StartedAt    *time.Time `gorm:"column:started_at" json:"started_at"`
	CompletedAt  *time.Time `gorm:"column:completed_at" json:"completed_at"`
	CreateTime   time.Time  `gorm:"column:create_time" json:"create_time"`
	UpdateTime   time.Time  `gorm:"column:update_time" json:"update_time"`
}

func (CrawlerTask) TableName() string {
	return "crawler_tasks"
}

// ProcessingTask 数据处理任务
type ProcessingTask struct {
	TaskID       uint       `gorm:"primaryKey;column:task_id" json:"task_id"`
	Name         string     `gorm:"size:200;not null" json:"name"`
	Description  string     `gorm:"type:text" json:"description"`
	TaskType     string     `gorm:"size:50;not null" json:"task_type"`
	InputFiles   string     `gorm:"type:json" json:"input_files"`
	OutputFiles  string     `gorm:"type:json" json:"output_files"`
	Config       string     `gorm:"type:json" json:"config"`
	Status       string     `gorm:"size:20;default:pending" json:"status"`
	Result       string     `gorm:"type:json" json:"result"`
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	CreatedBy    uint       `gorm:"column:created_by;not null" json:"created_by"`
	Creator      User       `gorm:"foreignKey:CreatedBy"`
	StartedAt    *time.Time `gorm:"column:started_at" json:"started_at"`
	CompletedAt  *time.Time `gorm:"column:completed_at" json:"completed_at"`
	CreateTime   time.Time  `gorm:"column:create_time" json:"create_time"`
	UpdateTime   time.Time  `gorm:"column:update_time" json:"update_time"`
}

func (ProcessingTask) TableName() string {
	return "processing_tasks"
}
