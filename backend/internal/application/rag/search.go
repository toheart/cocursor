package rag

import (
	"context"
	"fmt"
	"sort"

	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	"github.com/cocursor/backend/internal/infrastructure/embedding"
	"github.com/cocursor/backend/internal/infrastructure/vector"
	"github.com/qdrant/go-client/qdrant"
)

// SearchService 搜索服务
type SearchService struct {
	embeddingClient *embedding.Client
	qdrantManager   *vector.QdrantManager
	ragRepo         domainRAG.RAGRepository
}

// NewSearchService 创建搜索服务
func NewSearchService(
	embeddingClient *embedding.Client,
	qdrantManager *vector.QdrantManager,
	ragRepo domainRAG.RAGRepository,
) *SearchService {
	return &SearchService{
		embeddingClient: embeddingClient,
		qdrantManager:   qdrantManager,
		ragRepo:         ragRepo,
	}
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Query      string   `json:"query"`
	ProjectIDs []string `json:"project_ids,omitempty"` // 项目过滤（空则搜索所有项目）
	Limit      int      `json:"limit"`                 // 返回结果数量
}

// SearchResult 搜索结果
type SearchResult struct {
	Type        string   `json:"type"` // "message" 或 "turn"
	SessionID   string   `json:"session_id"`
	Score       float32  `json:"score"`
	Content     string   `json:"content"`
	UserText    string   `json:"user_text,omitempty"`  // 对话对才有
	AIText      string   `json:"ai_text,omitempty"`    // 对话对才有
	MessageID   string   `json:"message_id,omitempty"` // 消息才有
	TurnIndex   int      `json:"turn_index,omitempty"` // 对话对才有
	ProjectID   string   `json:"project_id"`
	ProjectName string   `json:"project_name"`
	Timestamp   int64    `json:"timestamp"`
	MessageIDs  []string `json:"message_ids,omitempty"` // 对话对包含的消息 ID
	Summary     string   `json:"summary"`               // 总结 JSON 字符串
}

// Search 执行语义搜索
func (s *SearchService) Search(ctx context.Context, req *SearchRequest) ([]*SearchResult, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// 1. 向量化查询文本
	queryVectors, err := s.embeddingClient.EmbedTexts([]string{req.Query})
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	if len(queryVectors) == 0 || len(queryVectors[0]) == 0 {
		return nil, fmt.Errorf("invalid embedding result")
	}

	queryVector := queryVectors[0]
	client := s.qdrantManager.GetClient()
	if client == nil {
		return nil, fmt.Errorf("qdrant client not initialized")
	}

	// 2. 只在对话对级别搜索（总结向量）
	turnResults, err := s.searchTurns(ctx, client, queryVector, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search turns: %w", err)
	}

	return turnResults, nil
}

// searchTurns 搜索对话对级别
func (s *SearchService) searchTurns(ctx context.Context, client *qdrant.Client, queryVector []float32, req *SearchRequest) ([]*SearchResult, error) {
	// 构建过滤条件
	filter := s.buildProjectFilter(req.ProjectIDs)

	// 执行搜索
	limit := uint64(req.Limit * 2)
	searchResp, err := client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: "cursor_sessions_turns",
		Query:          qdrant.NewQuery(queryVector...),
		Limit:          &limit, // 多取一些，因为会加权
		Filter:         filter,
	})
	if err != nil {
		return nil, err
	}

	var results []*SearchResult
	for _, hit := range searchResp {
		result := s.turnHitToResult(hit)
		if result != nil {
			results = append(results, result)
		}
	}

	return results, nil
}

// searchMessages 搜索消息级别
func (s *SearchService) searchMessages(ctx context.Context, client *qdrant.Client, queryVector []float32, req *SearchRequest) ([]*SearchResult, error) {
	// 构建过滤条件
	filter := s.buildProjectFilter(req.ProjectIDs)

	// 执行搜索
	limit := uint64(req.Limit * 2)
	searchResp, err := client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: "cursor_sessions_messages",
		Query:          qdrant.NewQuery(queryVector...),
		Limit:          &limit, // 多取一些，用于合并
		Filter:         filter,
	})
	if err != nil {
		return nil, err
	}

	var results []*SearchResult
	for _, hit := range searchResp {
		result := s.messageHitToResult(hit)
		if result != nil {
			results = append(results, result)
		}
	}

	return results, nil
}

// buildProjectFilter 构建项目过滤条件
func (s *SearchService) buildProjectFilter(projectIDs []string) *qdrant.Filter {
	if len(projectIDs) == 0 {
		return nil // 不过滤
	}

	// 如果只有一个项目 ID，使用 NewMatch
	if len(projectIDs) == 1 {
		return &qdrant.Filter{
			Must: []*qdrant.Condition{
				qdrant.NewMatch("project_id", projectIDs[0]),
			},
		}
	}

	// 多个项目 ID，使用 OR 逻辑（should）
	conditions := make([]*qdrant.Condition, len(projectIDs))
	for i, projectID := range projectIDs {
		conditions[i] = qdrant.NewMatch("project_id", projectID)
	}

	return &qdrant.Filter{
		Should: conditions,
	}
}

// turnHitToResult 将对话对命中转换为搜索结果
func (s *SearchService) turnHitToResult(hit *qdrant.ScoredPoint) *SearchResult {
	payload := hit.GetPayload()
	if payload == nil {
		return nil
	}

	result := &SearchResult{
		Type:  "turn",
		Score: hit.GetScore(),
	}

	// 从 payload 提取信息
	if val, ok := payload["session_id"]; ok {
		result.SessionID = s.extractStringValue(val)
	}
	if val, ok := payload["turn_index"]; ok {
		result.TurnIndex = int(s.extractIntValue(val))
	}
	if val, ok := payload["user_text"]; ok {
		result.UserText = s.extractStringValue(val)
	}
	if val, ok := payload["ai_text"]; ok {
		result.AIText = s.extractStringValue(val)
	}
	if val, ok := payload["combined_text"]; ok {
		result.Content = s.extractStringValue(val)
	}
	if val, ok := payload["project_id"]; ok {
		result.ProjectID = s.extractStringValue(val)
	}
	if val, ok := payload["project_name"]; ok {
		result.ProjectName = s.extractStringValue(val)
	}
	// 提取 summary (JSON 字符串)
	if val, ok := payload["summary"]; ok {
		result.Summary = s.extractStringValue(val)
	}

	return result
}

// messageHitToResult 将消息命中转换为搜索结果
func (s *SearchService) messageHitToResult(hit *qdrant.ScoredPoint) *SearchResult {
	payload := hit.GetPayload()
	if payload == nil {
		return nil
	}

	result := &SearchResult{
		Type:  "message",
		Score: hit.GetScore(),
	}

	// 从 payload 提取信息
	if val, ok := payload["session_id"]; ok {
		result.SessionID = s.extractStringValue(val)
	}
	if val, ok := payload["message_id"]; ok {
		result.MessageID = s.extractStringValue(val)
	}
	if val, ok := payload["content"]; ok {
		result.Content = s.extractStringValue(val)
	}
	if val, ok := payload["timestamp"]; ok {
		result.Timestamp = s.extractIntValue(val)
	}
	if val, ok := payload["project_id"]; ok {
		result.ProjectID = s.extractStringValue(val)
	}
	if val, ok := payload["project_name"]; ok {
		result.ProjectName = s.extractStringValue(val)
	}

	return result
}

// extractStringValue 从 Value 中提取字符串
func (s *SearchService) extractStringValue(val *qdrant.Value) string {
	if val == nil {
		return ""
	}
	if strVal, ok := val.Kind.(*qdrant.Value_StringValue); ok {
		return strVal.StringValue
	}
	return ""
}

// extractIntValue 从 Value 中提取整数
func (s *SearchService) extractIntValue(val *qdrant.Value) int64 {
	if val == nil {
		return 0
	}
	if intVal, ok := val.Kind.(*qdrant.Value_IntegerValue); ok {
		return intVal.IntegerValue
	}
	return 0
}

// mergeResults 合并搜索结果（对话对加权 20%，去重，排序）
func (s *SearchService) mergeResults(turnResults, messageResults []*SearchResult, limit int) []*SearchResult {
	// 对话对加权 20%
	for _, result := range turnResults {
		result.Score = result.Score * 1.2
	}

	// 合并结果
	allResults := append(turnResults, messageResults...)

	// 去重（基于 session_id + type + message_id/turn_index）
	seen := make(map[string]bool)
	var uniqueResults []*SearchResult
	for _, result := range allResults {
		key := s.getResultKey(result)
		if !seen[key] {
			seen[key] = true
			uniqueResults = append(uniqueResults, result)
		}
	}

	// 按分数排序
	sort.Slice(uniqueResults, func(i, j int) bool {
		return uniqueResults[i].Score > uniqueResults[j].Score
	})

	// 限制数量
	if len(uniqueResults) > limit {
		uniqueResults = uniqueResults[:limit]
	}

	return uniqueResults
}

// getResultKey 获取结果去重键
func (s *SearchService) getResultKey(result *SearchResult) string {
	if result.Type == "turn" {
		return fmt.Sprintf("%s:turn:%d", result.SessionID, result.TurnIndex)
	}
	return fmt.Sprintf("%s:message:%s", result.SessionID, result.MessageID)
}
