package storage

import (
	"database/sql"
	"time"

	domainRAG "github.com/cocursor/backend/internal/domain/rag"
)

// 确保 EnrichmentQueueRepositoryImpl 实现了 domainRAG.EnrichmentQueueRepository 接口
var _ domainRAG.EnrichmentQueueRepository = (*EnrichmentQueueRepositoryImpl)(nil)

// EnrichmentQueueRepositoryImpl 增强队列仓库实现
type EnrichmentQueueRepositoryImpl struct {
	db *sql.DB
}

// NewEnrichmentQueueRepository 创建增强队列仓库实例
func NewEnrichmentQueueRepository(db *sql.DB) domainRAG.EnrichmentQueueRepository {
	return &EnrichmentQueueRepositoryImpl{db: db}
}

// EnqueueTask 添加任务到队列
func (r *EnrichmentQueueRepositoryImpl) EnqueueTask(task *domainRAG.EnrichmentTask) error {
	query := `
		INSERT OR REPLACE INTO rag_enrichment_queue (
			chunk_id, priority, status, retry_count, max_retries,
			created_at, next_retry_at, last_error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(
		query,
		task.ChunkID,
		task.Priority,
		task.Status,
		task.RetryCount,
		task.MaxRetries,
		task.CreatedAt,
		task.NextRetryAt,
		task.LastError,
	)

	return err
}

// EnqueueTasks 批量添加任务
func (r *EnrichmentQueueRepositoryImpl) EnqueueTasks(tasks []*domainRAG.EnrichmentTask) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO rag_enrichment_queue (
			chunk_id, priority, status, retry_count, max_retries,
			created_at, next_retry_at, last_error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, task := range tasks {
		_, err := stmt.Exec(
			task.ChunkID,
			task.Priority,
			task.Status,
			task.RetryCount,
			task.MaxRetries,
			task.CreatedAt,
			task.NextRetryAt,
			task.LastError,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DequeueTasks 获取待处理的任务
func (r *EnrichmentQueueRepositoryImpl) DequeueTasks(limit int) ([]*domainRAG.EnrichmentTask, error) {
	now := time.Now().Unix()

	// 查询待处理的任务：状态为 pending 且 next_retry_at 为空或已到期
	query := `
		SELECT chunk_id, priority, status, retry_count, max_retries,
		       created_at, next_retry_at, last_error
		FROM rag_enrichment_queue
		WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= ?)
		ORDER BY priority DESC, created_at ASC
		LIMIT ?`

	rows, err := r.db.Query(query, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domainRAG.EnrichmentTask
	for rows.Next() {
		var task domainRAG.EnrichmentTask
		var nextRetryAt sql.NullInt64
		var lastError sql.NullString

		err := rows.Scan(
			&task.ChunkID,
			&task.Priority,
			&task.Status,
			&task.RetryCount,
			&task.MaxRetries,
			&task.CreatedAt,
			&nextRetryAt,
			&lastError,
		)
		if err != nil {
			return nil, err
		}

		if nextRetryAt.Valid {
			task.NextRetryAt = nextRetryAt.Int64
		}
		if lastError.Valid {
			task.LastError = lastError.String
		}

		results = append(results, &task)
	}

	return results, rows.Err()
}

// GetTask 获取任务
func (r *EnrichmentQueueRepositoryImpl) GetTask(chunkID string) (*domainRAG.EnrichmentTask, error) {
	query := `
		SELECT chunk_id, priority, status, retry_count, max_retries,
		       created_at, next_retry_at, last_error
		FROM rag_enrichment_queue
		WHERE chunk_id = ?`

	var task domainRAG.EnrichmentTask
	var nextRetryAt sql.NullInt64
	var lastError sql.NullString

	err := r.db.QueryRow(query, chunkID).Scan(
		&task.ChunkID,
		&task.Priority,
		&task.Status,
		&task.RetryCount,
		&task.MaxRetries,
		&task.CreatedAt,
		&nextRetryAt,
		&lastError,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if nextRetryAt.Valid {
		task.NextRetryAt = nextRetryAt.Int64
	}
	if lastError.Valid {
		task.LastError = lastError.String
	}

	return &task, nil
}

// UpdateTask 更新任务状态
func (r *EnrichmentQueueRepositoryImpl) UpdateTask(task *domainRAG.EnrichmentTask) error {
	query := `
		UPDATE rag_enrichment_queue
		SET priority = ?, status = ?, retry_count = ?, max_retries = ?,
		    next_retry_at = ?, last_error = ?
		WHERE chunk_id = ?`

	_, err := r.db.Exec(
		query,
		task.Priority,
		task.Status,
		task.RetryCount,
		task.MaxRetries,
		task.NextRetryAt,
		task.LastError,
		task.ChunkID,
	)

	return err
}

// DeleteTask 删除任务
func (r *EnrichmentQueueRepositoryImpl) DeleteTask(chunkID string) error {
	query := `DELETE FROM rag_enrichment_queue WHERE chunk_id = ?`
	_, err := r.db.Exec(query, chunkID)
	return err
}

// ResetFailedTasks 重置失败的任务
func (r *EnrichmentQueueRepositoryImpl) ResetFailedTasks() (int, error) {
	query := `
		UPDATE rag_enrichment_queue
		SET status = 'pending', retry_count = 0, next_retry_at = NULL, last_error = NULL
		WHERE status = 'failed'`

	result, err := r.db.Exec(query)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(affected), nil
}

// GetQueueStats 获取队列统计
func (r *EnrichmentQueueRepositoryImpl) GetQueueStats() (*domainRAG.EnrichmentStats, error) {
	query := `
		SELECT 
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending_count,
			SUM(CASE WHEN status = 'processing' THEN 1 ELSE 0 END) as processing_count,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed_count,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count
		FROM rag_enrichment_queue`

	var stats domainRAG.EnrichmentStats
	var pending, processing, completed, failed sql.NullInt64

	err := r.db.QueryRow(query).Scan(&pending, &processing, &completed, &failed)
	if err != nil {
		return nil, err
	}

	if pending.Valid {
		stats.PendingCount = int(pending.Int64)
	}
	if processing.Valid {
		stats.ProcessingCount = int(processing.Int64)
	}
	if completed.Valid {
		stats.CompletedCount = int(completed.Int64)
	}
	if failed.Valid {
		stats.FailedCount = int(failed.Int64)
	}

	return &stats, nil
}

// ClearQueue 清空队列
func (r *EnrichmentQueueRepositoryImpl) ClearQueue() error {
	_, err := r.db.Exec("DELETE FROM rag_enrichment_queue")
	return err
}
