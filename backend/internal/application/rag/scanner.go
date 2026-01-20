package rag

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"log/slog"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// ScanScheduler 扫描调度器（事件驱动 + 手动全量索引）
type ScanScheduler struct {
	chunkService    *ChunkService
	ragInitializer  *RAGInitializer // 用于延迟初始化
	projectManager  *appCursor.ProjectManager
	indexStatusRepo domainRAG.IndexStatusRepository
	config          *ScanConfig
	mu              sync.RWMutex
	stopChan        chan struct{}
	initialized     bool
	logger          *slog.Logger

	// 全量索引进度
	indexProgress *IndexProgress
	progressMu    sync.RWMutex
}

// ScanConfig 扫描配置（仅用于全量索引）
type ScanConfig struct {
	BatchSize   int // 每批处理的文件数
	Concurrency int // 并发处理数
}

// IndexProgress 索引进度
type IndexProgress struct {
	Status          string    `json:"status"`           // running, completed, failed, cancelled
	TotalFiles      int       `json:"total_files"`      // 总文件数
	ProcessedFiles  int       `json:"processed_files"`  // 已处理文件数
	IndexedMessages int       `json:"indexed_messages"` // 已索引消息数
	StartTime       time.Time `json:"start_time"`       // 开始时间
	ErrorMessage    string    `json:"error_message"`    // 错误信息（如果失败）
}

// ProgressCallback 进度回调函数
type ProgressCallback func(progress *IndexProgress)

// NewScanScheduler 创建扫描调度器
func NewScanScheduler(
	chunkService *ChunkService,
	projectManager *appCursor.ProjectManager,
	indexStatusRepo domainRAG.IndexStatusRepository,
	config *ScanConfig,
) *ScanScheduler {
	// 设置默认值
	if config == nil {
		config = &ScanConfig{
			BatchSize:   10,
			Concurrency: 3,
		}
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 10
	}
	if config.Concurrency <= 0 {
		config.Concurrency = 3
	}

	return &ScanScheduler{
		chunkService:    chunkService,
		projectManager:  projectManager,
		indexStatusRepo: indexStatusRepo,
		config:          config,
		stopChan:        make(chan struct{}),
		initialized:     false,
		logger:          log.NewModuleLogger("rag", "scanner"),
	}
}

// SetRAGInitializer 设置 RAG 初始化器（用于延迟初始化）
func (s *ScanScheduler) SetRAGInitializer(initializer *RAGInitializer) {
	s.ragInitializer = initializer
}

// Start 启动扫描调度器（现在只是初始化，不再有定时扫描）
func (s *ScanScheduler) Start() error {
	s.logger.Info("ScanScheduler started (event-driven mode)")
	return nil
}

// Stop 停止扫描调度器
func (s *ScanScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	close(s.stopChan)
	s.logger.Info("ScanScheduler stopped")

	return nil
}

// UpdateConfig 更新配置（仅更新 BatchSize 和 Concurrency）
func (s *ScanScheduler) UpdateConfig(config *ScanConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config.BatchSize > 0 {
		s.config.BatchSize = config.BatchSize
	}
	if config.Concurrency > 0 {
		s.config.Concurrency = config.Concurrency
	}

	// 尝试初始化 chunkService（如果还没有）
	if s.chunkService == nil && s.ragInitializer != nil {
		chunkService := s.ragInitializer.GetChunkService()
		if chunkService != nil {
			s.chunkService = chunkService
			s.initialized = true
			s.logger.Info("ChunkService initialized via UpdateConfig")
		}
	}
}

// scanAllFiles 扫描所有会话文件
func (s *ScanScheduler) scanAllFiles() []*FileToUpdate {
	var allFiles []*FileToUpdate

	homeDir, err := os.UserHomeDir()
	if err != nil {
		s.logger.Error("Failed to get home directory", "error", err)
		return allFiles
	}

	projectsDir := filepath.Join(homeDir, ".cursor", "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		s.logger.Error("Failed to read projects directory", "error", err)
		return allFiles
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectKey := entry.Name()
		transcriptsDir := filepath.Join(projectsDir, projectKey, "agent-transcripts")

		// 检查目录是否存在
		if _, err := os.Stat(transcriptsDir); os.IsNotExist(err) {
			continue
		}

		// 获取项目信息
		project := s.projectManager.GetProject(projectKey)
		var projectInfo *ProjectInfo
		if project == nil {
			projectInfo = &ProjectInfo{
				ProjectID:   projectKey,
				ProjectName: projectKey,
				WorkspaceID: "",
			}
		} else {
			projectInfo = s.toProjectInfo(project)
		}

		// 扫描会话文件
		transcriptFiles, err := os.ReadDir(transcriptsDir)
		if err != nil {
			continue
		}

		for _, file := range transcriptFiles {
			if !strings.HasSuffix(file.Name(), ".txt") {
				continue
			}

			sessionID := strings.TrimSuffix(file.Name(), ".txt")
			filePath := filepath.Join(transcriptsDir, file.Name())

			allFiles = append(allFiles, &FileToUpdate{
				SessionID:   sessionID,
				FilePath:    filePath,
				ProjectInfo: projectInfo,
			})
		}
	}

	return allFiles
}

// FileToUpdate 需要更新的文件
type FileToUpdate struct {
	SessionID   string
	FilePath    string
	ProjectInfo *ProjectInfo
}

// processBatchWithProgress 批量处理文件（带进度更新）
func (s *ScanScheduler) processBatchWithProgress(files []*FileToUpdate, callback ProgressCallback) {
	if len(files) == 0 {
		return
	}

	batchSize := s.config.BatchSize
	concurrency := s.config.Concurrency

	// 分批处理
	for i := 0; i < len(files); i += batchSize {
		// 检查是否取消
		select {
		case <-s.stopChan:
			return
		default:
		}

		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}

		batch := files[i:end]
		indexedCount := s.processBatch(batch, concurrency)

		// 更新进度
		s.progressMu.Lock()
		if s.indexProgress != nil {
			s.indexProgress.ProcessedFiles = end
			s.indexProgress.IndexedMessages += indexedCount
			if callback != nil {
				callback(s.indexProgress)
			}
		}
		s.progressMu.Unlock()
	}
}

// processBatch 处理一批文件，返回索引的消息数
func (s *ScanScheduler) processBatch(batch []*FileToUpdate, concurrency int) int {
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var indexedCount int
	var countMu sync.Mutex

	for _, file := range batch {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量

		go func(f *FileToUpdate) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			// 确保 chunkService 已初始化
			if s.chunkService == nil && s.ragInitializer != nil && !s.initialized {
				chunkService := s.ragInitializer.GetChunkService()
				if chunkService != nil {
					s.chunkService = chunkService
					s.initialized = true
				}
			}

			// 如果仍然没有 chunkService，跳过
			if s.chunkService == nil {
				return
			}

			// 检查内容哈希（精确检测）
			if s.needsReindex(f.SessionID, f.FilePath) {
				count, err := s.chunkService.IndexSessionWithCount(f.SessionID, f.FilePath)
				if err != nil {
					s.logger.Error("Failed to index session",
						"session_id", f.SessionID,
						"file_path", f.FilePath,
						"error", err,
					)
				} else {
					countMu.Lock()
					indexedCount += count
					countMu.Unlock()
				}
			} else {
				// 只更新文件修改时间
				fileInfo, _ := os.Stat(f.FilePath)
				if fileInfo != nil {
					s.indexStatusRepo.UpdateFileMtime(f.FilePath, fileInfo.ModTime().Unix())
				}
			}
		}(file)
	}

	wg.Wait()
	return indexedCount
}

// needsReindex 检查是否需要重新索引（精确检测：检查内容哈希）
func (s *ScanScheduler) needsReindex(sessionID, filePath string) bool {
	status, err := s.indexStatusRepo.GetIndexStatus(filePath)
	if err != nil || status == nil {
		return true // 没有状态，需要索引
	}

	// 计算当前文件哈希
	currentHash := s.calculateFileHash(filePath)
	if currentHash == "" {
		return true // 无法计算哈希，重新索引
	}

	// 比较内容哈希
	return currentHash != status.ContentHash
}

// calculateFileHash 计算文件内容哈希
func (s *ScanScheduler) calculateFileHash(filePath string) string {
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

// toProjectInfo 转换为 ProjectInfo
func (s *ScanScheduler) toProjectInfo(project *domainCursor.ProjectInfo) *ProjectInfo {
	if project == nil {
		return &ProjectInfo{
			ProjectID:   "unknown",
			ProjectName: "Unknown",
			WorkspaceID: "",
		}
	}

	workspaceID := ""
	if len(project.Workspaces) > 0 {
		workspaceID = project.Workspaces[0].WorkspaceID
	}

	return &ProjectInfo{
		ProjectID:   project.ProjectID,
		ProjectName: project.ProjectName,
		WorkspaceID: workspaceID,
	}
}

// TriggerFullScan 触发全量扫描（带进度回调）
func (s *ScanScheduler) TriggerFullScan(chunkService *ChunkService, batchSize, concurrency int, callback ProgressCallback) error {
	s.logger.Info("Triggering full RAG index",
		"batch_size", batchSize,
		"concurrency", concurrency,
	)

	// 更新 chunkService（如果传入）
	if chunkService != nil {
		s.chunkService = chunkService
		s.initialized = true
	}

	// 检查 chunkService 是否可用
	if s.chunkService == nil {
		if s.ragInitializer != nil {
			s.chunkService = s.ragInitializer.GetChunkService()
			if s.chunkService != nil {
				s.initialized = true
			}
		}
	}
	if s.chunkService == nil {
		return fmt.Errorf("RAG service not initialized, please configure RAG first")
	}

	// 更新配置
	if batchSize > 0 {
		s.config.BatchSize = batchSize
	}
	if concurrency > 0 {
		s.config.Concurrency = concurrency
	}

	// 异步执行全量扫描
	go s.runFullScanWithProgress(callback)

	return nil
}

// runFullScanWithProgress 执行全量扫描（带进度）
func (s *ScanScheduler) runFullScanWithProgress(callback ProgressCallback) {
	// 扫描所有文件
	allFiles := s.scanAllFiles()

	// 初始化进度
	s.progressMu.Lock()
	s.indexProgress = &IndexProgress{
		Status:          "running",
		TotalFiles:      len(allFiles),
		ProcessedFiles:  0,
		IndexedMessages: 0,
		StartTime:       time.Now(),
	}
	if callback != nil {
		callback(s.indexProgress)
	}
	s.progressMu.Unlock()

	s.logger.Info("Full scan started", "total_files", len(allFiles))

	// 批量处理
	s.processBatchWithProgress(allFiles, callback)

	// 完成
	s.progressMu.Lock()
	if s.indexProgress != nil {
		s.indexProgress.Status = "completed"
		if callback != nil {
			callback(s.indexProgress)
		}
	}
	s.progressMu.Unlock()

	s.logger.Info("Full scan completed",
		"total_files", len(allFiles),
		"indexed_messages", s.indexProgress.IndexedMessages,
	)
}

// GetProgress 获取当前索引进度
func (s *ScanScheduler) GetProgress() *IndexProgress {
	s.progressMu.RLock()
	defer s.progressMu.RUnlock()

	if s.indexProgress == nil {
		return nil
	}

	// 返回副本
	return &IndexProgress{
		Status:          s.indexProgress.Status,
		TotalFiles:      s.indexProgress.TotalFiles,
		ProcessedFiles:  s.indexProgress.ProcessedFiles,
		IndexedMessages: s.indexProgress.IndexedMessages,
		StartTime:       s.indexProgress.StartTime,
		ErrorMessage:    s.indexProgress.ErrorMessage,
	}
}

// IsRunning 检查是否正在运行全量索引
func (s *ScanScheduler) IsRunning() bool {
	s.progressMu.RLock()
	defer s.progressMu.RUnlock()

	return s.indexProgress != nil && s.indexProgress.Status == "running"
}

// ClearMetadata 清空元数据
func (s *ScanScheduler) ClearMetadata() error {
	s.logger.Info("Clearing RAG metadata")
	return s.indexStatusRepo.ClearAllStatus()
}

// ===== 事件驱动接口 =====
// 以下方法实现 events.Handler 接口，用于接收 FileWatcher 的事件

// HandleEvent 实现 events.Handler 接口
// 处理会话文件变更事件
func (s *ScanScheduler) HandleEvent(event interface{}) error {
	// 类型断言为 SessionFileEvent
	sessionEvent, ok := event.(interface {
		Type() interface{}
		SessionID() string
		ProjectKey() string
		FilePath() string
	})

	if !ok {
		return nil
	}

	return s.HandleSessionFileEvent(
		sessionEvent.SessionID(),
		sessionEvent.ProjectKey(),
		sessionEvent.FilePath(),
	)
}

// HandleSessionFileEvent 处理会话文件事件
// 这是事件驱动模式的入口，由 FileWatcher 触发
func (s *ScanScheduler) HandleSessionFileEvent(sessionID, projectKey, filePath string) error {
	s.logger.Debug("Handling session file event",
		"session_id", sessionID,
		"project_key", projectKey,
		"file_path", filePath,
	)

	// 确保 chunkService 已初始化
	if s.chunkService == nil && s.ragInitializer != nil && !s.initialized {
		chunkService := s.ragInitializer.GetChunkService()
		if chunkService != nil {
			s.chunkService = chunkService
			s.initialized = true
		}
	}

	// 如果仍然没有 chunkService，跳过
	if s.chunkService == nil {
		s.logger.Debug("ChunkService not initialized, skipping event")
		return nil
	}

	// 检查是否需要重新索引
	if s.needsReindex(sessionID, filePath) {
		if err := s.chunkService.IndexSession(sessionID, filePath); err != nil {
			s.logger.Error("Failed to index session from event",
				"session_id", sessionID,
				"file_path", filePath,
				"error", err,
			)
			return err
		}
		s.logger.Info("Session indexed from event",
			"session_id", sessionID,
		)
	} else {
		// 只更新文件修改时间
		fileInfo, _ := os.Stat(filePath)
		if fileInfo != nil {
			s.indexStatusRepo.UpdateFileMtime(filePath, fileInfo.ModTime().Unix())
		}
	}

	return nil
}
