package rag

// ChunkRepository 知识片段仓库接口
type ChunkRepository interface {
	// 保存知识片段
	SaveChunk(chunk *KnowledgeChunk) error

	// 批量保存知识片段
	SaveChunks(chunks []*KnowledgeChunk) error

	// 获取知识片段
	GetChunk(id string) (*KnowledgeChunk, error)

	// 按会话获取所有知识片段
	GetChunksBySession(sessionID string) ([]*KnowledgeChunk, error)

	// 按项目获取知识片段
	GetChunksByProject(projectID string, limit, offset int) ([]*KnowledgeChunk, error)

	// 删除会话的所有知识片段
	DeleteChunksBySession(sessionID string) error

	// 更新增强内容
	UpdateChunkEnrichment(id string, enrichment *ChunkEnrichment) error

	// 更新增强状态
	UpdateChunkEnrichmentStatus(id string, status string, errMsg string) error

	// 获取待增强的知识片段
	GetPendingEnrichmentChunks(limit int) ([]*KnowledgeChunk, error)

	// 清空所有知识片段
	ClearAllChunks() error
}

// IndexStatusRepository 索引状态仓库接口
type IndexStatusRepository interface {
	// 保存索引状态
	SaveIndexStatus(status *IndexStatus) error

	// 获取索引状态
	GetIndexStatus(filePath string) (*IndexStatus, error)

	// 更新文件修改时间
	UpdateFileMtime(filePath string, mtime int64) error

	// 删除索引状态
	DeleteIndexStatus(filePath string) error

	// 获取所有索引状态
	GetAllIndexStatus() ([]*IndexStatus, error)

	// 清空所有索引状态
	ClearAllStatus() error
}

// EnrichmentQueueRepository 增强队列仓库接口
type EnrichmentQueueRepository interface {
	// 添加任务到队列
	EnqueueTask(task *EnrichmentTask) error

	// 批量添加任务
	EnqueueTasks(tasks []*EnrichmentTask) error

	// 获取待处理的任务
	DequeueTasks(limit int) ([]*EnrichmentTask, error)

	// 获取任务
	GetTask(chunkID string) (*EnrichmentTask, error)

	// 更新任务状态
	UpdateTask(task *EnrichmentTask) error

	// 删除任务
	DeleteTask(chunkID string) error

	// 重置失败的任务
	ResetFailedTasks() (int, error)

	// 获取队列统计
	GetQueueStats() (*EnrichmentStats, error)

	// 清空队列
	ClearQueue() error
}
