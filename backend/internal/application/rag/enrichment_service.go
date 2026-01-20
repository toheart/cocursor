package rag

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/cocursor/backend/internal/infrastructure/vector"
	"github.com/qdrant/go-client/qdrant"
)

// EnrichmentService LLM 增强服务
type EnrichmentService struct {
	chunkRepo       domainRAG.ChunkRepository
	enrichmentQueue domainRAG.EnrichmentQueueRepository
	llmClient       *LLMClient
	qdrantManager   *vector.QdrantManager
	logger          *slog.Logger

	// Worker 控制
	workerCount   int
	stopChan      chan struct{}
	wg            sync.WaitGroup
	isRunning     bool
	mu            sync.Mutex
	pollInterval  time.Duration
	batchSize     int
}

// NewEnrichmentService 创建增强服务
func NewEnrichmentService(
	chunkRepo domainRAG.ChunkRepository,
	enrichmentQueue domainRAG.EnrichmentQueueRepository,
	llmClient *LLMClient,
	qdrantManager *vector.QdrantManager,
) *EnrichmentService {
	return &EnrichmentService{
		chunkRepo:       chunkRepo,
		enrichmentQueue: enrichmentQueue,
		llmClient:       llmClient,
		qdrantManager:   qdrantManager,
		logger:          log.NewModuleLogger("rag", "enrichment"),
		workerCount:     2,
		pollInterval:    30 * time.Second,
		batchSize:       5,
	}
}

// StartWorkers 启动后台 Worker
func (s *EnrichmentService) StartWorkers() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return
	}

	// 如果没有 LLM 客户端，不启动 Worker
	if s.llmClient == nil {
		s.logger.Info("LLM client not configured, enrichment worker not started")
		return
	}

	s.stopChan = make(chan struct{})
	s.isRunning = true

	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	s.logger.Info("Enrichment workers started", "count", s.workerCount)
}

// StopWorkers 停止后台 Worker
func (s *EnrichmentService) StopWorkers() {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = false
	close(s.stopChan)
	s.mu.Unlock()

	s.wg.Wait()
	s.logger.Info("Enrichment workers stopped")
}

// worker 后台工作协程
func (s *EnrichmentService) worker(id int) {
	defer s.wg.Done()

	s.logger.Info("Enrichment worker started", "worker_id", id)

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			s.logger.Info("Enrichment worker stopping", "worker_id", id)
			return
		case <-ticker.C:
			s.processQueue(id)
		}
	}
}

// processQueue 处理队列中的任务
func (s *EnrichmentService) processQueue(workerID int) {
	// 获取待处理任务
	tasks, err := s.enrichmentQueue.DequeueTasks(s.batchSize)
	if err != nil {
		s.logger.Error("Failed to dequeue tasks", "worker_id", workerID, "error", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	s.logger.Debug("Processing tasks", "worker_id", workerID, "count", len(tasks))

	for _, task := range tasks {
		select {
		case <-s.stopChan:
			return
		default:
			s.processTask(task)
		}
	}
}

// processTask 处理单个任务
func (s *EnrichmentService) processTask(task *domainRAG.EnrichmentTask) {
	// 标记为处理中
	task.MarkProcessing()
	if err := s.enrichmentQueue.UpdateTask(task); err != nil {
		s.logger.Error("Failed to update task status", "chunk_id", task.ChunkID, "error", err)
		return
	}

	// 获取 chunk
	chunk, err := s.chunkRepo.GetChunk(task.ChunkID)
	if err != nil || chunk == nil {
		s.logger.Error("Failed to get chunk", "chunk_id", task.ChunkID, "error", err)
		task.MarkFailed("chunk not found")
		s.enrichmentQueue.UpdateTask(task)
		return
	}

	// 调用 LLM 进行总结
	enrichment, err := s.enrichChunk(chunk)
	if err != nil {
		s.logger.Warn("Failed to enrich chunk", "chunk_id", task.ChunkID, "error", err)
		task.MarkFailed(err.Error())
		s.enrichmentQueue.UpdateTask(task)

		// 更新 chunk 状态
		if task.Status == domainRAG.TaskStatusFailed {
			s.chunkRepo.UpdateChunkEnrichmentStatus(task.ChunkID, domainRAG.EnrichmentStatusFailed, err.Error())
		}
		return
	}

	// 更新 chunk 增强内容
	if err := s.chunkRepo.UpdateChunkEnrichment(task.ChunkID, enrichment); err != nil {
		s.logger.Error("Failed to update chunk enrichment", "chunk_id", task.ChunkID, "error", err)
		task.MarkFailed(err.Error())
		s.enrichmentQueue.UpdateTask(task)
		return
	}

	// 更新 Qdrant payload（添加 summary）
	if err := s.updateQdrantPayload(task.ChunkID, enrichment); err != nil {
		s.logger.Warn("Failed to update qdrant payload", "chunk_id", task.ChunkID, "error", err)
		// 不影响成功状态
	}

	// 标记任务完成
	task.MarkCompleted()
	if err := s.enrichmentQueue.UpdateTask(task); err != nil {
		s.logger.Error("Failed to update task status", "chunk_id", task.ChunkID, "error", err)
	}

	s.logger.Info("Chunk enriched successfully", "chunk_id", task.ChunkID)
}

// enrichChunk 对 chunk 进行 LLM 总结
func (s *EnrichmentService) enrichChunk(chunk *domainRAG.KnowledgeChunk) (*domainRAG.ChunkEnrichment, error) {
	// 调用 LLM 总结
	summary, err := s.llmClient.SummarizeTurn(chunk.UserQuery, chunk.AIResponseCore)
	if err != nil {
		return nil, err
	}

	return &domainRAG.ChunkEnrichment{
		Summary:   summary.Summary,
		MainTopic: summary.MainTopic,
		Tags:      summary.Tags,
	}, nil
}

// updateQdrantPayload 更新 Qdrant 中的 payload
func (s *EnrichmentService) updateQdrantPayload(chunkID string, enrichment *domainRAG.ChunkEnrichment) error {
	client := s.qdrantManager.GetClient()
	if client == nil {
		return nil // 客户端未初始化，跳过
	}

	ctx := context.Background()

	// 序列化 tags
	tagsJSON, _ := json.Marshal(enrichment.Tags)

	// 更新 payload
	_, err := client.SetPayload(ctx, &qdrant.SetPayloadPoints{
		CollectionName: "cursor_knowledge",
		Payload: qdrant.NewValueMap(map[string]interface{}{
			"summary":    enrichment.Summary,
			"main_topic": enrichment.MainTopic,
			"tags":       string(tagsJSON),
		}),
		PointsSelector: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{qdrant.NewID(chunkID)},
				},
			},
		},
	})

	return err
}

// GetQueueStats 获取队列统计
func (s *EnrichmentService) GetQueueStats() (*domainRAG.EnrichmentStats, error) {
	return s.enrichmentQueue.GetQueueStats()
}

// RetryFailed 重试失败的任务
func (s *EnrichmentService) RetryFailed() (int, error) {
	count, err := s.enrichmentQueue.ResetFailedTasks()
	if err != nil {
		return 0, err
	}

	s.logger.Info("Reset failed tasks", "count", count)
	return count, nil
}

// IsRunning 检查 Worker 是否运行中
func (s *EnrichmentService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isRunning
}

// SetWorkerCount 设置 Worker 数量（需要重启生效）
func (s *EnrichmentService) SetWorkerCount(count int) {
	if count < 1 {
		count = 1
	}
	if count > 10 {
		count = 10
	}
	s.workerCount = count
}

// SetPollInterval 设置轮询间隔
func (s *EnrichmentService) SetPollInterval(interval time.Duration) {
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	s.pollInterval = interval
}
