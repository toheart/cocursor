package rag

// MessageMetadata 消息级别元数据
type MessageMetadata struct {
	SessionID    string
	MessageID    string
	WorkspaceID  string
	ProjectID    string
	ProjectName  string
	MessageType  string
	MessageIndex int
	TurnIndex    int
	VectorID     string
	ContentHash  string
	FilePath     string
	FileMtime    int64
	IndexedAt    int64
}

// TurnMetadata 对话对级别元数据
type TurnMetadata struct {
	SessionID      string
	TurnIndex      int
	WorkspaceID    string
	ProjectID      string
	ProjectName    string
	UserMessageIDs []string
	AIMessageIDs   []string
	MessageCount   int
	VectorID       string
	ContentHash    string
	FilePath       string
	FileMtime      int64
	IndexedAt      int64
	IsIncomplete   bool
	Summary        *TurnSummary `json:"summary,omitempty"` // 对话总结 (JSON 字符串存储)
}
