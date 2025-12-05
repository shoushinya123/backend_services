package models

import (
	"time"
)

// MCPServer MCP服务表
type MCPServer struct {
	ServerID      uint      `gorm:"primaryKey;column:server_id" json:"server_id"`
	Name          string    `gorm:"size:100;not null" json:"name"`
	Description   string    `gorm:"type:text" json:"description"`
	AuthorID      *uint     `gorm:"column:author_id" json:"author_id"`
	ServerURL     string    `gorm:"column:server_url;size:500;not null" json:"server_url"`
	ServerType    string    `gorm:"column:server_type;size:50;not null" json:"server_type"` // HTTP/SSE/STDIO
	AuthType      string    `gorm:"column:auth_type;size:50" json:"auth_type"`               // NONE/API_KEY/BEARER/OAUTH
	AuthConfig    string    `gorm:"type:text;column:auth_config" json:"auth_config"`         // JSON配置
	Category      string    `gorm:"size:50" json:"category"`                                 // TOOL/RESOURCE/PROMPT/MIXED
	Tags          string    `gorm:"type:text" json:"tags"`                                   // JSON数组
	IconURL       string    `gorm:"column:icon_url;size:500" json:"icon_url"`
	CoverImage    string    `gorm:"column:cover_image;size:500" json:"cover_image"`
	Version       string    `gorm:"size:20" json:"version"`
	Status        string    `gorm:"size:20;default:ACTIVE" json:"status"` // ACTIVE/INACTIVE/REVIEWING/BANNED
	IsPublic      bool      `gorm:"column:is_public;default:false" json:"is_public"`
	IsVerified    bool      `gorm:"column:is_verified;default:false" json:"is_verified"`
	TotalInstalls int       `gorm:"column:total_installs;default:0" json:"total_installs"`
	TotalRatings  int       `gorm:"column:total_ratings;default:0" json:"total_ratings"`
	AverageRating float64   `gorm:"column:average_rating;type:decimal(3,2);default:0.00" json:"average_rating"`
	CreateTime    time.Time `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime    time.Time `gorm:"column:update_time;not null" json:"update_time"`

	// 关系
	Author    *User            `gorm:"foreignKey:AuthorID"`
	Tools     []MCPTool        `gorm:"foreignKey:ServerID"`
	Resources []MCPResource    `gorm:"foreignKey:ServerID"`
	Prompts   []MCPPrompt      `gorm:"foreignKey:ServerID"`
	UserServers []UserMCPServer `gorm:"foreignKey:ServerID"`
	Ratings   []MCPServerRating `gorm:"foreignKey:ServerID"`
}

func (MCPServer) TableName() string {
	return "mcp_server"
}

// MCPTool MCP工具表
type MCPTool struct {
	ToolID      uint      `gorm:"primaryKey;column:tool_id" json:"tool_id"`
	ServerID    uint      `gorm:"column:server_id;not null;index" json:"server_id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	InputSchema string    `gorm:"type:text;column:input_schema" json:"input_schema"`   // JSON Schema
	OutputSchema string   `gorm:"type:text;column:output_schema" json:"output_schema"` // JSON Schema
	Category    string    `gorm:"size:50" json:"category"`
	IsActive    bool      `gorm:"column:is_active;default:true" json:"is_active"`
	CreateTime  time.Time `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time;not null" json:"update_time"`

	// 关系
	Server MCPServer `gorm:"foreignKey:ServerID"`
}

func (MCPTool) TableName() string {
	return "mcp_tool"
}

// MCPResource MCP资源表
type MCPResource struct {
	ResourceID  uint      `gorm:"primaryKey;column:resource_id" json:"resource_id"`
	ServerID    uint      `gorm:"column:server_id;not null;index" json:"server_id"`
	URI         string    `gorm:"size:500;not null" json:"uri"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	MimeType    string    `gorm:"column:mime_type;size:100" json:"mime_type"`
	Size        int64     `gorm:"type:bigint" json:"size"`
	IsActive    bool      `gorm:"column:is_active;default:true" json:"is_active"`
	CreateTime  time.Time `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time;not null" json:"update_time"`

	// 关系
	Server MCPServer `gorm:"foreignKey:ServerID"`
}

func (MCPResource) TableName() string {
	return "mcp_resource"
}

// MCPPrompt MCP提示词表
type MCPPrompt struct {
	PromptID    uint      `gorm:"primaryKey;column:prompt_id" json:"prompt_id"`
	ServerID    uint      `gorm:"column:server_id;not null;index" json:"server_id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Template    string    `gorm:"type:text;not null" json:"template"`
	Arguments   string    `gorm:"type:text" json:"arguments"` // JSON Schema for arguments
	Category    string    `gorm:"size:50" json:"category"`
	IsActive    bool      `gorm:"column:is_active;default:true" json:"is_active"`
	CreateTime  time.Time `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time;not null" json:"update_time"`

	// 关系
	Server MCPServer `gorm:"foreignKey:ServerID"`
}

func (MCPPrompt) TableName() string {
	return "mcp_prompt"
}

// UserMCPServer 用户MCP服务关联表
type UserMCPServer struct {
	ID              uint       `gorm:"primaryKey;column:id" json:"id"`
	UserID          uint       `gorm:"column:user_id;not null;index" json:"user_id"`
	ServerID        uint       `gorm:"column:server_id;not null;index" json:"server_id"`
	CustomConfig    string     `gorm:"type:text;column:custom_config" json:"custom_config"` // JSON
	ConnectionStatus string    `gorm:"column:connection_status;size:20;default:DISCONNECTED" json:"connection_status"` // CONNECTED/DISCONNECTED/ERROR
	LastConnectedAt *time.Time `gorm:"column:last_connected_at" json:"last_connected_at"`
	LastError       string     `gorm:"type:text;column:last_error" json:"last_error"`
	IsFavorite      bool       `gorm:"column:is_favorite;default:false" json:"is_favorite"`
	CreateTime      time.Time  `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime      time.Time  `gorm:"column:update_time;not null" json:"update_time"`

	// 关系
	User   User      `gorm:"foreignKey:UserID"`
	Server MCPServer `gorm:"foreignKey:ServerID"`
}

func (UserMCPServer) TableName() string {
	return "user_mcp_server"
}

// MCPServerRating MCP服务评分表
type MCPServerRating struct {
	RatingID  uint      `gorm:"primaryKey;column:rating_id" json:"rating_id"`
	ServerID  uint      `gorm:"column:server_id;not null;index" json:"server_id"`
	UserID    uint      `gorm:"column:user_id;not null;index" json:"user_id"`
	Rating    int       `gorm:"not null;check:rating >= 1 AND rating <= 5" json:"rating"`
	Comment   string    `gorm:"type:text" json:"comment"`
	CreateTime time.Time `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;not null" json:"update_time"`

	// 关系
	Server MCPServer `gorm:"foreignKey:ServerID"`
	User   User      `gorm:"foreignKey:UserID"`
}

func (MCPServerRating) TableName() string {
	return "mcp_server_rating"
}

// MCPToolCall MCP工具调用记录表
type MCPToolCall struct {
	CallID        uint      `gorm:"primaryKey;column:call_id" json:"call_id"`
	UserID        uint      `gorm:"column:user_id;not null;index" json:"user_id"`
	ServerID      uint      `gorm:"column:server_id;not null;index" json:"server_id"`
	ToolID        uint      `gorm:"column:tool_id;not null;index" json:"tool_id"`
	InputData     string    `gorm:"type:text;column:input_data" json:"input_data"`     // JSON
	OutputData    string    `gorm:"type:text;column:output_data" json:"output_data"`   // JSON
	Status        string    `gorm:"size:20;not null" json:"status"`                     // SUCCESS/FAILED/TIMEOUT
	ErrorMessage  string    `gorm:"type:text;column:error_message" json:"error_message"`
	ExecutionTime int       `gorm:"column:execution_time_ms" json:"execution_time_ms"` // 执行时间（毫秒）
	CreateTime    time.Time `gorm:"column:create_time;not null" json:"create_time"`

	// 关系
	User   User      `gorm:"foreignKey:UserID"`
	Server MCPServer `gorm:"foreignKey:ServerID"`
	Tool   MCPTool   `gorm:"foreignKey:ToolID"`
}

func (MCPToolCall) TableName() string {
	return "mcp_tool_call"
}

