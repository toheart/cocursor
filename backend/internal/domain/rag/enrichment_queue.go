package rag

import "time"

// EnrichmentTask LLM 增强任务
type EnrichmentTask struct {
	ChunkID     string // 关联的 chunk ID（主键）
	Priority    int    // 优先级，数值越大越优先
	Status      string // pending/processing/completed/failed
	RetryCount  int    // 已重试次数
	MaxRetries  int    // 最大重试次数
	CreatedAt   int64  // 创建时间
	NextRetryAt int64  // 下次重试时间（Unix 时间戳）
	LastError   string // 最后一次错误信息
}

// 任务状态常量
const (
	TaskStatusPending    = "pending"
	TaskStatusProcessing = "processing"
	TaskStatusCompleted  = "completed"
	TaskStatusFailed     = "failed"
)

// 默认配置
const (
	DefaultMaxRetries = 3
	DefaultPriority   = 0
)

// 重试延迟配置（指数退避）
var retryDelays = []time.Duration{
	1 * time.Minute,  // 第 1 次重试
	5 * time.Minute,  // 第 2 次重试
	15 * time.Minute, // 第 3 次重试
}

// NewEnrichmentTask 创建新的增强任务
func NewEnrichmentTask(chunkID string) *EnrichmentTask {
	return &EnrichmentTask{
		ChunkID:    chunkID,
		Priority:   DefaultPriority,
		Status:     TaskStatusPending,
		RetryCount: 0,
		MaxRetries: DefaultMaxRetries,
		CreatedAt:  time.Now().Unix(),
	}
}

// CanRetry 检查是否可以重试
func (t *EnrichmentTask) CanRetry() bool {
	return t.RetryCount < t.MaxRetries
}

// ShouldProcess 检查是否应该处理
func (t *EnrichmentTask) ShouldProcess() bool {
	if t.Status != TaskStatusPending {
		return false
	}
	if t.NextRetryAt == 0 {
		return true
	}
	return time.Now().Unix() >= t.NextRetryAt
}

// MarkProcessing 标记为处理中
func (t *EnrichmentTask) MarkProcessing() {
	t.Status = TaskStatusProcessing
}

// MarkCompleted 标记为完成
func (t *EnrichmentTask) MarkCompleted() {
	t.Status = TaskStatusCompleted
}

// MarkFailed 标记为失败
func (t *EnrichmentTask) MarkFailed(err string) {
	t.RetryCount++
	t.LastError = err

	if t.CanRetry() {
		// 可以重试，设置下次重试时间
		t.Status = TaskStatusPending
		delayIndex := t.RetryCount - 1
		if delayIndex >= len(retryDelays) {
			delayIndex = len(retryDelays) - 1
		}
		t.NextRetryAt = time.Now().Add(retryDelays[delayIndex]).Unix()
	} else {
		// 超过最大重试次数，标记为失败
		t.Status = TaskStatusFailed
	}
}

// Reset 重置任务状态（用于手动重试）
func (t *EnrichmentTask) Reset() {
	t.Status = TaskStatusPending
	t.RetryCount = 0
	t.NextRetryAt = 0
	t.LastError = ""
}

// EnrichmentStats 增强队列统计
type EnrichmentStats struct {
	PendingCount    int `json:"pending_count"`
	ProcessingCount int `json:"processing_count"`
	CompletedCount  int `json:"completed_count"`
	FailedCount     int `json:"failed_count"`
}
