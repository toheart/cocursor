package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	"github.com/cocursor/backend/internal/infrastructure/embedding"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/cocursor/backend/internal/infrastructure/vector"
	"github.com/qdrant/go-client/qdrant"
)

// SearchService 搜索服务
type SearchService struct {
	embeddingClient *embedding.Client
	qdrantManager   *vector.QdrantManager
	chunkRepo       domainRAG.ChunkRepository
	logger          *slog.Logger
}

// NewSearchService 创建搜索服务
func NewSearchService(
	embeddingClient *embedding.Client,
	qdrantManager *vector.QdrantManager,
	chunkRepo domainRAG.ChunkRepository,
) *SearchService {
	return &SearchService{
		embeddingClient: embeddingClient,
		qdrantManager:   qdrantManager,
		chunkRepo:       chunkRepo,
		logger:          log.NewModuleLogger("rag", "search"),
	}
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Query      string   `json:"query"`
	ProjectIDs []string `json:"project_ids,omitempty"` // 项目过滤（空则搜索所有项目）
	Limit      int      `json:"limit"`                 // 返回结果数量
}

// ChunkSearchResult 知识片段搜索结果
type ChunkSearchResult struct {
	ChunkID          string   `json:"chunk_id"`
	SessionID        string   `json:"session_id"`
	Score            float32  `json:"score"`
	ProjectID        string   `json:"project_id"`
	ProjectName      string   `json:"project_name"`
	UserQueryPreview string   `json:"user_query_preview"`
	Summary          string   `json:"summary,omitempty"`
	MainTopic        string   `json:"main_topic,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	ToolsUsed        []string `json:"tools_used,omitempty"`
	FilesModified    []string `json:"files_modified,omitempty"`
	HasCode          bool     `json:"has_code"`
	Timestamp        int64    `json:"timestamp"`
	IsEnriched       bool     `json:"is_enriched"`
}

// ChunkDetail 知识片段详情
type ChunkDetail struct {
	ChunkSearchResult
	UserQuery        string `json:"user_query"`
	AIResponseCore   string `json:"ai_response_core"`
	EnrichmentStatus string `json:"enrichment_status"`
	EnrichmentError  string `json:"enrichment_error,omitempty"`
}

// Search 执行语义搜索（使用 cursor_knowledge 集合）
func (s *SearchService) Search(ctx context.Context, req *SearchRequest) ([]*ChunkSearchResult, error) {
	return s.SearchChunks(ctx, req)
}

// SearchChunks 搜索知识片段
func (s *SearchService) SearchChunks(ctx context.Context, req *SearchRequest) ([]*ChunkSearchResult, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	s.logger.Info("Starting search",
		"query", req.Query,
		"limit", req.Limit,
		"project_ids", req.ProjectIDs,
	)

	// 1. 向量化查询文本
	queryVectors, err := s.embeddingClient.EmbedTexts([]string{req.Query})
	if err != nil {
		s.logger.Error("Failed to embed query", "error", err)
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	if len(queryVectors) == 0 || len(queryVectors[0]) == 0 {
		s.logger.Error("Invalid embedding result", "vectors_count", len(queryVectors))
		return nil, fmt.Errorf("invalid embedding result")
	}

	queryVector := queryVectors[0]
	s.logger.Debug("Query embedded", "vector_dim", len(queryVector))

	client := s.qdrantManager.GetClient()
	if client == nil {
		s.logger.Error("Qdrant client not initialized")
		return nil, fmt.Errorf("qdrant client not initialized")
	}

	// 2. 构建过滤条件
	filter := s.buildProjectFilter(req.ProjectIDs)

	// 3. 执行搜索
	limit := uint64(req.Limit)
	searchResp, err := client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: "cursor_knowledge",
		Query:          qdrant.NewQuery(queryVector...),
		Limit:          &limit,
		Filter:         filter,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		s.logger.Error("Failed to query qdrant", "error", err)
		return nil, fmt.Errorf("failed to query qdrant: %w", err)
	}

	s.logger.Info("Qdrant search completed", "hits_count", len(searchResp))

	// 4. 转换结果
	results := make([]*ChunkSearchResult, 0, len(searchResp))
	for _, hit := range searchResp {
		result := s.chunkHitToResult(hit)
		if result != nil {
			results = append(results, result)
		}
	}

	s.logger.Info("Search completed", "results_count", len(results))

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

// chunkHitToResult 将知识片段命中转换为搜索结果
func (s *SearchService) chunkHitToResult(hit *qdrant.ScoredPoint) *ChunkSearchResult {
	payload := hit.GetPayload()
	if payload == nil {
		return nil
	}

	result := &ChunkSearchResult{
		Score: hit.GetScore(),
	}

	// 从 payload 提取信息
	if val, ok := payload["chunk_id"]; ok {
		result.ChunkID = extractStringValue(val)
	}
	if val, ok := payload["session_id"]; ok {
		result.SessionID = extractStringValue(val)
	}
	if val, ok := payload["project_id"]; ok {
		result.ProjectID = extractStringValue(val)
	}
	if val, ok := payload["project_name"]; ok {
		result.ProjectName = extractStringValue(val)
	}
	if val, ok := payload["user_query_preview"]; ok {
		result.UserQueryPreview = extractStringValue(val)
	}
	if val, ok := payload["summary"]; ok {
		result.Summary = extractStringValue(val)
	}
	if val, ok := payload["main_topic"]; ok {
		result.MainTopic = extractStringValue(val)
	}
	if val, ok := payload["tags"]; ok {
		tagsStr := extractStringValue(val)
		if tagsStr != "" {
			var tags []string
			if err := json.Unmarshal([]byte(tagsStr), &tags); err == nil {
				result.Tags = tags
			}
		}
	}
	if val, ok := payload["tools_used"]; ok {
		toolsStr := extractStringValue(val)
		if toolsStr != "" {
			var tools []string
			if err := json.Unmarshal([]byte(toolsStr), &tools); err == nil {
				result.ToolsUsed = tools
			}
		}
	}
	if val, ok := payload["has_code"]; ok {
		result.HasCode = extractBoolValue(val)
	}
	if val, ok := payload["timestamp"]; ok {
		result.Timestamp = extractIntValue(val)
	}

	// 判断是否已增强
	result.IsEnriched = result.Summary != ""

	return result
}

// GetChunkDetail 获取知识片段详情
func (s *SearchService) GetChunkDetail(chunkID string) (*ChunkDetail, error) {
	if s.chunkRepo == nil {
		return nil, fmt.Errorf("chunk repository not initialized")
	}

	chunk, err := s.chunkRepo.GetChunk(chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}
	if chunk == nil {
		return nil, fmt.Errorf("chunk not found")
	}

	// 从数据库获取完整信息
	detail := &ChunkDetail{
		ChunkSearchResult: ChunkSearchResult{
			ChunkID:          chunk.ID,
			SessionID:        chunk.SessionID,
			Score:            0, // 详情不包含分数
			ProjectID:        chunk.ProjectID,
			ProjectName:      chunk.ProjectName,
			UserQueryPreview: chunk.UserQueryPreview(),
			Summary:          chunk.Summary,
			MainTopic:        chunk.MainTopic,
			Tags:             chunk.Tags,
			ToolsUsed:        chunk.ToolsUsed,
			FilesModified:    chunk.FilesModified,
			HasCode:          chunk.HasCode,
			Timestamp:        chunk.Timestamp,
			IsEnriched:       chunk.IsEnriched(),
		},
		UserQuery:        chunk.UserQuery,
		AIResponseCore:   chunk.AIResponseCore,
		EnrichmentStatus: chunk.EnrichmentStatus,
		EnrichmentError:  chunk.EnrichmentError,
	}

	return detail, nil
}

// extractStringValue 从 qdrant.Value 提取字符串值
func extractStringValue(val *qdrant.Value) string {
	if val == nil {
		return ""
	}
	if strVal := val.GetStringValue(); strVal != "" {
		return strVal
	}
	return ""
}

// extractIntValue 从 qdrant.Value 提取整数值
func extractIntValue(val *qdrant.Value) int64 {
	if val == nil {
		return 0
	}
	if intVal := val.GetIntegerValue(); intVal != 0 {
		return intVal
	}
	if dblVal := val.GetDoubleValue(); dblVal != 0 {
		return int64(dblVal)
	}
	return 0
}

// extractBoolValue 从 qdrant.Value 提取布尔值
func extractBoolValue(val *qdrant.Value) bool {
	if val == nil {
		return false
	}
	return val.GetBoolValue()
}
