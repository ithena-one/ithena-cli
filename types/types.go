package types

// AuditRecord defines the structure for a log entry that can be sent to the platform
// or stored locally.
// Note: Fields that are pointers can be omitted (omitempty) if nil when marshalled to JSON.
// For SQLite storage, these will need to be handled as sql.NullString, sql.NullInt64 etc.
type AuditRecord struct {
	ID                string      `json:"id"`
	McpMethod         *string     `json:"mcp_method,omitempty"`
	ToolName          *string     `json:"tool_name,omitempty"`
	DurationMs        *int64      `json:"duration_ms,omitempty"`
	Status            string      `json:"status"` // "success" or "failure"
	ProxyVersion      *string     `json:"proxy_version,omitempty"`
	TargetServerAlias *string     `json:"target_server_alias,omitempty"`
	RequestPreview    interface{} `json:"request_preview,omitempty"`
	ResponsePreview   interface{} `json:"response_preview,omitempty"`
	ErrorDetails      interface{} `json:"error_details,omitempty"`
	Timestamp         string      `json:"timestamp"` // ISO 8601 format string
} 