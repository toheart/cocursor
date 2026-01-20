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

// ScanScheduler 扫描调度器
type ScanScheduler struct {
	chunkService    *ChunkService
	ragInitializer  *RAGInitializer // 用于延迟初始化
	projectManager  *appCursor.ProjectManager
	indexStatusRepo domainRAG.IndexStatusRepository
	config          *ScanConfig
	mu              sync.RWMutex
	stopChan        chan struct{}
	ticker          *time.Ticker
	initialized     bool
	logger          *slog.Logger
}

// ScanConfig 扫描配置
type ScanConfig struct {
	Enabled     bool
	Interval    time.Duration
	BatchSize   int
	Concurrency int
}

// NewScanScheduler 创建扫描调度器
func NewScanScheduler(
	chunkService *ChunkService,
	projectManager *appCursor.ProjectManager,
	indexStatusRepo domainRAG.IndexStatusRepository,
	config *ScanConfig,
) *ScanScheduler {
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

// Start 启动扫描调度器
func (s *ScanScheduler) Start() error {
	if !s.config.Enabled {
		return nil
	}

	// 如果 chunkService 为 nil，尝试通过 initializer 初始化
	if s.chunkService == nil && s.ragInitializer != nil && !s.initialized {
		chunkService := s.ragInitializer.GetChunkService()
		if chunkService != nil {
			s.chunkService = chunkService
			s.initialized = true
		} else {
			// 初始化失败，禁用调度器
			s.config.Enabled = false
			return nil
		}
	}

	// 如果仍然没有 chunkService，不启动
	if s.chunkService == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 不再启动时自动全量扫描，改为手动触发
	// 只启动定时扫描（如果配置了 interval）
	if s.config.Interval > 0 {
		s.ticker = time.NewTicker(s.config.Interval)
		go s.runPeriodicScan()
	}

	return nil
}

// Stop 停止扫描调度器
func (s *ScanScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopChan)

	return nil
}

// runPeriodicScan 运行定时扫描
func (s *ScanScheduler) runPeriodicScan() {
	for {
		select {
		case <-s.ticker.C:
			s.scanOnce()
		case <-s.stopChan:
			return
		}
	}
}

// scanOnce 执行一次扫描
func (s *ScanScheduler) scanOnce() {
	// 阶段 1: 快速扫描（只读文件信息）
	filesToUpdate := s.phase1QuickScan()

	// 阶段 2: 批量处理（只处理更新的文件）
	s.phase2BatchProcess(filesToUpdate)
}

// phase1QuickScan 阶段 1: 快速扫描（只读文件信息）
func (s *ScanScheduler) phase1QuickScan() []*FileToUpdate {
	var filesToUpdate []*FileToUpdate

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filesToUpdate
	}

	projectsDir := filepath.Join(homeDir, ".cursor", "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return filesToUpdate
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
			// 如果项目不存在，使用 projectKey 作为 ProjectID
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

			fileInfo, err := os.Stat(filePath)
			if err != nil {
				continue
			}

			// 检查是否需要更新
			if s.needsUpdate(sessionID, filePath, fileInfo.ModTime()) {
				filesToUpdate = append(filesToUpdate, &FileToUpdate{
					SessionID:   sessionID,
					FilePath:    filePath,
					ProjectInfo: projectInfo,
				})
			}
		}
	}

	return filesToUpdate
}

// FileToUpdate 需要更新的文件
type FileToUpdate struct {
	SessionID   string
	FilePath    string
	ProjectInfo *ProjectInfo
}

// phase2BatchProcess 阶段 2: 批量处理
func (s *ScanScheduler) phase2BatchProcess(filesToUpdate []*FileToUpdate) {
	if len(filesToUpdate) == 0 {
		return
	}

	// 使用简单的并发控制（不使用 conc 库，保持依赖简单）
	batchSize := s.config.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}

	concurrency := s.config.Concurrency
	if concurrency <= 0 {
		concurrency = 3
	}

	// 分批处理
	for i := 0; i < len(filesToUpdate); i += batchSize {
		end := i + batchSize
		if end > len(filesToUpdate) {
			end = len(filesToUpdate)
		}

		batch := filesToUpdate[i:end]
		s.processBatch(batch, concurrency)
	}
}

// processBatch 处理一批文件
func (s *ScanScheduler) processBatch(batch []*FileToUpdate, concurrency int) {
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

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
				err := s.chunkService.IndexSession(f.SessionID, f.FilePath)
				if err != nil {
					s.logger.Error("Failed to index session",
						"session_id", f.SessionID,
						"file_path", f.FilePath,
						"error", err,
					)
				}
			} else {
				// 只更新文件修改时间（使用 indexStatusRepo）
				fileInfo, _ := os.Stat(f.FilePath)
				if fileInfo != nil {
					s.indexStatusRepo.UpdateFileMtime(f.FilePath, fileInfo.ModTime().Unix())
				}
			}
		}(file)
	}

	wg.Wait()
}

// needsUpdate 检查文件是否需要更新（快速检测：只检查 mtime）
func (s *ScanScheduler) needsUpdate(sessionID, filePath string, fileMtime time.Time) bool {
	status, err := s.indexStatusRepo.GetIndexStatus(filePath)
	if err != nil || status == nil {
		return true // 没有状态，需要索引
	}

	// 比较文件修改时间
	return fileMtime.Unix() > status.FileMtime
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

// TriggerScan 手动触发扫描
func (s *ScanScheduler) TriggerScan() {
	go s.scanOnce()
}

// UpdateConfig 更新扫描配置
func (s *ScanScheduler) UpdateConfig(config *ScanConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldEnabled := s.config.Enabled
	s.config = config

	// 如果从禁用变为启用，启动调度器
	if config.Enabled && !oldEnabled {
		// 尝试初始化服务
		if s.chunkService == nil && s.ragInitializer != nil && !s.initialized {
			chunkService := s.ragInitializer.GetChunkService()
			if chunkService != nil {
				s.chunkService = chunkService
				s.initialized = true
			} else {
				// 初始化失败，禁用调度器
				s.config.Enabled = false
				return
			}
		}

		// 如果有 chunkService，启动定时器
		if s.chunkService != nil && config.Interval > 0 {
			if s.ticker != nil {
				s.ticker.Stop()
			}
			s.ticker = time.NewTicker(config.Interval)
			go s.runPeriodicScan()
		}
	}

	// 如果从启用变为禁用，停止调度器
	if !config.Enabled && oldEnabled {
		if s.ticker != nil {
			s.ticker.Stop()
			s.ticker = nil
		}
	}

	// 如果间隔时间变化，重启定时器
	if config.Enabled && oldEnabled && config.Interval != s.config.Interval {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		s.ticker = time.NewTicker(config.Interval)
		go s.runPeriodicScan()
	}
}

// TriggerFullScan 触发全量扫描
func (s *ScanScheduler) TriggerFullScan(chunkService *ChunkService) {
	s.logger.Info("Triggering full RAG index")
	// 更新 chunkService（如果传入）
	if chunkService != nil {
		s.chunkService = chunkService
	}
	// 执行全量扫描（覆盖 needsUpdate 逻辑）
	go s.runFullScan()
}

// runFullScan 执行全量扫描
func (s *ScanScheduler) runFullScan() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		s.logger.Error("Failed to get home directory", "error", err)
		return
	}

	projectsDir := filepath.Join(homeDir, ".cursor", "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		s.logger.Error("Failed to read projects directory", "error", err)
		return
	}

	var allFiles []*FileToUpdate
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

		// 扫描所有会话文件（不考虑 mtime）
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
			
			// 全量扫描时，添加所有文件
			allFiles = append(allFiles, &FileToUpdate{
				SessionID:   sessionID,
				FilePath:    filePath,
				ProjectInfo: projectInfo,
			})
		}
	}

	s.logger.Info("Full scan found files", "count", len(allFiles))
	s.phase2BatchProcess(allFiles)
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
