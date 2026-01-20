package rag

// KnowledgeChunk 知识片段模型
// 表示一个完整的问答对，是 RAG 检索的最小有意义单位
type KnowledgeChunk struct {
	// 基础标识
	ID         string // UUID，同时作为 Qdrant point_id
	SessionID  string // 会话 ID
	ChunkIndex int    // 在会话中的索引

	// 项目信息
	ProjectID   string // 项目 ID
	ProjectName string // 项目名称
	WorkspaceID string // 工作区 ID

	// 核心内容
	UserQuery      string // 用户问题（完整）
	AIResponseCore string // AI 核心回答（去除代码块后）
	VectorText     string // 组合后的向量化文本

	// 元数据
	ToolsUsed     []string // 使用的工具列表
	FilesModified []string // 修改的文件列表
	CodeLanguages []string // 代码块语言列表
	HasCode       bool     // 是否包含代码

	// LLM 增强内容
	Summary          string   // 一句话总结
	MainTopic        string   // 主要主题
	Tags             []string // 标签
	EnrichmentStatus string   // pending/processing/completed/failed
	EnrichmentError  string   // 增强失败的错误信息

	// 索引信息
	Timestamp   int64  // 对话时间戳
	ContentHash string // 内容哈希，用于去重
	FilePath    string // 源文件路径
	IndexedAt   int64  // 索引时间
}

// ChunkEnrichment 增强内容
type ChunkEnrichment struct {
	Summary   string   // 一句话总结
	MainTopic string   // 主要主题
	Tags      []string // 标签
}

// 增强状态常量
const (
	EnrichmentStatusPending    = "pending"
	EnrichmentStatusProcessing = "processing"
	EnrichmentStatusCompleted  = "completed"
	EnrichmentStatusFailed     = "failed"
)

// IsEnriched 检查是否已完成增强
func (c *KnowledgeChunk) IsEnriched() bool {
	return c.EnrichmentStatus == EnrichmentStatusCompleted
}

// UserQueryPreview 获取用户问题预览（前 200 字符）
func (c *KnowledgeChunk) UserQueryPreview() string {
	if len(c.UserQuery) <= 200 {
		return c.UserQuery
	}
	return c.UserQuery[:200] + "..."
}
