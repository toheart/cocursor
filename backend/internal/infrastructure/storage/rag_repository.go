package storage

import (
	"database/sql"
	"encoding/json"

	domainRAG "github.com/cocursor/backend/internal/domain/rag"
)

// 确保 RAGRepository 实现了 domainRAG.RAGRepository 接口
var _ domainRAG.RAGRepository = (*RAGRepository)(nil)

// RAGRepository RAG 元数据仓库实现
type RAGRepository struct {
	db *sql.DB
}

// NewRAGRepository 创建 RAG 仓库实例
func NewRAGRepository(db *sql.DB) domainRAG.RAGRepository {
	return &RAGRepository{
		db: db,
	}
}

// SaveMessageMetadata 保存消息元数据
func (r *RAGRepository) SaveMessageMetadata(metadata *domainRAG.MessageMetadata) error {
	query := `
		INSERT OR REPLACE INTO rag_message_metadata (
			session_id, message_id, workspace_id, project_id, project_name,
			message_type, message_index, turn_index, vector_id,
			content_hash, file_path, file_mtime, indexed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(
		query,
		metadata.SessionID,
		metadata.MessageID,
		metadata.WorkspaceID,
		metadata.ProjectID,
		metadata.ProjectName,
		metadata.MessageType,
		metadata.MessageIndex,
		metadata.TurnIndex,
		metadata.VectorID,
		metadata.ContentHash,
		metadata.FilePath,
		metadata.FileMtime,
		metadata.IndexedAt,
	)

	return err
}

// GetMessageMetadata 获取消息元数据
func (r *RAGRepository) GetMessageMetadata(sessionID, messageID string) (*domainRAG.MessageMetadata, error) {
	query := `
		SELECT session_id, message_id, workspace_id, project_id, project_name,
		       message_type, message_index, turn_index, vector_id,
		       content_hash, file_path, file_mtime, indexed_at
		FROM rag_message_metadata
		WHERE session_id = ? AND message_id = ?`

	var metadata domainRAG.MessageMetadata
	err := r.db.QueryRow(query, sessionID, messageID).Scan(
		&metadata.SessionID,
		&metadata.MessageID,
		&metadata.WorkspaceID,
		&metadata.ProjectID,
		&metadata.ProjectName,
		&metadata.MessageType,
		&metadata.MessageIndex,
		&metadata.TurnIndex,
		&metadata.VectorID,
		&metadata.ContentHash,
		&metadata.FilePath,
		&metadata.FileMtime,
		&metadata.IndexedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}

// GetMessageMetadataBySession 获取会话的所有消息元数据
func (r *RAGRepository) GetMessageMetadataBySession(sessionID string) ([]*domainRAG.MessageMetadata, error) {
	query := `
		SELECT session_id, message_id, workspace_id, project_id, project_name,
		       message_type, message_index, turn_index, vector_id,
		       content_hash, file_path, file_mtime, indexed_at
		FROM rag_message_metadata
		WHERE session_id = ?
		ORDER BY message_index`

	rows, err := r.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domainRAG.MessageMetadata
	for rows.Next() {
		var metadata domainRAG.MessageMetadata
		err := rows.Scan(
			&metadata.SessionID,
			&metadata.MessageID,
			&metadata.WorkspaceID,
			&metadata.ProjectID,
			&metadata.ProjectName,
			&metadata.MessageType,
			&metadata.MessageIndex,
			&metadata.TurnIndex,
			&metadata.VectorID,
			&metadata.ContentHash,
			&metadata.FilePath,
			&metadata.FileMtime,
			&metadata.IndexedAt,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, &metadata)
	}

	return results, rows.Err()
}

// GetMessageMetadataByProject 获取项目的所有消息元数据
func (r *RAGRepository) GetMessageMetadataByProject(projectID string) ([]*domainRAG.MessageMetadata, error) {
	query := `
		SELECT session_id, message_id, workspace_id, project_id, project_name,
		       message_type, message_index, turn_index, vector_id,
		       content_hash, file_path, file_mtime, indexed_at
		FROM rag_message_metadata
		WHERE project_id = ?
		ORDER BY session_id, message_index`

	rows, err := r.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domainRAG.MessageMetadata
	for rows.Next() {
		var metadata domainRAG.MessageMetadata
		err := rows.Scan(
			&metadata.SessionID,
			&metadata.MessageID,
			&metadata.WorkspaceID,
			&metadata.ProjectID,
			&metadata.ProjectName,
			&metadata.MessageType,
			&metadata.MessageIndex,
			&metadata.TurnIndex,
			&metadata.VectorID,
			&metadata.ContentHash,
			&metadata.FilePath,
			&metadata.FileMtime,
			&metadata.IndexedAt,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, &metadata)
	}

	return results, rows.Err()
}

// DeleteMessageMetadata 删除消息元数据
func (r *RAGRepository) DeleteMessageMetadata(sessionID, messageID string) error {
	query := `DELETE FROM rag_message_metadata WHERE session_id = ? AND message_id = ?`
	_, err := r.db.Exec(query, sessionID, messageID)
	return err
}

// SaveTurnMetadata 保存对话对元数据
func (r *RAGRepository) SaveTurnMetadata(metadata *domainRAG.TurnMetadata) error {
	userIDsJSON, _ := json.Marshal(metadata.UserMessageIDs)
	aiIDsJSON, _ := json.Marshal(metadata.AIMessageIDs)

	isIncomplete := 0
	if metadata.IsIncomplete {
		isIncomplete = 1
	}

	query := `
		INSERT OR REPLACE INTO rag_turn_metadata (
			session_id, turn_index, workspace_id, project_id, project_name,
			user_message_ids, ai_message_ids, message_count,
			vector_id, content_hash, file_path, file_mtime, indexed_at, is_incomplete
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(
		query,
		metadata.SessionID,
		metadata.TurnIndex,
		metadata.WorkspaceID,
		metadata.ProjectID,
		metadata.ProjectName,
		string(userIDsJSON),
		string(aiIDsJSON),
		metadata.MessageCount,
		metadata.VectorID,
		metadata.ContentHash,
		metadata.FilePath,
		metadata.FileMtime,
		metadata.IndexedAt,
		isIncomplete,
	)

	return err
}

// GetTurnMetadata 获取对话对元数据
func (r *RAGRepository) GetTurnMetadata(sessionID string, turnIndex int) (*domainRAG.TurnMetadata, error) {
	query := `
		SELECT session_id, turn_index, workspace_id, project_id, project_name,
		       user_message_ids, ai_message_ids, message_count,
		       vector_id, content_hash, file_path, file_mtime, indexed_at, is_incomplete
		FROM rag_turn_metadata
		WHERE session_id = ? AND turn_index = ?`

	var metadata domainRAG.TurnMetadata
	var userIDsJSON, aiIDsJSON string
	var isIncomplete int

	err := r.db.QueryRow(query, sessionID, turnIndex).Scan(
		&metadata.SessionID,
		&metadata.TurnIndex,
		&metadata.WorkspaceID,
		&metadata.ProjectID,
		&metadata.ProjectName,
		&userIDsJSON,
		&aiIDsJSON,
		&metadata.MessageCount,
		&metadata.VectorID,
		&metadata.ContentHash,
		&metadata.FilePath,
		&metadata.FileMtime,
		&metadata.IndexedAt,
		&isIncomplete,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(userIDsJSON), &metadata.UserMessageIDs)
	json.Unmarshal([]byte(aiIDsJSON), &metadata.AIMessageIDs)
	metadata.IsIncomplete = isIncomplete == 1

	return &metadata, nil
}

// GetTurnMetadataBySession 获取会话的所有对话对元数据
func (r *RAGRepository) GetTurnMetadataBySession(sessionID string) ([]*domainRAG.TurnMetadata, error) {
	query := `
		SELECT session_id, turn_index, workspace_id, project_id, project_name,
		       user_message_ids, ai_message_ids, message_count,
		       vector_id, content_hash, file_path, file_mtime, indexed_at, is_incomplete
		FROM rag_turn_metadata
		WHERE session_id = ?
		ORDER BY turn_index`

	rows, err := r.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domainRAG.TurnMetadata
	for rows.Next() {
		var metadata domainRAG.TurnMetadata
		var userIDsJSON, aiIDsJSON string
		var isIncomplete int

		err := rows.Scan(
			&metadata.SessionID,
			&metadata.TurnIndex,
			&metadata.WorkspaceID,
			&metadata.ProjectID,
			&metadata.ProjectName,
			&userIDsJSON,
			&aiIDsJSON,
			&metadata.MessageCount,
			&metadata.VectorID,
			&metadata.ContentHash,
			&metadata.FilePath,
			&metadata.FileMtime,
			&metadata.IndexedAt,
			&isIncomplete,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(userIDsJSON), &metadata.UserMessageIDs)
		json.Unmarshal([]byte(aiIDsJSON), &metadata.AIMessageIDs)
		metadata.IsIncomplete = isIncomplete == 1

		results = append(results, &metadata)
	}

	return results, rows.Err()
}

// GetTurnMetadataByProject 获取项目的所有对话对元数据
func (r *RAGRepository) GetTurnMetadataByProject(projectID string) ([]*domainRAG.TurnMetadata, error) {
	query := `
		SELECT session_id, turn_index, workspace_id, project_id, project_name,
		       user_message_ids, ai_message_ids, message_count,
		       vector_id, content_hash, file_path, file_mtime, indexed_at, is_incomplete
		FROM rag_turn_metadata
		WHERE project_id = ?
		ORDER BY session_id, turn_index`

	rows, err := r.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domainRAG.TurnMetadata
	for rows.Next() {
		var metadata domainRAG.TurnMetadata
		var userIDsJSON, aiIDsJSON string
		var isIncomplete int

		err := rows.Scan(
			&metadata.SessionID,
			&metadata.TurnIndex,
			&metadata.WorkspaceID,
			&metadata.ProjectID,
			&metadata.ProjectName,
			&userIDsJSON,
			&aiIDsJSON,
			&metadata.MessageCount,
			&metadata.VectorID,
			&metadata.ContentHash,
			&metadata.FilePath,
			&metadata.FileMtime,
			&metadata.IndexedAt,
			&isIncomplete,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(userIDsJSON), &metadata.UserMessageIDs)
		json.Unmarshal([]byte(aiIDsJSON), &metadata.AIMessageIDs)
		metadata.IsIncomplete = isIncomplete == 1

		results = append(results, &metadata)
	}

	return results, rows.Err()
}

// GetIncompleteTurns 获取未完成的对话对
func (r *RAGRepository) GetIncompleteTurns(sessionID string) ([]*domainRAG.TurnMetadata, error) {
	query := `
		SELECT session_id, turn_index, workspace_id, project_id, project_name,
		       user_message_ids, ai_message_ids, message_count,
		       vector_id, content_hash, file_path, file_mtime, indexed_at, is_incomplete
		FROM rag_turn_metadata
		WHERE session_id = ? AND is_incomplete = 1
		ORDER BY turn_index`

	rows, err := r.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domainRAG.TurnMetadata
	for rows.Next() {
		var metadata domainRAG.TurnMetadata
		var userIDsJSON, aiIDsJSON string
		var isIncomplete int

		err := rows.Scan(
			&metadata.SessionID,
			&metadata.TurnIndex,
			&metadata.WorkspaceID,
			&metadata.ProjectID,
			&metadata.ProjectName,
			&userIDsJSON,
			&aiIDsJSON,
			&metadata.MessageCount,
			&metadata.VectorID,
			&metadata.ContentHash,
			&metadata.FilePath,
			&metadata.FileMtime,
			&metadata.IndexedAt,
			&isIncomplete,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(userIDsJSON), &metadata.UserMessageIDs)
		json.Unmarshal([]byte(aiIDsJSON), &metadata.AIMessageIDs)
		metadata.IsIncomplete = isIncomplete == 1

		results = append(results, &metadata)
	}

	return results, rows.Err()
}

// DeleteTurnMetadata 删除对话对元数据
func (r *RAGRepository) DeleteTurnMetadata(sessionID string, turnIndex int) error {
	query := `DELETE FROM rag_turn_metadata WHERE session_id = ? AND turn_index = ?`
	_, err := r.db.Exec(query, sessionID, turnIndex)
	return err
}

// GetFileMetadata 获取文件元数据（从消息元数据表中聚合）
func (r *RAGRepository) GetFileMetadata(filePath string) (*domainRAG.FileMetadata, error) {
	query := `
		SELECT file_path, MAX(file_mtime) as file_mtime, 
		       MAX(indexed_at) as last_indexed,
		       GROUP_CONCAT(DISTINCT content_hash) as content_hash
		FROM rag_message_metadata
		WHERE file_path = ?
		GROUP BY file_path`

	var metadata domainRAG.FileMetadata
	var contentHashes string

	err := r.db.QueryRow(query, filePath).Scan(
		&metadata.FilePath,
		&metadata.FileMtime,
		&metadata.LastIndexed,
		&contentHashes,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// 简化：使用第一个 hash（实际应该计算所有 hash 的组合）
	if contentHashes != "" {
		hashes := []rune(contentHashes)
		if len(hashes) > 0 {
			metadata.ContentHash = string(hashes[0])
		}
	}

	return &metadata, nil
}

// UpdateFileMtime 更新文件修改时间（批量更新该文件的所有元数据）
func (r *RAGRepository) UpdateFileMtime(filePath string, mtime int64) error {
	query := `
		UPDATE rag_message_metadata
		SET file_mtime = ?
		WHERE file_path = ?`

	_, err := r.db.Exec(query, mtime, filePath)
	if err != nil {
		return err
	}

	query = `
		UPDATE rag_turn_metadata
		SET file_mtime = ?
		WHERE file_path = ?`

	_, err = r.db.Exec(query, mtime, filePath)
	return err
}

// ClearAllMetadata 清空所有元数据
func (r *RAGRepository) ClearAllMetadata() error {
	// 清空消息元数据
	if _, err := r.db.Exec("DELETE FROM rag_message_metadata"); err != nil {
		return err
	}

	// 清空对话对元数据
	if _, err := r.db.Exec("DELETE FROM rag_turn_metadata"); err != nil {
		return err
	}

	return nil
}
