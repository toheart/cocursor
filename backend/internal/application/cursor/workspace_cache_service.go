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
	projectManager    *ProjectManager
	pathResolver      *infraCursor.PathResolver
	dbReader          *infraCursor.DBReader
	tiktokenEstimator *infraCursor.TiktokenEstimator

	sessionRepo  storage.WorkspaceSessionRepository
	metadataRepo storage.WorkspaceFileMetadataRepository

	mu             sync.RWMutex
	syncInProgress map[string]bool

	// 面板到会话的映射缓存（key: workspaceID, value: map[panelID]composerID）
	panelMappingCache   map[string]map[string]string
	panelMappingCacheMu sync.RWMutex

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewWorkspaceCacheService 创建工作区缓存服务实例（接受 Repository 作为参数）
func NewWorkspaceCacheService(
	projectManager *ProjectManager,
	sessionRepo storage.WorkspaceSessionRepository,
	metadataRepo storage.WorkspaceFileMetadataRepository,
) *WorkspaceCacheService {
	// 初始化 tiktoken 估算器
	estimator, err := infraCursor.GetTiktokenEstimator()
	if err != nil {
		log.Printf("[WorkspaceCacheService] tiktoken 初始化失败，将使用字符估算: %v", err)
	}

	service := &WorkspaceCacheService{
		projectManager:    projectManager,
		pathResolver:      infraCursor.NewPathResolver(),
		dbReader:          infraCursor.NewDBReader(),
		tiktokenEstimator: estimator,
		sessionRepo:       sessionRepo,
		metadataRepo:      metadataRepo,
		syncInProgress:    make(map[string]bool),
		panelMappingCache: make(map[string]map[string]string),
		stopCh:            make(chan struct{}),
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
	s.wg.Add(3)
	go s.startPeriodicSync()
	go s.startPeriodicScan()
	go s.startRuntimeStateSync()

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

		// 计算会话的 Token 数量（基于会话名称和文件引用的估算）
		// 注意：完整的 Token 计算需要读取 transcript 文件，但这里使用简化的估算
		// 基于会话时长和复杂度进行估算
		tokenCount := s.estimateSessionTokens(&composer)

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
			TokenCount:          tokenCount,
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

		// 检查 folder 字段是否为空（无效工作区配置，跳过）
		if workspace.Folder == "" {
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

// estimateSessionTokens 估算会话的 Token 数量
// 基于会话时长、代码变更量和文件引用进行估算
// 这是一个简化的估算方法，避免读取完整的 transcript 文件
func (s *WorkspaceCacheService) estimateSessionTokens(composer *domainCursor.ComposerData) int {
	// 基础 Token 估算公式：
	// 1. 会话时长贡献：每分钟约 500 Token（用户输入 + AI 回复）
	// 2. 代码变更贡献：每行变更约 10 Token
	// 3. 文件引用贡献：每个文件约 200 Token（文件名 + 上下文）

	// 计算会话时长（分钟）
	durationMs := composer.LastUpdatedAt - composer.CreatedAt
	durationMinutes := float64(durationMs) / 1000.0 / 60.0
	if durationMinutes < 1 {
		durationMinutes = 1 // 最少 1 分钟
	}
	if durationMinutes > 120 {
		durationMinutes = 120 // 最多 2 小时（避免异常值）
	}

	// 时长贡献
	durationTokens := int(durationMinutes * 500)

	// 代码变更贡献
	codeChangeTokens := (composer.TotalLinesAdded + composer.TotalLinesRemoved) * 10

	// 文件引用贡献
	fileCount := composer.FilesChangedCount
	if fileCount == 0 && composer.Subtitle != "" {
		// 从 Subtitle 统计文件数（逗号分隔）
		fileCount = strings.Count(composer.Subtitle, ",") + 1
	}
	fileTokens := fileCount * 200

	// 总 Token 估算
	totalTokens := durationTokens + codeChangeTokens + fileTokens

	// 上下文使用率修正：上下文使用率越高，Token 消耗越多
	if composer.ContextUsagePercent > 0 {
		contextMultiplier := 1.0 + (composer.ContextUsagePercent / 100.0)
		totalTokens = int(float64(totalTokens) * contextMultiplier)
	}

	// 最小值保护
	if totalTokens < 100 {
		totalTokens = 100
	}

	return totalTokens
}

// ===== 事件驱动接口 =====
// 以下方法用于接收 FileWatcher 的事件

// HandleWorkspaceEvent 处理工作区事件
// 这是事件驱动模式的入口，由 FileWatcher 触发
func (s *WorkspaceCacheService) HandleWorkspaceEvent(workspaceID, projectPath string) error {
	log.Printf("[WorkspaceCacheService] Handling workspace event: workspace_id=%s, project_path=%s", workspaceID, projectPath)

	// 检查 ProjectManager 中是否已存在
	if s.projectManager.HasWorkspace(workspaceID) {
		log.Printf("[WorkspaceCacheService] Workspace already registered: %s", workspaceID)
		return nil
	}

	// 如果 projectPath 是 file:// URI，需要解析
	folderPath := projectPath
	if strings.HasPrefix(projectPath, "file://") {
		var err error
		folderPath, err = s.parseFolderURI(projectPath)
		if err != nil {
			log.Printf("[WorkspaceCacheService] Failed to parse folder URI: %v", err)
			return err
		}
	}

	// 如果路径为空，尝试从 workspace.json 获取
	if folderPath == "" {
		workspaceDir, err := s.pathResolver.GetWorkspaceStorageDir()
		if err != nil {
			return err
		}

		workspaceJSONPath := filepath.Join(workspaceDir, workspaceID, "workspace.json")
		data, err := os.ReadFile(workspaceJSONPath)
		if err != nil {
			log.Printf("[WorkspaceCacheService] Failed to read workspace.json: %v", err)
			return nil // 不是错误，可能只是工作区还没准备好
		}

		var workspace struct {
			Folder string `json:"folder"`
		}
		if err := json.Unmarshal(data, &workspace); err != nil {
			return nil
		}

		if workspace.Folder == "" {
			return nil
		}

		folderPath, err = s.parseFolderURI(workspace.Folder)
		if err != nil {
			return nil
		}
	}

	// 注册工作区
	_, err := s.projectManager.RegisterWorkspace(folderPath)
	if err != nil {
		log.Printf("[WorkspaceCacheService] Failed to register workspace from event: %v", err)
		return err
	}

	log.Printf("[WorkspaceCacheService] Workspace registered from event: %s -> %s", workspaceID, folderPath)
	return nil
}

// ===== 运行时状态扫描 =====

// startRuntimeStateSync 启动运行时状态同步任务（每1分钟）
func (s *WorkspaceCacheService) startRuntimeStateSync() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// 首次启动时立即执行一次
	s.syncAllRuntimeStates()

	for {
		select {
		case <-ticker.C:
			s.syncAllRuntimeStates()
		case <-s.stopCh:
			return
		}
	}
}

// syncAllRuntimeStates 同步所有工作区的运行时状态
func (s *WorkspaceCacheService) syncAllRuntimeStates() {
	projects := s.projectManager.ListAllProjects()

	for _, project := range projects {
		for _, ws := range project.Workspaces {
			if err := s.SyncRuntimeState(ws.WorkspaceID); err != nil {
				log.Printf("[WorkspaceCacheService] failed to sync runtime state for workspace %s: %v", ws.WorkspaceID, err)
			}
		}
	}
}

// SyncRuntimeState 同步单个工作区的运行时状态
func (s *WorkspaceCacheService) SyncRuntimeState(workspaceID string) error {
	// 获取工作区数据库路径
	workspaceDBPath, err := s.pathResolver.GetWorkspaceDBPath(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get workspace DB path: %w", err)
	}

	// 检查文件是否存在
	if _, err := os.Stat(workspaceDBPath); os.IsNotExist(err) {
		return nil // 文件不存在，跳过
	}

	// 读取面板可见性状态
	visiblePanels, err := s.readVisiblePanels(workspaceDBPath)
	if err != nil {
		log.Printf("[WorkspaceCacheService] failed to read visible panels for %s: %v", workspaceID, err)
		// 继续执行，使用空的可见面板列表
		visiblePanels = make(map[string]bool)
	}

	// 读取当前聚焦面板
	focusedPanelID, err := s.readFocusedPanel(workspaceDBPath)
	if err != nil {
		log.Printf("[WorkspaceCacheService] failed to read focused panel for %s: %v", workspaceID, err)
		focusedPanelID = ""
	}

	// 获取或更新面板映射
	panelMapping, err := s.getPanelMapping(workspaceID, workspaceDBPath, visiblePanels)
	if err != nil {
		log.Printf("[WorkspaceCacheService] failed to get panel mapping for %s: %v", workspaceID, err)
		panelMapping = make(map[string]string)
	}

	// 先重置所有会话的运行时状态
	if err := s.sessionRepo.ResetRuntimeState(workspaceID); err != nil {
		return fmt.Errorf("failed to reset runtime state: %w", err)
	}

	// 构建运行时状态更新
	updates := make([]*storage.RuntimeStateUpdate, 0)

	// 收集会话可见性（一个 composerID 可能对应多个 panelID）
	composerVisibility := make(map[string]struct {
		isVisible bool
		isFocused bool
		panelID   string
	})

	for panelID, composerID := range panelMapping {
		if composerID == "" {
			continue
		}

		isVisible := visiblePanels[panelID]
		isFocused := panelID == focusedPanelID

		existing, ok := composerVisibility[composerID]
		if !ok {
			composerVisibility[composerID] = struct {
				isVisible bool
				isFocused bool
				panelID   string
			}{
				isVisible: isVisible,
				isFocused: isFocused,
				panelID:   panelID,
			}
		} else {
			// 合并：任意一个 visible 即为 visible，任意一个 focused 即为 focused
			composerVisibility[composerID] = struct {
				isVisible bool
				isFocused bool
				panelID   string
			}{
				isVisible: existing.isVisible || isVisible,
				isFocused: existing.isFocused || isFocused,
				panelID:   panelID, // 使用最新的 panelID
			}
		}
	}

	// 生成更新列表
	for composerID, state := range composerVisibility {
		activeLevel := domainCursor.CalculateActiveLevel(false, state.isVisible, state.isFocused)

		updates = append(updates, &storage.RuntimeStateUpdate{
			ComposerID:  composerID,
			IsVisible:   state.isVisible,
			IsFocused:   state.isFocused,
			ActiveLevel: activeLevel,
			PanelID:     state.panelID,
		})
	}

	// 批量更新运行时状态
	if len(updates) > 0 {
		if err := s.sessionRepo.UpdateRuntimeState(workspaceID, updates); err != nil {
			return fmt.Errorf("failed to update runtime state: %w", err)
		}
	}

	return nil
}

// readVisiblePanels 读取可见面板列表
func (s *WorkspaceCacheService) readVisiblePanels(workspaceDBPath string) (map[string]bool, error) {
	result := make(map[string]bool)

	// 读取 workbench.auxiliarybar.viewContainersWorkspaceState
	value, err := s.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "workbench.auxiliarybar.viewContainersWorkspaceState")
	if err != nil {
		// key 不存在是正常情况
		if strings.Contains(err.Error(), "key not found") {
			return result, nil
		}
		return nil, err
	}

	// 解析 JSON 数组
	var panels []struct {
		ID      string `json:"id"`
		Visible bool   `json:"visible"`
	}

	if err := json.Unmarshal(value, &panels); err != nil {
		return nil, fmt.Errorf("failed to parse viewContainersWorkspaceState: %w", err)
	}

	for _, panel := range panels {
		if panel.Visible {
			// 提取面板 ID（去掉 workbench.panel.aichat. 前缀）
			panelID := panel.ID
			if strings.HasPrefix(panelID, "workbench.panel.aichat.") {
				panelID = strings.TrimPrefix(panelID, "workbench.panel.aichat.")
			}
			result[panelID] = true
		}
	}

	return result, nil
}

// readFocusedPanel 读取当前聚焦面板 ID
func (s *WorkspaceCacheService) readFocusedPanel(workspaceDBPath string) (string, error) {
	value, err := s.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "workbench.auxiliarybar.activepanelid")
	if err != nil {
		if strings.Contains(err.Error(), "key not found") {
			return "", nil
		}
		return "", err
	}

	// 值是一个字符串，可能带引号
	panelID := strings.Trim(string(value), "\"")

	// 提取面板 ID（去掉 workbench.panel.aichat. 前缀）
	if strings.HasPrefix(panelID, "workbench.panel.aichat.") {
		panelID = strings.TrimPrefix(panelID, "workbench.panel.aichat.")
	}

	return panelID, nil
}

// getPanelMapping 获取面板到会话的映射
func (s *WorkspaceCacheService) getPanelMapping(workspaceID, workspaceDBPath string, visiblePanels map[string]bool) (map[string]string, error) {
	s.panelMappingCacheMu.RLock()
	cached, ok := s.panelMappingCache[workspaceID]
	s.panelMappingCacheMu.RUnlock()

	if !ok {
		cached = make(map[string]string)
	}

	// 检查可见面板是否都有映射，如果缺失则读取
	needUpdate := false
	for panelID := range visiblePanels {
		if _, exists := cached[panelID]; !exists {
			needUpdate = true
			break
		}
	}

	if needUpdate {
		// 读取所有面板映射
		newMapping, err := s.readAllPanelMappings(workspaceDBPath)
		if err != nil {
			log.Printf("[WorkspaceCacheService] failed to read panel mappings: %v", err)
		} else {
			// 合并到缓存
			for k, v := range newMapping {
				cached[k] = v
			}

			s.panelMappingCacheMu.Lock()
			s.panelMappingCache[workspaceID] = cached
			s.panelMappingCacheMu.Unlock()
		}
	}

	return cached, nil
}

// readAllPanelMappings 读取所有面板到会话的映射
func (s *WorkspaceCacheService) readAllPanelMappings(workspaceDBPath string) (map[string]string, error) {
	result := make(map[string]string)

	// 读取所有以 workbench.panel.composerChatViewPane. 开头的 key
	keys, err := s.dbReader.ReadKeysWithPrefixFromWorkspaceDB(workspaceDBPath, "workbench.panel.composerChatViewPane.")
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		value, err := s.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, key)
		if err != nil {
			continue
		}

		// 解析面板 ID
		panelID := strings.TrimPrefix(key, "workbench.panel.composerChatViewPane.")

		// 值格式是 JSON: {"workbench.panel.aichat.view.{composerId}": {...}}
		// 解析 JSON 获取第一个 key，从中提取 composerID
		var panelState map[string]interface{}
		if err := json.Unmarshal(value, &panelState); err != nil {
			// 如果不是 JSON，尝试旧的字符串格式
			valueStr := strings.Trim(string(value), "\"")
			if strings.HasPrefix(valueStr, "workbench.panel.aichat.view.") {
				composerID := strings.TrimPrefix(valueStr, "workbench.panel.aichat.view.")
				result[panelID] = composerID
			}
			continue
		}

		// 从 JSON 的 key 中提取 composerID
		for viewKey := range panelState {
			if strings.HasPrefix(viewKey, "workbench.panel.aichat.view.") {
				composerID := strings.TrimPrefix(viewKey, "workbench.panel.aichat.view.")
				result[panelID] = composerID
				break // 只取第一个
			}
		}
	}

	return result, nil
}

// GetActiveSessionsOverview 获取活跃会话概览
func (s *WorkspaceCacheService) GetActiveSessionsOverview(workspaceID string) (*domainCursor.ActiveSessionsOverview, error) {
	// 查询活跃会话
	activeSessions, err := s.sessionRepo.FindActiveByWorkspaceID(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find active sessions: %w", err)
	}

	// 查询所有会话以获取统计信息
	allSessions, err := s.sessionRepo.FindByWorkspaceID(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find all sessions: %w", err)
	}

	// 统计关闭和归档数量
	closedCount := 0
	archivedCount := 0
	for _, session := range allSessions {
		switch session.ActiveLevel {
		case domainCursor.ActiveLevelClosed:
			closedCount++
		case domainCursor.ActiveLevelArchived:
			archivedCount++
		}
	}

	// 构建返回结果
	overview := &domainCursor.ActiveSessionsOverview{
		ClosedCount:   closedCount,
		ArchivedCount: archivedCount,
		OpenSessions:  make([]*domainCursor.ActiveSession, 0),
	}

	for _, session := range activeSessions {
		// 计算熵值
		entropy := domainCursor.CalculateEntropy(
			session.TotalLinesAdded,
			session.TotalLinesRemoved,
			session.FilesChangedCount,
			session.ContextUsagePercent,
		)

		// 计算健康状态
		status, warning := domainCursor.CalculateHealthStatus(entropy, session.ContextUsagePercent)

		activeSession := &domainCursor.ActiveSession{
			ComposerID:          session.ComposerID,
			Name:                session.Name,
			Entropy:             entropy,
			ContextUsagePercent: session.ContextUsagePercent,
			Status:              status,
			Warning:             warning,
			LastUpdatedAt:       session.LastUpdatedAt,
		}

		if session.IsFocused {
			overview.Focused = activeSession
		} else {
			overview.OpenSessions = append(overview.OpenSessions, activeSession)
		}
	}

	return overview, nil
}
