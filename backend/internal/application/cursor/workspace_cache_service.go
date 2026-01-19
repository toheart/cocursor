package cursor

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/cocursor/backend/internal/infrastructure/storage"
)

// WorkspaceCacheService 工作区缓存服务
type WorkspaceCacheService struct {
	projectManager *ProjectManager
	pathResolver   *infraCursor.PathResolver
	dbReader       *infraCursor.DBReader

	sessionRepo  storage.WorkspaceSessionRepository
	metadataRepo storage.WorkspaceFileMetadataRepository

	mu             sync.RWMutex
	syncInProgress map[string]bool

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewWorkspaceCacheService 创建工作区缓存服务实例（接受 Repository 作为参数）
func NewWorkspaceCacheService(
	projectManager *ProjectManager,
	sessionRepo storage.WorkspaceSessionRepository,
	metadataRepo storage.WorkspaceFileMetadataRepository,
) *WorkspaceCacheService {
	service := &WorkspaceCacheService{
		projectManager: projectManager,
		pathResolver:   infraCursor.NewPathResolver(),
		dbReader:       infraCursor.NewDBReader(),
		sessionRepo:    sessionRepo,
		metadataRepo:   metadataRepo,
		syncInProgress: make(map[string]bool),
		stopCh:         make(chan struct{}),
	}

	// 注册回调
	projectManager.RegisterWorkspaceChangeCallback(service.onWorkspaceChange)

	return service
}

// Start 启动缓存服务
func (s *WorkspaceCacheService) Start() error {
	log.Println("[WorkspaceCacheService] 启动工作区缓存服务...")

	// 首次全局扫描
	if err := s.performInitialScan(); err != nil {
		log.Printf("[WorkspaceCacheService] 首次全局扫描失败: %v", err)
		// 不阻止启动，继续执行
	}

	// 启动定时任务
	s.wg.Add(2)
	go s.startPeriodicSync()
	go s.startPeriodicScan()

	log.Println("[WorkspaceCacheService] 工作区缓存服务启动完成")
	return nil
}

// Stop 停止缓存服务
func (s *WorkspaceCacheService) Stop() error {
	log.Println("[WorkspaceCacheService] 停止工作区缓存服务...")

	// 关闭 stopCh 通知所有 goroutine 停止
	close(s.stopCh)

	// 等待所有 goroutine 完成
	s.wg.Wait()

	log.Println("[WorkspaceCacheService] 工作区缓存服务已停止")
	return nil
}

// onWorkspaceChange 工作区变化回调处理
func (s *WorkspaceCacheService) onWorkspaceChange(workspaceID string, action string) {
	switch action {
	case "added":
		// 新工作区，立即同步
		go s.SyncWorkspace(workspaceID)
	case "updated":
		// 工作区更新，检查是否需要同步
		go s.CheckAndSyncWorkspace(workspaceID)
	}
}

// CheckAndSyncWorkspace 检查并同步工作区（如果文件有改动）
func (s *WorkspaceCacheService) CheckAndSyncWorkspace(workspaceID string) error {
	// 获取工作区数据库路径
	workspaceDBPath, err := s.pathResolver.GetWorkspaceDBPath(workspaceID)
	if err != nil {
		return err
	}

	// 检查文件信息
	fileInfo, err := os.Stat(workspaceDBPath)
	if err != nil {
		return err
	}

	currentMtime := fileInfo.ModTime().Unix()
	currentSize := fileInfo.Size()

	// 查询元数据
	metadata, err := s.metadataRepo.FindByWorkspaceID(workspaceID)
	if err != nil || metadata == nil {
		// 没有元数据，需要同步
		return s.SyncWorkspace(workspaceID)
	}

	// 检查文件是否改动
	if metadata.FileMtime != currentMtime || metadata.FileSize != currentSize {
		// 文件有改动，执行同步
		return s.SyncWorkspace(workspaceID)
	}

	return nil
}

// SyncWorkspace 同步工作区数据（增量同步）
func (s *WorkspaceCacheService) SyncWorkspace(workspaceID string) error {
	// 防止重复同步
	s.mu.Lock()
	if s.syncInProgress[workspaceID] {
		s.mu.Unlock()
		return nil // 已在同步中，跳过
	}
	s.syncInProgress[workspaceID] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.syncInProgress, workspaceID)
		s.mu.Unlock()
	}()

	// 获取工作区数据库路径
	workspaceDBPath, err := s.pathResolver.GetWorkspaceDBPath(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get workspace DB path: %w", err)
	}

	// 检查文件改动
	fileInfo, err := os.Stat(workspaceDBPath)
	if err != nil {
		return fmt.Errorf("workspace DB not found: %w", err)
	}

	currentMtime := fileInfo.ModTime().Unix()
	currentSize := fileInfo.Size()

	// 读取或创建 metadata
	metadata, err := s.metadataRepo.FindByWorkspaceID(workspaceID)
	if err != nil || metadata == nil {
		// 创建新的 metadata
		metadata = &storage.WorkspaceFileMetadata{
			WorkspaceID:  workspaceID,
			DBPath:       workspaceDBPath,
			FileMtime:    currentMtime,
			FileSize:     currentSize,
			LastScanTime: time.Now().Unix(),
			CreatedAt:    time.Now().Unix(),
			UpdatedAt:    time.Now().Unix(),
		}
	}

	// 检查是否需要同步
	if metadata.LastSyncTime != nil {
		// 检查文件是否改动
		if metadata.FileMtime == currentMtime && metadata.FileSize == currentSize {
			// 文件未改动，跳过
			return nil
		}
	}

	// 读取 Cursor 数据库
	composerDataValue, err := s.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
	if err != nil {
		// key 不存在是正常情况（工作区可能从未使用过 Composer），记录信息日志并更新 metadata
		if strings.Contains(err.Error(), "key not found") {
			log.Printf("[WorkspaceCacheService] workspace %s has no composer data, skipping", workspaceID)
			// 更新 metadata，标记为已扫描但没有会话数据
			now := time.Now().Unix()
			metadata.FileMtime = currentMtime
			metadata.FileSize = currentSize
			metadata.LastScanTime = now
			syncTime := now
			metadata.LastSyncTime = &syncTime
			metadata.SessionsCount = 0
			metadata.UpdatedAt = now
			if err := s.metadataRepo.Save(metadata); err != nil {
				return fmt.Errorf("failed to update metadata: %w", err)
			}
			return nil // 正常情况，不返回错误
		}
		// 其他错误（如数据库读取失败）才返回错误
		return fmt.Errorf("failed to read composer data: %w", err)
	}

	// 解析 JSON
	composers, err := domainCursor.ParseComposerData(string(composerDataValue))
	if err != nil {
		return fmt.Errorf("failed to parse composer data: %w", err)
	}

	// 判断是否需要全量同步
	// 如果 metadata 不存在或 LastSyncTime 为 nil，说明是首次扫描，需要全量同步
	isFullSync := metadata.LastSyncTime == nil

	// 同步会话数据（全量或增量）
	if err := s.syncSessions(workspaceID, composers, isFullSync); err != nil {
		return fmt.Errorf("failed to sync sessions: %w", err)
	}

	// 更新 metadata
	now := time.Now().Unix()
	metadata.FileMtime = currentMtime
	metadata.FileSize = currentSize
	metadata.LastScanTime = now
	syncTime := now
	metadata.LastSyncTime = &syncTime
	metadata.SessionsCount = len(composers)
	metadata.UpdatedAt = now

	if err := s.metadataRepo.Save(metadata); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return nil
}

// syncSessions 同步会话数据（全量或增量）
// workspaceID: 工作区 ID
// composers: 从 Cursor 数据库读取的会话列表
// isFullSync: 是否全量同步（true=全量，false=增量）
func (s *WorkspaceCacheService) syncSessions(workspaceID string, composers []domainCursor.ComposerData, isFullSync bool) error {
	now := time.Now().UnixMilli()

	// 全量同步：先删除该工作区的所有旧数据
	if isFullSync {
		log.Printf("[WorkspaceCacheService] performing full sync for workspace %s, %d sessions", workspaceID, len(composers))
		// 注意：这里不删除数据，而是通过 INSERT OR REPLACE 来更新
		// 如果 Cursor 数据库中已删除的会话，会在后续的清理逻辑中处理
	} else {
		log.Printf("[WorkspaceCacheService] performing incremental sync for workspace %s, %d sessions", workspaceID, len(composers))
	}

	// 查询已缓存的 composer_id 列表（用于增量同步时的跳过逻辑）
	var cachedMap map[string]bool
	if !isFullSync {
		cachedIDs, err := s.sessionRepo.GetCachedComposerIDs(workspaceID)
		if err != nil {
			return fmt.Errorf("failed to get cached composer IDs: %w", err)
		}

		cachedMap = make(map[string]bool)
		for _, id := range cachedIDs {
			cachedMap[id] = true
		}
	}

	// 处理每个会话
	syncedCount := 0
	skippedCount := 0
	for _, composer := range composers {
		// 增量同步时，检查是否需要更新
		if !isFullSync && cachedMap[composer.ComposerID] {
			// 已存在，检查是否需要更新
			existing, err := s.sessionRepo.FindByWorkspaceIDAndComposerID(workspaceID, composer.ComposerID)
			if err != nil {
				log.Printf("[WorkspaceCacheService] failed to query existing session: %v", err)
				continue
			}
			if existing != nil && existing.LastUpdatedAt == composer.LastUpdatedAt {
				// last_updated_at 相同，跳过
				skippedCount++
				continue
			}
		}

		// 插入或更新会话
		session := &storage.WorkspaceSession{
			WorkspaceID:         workspaceID,
			ComposerID:          composer.ComposerID,
			Name:                composer.Name,
			Type:                composer.Type,
			CreatedAt:           composer.CreatedAt,
			LastUpdatedAt:       composer.LastUpdatedAt,
			UnifiedMode:         composer.UnifiedMode,
			Subtitle:            composer.Subtitle,
			TotalLinesAdded:     composer.TotalLinesAdded,
			TotalLinesRemoved:   composer.TotalLinesRemoved,
			FilesChangedCount:   composer.FilesChangedCount,
			ContextUsagePercent: composer.ContextUsagePercent,
			IsArchived:          composer.IsArchived,
			CreatedOnBranch:     composer.CreatedOnBranch,
			CachedAt:            now,
		}

		if err := s.sessionRepo.Save(session); err != nil {
			log.Printf("[WorkspaceCacheService] failed to save session %s: %v", composer.ComposerID, err)
			continue
		}
		syncedCount++
	}

	log.Printf("[WorkspaceCacheService] workspace %s sync completed: synced=%d, skipped=%d, total=%d",
		workspaceID, syncedCount, skippedCount, len(composers))

	return nil
}

// performInitialScan 执行首次全局扫描
func (s *WorkspaceCacheService) performInitialScan() error {
	log.Println("[WorkspaceCacheService] 开始首次全局扫描...")

	// 从 ProjectManager 获取所有工作区
	projects := s.projectManager.ListAllProjects()
	workspaceCount := 0

	for _, project := range projects {
		for _, ws := range project.Workspaces {
			workspaceCount++
			// 检查是否需要同步（忽略错误，继续处理其他工作区）
			if err := s.CheckAndSyncWorkspace(ws.WorkspaceID); err != nil {
				log.Printf("[WorkspaceCacheService] failed to sync workspace %s: %v", ws.WorkspaceID, err)
				// 继续处理下一个工作区，不中断扫描
			}
		}
	}

	log.Printf("[WorkspaceCacheService] 首次全局扫描完成，处理了 %d 个工作区", workspaceCount)
	return nil
}

// startPeriodicSync 启动定时同步任务（每5分钟）
func (s *WorkspaceCacheService) startPeriodicSync() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performPeriodicSync()
		case <-s.stopCh:
			return
		}
	}
}

// performPeriodicSync 执行定时同步（检查文件改动）
func (s *WorkspaceCacheService) performPeriodicSync() {
	// 从 ProjectManager 获取活跃工作区
	projects := s.projectManager.ListAllProjects()

	for _, project := range projects {
		for _, ws := range project.Workspaces {
			// 检查并同步（如果文件有改动）
			s.CheckAndSyncWorkspace(ws.WorkspaceID)
		}
	}
}

// startPeriodicScan 启动定时扫描任务（每5分钟，发现新工作区）
func (s *WorkspaceCacheService) startPeriodicScan() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.scanForNewWorkspaces()
		case <-s.stopCh:
			return
		}
	}
}

// scanForNewWorkspaces 扫描新工作区
func (s *WorkspaceCacheService) scanForNewWorkspaces() {
	// 扫描 workspaceStorage 目录
	workspaceDir, err := s.pathResolver.GetWorkspaceStorageDir()
	if err != nil {
		return
	}

	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return
	}

	// 查询已缓存的工作区列表
	cachedWorkspaceIDs, err := s.metadataRepo.FindAllWorkspaceIDs()
	if err != nil {
		return
	}

	cachedMap := make(map[string]bool)
	for _, id := range cachedWorkspaceIDs {
		cachedMap[id] = true
	}

	// 对比，找出新工作区
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workspaceID := entry.Name()

		// 检查是否已缓存
		if cachedMap[workspaceID] {
			continue
		}

		// 检查 ProjectManager 中是否已存在
		if s.projectManager.HasWorkspace(workspaceID) {
			continue
		}

		// 检查 workspace.json 是否存在
		workspaceJSONPath := filepath.Join(workspaceDir, workspaceID, "workspace.json")
		data, err := os.ReadFile(workspaceJSONPath)
		if err != nil {
			continue
		}

		// 解析 workspace.json
		var workspace struct {
			Folder string `json:"folder"`
		}
		if err := json.Unmarshal(data, &workspace); err != nil {
			continue
		}

		// 检查 folder 字段是否为空
		if workspace.Folder == "" {
			log.Printf("[WorkspaceCacheService] workspace %s has empty folder field, skipping", workspaceID)
			continue
		}

		// 解析 folder URI 为文件系统路径
		folderPath, err := s.parseFolderURI(workspace.Folder)
		if err != nil {
			log.Printf("[WorkspaceCacheService] failed to parse folder URI for workspace %s: %v", workspaceID, err)
			continue
		}

		// 调用 ProjectManager.RegisterWorkspace（会触发回调）
		_, err = s.projectManager.RegisterWorkspace(folderPath)
		if err != nil {
			log.Printf("[WorkspaceCacheService] failed to register workspace %s: %v", workspaceID, err)
		}
	}
}

// parseFolderURI 解析 folder URI 为文件系统路径（与 PathResolver 逻辑相同）
func (s *WorkspaceCacheService) parseFolderURI(uri string) (string, error) {
	// 检查 URI 是否为空
	if uri == "" {
		return "", fmt.Errorf("empty URI")
	}

	// 解析 URI
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("failed to parse URI: %w", err)
	}

	// 检查协议
	if parsedURL.Scheme == "" {
		return "", fmt.Errorf("missing scheme in URI: %s", uri)
	}
	if parsedURL.Scheme != "file" {
		return "", fmt.Errorf("unsupported scheme: %s", parsedURL.Scheme)
	}

	// 获取路径部分
	path := parsedURL.Path

	// 手动处理 URL 编码
	decodedPath, err := url.PathUnescape(path)
	if err != nil {
		decodedPath = path
	}

	// 区分 Windows 和 Unix 路径
	if len(decodedPath) > 2 && decodedPath[1] == ':' {
		// Windows 路径: 移除开头的斜杠
		if len(decodedPath) > 0 && decodedPath[0] == '/' {
			decodedPath = decodedPath[1:]
		}
	}

	// 转换为系统路径格式
	systemPath := filepath.FromSlash(decodedPath)

	// 清理路径：移除开头的单个反斜杠（仅 Windows，且非 UNC 路径）
	if runtime.GOOS == "windows" && len(systemPath) > 0 && systemPath[0] == '\\' {
		if len(systemPath) <= 1 || systemPath[1] != '\\' {
			systemPath = systemPath[1:]
		}
	}

	return systemPath, nil
}
