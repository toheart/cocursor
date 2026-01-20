package rag

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"log/slog"

	"github.com/google/uuid"

	domainCursor "github.com/cocursor/backend/internal/application/cursor"
	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	"github.com/cocursor/backend/internal/infrastructure/embedding"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/cocursor/backend/internal/infrastructure/vector"
	"github.com/qdrant/go-client/qdrant"
)

// ChunkService 知识片段服务
// 使用新的 KnowledgeChunk 模型进行索引
type ChunkService struct {
	sessionService    *domainCursor.SessionService
	embeddingClient   *embedding.Client
	qdrantManager     *vector.QdrantManager
	chunkRepo         domainRAG.ChunkRepository
	indexStatusRepo   domainRAG.IndexStatusRepository
	enrichmentQueue   domainRAG.EnrichmentQueueRepository
	projectManager    *domainCursor.ProjectManager
	contentExtractor  *ContentExtractor
	logger            *slog.Logger
}

// NewChunkService 创建知识片段服务
func NewChunkService(
	sessionService *domainCursor.SessionService,
	embeddingClient *embedding.Client,
	qdrantManager *vector.QdrantManager,
	chunkRepo domainRAG.ChunkRepository,
	indexStatusRepo domainRAG.IndexStatusRepository,
	enrichmentQueue domainRAG.EnrichmentQueueRepository,
	projectManager *domainCursor.ProjectManager,
) *ChunkService {
	return &ChunkService{
		sessionService:    sessionService,
		embeddingClient:   embeddingClient,
		qdrantManager:     qdrantManager,
		chunkRepo:         chunkRepo,
		indexStatusRepo:   indexStatusRepo,
		enrichmentQueue:   enrichmentQueue,
		projectManager:    projectManager,
		contentExtractor:  NewContentExtractor(),
		logger:            log.NewModuleLogger("rag", "chunk_service"),
	}
}

// IndexSession 使用新模型索引会话
func (s *ChunkService) IndexSession(sessionID, filePath string) error {
	s.logger.Info("Indexing session with new chunk model",
		"session_id", sessionID,
		"file_path", filePath,
	)

	// 1. 获取项目信息
	projectInfo, err := s.getProjectInfo(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get project info: %w", err)
	}

	// 2. 读取会话消息
	messages, err := s.sessionService.GetSessionTextContent(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session text content: %w", err)
	}

	if len(messages) == 0 {
		s.logger.Debug("No messages in session, skipping", "session_id", sessionID)
		return nil
	}

	// 3. 配对消息为对话对
	turns := PairMessages(messages, sessionID)
	if len(turns) == 0 {
		s.logger.Debug("No turns after pairing, skipping", "session_id", sessionID)
		return nil
	}

	// 4. 计算文件信息
	fileInfo, _ := os.Stat(filePath)
	fileMtime := int64(0)
	if fileInfo != nil {
		fileMtime = fileInfo.ModTime().Unix()
	}
	contentHash := s.calculateFileHash(filePath)
	indexedAt := time.Now().Unix()

	// 5. 提取内容并创建 KnowledgeChunks
	chunks := make([]*domainRAG.KnowledgeChunk, 0, len(turns))
	vectorTexts := make([]string, 0, len(turns))

	for i, turn := range turns {
		// 跳过未完成的对话对
		if turn.IsIncomplete {
			s.logger.Debug("Skipping incomplete turn", "turn_index", turn.TurnIndex)
			continue
		}

		// 提取内容
		extraction := s.contentExtractor.ExtractFromTurn(turn)

		// 跳过空内容
		if extraction.UserQuery == "" && extraction.AIResponseCore == "" {
			s.logger.Debug("Skipping empty turn", "turn_index", turn.TurnIndex)
			continue
		}

		// 创建 KnowledgeChunk
		chunk := &domainRAG.KnowledgeChunk{
			ID:               uuid.New().String(),
			SessionID:        sessionID,
			ChunkIndex:       i,
			ProjectID:        projectInfo.ProjectID,
			ProjectName:      projectInfo.ProjectName,
			WorkspaceID:      projectInfo.WorkspaceID,
			UserQuery:        extraction.UserQuery,
			AIResponseCore:   extraction.AIResponseCore,
			VectorText:       extraction.VectorText,
			ToolsUsed:        extraction.ToolsUsed,
			FilesModified:    extraction.FilesModified,
			CodeLanguages:    extraction.CodeLanguages,
			HasCode:          extraction.HasCode,
			EnrichmentStatus: domainRAG.EnrichmentStatusPending,
			Timestamp:        turn.Timestamp,
			ContentHash:      contentHash,
			FilePath:         filePath,
			IndexedAt:        indexedAt,
		}

		chunks = append(chunks, chunk)
		vectorTexts = append(vectorTexts, extraction.VectorText)
	}

	if len(chunks) == 0 {
		s.logger.Info("No valid chunks to index", "session_id", sessionID)
		return nil
	}

	// 6. 批量向量化
	vectors, err := s.embeddingClient.EmbedTexts(vectorTexts)
	if err != nil {
		return fmt.Errorf("failed to embed texts: %w", err)
	}

	// 7. 构建 Qdrant 点并写入
	points := s.buildChunkPoints(chunks, vectors)

	client := s.qdrantManager.GetClient()
	if client == nil {
		return fmt.Errorf("qdrant client not initialized")
	}

	ctx := context.Background()
	_, err = client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: "cursor_knowledge",
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert chunks: %w", err)
	}

	// 8. 保存元数据到 SQLite
	if err := s.chunkRepo.SaveChunks(chunks); err != nil {
		return fmt.Errorf("failed to save chunk metadata: %w", err)
	}

	// 9. 更新索引状态
	indexStatus := &domainRAG.IndexStatus{
		FilePath:      filePath,
		SessionID:     sessionID,
		ProjectID:     projectInfo.ProjectID,
		ContentHash:   contentHash,
		ChunkCount:    len(chunks),
		FileMtime:     fileMtime,
		LastIndexedAt: indexedAt,
		Status:        domainRAG.IndexStatusIndexed,
	}
	if err := s.indexStatusRepo.SaveIndexStatus(indexStatus); err != nil {
		return fmt.Errorf("failed to save index status: %w", err)
	}

	// 10. 添加到增强队列
	tasks := make([]*domainRAG.EnrichmentTask, len(chunks))
	for i, chunk := range chunks {
		tasks[i] = domainRAG.NewEnrichmentTask(chunk.ID)
	}
	if err := s.enrichmentQueue.EnqueueTasks(tasks); err != nil {
		// 入队失败不影响索引成功
		s.logger.Warn("Failed to enqueue enrichment tasks", "error", err)
	}

	s.logger.Info("Session indexed successfully",
		"session_id", sessionID,
		"chunk_count", len(chunks),
	)

	return nil
}

// IndexSessionWithCount 索引会话并返回索引的消息数
func (s *ChunkService) IndexSessionWithCount(sessionID, filePath string) (int, error) {
	s.logger.Info("Indexing session with count",
		"session_id", sessionID,
		"file_path", filePath,
	)

	// 1. 获取项目信息
	projectInfo, err := s.getProjectInfo(sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to get project info: %w", err)
	}

	// 2. 读取会话消息
	messages, err := s.sessionService.GetSessionTextContent(sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to get session text content: %w", err)
	}

	if len(messages) == 0 {
		s.logger.Debug("No messages in session, skipping", "session_id", sessionID)
		return 0, nil
	}

	// 3. 配对消息为对话对
	turns := PairMessages(messages, sessionID)
	if len(turns) == 0 {
		s.logger.Debug("No turns after pairing, skipping", "session_id", sessionID)
		return 0, nil
	}

	// 4. 计算文件信息
	fileInfo, _ := os.Stat(filePath)
	fileMtime := int64(0)
	if fileInfo != nil {
		fileMtime = fileInfo.ModTime().Unix()
	}
	contentHash := s.calculateFileHash(filePath)
	indexedAt := time.Now().Unix()

	// 5. 提取内容并创建 KnowledgeChunks
	chunks := make([]*domainRAG.KnowledgeChunk, 0, len(turns))
	vectorTexts := make([]string, 0, len(turns))

	for i, turn := range turns {
		// 跳过未完成的对话对
		if turn.IsIncomplete {
			continue
		}

		// 提取内容
		extraction := s.contentExtractor.ExtractFromTurn(turn)

		// 跳过空内容
		if extraction.UserQuery == "" && extraction.AIResponseCore == "" {
			continue
		}

		// 创建 KnowledgeChunk
		chunk := &domainRAG.KnowledgeChunk{
			ID:               uuid.New().String(),
			SessionID:        sessionID,
			ChunkIndex:       i,
			ProjectID:        projectInfo.ProjectID,
			ProjectName:      projectInfo.ProjectName,
			WorkspaceID:      projectInfo.WorkspaceID,
			UserQuery:        extraction.UserQuery,
			AIResponseCore:   extraction.AIResponseCore,
			VectorText:       extraction.VectorText,
			ToolsUsed:        extraction.ToolsUsed,
			FilesModified:    extraction.FilesModified,
			CodeLanguages:    extraction.CodeLanguages,
			HasCode:          extraction.HasCode,
			EnrichmentStatus: domainRAG.EnrichmentStatusPending,
			Timestamp:        turn.Timestamp,
			ContentHash:      contentHash,
			FilePath:         filePath,
			IndexedAt:        indexedAt,
		}

		chunks = append(chunks, chunk)
		vectorTexts = append(vectorTexts, extraction.VectorText)
	}

	if len(chunks) == 0 {
		s.logger.Info("No valid chunks to index", "session_id", sessionID)
		return 0, nil
	}

	// 6. 批量向量化
	vectors, err := s.embeddingClient.EmbedTexts(vectorTexts)
	if err != nil {
		return 0, fmt.Errorf("failed to embed texts: %w", err)
	}

	// 7. 构建 Qdrant 点并写入
	points := s.buildChunkPoints(chunks, vectors)

	client := s.qdrantManager.GetClient()
	if client == nil {
		return 0, fmt.Errorf("qdrant client not initialized")
	}

	ctx := context.Background()
	_, err = client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: "cursor_knowledge",
		Points:         points,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to upsert chunks: %w", err)
	}

	// 8. 保存元数据到 SQLite
	if err := s.chunkRepo.SaveChunks(chunks); err != nil {
		return 0, fmt.Errorf("failed to save chunk metadata: %w", err)
	}

	// 9. 更新索引状态
	indexStatus := &domainRAG.IndexStatus{
		FilePath:      filePath,
		SessionID:     sessionID,
		ProjectID:     projectInfo.ProjectID,
		ContentHash:   contentHash,
		ChunkCount:    len(chunks),
		FileMtime:     fileMtime,
		LastIndexedAt: indexedAt,
		Status:        domainRAG.IndexStatusIndexed,
	}
	if err := s.indexStatusRepo.SaveIndexStatus(indexStatus); err != nil {
		return 0, fmt.Errorf("failed to save index status: %w", err)
	}

	// 10. 添加到增强队列
	tasks := make([]*domainRAG.EnrichmentTask, len(chunks))
	for i, chunk := range chunks {
		tasks[i] = domainRAG.NewEnrichmentTask(chunk.ID)
	}
	if err := s.enrichmentQueue.EnqueueTasks(tasks); err != nil {
		s.logger.Warn("Failed to enqueue enrichment tasks", "error", err)
	}

	s.logger.Info("Session indexed successfully with count",
		"session_id", sessionID,
		"chunk_count", len(chunks),
	)

	return len(chunks), nil
}

// sanitizeUTF8 清理字符串中的无效 UTF-8 字符
// Qdrant 客户端要求所有字符串必须是有效的 UTF-8
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	// 使用 strings.ToValidUTF8 替换无效字符为空字符串
	return strings.ToValidUTF8(s, "")
}

// buildChunkPoints 构建 Qdrant 点
func (s *ChunkService) buildChunkPoints(chunks []*domainRAG.KnowledgeChunk, vectors [][]float32) []*qdrant.PointStruct {
	points := make([]*qdrant.PointStruct, len(chunks))

	for i, chunk := range chunks {
		// 将 []float32 转换为可变参数
		vectorArgs := make([]float32, len(vectors[i]))
		copy(vectorArgs, vectors[i])

		// 序列化工具列表
		toolsUsedJSON, _ := json.Marshal(chunk.ToolsUsed)

		// 清理所有字符串字段，确保 UTF-8 有效
		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewID(chunk.ID),
			Vectors: qdrant.NewVectors(vectorArgs...),
			Payload: qdrant.NewValueMap(map[string]interface{}{
				"chunk_id":           chunk.ID,
				"session_id":         chunk.SessionID,
				"project_id":         sanitizeUTF8(chunk.ProjectID),
				"project_name":       sanitizeUTF8(chunk.ProjectName),
				"timestamp":          chunk.Timestamp,
				"has_code":           chunk.HasCode,
				"tools_used":         string(toolsUsedJSON),
				"user_query_preview": sanitizeUTF8(chunk.UserQueryPreview()),
			}),
		}
	}

	return points
}

// getProjectInfo 获取项目信息
func (s *ChunkService) getProjectInfo(sessionID string) (*ProjectInfo, error) {
	// 从文件路径中提取 projectKey
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &ProjectInfo{
			ProjectID:   "unknown",
			ProjectName: "Unknown",
			WorkspaceID: "unknown",
		}, nil
	}

	projectsDir := filepath.Join(homeDir, ".cursor", "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return &ProjectInfo{
			ProjectID:   "unknown",
			ProjectName: "Unknown",
			WorkspaceID: "unknown",
		}, nil
	}

	// 查找包含该 sessionID 的项目
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectKey := entry.Name()
		transcriptPath := filepath.Join(projectsDir, projectKey, "agent-transcripts", sessionID+".txt")
		if _, err := os.Stat(transcriptPath); err == nil {
			// 找到项目
			project := s.projectManager.GetProject(projectKey)
			if project == nil {
				return &ProjectInfo{
					ProjectID:   projectKey,
					ProjectName: projectKey,
					WorkspaceID: "",
				}, nil
			}

			workspaceID := ""
			if len(project.Workspaces) > 0 {
				workspaceID = project.Workspaces[0].WorkspaceID
			}

			return &ProjectInfo{
				ProjectID:   project.ProjectID,
				ProjectName: project.ProjectName,
				WorkspaceID: workspaceID,
			}, nil
		}
	}

	return &ProjectInfo{
		ProjectID:   "unknown",
		ProjectName: "Unknown",
		WorkspaceID: "unknown",
	}, nil
}

// calculateFileHash 计算文件内容哈希
func (s *ChunkService) calculateFileHash(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

// DeleteSessionChunks 删除会话的所有知识片段
func (s *ChunkService) DeleteSessionChunks(sessionID string) error {
	// 1. 从数据库删除
	if err := s.chunkRepo.DeleteChunksBySession(sessionID); err != nil {
		return fmt.Errorf("failed to delete chunks from database: %w", err)
	}

	// 2. 从 Qdrant 删除（通过 session_id 过滤）
	client := s.qdrantManager.GetClient()
	if client == nil {
		return fmt.Errorf("qdrant client not initialized")
	}

	ctx := context.Background()
	_, err := client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: "cursor_knowledge",
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: &qdrant.Filter{
					Must: []*qdrant.Condition{
						{
							ConditionOneOf: &qdrant.Condition_Field{
								Field: &qdrant.FieldCondition{
									Key: "session_id",
									Match: &qdrant.Match{
										MatchValue: &qdrant.Match_Keyword{
											Keyword: sessionID,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	return err
}

// GetIndexStats 获取索引统计信息
func (s *ChunkService) GetIndexStats() (*IndexStats, error) {
	allStatus, err := s.indexStatusRepo.GetAllIndexStatus()
	if err != nil {
		return nil, err
	}

	stats := &IndexStats{
		TotalFiles:   len(allStatus),
		TotalChunks:  0,
		LastScanTime: 0,
	}

	for _, status := range allStatus {
		stats.TotalChunks += status.ChunkCount
		if status.LastIndexedAt > stats.LastScanTime {
			stats.LastScanTime = status.LastIndexedAt
		}
	}

	return stats, nil
}

// IndexStats 索引统计
type IndexStats struct {
	TotalFiles   int   `json:"total_files"`
	TotalChunks  int   `json:"total_chunks"`
	LastScanTime int64 `json:"last_scan_time"`
}

// ProjectInfo 项目信息
type ProjectInfo struct {
	ProjectID   string
	ProjectName string
	WorkspaceID string
}
