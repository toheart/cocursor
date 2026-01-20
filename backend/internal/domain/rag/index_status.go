package rag

// IndexStatus 文件索引状态
// 用于追踪文件的索引状态，支持增量索引
type IndexStatus struct {
	FilePath      string // 源文件路径（主键）
	SessionID     string // 会话 ID
	ProjectID     string // 项目 ID
	ContentHash   string // 文件内容哈希
	ChunkCount    int    // 生成的 chunk 数量
	FileMtime     int64  // 文件修改时间
	LastIndexedAt int64  // 最后索引时间
	Status        string // indexed/indexing/failed
}

// 索引状态常量
const (
	IndexStatusIndexed  = "indexed"
	IndexStatusIndexing = "indexing"
	IndexStatusFailed   = "failed"
)

// NeedsReindex 判断是否需要重新索引
func (s *IndexStatus) NeedsReindex(newMtime int64, newHash string) bool {
	// mtime 不同且 hash 也不同，需要重新索引
	return s.FileMtime != newMtime && s.ContentHash != newHash
}

// NeedsMtimeUpdate 判断是否只需要更新 mtime
func (s *IndexStatus) NeedsMtimeUpdate(newMtime int64, newHash string) bool {
	// mtime 不同但 hash 相同，只更新 mtime
	return s.FileMtime != newMtime && s.ContentHash == newHash
}
