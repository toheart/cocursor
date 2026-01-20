package storage

import (
	"database/sql"

	domainRAG "github.com/cocursor/backend/internal/domain/rag"
)

// 确保 IndexStatusRepositoryImpl 实现了 domainRAG.IndexStatusRepository 接口
var _ domainRAG.IndexStatusRepository = (*IndexStatusRepositoryImpl)(nil)

// IndexStatusRepositoryImpl 索引状态仓库实现
type IndexStatusRepositoryImpl struct {
	db *sql.DB
}

// NewIndexStatusRepository 创建索引状态仓库实例
func NewIndexStatusRepository(db *sql.DB) domainRAG.IndexStatusRepository {
	return &IndexStatusRepositoryImpl{db: db}
}

// SaveIndexStatus 保存索引状态
func (r *IndexStatusRepositoryImpl) SaveIndexStatus(status *domainRAG.IndexStatus) error {
	query := `
		INSERT OR REPLACE INTO rag_index_status (
			file_path, session_id, project_id, content_hash,
			chunk_count, file_mtime, last_indexed_at, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(
		query,
		status.FilePath,
		status.SessionID,
		status.ProjectID,
		status.ContentHash,
		status.ChunkCount,
		status.FileMtime,
		status.LastIndexedAt,
		status.Status,
	)

	return err
}

// GetIndexStatus 获取索引状态
func (r *IndexStatusRepositoryImpl) GetIndexStatus(filePath string) (*domainRAG.IndexStatus, error) {
	query := `
		SELECT file_path, session_id, project_id, content_hash,
		       chunk_count, file_mtime, last_indexed_at, status
		FROM rag_index_status
		WHERE file_path = ?`

	var status domainRAG.IndexStatus
	err := r.db.QueryRow(query, filePath).Scan(
		&status.FilePath,
		&status.SessionID,
		&status.ProjectID,
		&status.ContentHash,
		&status.ChunkCount,
		&status.FileMtime,
		&status.LastIndexedAt,
		&status.Status,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &status, nil
}

// UpdateFileMtime 更新文件修改时间
func (r *IndexStatusRepositoryImpl) UpdateFileMtime(filePath string, mtime int64) error {
	query := `UPDATE rag_index_status SET file_mtime = ? WHERE file_path = ?`
	_, err := r.db.Exec(query, mtime, filePath)
	return err
}

// DeleteIndexStatus 删除索引状态
func (r *IndexStatusRepositoryImpl) DeleteIndexStatus(filePath string) error {
	query := `DELETE FROM rag_index_status WHERE file_path = ?`
	_, err := r.db.Exec(query, filePath)
	return err
}

// GetAllIndexStatus 获取所有索引状态
func (r *IndexStatusRepositoryImpl) GetAllIndexStatus() ([]*domainRAG.IndexStatus, error) {
	query := `
		SELECT file_path, session_id, project_id, content_hash,
		       chunk_count, file_mtime, last_indexed_at, status
		FROM rag_index_status
		ORDER BY last_indexed_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domainRAG.IndexStatus
	for rows.Next() {
		var status domainRAG.IndexStatus
		err := rows.Scan(
			&status.FilePath,
			&status.SessionID,
			&status.ProjectID,
			&status.ContentHash,
			&status.ChunkCount,
			&status.FileMtime,
			&status.LastIndexedAt,
			&status.Status,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, &status)
	}

	return results, rows.Err()
}

// ClearAllStatus 清空所有索引状态
func (r *IndexStatusRepositoryImpl) ClearAllStatus() error {
	_, err := r.db.Exec("DELETE FROM rag_index_status")
	return err
}
