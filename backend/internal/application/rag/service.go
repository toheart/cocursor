package rag

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	domainCursor "github.com/cocursor/backend/internal/application/cursor"
	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	"github.com/cocursor/backend/internal/infrastructure/embedding"
	"github.com/cocursor/backend/internal/infrastructure/vector"
	"github.com/qdrant/go-client/qdrant"
)

// RAGService RAG 服务
type RAGService struct {
	sessionService *domainCursor.SessionService
	embeddingClient *embedding.Client
	qdrantManager  *vector.QdrantManager
	ragRepo        domainRAG.RAGRepository
	projectManager *domainCursor.ProjectManager
}

// NewRAGService 创建 RAG 服务
func NewRAGService(
	sessionService *domainCursor.SessionService,
	embeddingClient *embedding.Client,
	qdrantManager *vector.QdrantManager,
	ragRepo domainRAG.RAGRepository,
	projectManager *domainCursor.ProjectManager,
) *RAGService {
	return &RAGService{
		sessionService:  sessionService,
		embeddingClient: embeddingClient,
		qdrantManager:   qdrantManager,
		ragRepo:         ragRepo,
		projectManager:  projectManager,
	}
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

	// 5. 提取文本内容
	messageTexts := ExtractMessageTexts(indexedMessages)
	turnTexts := ExtractTurnTexts(turns)

	// 6. 批量向量化
	messageVectors, err := s.embeddingClient.EmbedTexts(messageTexts)
	if err != nil {
		return fmt.Errorf("failed to embed message texts: %w", err)
	}

	turnVectors, err := s.embeddingClient.EmbedTexts(turnTexts)
	if err != nil {
		return fmt.Errorf("failed to embed turn texts: %w", err)
	}

	// 7. 构建 Qdrant 点
	messagePoints := s.buildMessagePoints(sessionID, projectInfo, indexedMessages, messageVectors)
	turnPoints := s.buildTurnPoints(sessionID, projectInfo, turns, turnVectors)

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
			SessionID:   sessionID,
			MessageID:   im.MessageID,
			WorkspaceID: projectInfo.WorkspaceID,
			ProjectID:   projectInfo.ProjectID,
			ProjectName: projectInfo.ProjectName,
			MessageType: string(im.Message.Type),
			MessageIndex: im.Index,
			TurnIndex:   turnIndex,
			VectorID:    fmt.Sprintf("%s:%s", sessionID, im.MessageID),
			ContentHash: contentHash,
			FilePath:    filePath,
			FileMtime:   fileMtime,
			IndexedAt:   indexedAt,
		})
		if err != nil {
			return fmt.Errorf("failed to save message metadata: %w", err)
		}
	}

	// 更新对话对元数据
	for _, turn := range turns {
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
			VectorID:       fmt.Sprintf("%s:turn-%d", sessionID, turn.TurnIndex),
			ContentHash:    turnHash,
			FilePath:       filePath,
			FileMtime:      fileMtime,
			IndexedAt:      indexedAt,
			IsIncomplete:   turn.IsIncomplete,
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
		vectorID := fmt.Sprintf("%s:%s", sessionID, im.MessageID)
		
		// 将 []float32 转换为可变参数
		vectorArgs := make([]float32, len(vectors[i]))
		copy(vectorArgs, vectors[i])
		
		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewID(vectorID),
			Vectors: qdrant.NewVectors(vectorArgs...),
			Payload: qdrant.NewValueMap(map[string]interface{}{
				"session_id":    sessionID,
				"message_id":    im.MessageID,
				"message_type":  string(im.Message.Type),
				"content":       im.Message.Text,
				"timestamp":     im.Message.Timestamp,
				"message_index": im.Index,
				"project_id":    projectInfo.ProjectID,
				"project_name":   projectInfo.ProjectName,
				"workspace_id":   projectInfo.WorkspaceID,
			}),
		}
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
	points := make([]*qdrant.PointStruct, len(turns))
	for i, turn := range turns {
		vectorID := fmt.Sprintf("%s:turn-%d", sessionID, turn.TurnIndex)
		
		// 将 []float32 转换为可变参数
		vectorArgs := make([]float32, len(vectors[i]))
		copy(vectorArgs, vectors[i])
		
		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewID(vectorID),
			Vectors: qdrant.NewVectors(vectorArgs...),
			Payload: qdrant.NewValueMap(map[string]interface{}{
				"session_id":    sessionID,
				"turn_index":    turn.TurnIndex,
				"user_text":     turn.UserText,
				"ai_text":       turn.AIText,
				"combined_text": turn.CombinedText,
				"message_count": len(turn.UserMessages) + len(turn.AIMessages),
				"project_id":    projectInfo.ProjectID,
				"project_name":  projectInfo.ProjectName,
				"workspace_id":  projectInfo.WorkspaceID,
				"is_incomplete": turn.IsIncomplete,
			}),
		}
	}
	return points
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
