package storage

import (
	"database/sql"
	"encoding/json"

	domainRAG "github.com/cocursor/backend/internal/domain/rag"
)

// 确保 ChunkRepositoryImpl 实现了 domainRAG.ChunkRepository 接口
var _ domainRAG.ChunkRepository = (*ChunkRepositoryImpl)(nil)

// ChunkRepositoryImpl 知识片段仓库实现
type ChunkRepositoryImpl struct {
	db *sql.DB
}

// NewChunkRepository 创建知识片段仓库实例
func NewChunkRepository(db *sql.DB) domainRAG.ChunkRepository {
	return &ChunkRepositoryImpl{db: db}
}

// SaveChunk 保存知识片段
func (r *ChunkRepositoryImpl) SaveChunk(chunk *domainRAG.KnowledgeChunk) error {
	toolsUsedJSON, _ := json.Marshal(chunk.ToolsUsed)
	filesModifiedJSON, _ := json.Marshal(chunk.FilesModified)
	codeLanguagesJSON, _ := json.Marshal(chunk.CodeLanguages)
	tagsJSON, _ := json.Marshal(chunk.Tags)

	hasCode := 0
	if chunk.HasCode {
		hasCode = 1
	}

	query := `
		INSERT OR REPLACE INTO rag_knowledge_chunks (
			id, session_id, chunk_index, project_id, project_name, workspace_id,
			user_query, ai_response_core, vector_text,
			tools_used, files_modified, code_languages, has_code,
			summary, main_topic, tags, enrichment_status, enrichment_error,
			timestamp, content_hash, file_path, indexed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(
		query,
		chunk.ID,
		chunk.SessionID,
		chunk.ChunkIndex,
		chunk.ProjectID,
		chunk.ProjectName,
		chunk.WorkspaceID,
		chunk.UserQuery,
		chunk.AIResponseCore,
		chunk.VectorText,
		string(toolsUsedJSON),
		string(filesModifiedJSON),
		string(codeLanguagesJSON),
		hasCode,
		chunk.Summary,
		chunk.MainTopic,
		string(tagsJSON),
		chunk.EnrichmentStatus,
		chunk.EnrichmentError,
		chunk.Timestamp,
		chunk.ContentHash,
		chunk.FilePath,
		chunk.IndexedAt,
	)

	return err
}

// SaveChunks 批量保存知识片段
func (r *ChunkRepositoryImpl) SaveChunks(chunks []*domainRAG.KnowledgeChunk) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO rag_knowledge_chunks (
			id, session_id, chunk_index, project_id, project_name, workspace_id,
			user_query, ai_response_core, vector_text,
			tools_used, files_modified, code_languages, has_code,
			summary, main_topic, tags, enrichment_status, enrichment_error,
			timestamp, content_hash, file_path, indexed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, chunk := range chunks {
		toolsUsedJSON, _ := json.Marshal(chunk.ToolsUsed)
		filesModifiedJSON, _ := json.Marshal(chunk.FilesModified)
		codeLanguagesJSON, _ := json.Marshal(chunk.CodeLanguages)
		tagsJSON, _ := json.Marshal(chunk.Tags)

		hasCode := 0
		if chunk.HasCode {
			hasCode = 1
		}

		_, err := stmt.Exec(
			chunk.ID,
			chunk.SessionID,
			chunk.ChunkIndex,
			chunk.ProjectID,
			chunk.ProjectName,
			chunk.WorkspaceID,
			chunk.UserQuery,
			chunk.AIResponseCore,
			chunk.VectorText,
			string(toolsUsedJSON),
			string(filesModifiedJSON),
			string(codeLanguagesJSON),
			hasCode,
			chunk.Summary,
			chunk.MainTopic,
			string(tagsJSON),
			chunk.EnrichmentStatus,
			chunk.EnrichmentError,
			chunk.Timestamp,
			chunk.ContentHash,
			chunk.FilePath,
			chunk.IndexedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetChunk 获取知识片段
func (r *ChunkRepositoryImpl) GetChunk(id string) (*domainRAG.KnowledgeChunk, error) {
	query := `
		SELECT id, session_id, chunk_index, project_id, project_name, workspace_id,
		       user_query, ai_response_core, vector_text,
		       tools_used, files_modified, code_languages, has_code,
		       summary, main_topic, tags, enrichment_status, enrichment_error,
		       timestamp, content_hash, file_path, indexed_at
		FROM rag_knowledge_chunks
		WHERE id = ?`

	return r.scanChunk(r.db.QueryRow(query, id))
}

// GetChunksBySession 按会话获取所有知识片段
func (r *ChunkRepositoryImpl) GetChunksBySession(sessionID string) ([]*domainRAG.KnowledgeChunk, error) {
	query := `
		SELECT id, session_id, chunk_index, project_id, project_name, workspace_id,
		       user_query, ai_response_core, vector_text,
		       tools_used, files_modified, code_languages, has_code,
		       summary, main_topic, tags, enrichment_status, enrichment_error,
		       timestamp, content_hash, file_path, indexed_at
		FROM rag_knowledge_chunks
		WHERE session_id = ?
		ORDER BY chunk_index`

	rows, err := r.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanChunks(rows)
}

// GetChunksByProject 按项目获取知识片段
func (r *ChunkRepositoryImpl) GetChunksByProject(projectID string, limit, offset int) ([]*domainRAG.KnowledgeChunk, error) {
	query := `
		SELECT id, session_id, chunk_index, project_id, project_name, workspace_id,
		       user_query, ai_response_core, vector_text,
		       tools_used, files_modified, code_languages, has_code,
		       summary, main_topic, tags, enrichment_status, enrichment_error,
		       timestamp, content_hash, file_path, indexed_at
		FROM rag_knowledge_chunks
		WHERE project_id = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanChunks(rows)
}

// DeleteChunksBySession 删除会话的所有知识片段
func (r *ChunkRepositoryImpl) DeleteChunksBySession(sessionID string) error {
	query := `DELETE FROM rag_knowledge_chunks WHERE session_id = ?`
	_, err := r.db.Exec(query, sessionID)
	return err
}

// UpdateChunkEnrichment 更新增强内容
func (r *ChunkRepositoryImpl) UpdateChunkEnrichment(id string, enrichment *domainRAG.ChunkEnrichment) error {
	tagsJSON, _ := json.Marshal(enrichment.Tags)

	query := `
		UPDATE rag_knowledge_chunks
		SET summary = ?, main_topic = ?, tags = ?, 
		    enrichment_status = 'completed', enrichment_error = NULL
		WHERE id = ?`

	_, err := r.db.Exec(query, enrichment.Summary, enrichment.MainTopic, string(tagsJSON), id)
	return err
}

// UpdateChunkEnrichmentStatus 更新增强状态
func (r *ChunkRepositoryImpl) UpdateChunkEnrichmentStatus(id string, status string, errMsg string) error {
	query := `
		UPDATE rag_knowledge_chunks
		SET enrichment_status = ?, enrichment_error = ?
		WHERE id = ?`

	_, err := r.db.Exec(query, status, errMsg, id)
	return err
}

// GetPendingEnrichmentChunks 获取待增强的知识片段
func (r *ChunkRepositoryImpl) GetPendingEnrichmentChunks(limit int) ([]*domainRAG.KnowledgeChunk, error) {
	query := `
		SELECT id, session_id, chunk_index, project_id, project_name, workspace_id,
		       user_query, ai_response_core, vector_text,
		       tools_used, files_modified, code_languages, has_code,
		       summary, main_topic, tags, enrichment_status, enrichment_error,
		       timestamp, content_hash, file_path, indexed_at
		FROM rag_knowledge_chunks
		WHERE enrichment_status = 'pending'
		ORDER BY timestamp DESC
		LIMIT ?`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanChunks(rows)
}

// ClearAllChunks 清空所有知识片段
func (r *ChunkRepositoryImpl) ClearAllChunks() error {
	_, err := r.db.Exec("DELETE FROM rag_knowledge_chunks")
	return err
}

// scanChunk 扫描单行数据到 KnowledgeChunk
func (r *ChunkRepositoryImpl) scanChunk(row *sql.Row) (*domainRAG.KnowledgeChunk, error) {
	var chunk domainRAG.KnowledgeChunk
	var toolsUsedJSON, filesModifiedJSON, codeLanguagesJSON, tagsJSON sql.NullString
	var summary, mainTopic, enrichmentError sql.NullString
	var hasCode int

	err := row.Scan(
		&chunk.ID,
		&chunk.SessionID,
		&chunk.ChunkIndex,
		&chunk.ProjectID,
		&chunk.ProjectName,
		&chunk.WorkspaceID,
		&chunk.UserQuery,
		&chunk.AIResponseCore,
		&chunk.VectorText,
		&toolsUsedJSON,
		&filesModifiedJSON,
		&codeLanguagesJSON,
		&hasCode,
		&summary,
		&mainTopic,
		&tagsJSON,
		&chunk.EnrichmentStatus,
		&enrichmentError,
		&chunk.Timestamp,
		&chunk.ContentHash,
		&chunk.FilePath,
		&chunk.IndexedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	chunk.HasCode = hasCode == 1
	if toolsUsedJSON.Valid {
		json.Unmarshal([]byte(toolsUsedJSON.String), &chunk.ToolsUsed)
	}
	if filesModifiedJSON.Valid {
		json.Unmarshal([]byte(filesModifiedJSON.String), &chunk.FilesModified)
	}
	if codeLanguagesJSON.Valid {
		json.Unmarshal([]byte(codeLanguagesJSON.String), &chunk.CodeLanguages)
	}
	if tagsJSON.Valid {
		json.Unmarshal([]byte(tagsJSON.String), &chunk.Tags)
	}
	if summary.Valid {
		chunk.Summary = summary.String
	}
	if mainTopic.Valid {
		chunk.MainTopic = mainTopic.String
	}
	if enrichmentError.Valid {
		chunk.EnrichmentError = enrichmentError.String
	}

	return &chunk, nil
}

// scanChunks 扫描多行数据到 KnowledgeChunk 切片
func (r *ChunkRepositoryImpl) scanChunks(rows *sql.Rows) ([]*domainRAG.KnowledgeChunk, error) {
	var results []*domainRAG.KnowledgeChunk

	for rows.Next() {
		var chunk domainRAG.KnowledgeChunk
		var toolsUsedJSON, filesModifiedJSON, codeLanguagesJSON, tagsJSON sql.NullString
		var summary, mainTopic, enrichmentError sql.NullString
		var hasCode int

		err := rows.Scan(
			&chunk.ID,
			&chunk.SessionID,
			&chunk.ChunkIndex,
			&chunk.ProjectID,
			&chunk.ProjectName,
			&chunk.WorkspaceID,
			&chunk.UserQuery,
			&chunk.AIResponseCore,
			&chunk.VectorText,
			&toolsUsedJSON,
			&filesModifiedJSON,
			&codeLanguagesJSON,
			&hasCode,
			&summary,
			&mainTopic,
			&tagsJSON,
			&chunk.EnrichmentStatus,
			&enrichmentError,
			&chunk.Timestamp,
			&chunk.ContentHash,
			&chunk.FilePath,
			&chunk.IndexedAt,
		)
		if err != nil {
			return nil, err
		}

		chunk.HasCode = hasCode == 1
		if toolsUsedJSON.Valid {
			json.Unmarshal([]byte(toolsUsedJSON.String), &chunk.ToolsUsed)
		}
		if filesModifiedJSON.Valid {
			json.Unmarshal([]byte(filesModifiedJSON.String), &chunk.FilesModified)
		}
		if codeLanguagesJSON.Valid {
			json.Unmarshal([]byte(codeLanguagesJSON.String), &chunk.CodeLanguages)
		}
		if tagsJSON.Valid {
			json.Unmarshal([]byte(tagsJSON.String), &chunk.Tags)
		}
		if summary.Valid {
			chunk.Summary = summary.String
		}
		if mainTopic.Valid {
			chunk.MainTopic = mainTopic.String
		}
		if enrichmentError.Valid {
			chunk.EnrichmentError = enrichmentError.String
		}

		results = append(results, &chunk)
	}

	return results, rows.Err()
}
