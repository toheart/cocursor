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

	"log/slog"

	"github.com/google/uuid"

	domainCursor "github.com/cocursor/backend/internal/application/cursor"
	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	"github.com/cocursor/backend/internal/infrastructure/embedding"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/cocursor/backend/internal/infrastructure/vector"
	"github.com/qdrant/go-client/qdrant"
)

// RAGService RAG 服务
type RAGService struct {
	sessionService  *domainCursor.SessionService
	embeddingClient *embedding.Client
	llmClient       *LLMClient  // LLM 客户端（可选）
	summarizer      *Summarizer // 总结服务（可选）
	qdrantManager   *vector.QdrantManager
	ragRepo         domainRAG.RAGRepository
	projectManager  *domainCursor.ProjectManager
	logger          *slog.Logger
}

// NewRAGService 创建 RAG 服务
func NewRAGService(
	sessionService *domainCursor.SessionService,
	embeddingClient *embedding.Client,
	llmClient *LLMClient,
	qdrantManager *vector.QdrantManager,
	ragRepo domainRAG.RAGRepository,
	projectManager *domainCursor.ProjectManager,
) *RAGService {
	ragService := &RAGService{
		sessionService:  sessionService,
		embeddingClient: embeddingClient,
		llmClient:       llmClient, // 可能为 nil
		qdrantManager:   qdrantManager,
		ragRepo:         ragRepo,
		projectManager:  projectManager,
		logger:          log.NewModuleLogger("rag", "service"),
	}

	// 如果 LLMClient 可用，创建 Summarizer
	if llmClient != nil {
		ragService.summarizer = NewSummarizer(llmClient)
	}

	return ragService
}

// ProjectInfo 项目信息
type ProjectInfo struct {
	ProjectID   string
	ProjectName string
	WorkspaceID string
}

// IndexSession 索引单个会话
func (s *RAGService) IndexSession(sessionID, filePath string) error {
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
		return nil // 没有消息，跳过
	}

	// 3. 生成消息 ID 和索引消息
	indexedMessages := make([]*IndexedMessage, 0, len(messages))
	for i, msg := range messages {
		messageID := fmt.Sprintf("%s-%d-%d", sessionID, i, msg.Timestamp)
		indexedMessages = append(indexedMessages, &IndexedMessage{
			Message:   msg,
			MessageID: messageID,
			Index:     i,
		})
	}

	// 4. 配对消息为对话对
	turns := PairMessages(messages, sessionID)

	// 4.5 对每个对话对进行总结（如果 LLMClient 可用）
	successCount := 0
	failedCount := 0
	if s.summarizer != nil {
		for _, turn := range turns {
			summary, err := s.summarizer.SummarizeTurn(turn)
			if err != nil {
				// 总结失败，记录但跳过该对话对
				s.logger.Warn("Failed to summarize turn, skipping",
					"turn_index", turn.TurnIndex,
					"error", err,
				)
				failedCount++
				// 标记总结为 nil，后续向量化时会跳过
				turn.Summary = nil
			} else {
				turn.Summary = summary
				successCount++
			}
		}
		s.logger.Info("Summarization completed",
			"total_turns", len(turns),
			"success_count", successCount,
			"failed_count", failedCount,
		)
	}

	// 5. 提取文本内容
	messageTexts := ExtractMessageTexts(indexedMessages)

	// 5.5 提取总结文本用于向量化（只向量化总结）
	summaryTexts := s.extractSummaryTexts(turns)

	// 6. 批量向量化
	messageVectors, err := s.embeddingClient.EmbedTexts(messageTexts)
	if err != nil {
		return fmt.Errorf("failed to embed message texts: %w", err)
	}

	// 向量化总结内容（如果有）
	var summaryVectors [][]float32
	if len(summaryTexts) > 0 {
		summaryVectors, err = s.embeddingClient.EmbedTexts(summaryTexts)
		if err != nil {
			return fmt.Errorf("failed to embed summary texts: %w", err)
		}
	}

	// 7. 构建 Qdrant 点
	messagePoints := s.buildMessagePoints(sessionID, projectInfo, indexedMessages, messageVectors)
	turnPoints := s.buildTurnPoints(sessionID, projectInfo, turns, summaryVectors)

	// 8. Upsert 到 Qdrant
	client := s.qdrantManager.GetClient()
	if client == nil {
		return fmt.Errorf("qdrant client not initialized")
	}

	ctx := context.Background()

	// 写入消息级别向量
	_, err = client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: "cursor_sessions_messages",
		Points:         messagePoints,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert messages: %w", err)
	}

	// 写入对话对级别向量
	_, err = client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: "cursor_sessions_turns",
		Points:         turnPoints,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert turns: %w", err)
	}

	// 9. 更新元数据表
	fileInfo, _ := os.Stat(filePath)
	fileMtime := int64(0)
	if fileInfo != nil {
		fileMtime = fileInfo.ModTime().Unix()
	}
	contentHash := s.calculateFileHash(filePath)
	indexedAt := time.Now().Unix()

	// 更新消息元数据
	for _, im := range indexedMessages {
		turnIndex := s.findTurnIndex(sessionID, im.MessageID, turns)
		err := s.ragRepo.SaveMessageMetadata(&domainRAG.MessageMetadata{
			SessionID:    sessionID,
			MessageID:    im.MessageID,
			WorkspaceID:  projectInfo.WorkspaceID,
			ProjectID:    projectInfo.ProjectID,
			ProjectName:  projectInfo.ProjectName,
			MessageType:  string(im.Message.Type),
			MessageIndex: im.Index,
			TurnIndex:    turnIndex,
			VectorID:     im.VectorID, // 使用生成的 UUID
			ContentHash:  contentHash,
			FilePath:     filePath,
			FileMtime:    fileMtime,
			IndexedAt:    indexedAt,
		})
		if err != nil {
			return fmt.Errorf("failed to save message metadata: %w", err)
		}
	}

	// 更新对话对元数据
	for _, turn := range turns {
		// 只保存有总结的对话对
		if turn.Summary == nil {
			continue
		}

		turnHash := s.calculateTurnHash(turn)
		err := s.ragRepo.SaveTurnMetadata(&domainRAG.TurnMetadata{
			SessionID:      sessionID,
			TurnIndex:      turn.TurnIndex,
			WorkspaceID:    projectInfo.WorkspaceID,
			ProjectID:      projectInfo.ProjectID,
			ProjectName:    projectInfo.ProjectName,
			UserMessageIDs: turn.UserMessageIDs,
			AIMessageIDs:   turn.AIMessageIDs,
			MessageCount:   len(turn.UserMessages) + len(turn.AIMessages),
			VectorID:       turn.VectorID, // 使用生成的 UUID
			ContentHash:    turnHash,
			FilePath:       filePath,
			FileMtime:      fileMtime,
			IndexedAt:      indexedAt,
			IsIncomplete:   turn.IsIncomplete,
			Summary:        turn.Summary, // 保存总结
		})
		if err != nil {
			return fmt.Errorf("failed to save turn metadata: %w", err)
		}
	}

	return nil
}

// buildMessagePoints 构建消息级别的 Qdrant 点
func (s *RAGService) buildMessagePoints(
	sessionID string,
	projectInfo *ProjectInfo,
	indexedMessages []*IndexedMessage,
	vectors [][]float32,
) []*qdrant.PointStruct {
	points := make([]*qdrant.PointStruct, len(indexedMessages))
	for i, im := range indexedMessages {
		// 生成 UUID 作为 Qdrant 点 ID
		pointID := uuid.New().String()

		// 将 []float32 转换为可变参数
		vectorArgs := make([]float32, len(vectors[i]))
		copy(vectorArgs, vectors[i])

		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewID(pointID),
			Vectors: qdrant.NewVectors(vectorArgs...),
			Payload: qdrant.NewValueMap(map[string]interface{}{
				"session_id":    sessionID,
				"message_id":    im.MessageID,
				"message_type":  string(im.Message.Type),
				"content":       im.Message.Text,
				"timestamp":     im.Message.Timestamp,
				"message_index": im.Index,
				"project_id":    projectInfo.ProjectID,
				"project_name":  projectInfo.ProjectName,
				"workspace_id":  projectInfo.WorkspaceID,
			}),
		}

		// 保存生成的 UUID 到 VectorID（用于元数据）
		im.VectorID = pointID
	}
	return points
}

// buildTurnPoints 构建对话对级别的 Qdrant 点
func (s *RAGService) buildTurnPoints(
	sessionID string,
	projectInfo *ProjectInfo,
	turns []*ConversationTurn,
	vectors [][]float32,
) []*qdrant.PointStruct {
	points := make([]*qdrant.PointStruct, 0, len(turns)) // 预分配容量，实际长度可能更少
	pointIndex := 0

	for _, turn := range turns {
		// 跳过没有总结且没有向量的对话对
		if turn.Summary == nil || vectors[pointIndex] == nil || len(vectors[pointIndex]) == 0 {
			s.logger.Debug("Skipping turn without summary or vector",
				"turn_index", turn.TurnIndex,
			)
			continue
		}

		// 生成 UUID 作为 Qdrant 点 ID
		pointID := uuid.New().String()

		// 将 []float32 转换为可变参数
		vectorArgs := make([]float32, len(vectors[pointIndex]))
		copy(vectorArgs, vectors[pointIndex])

		// 序列化总结为 JSON
		summaryJSON, _ := json.Marshal(turn.Summary)

		points[pointIndex] = &qdrant.PointStruct{
			Id:      qdrant.NewID(pointID),
			Vectors: qdrant.NewVectors(vectorArgs...),
			Payload: qdrant.NewValueMap(map[string]interface{}{
				"session_id":    sessionID,
				"turn_index":    turn.TurnIndex,
				"summary":       string(summaryJSON), // 总结 JSON 字符串
				"message_count": len(turn.UserMessages) + len(turn.AIMessages),
				"project_id":    projectInfo.ProjectID,
				"project_name":  projectInfo.ProjectName,
				"workspace_id":  projectInfo.WorkspaceID,
				"is_incomplete": turn.IsIncomplete,
			}),
		}

		// 保存生成的 UUID 到 VectorID（用于元数据）
		turn.VectorID = pointID
		pointIndex++
	}

	// 返回实际使用的点（可能少于原始 turns 数量）
	return points[:pointIndex]
}

// getProjectInfo 获取项目信息
func (s *RAGService) getProjectInfo(sessionID string) (*ProjectInfo, error) {
	// 从文件路径中提取 projectKey
	// 简化实现：通过扫描 projects 目录查找
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

// findTurnIndex 查找消息所属的对话对索引
func (s *RAGService) findTurnIndex(sessionID, messageID string, turns []*ConversationTurn) int {
	for _, turn := range turns {
		for _, id := range turn.UserMessageIDs {
			if id == messageID {
				return turn.TurnIndex
			}
		}
		for _, id := range turn.AIMessageIDs {
			if id == messageID {
				return turn.TurnIndex
			}
		}
	}
	return -1
}

// calculateFileHash 计算文件内容哈希
func (s *RAGService) calculateFileHash(filePath string) string {
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

// calculateTurnHash 计算对话对内容哈希
func (s *RAGService) calculateTurnHash(turn *ConversationTurn) string {
	hash := sha256.New()
	hash.Write([]byte(turn.CombinedText))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// extractSummaryTexts 提取总结文本用于向量化
func (s *RAGService) extractSummaryTexts(turns []*ConversationTurn) []string {
	texts := make([]string, len(turns))
	for i, turn := range turns {
		// 只向量化有总结的对话对
		if turn.Summary != nil {
			texts[i] = s.extractTurnSummaryVectorText(turn.Summary)
		} else {
			// 如果没有总结，使用空字符串（后续会被过滤）
			texts[i] = ""
		}
	}
	return texts
}

// extractTurnSummaryVectorText 组合总结字段为一段文本
func (s *RAGService) extractTurnSummaryVectorText(summary *domainRAG.TurnSummary) string {
	parts := []string{
		fmt.Sprintf("MainTopic: %s", summary.MainTopic),
		fmt.Sprintf("KeyPoints: %s", joinStrings(summary.KeyPoints, "; ")),
		fmt.Sprintf("Tags: %s", joinStrings(summary.Tags, ", ")),
		summary.Summary,
	}
	return fmt.Sprintf("%s\n\n%s", strings.Join(parts, "\n"), summary.Context)
}

// joinStrings 辅助函数：连接字符串数组
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	return strings.Join(strs, sep)
}
