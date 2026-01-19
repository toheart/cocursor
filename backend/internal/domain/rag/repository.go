package rag

// RAGRepository RAG 元数据仓库接口
type RAGRepository interface {
	// MessageMetadata 相关方法
	SaveMessageMetadata(metadata *MessageMetadata) error
	GetMessageMetadata(sessionID, messageID string) (*MessageMetadata, error)
	GetMessageMetadataBySession(sessionID string) ([]*MessageMetadata, error)
	GetMessageMetadataByProject(projectID string) ([]*MessageMetadata, error)
	DeleteMessageMetadata(sessionID, messageID string) error

	// TurnMetadata 相关方法
	SaveTurnMetadata(metadata *TurnMetadata) error
	GetTurnMetadata(sessionID string, turnIndex int) (*TurnMetadata, error)
	GetTurnMetadataBySession(sessionID string) ([]*TurnMetadata, error)
	GetTurnMetadataByProject(projectID string) ([]*TurnMetadata, error)
	GetIncompleteTurns(sessionID string) ([]*TurnMetadata, error)
	DeleteTurnMetadata(sessionID string, turnIndex int) error

	// 文件元数据查询
	GetFileMetadata(filePath string) (*FileMetadata, error)
	UpdateFileMtime(filePath string, mtime int64) error
	
	// 清空数据
	ClearAllMetadata() error
}

// FileMetadata 文件元数据（用于快速检测文件变化）
type FileMetadata struct {
	FilePath    string
	FileMtime   int64
	ContentHash string
	LastIndexed int64
}
