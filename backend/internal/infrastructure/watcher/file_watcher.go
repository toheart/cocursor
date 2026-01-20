package watcher

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cocursor/backend/internal/domain/events"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/fsnotify/fsnotify"
)

// WatchConfig FileWatcher 配置
type WatchConfig struct {
	// SessionDir 会话文件目录（~/.cursor/projects）
	SessionDir string
	// WorkspaceDir 工作区目录（workspaceStorage）
	WorkspaceDir string
	// DebounceDelay 防抖延迟
	DebounceDelay time.Duration
	// FullScanThreshold 全量扫描阈值（距上次扫描超过此时间则执行全量扫描）
	FullScanThreshold time.Duration
}

// DefaultWatchConfig 返回默认配置
func DefaultWatchConfig() WatchConfig {
	homeDir, _ := os.UserHomeDir()

	return WatchConfig{
		SessionDir:        filepath.Join(homeDir, ".cursor", "projects"),
		WorkspaceDir:      "", // 需要通过 PathResolver 获取
		DebounceDelay:     500 * time.Millisecond,
		FullScanThreshold: 24 * time.Hour,
	}
}

// FileWatcher 统一文件监听器
type FileWatcher struct {
	config   WatchConfig
	eventBus events.EventBus
	watcher  *fsnotify.Watcher
	logger   *slog.Logger

	// 防抖相关
	debounceTimers map[string]*time.Timer
	debounceMu     sync.Mutex

	// 控制
	stopCh chan struct{}
	wg     sync.WaitGroup

	// 扫描元数据
	metadata *ScanMetadata
}

// NewFileWatcher 创建文件监听器
func NewFileWatcher(config WatchConfig, eventBus events.EventBus) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &FileWatcher{
		config:         config,
		eventBus:       eventBus,
		watcher:        watcher,
		logger:         log.NewModuleLogger("watcher", "file_watcher"),
		debounceTimers: make(map[string]*time.Timer),
		stopCh:         make(chan struct{}),
		metadata:       NewScanMetadata(),
	}, nil
}

// Start 启动文件监听
func (fw *FileWatcher) Start() error {
	fw.logger.Info("Starting file watcher",
		"session_dir", fw.config.SessionDir,
		"workspace_dir", fw.config.WorkspaceDir,
	)

	// 检查是否需要全量扫描
	if fw.needsFullScan() {
		fw.logger.Info("Performing full scan on startup")
		fw.performFullScan()
	}

	// 添加监听目录
	if err := fw.addWatchDirs(); err != nil {
		return err
	}

	// 启动事件处理循环
	fw.wg.Add(1)
	go fw.watchLoop()

	return nil
}

// Stop 停止文件监听
func (fw *FileWatcher) Stop() {
	fw.logger.Info("Stopping file watcher")

	close(fw.stopCh)
	fw.watcher.Close()
	fw.wg.Wait()

	// 取消所有防抖定时器
	fw.debounceMu.Lock()
	for _, timer := range fw.debounceTimers {
		timer.Stop()
	}
	fw.debounceMu.Unlock()

	fw.logger.Info("File watcher stopped")
}

// needsFullScan 判断是否需要全量扫描
func (fw *FileWatcher) needsFullScan() bool {
	lastScan := fw.metadata.GetLastScanTime()

	// 从未扫描过
	if lastScan.IsZero() {
		fw.logger.Info("No previous scan found, full scan required")
		return true
	}

	// 距上次扫描超过阈值
	elapsed := time.Since(lastScan)
	if elapsed > fw.config.FullScanThreshold {
		fw.logger.Info("Last scan too old, full scan required",
			"last_scan", lastScan,
			"elapsed", elapsed,
			"threshold", fw.config.FullScanThreshold,
		)
		return true
	}

	fw.logger.Info("Recent scan found, skipping full scan",
		"last_scan", lastScan,
		"elapsed", elapsed,
	)
	return false
}

// performFullScan 执行全量扫描
func (fw *FileWatcher) performFullScan() {
	startTime := time.Now()

	// 扫描会话目录
	sessionCount := fw.scanSessionDirectory()

	// 扫描工作区目录
	workspaceCount := fw.scanWorkspaceDirectory()

	// 更新扫描时间
	fw.metadata.SetLastScanTime(time.Now())

	fw.logger.Info("Full scan completed",
		"sessions", sessionCount,
		"workspaces", workspaceCount,
		"duration", time.Since(startTime),
	)
}

// scanSessionDirectory 扫描会话目录
func (fw *FileWatcher) scanSessionDirectory() int {
	count := 0

	if fw.config.SessionDir == "" {
		return count
	}

	entries, err := os.ReadDir(fw.config.SessionDir)
	if err != nil {
		fw.logger.Error("Failed to read session directory", "error", err)
		return count
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectKey := entry.Name()
		transcriptsDir := filepath.Join(fw.config.SessionDir, projectKey, "agent-transcripts")

		// 检查目录是否存在
		if _, err := os.Stat(transcriptsDir); os.IsNotExist(err) {
			continue
		}

		// 扫描会话文件
		files, err := os.ReadDir(transcriptsDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".txt") {
				continue
			}

			sessionID := strings.TrimSuffix(file.Name(), ".txt")
			filePath := filepath.Join(transcriptsDir, file.Name())

			fileInfo, err := os.Stat(filePath)
			if err != nil {
				continue
			}

			// 发布 Created 事件
			fw.eventBus.Publish(&events.SessionFileEvent{
				EventType: events.SessionFileCreated,
				SessionID: sessionID,
				ProjectKey: projectKey,
				FilePath:   filePath,
				ModTime:    fileInfo.ModTime(),
				FileSize:   fileInfo.Size(),
				EventTime:  time.Now(),
			})
			count++
		}
	}

	return count
}

// scanWorkspaceDirectory 扫描工作区目录
func (fw *FileWatcher) scanWorkspaceDirectory() int {
	count := 0

	if fw.config.WorkspaceDir == "" {
		return count
	}

	entries, err := os.ReadDir(fw.config.WorkspaceDir)
	if err != nil {
		fw.logger.Error("Failed to read workspace directory", "error", err)
		return count
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workspaceID := entry.Name()
		projectPath := fw.getProjectPathFromWorkspace(workspaceID)

		fw.eventBus.Publish(&events.WorkspaceEvent{
			EventType:   events.WorkspaceCreated,
			WorkspaceID: workspaceID,
			ProjectPath: projectPath,
			EventTime:   time.Now(),
		})
		count++
	}

	return count
}

// getProjectPathFromWorkspace 从工作区目录获取项目路径
func (fw *FileWatcher) getProjectPathFromWorkspace(workspaceID string) string {
	workspaceJSONPath := filepath.Join(fw.config.WorkspaceDir, workspaceID, "workspace.json")

	data, err := os.ReadFile(workspaceJSONPath)
	if err != nil {
		return ""
	}

	var workspace struct {
		Folder string `json:"folder"`
	}
	if err := json.Unmarshal(data, &workspace); err != nil {
		return ""
	}

	return workspace.Folder
}

// addWatchDirs 添加监听目录
func (fw *FileWatcher) addWatchDirs() error {
	// 添加会话目录及其子目录
	if fw.config.SessionDir != "" {
		if err := fw.addDirRecursive(fw.config.SessionDir); err != nil {
			fw.logger.Warn("Failed to add session directory to watch", "error", err)
		}
	}

	// 添加工作区目录
	if fw.config.WorkspaceDir != "" {
		if err := fw.watcher.Add(fw.config.WorkspaceDir); err != nil {
			fw.logger.Warn("Failed to add workspace directory to watch", "error", err)
		}
	}

	return nil
}

// addDirRecursive 递归添加目录监听
func (fw *FileWatcher) addDirRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略无法访问的目录
		}

		if info.IsDir() {
			// 只监听 agent-transcripts 目录
			if strings.HasSuffix(path, "agent-transcripts") || path == dir {
				if err := fw.watcher.Add(path); err != nil {
					fw.logger.Debug("Failed to add directory to watch",
						"path", path,
						"error", err,
					)
				} else {
					fw.logger.Debug("Added directory to watch", "path", path)
				}
			}
		}
		return nil
	})
}

// watchLoop 事件监听循环
func (fw *FileWatcher) watchLoop() {
	defer fw.wg.Done()

	for {
		select {
		case <-fw.stopCh:
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleFsEvent(event)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			fw.logger.Error("Watcher error", "error", err)
		}
	}
}

// handleFsEvent 处理文件系统事件
func (fw *FileWatcher) handleFsEvent(event fsnotify.Event) {
	// 判断事件类型和目标
	if fw.isSessionFile(event.Name) {
		fw.handleSessionFileEvent(event)
	} else if fw.isWorkspaceDir(event.Name) {
		fw.handleWorkspaceEvent(event)
	} else if event.Has(fsnotify.Create) && fw.isAgentTranscriptsDir(event.Name) {
		// 新创建的 agent-transcripts 目录需要添加监听
		fw.watcher.Add(event.Name)
	}
}

// isSessionFile 判断是否为会话文件
func (fw *FileWatcher) isSessionFile(path string) bool {
	return strings.Contains(path, "agent-transcripts") && strings.HasSuffix(path, ".txt")
}

// isWorkspaceDir 判断是否为工作区目录
func (fw *FileWatcher) isWorkspaceDir(path string) bool {
	if fw.config.WorkspaceDir == "" {
		return false
	}
	return strings.HasPrefix(path, fw.config.WorkspaceDir) &&
		path != fw.config.WorkspaceDir
}

// isAgentTranscriptsDir 判断是否为 agent-transcripts 目录
func (fw *FileWatcher) isAgentTranscriptsDir(path string) bool {
	return strings.HasSuffix(path, "agent-transcripts")
}

// handleSessionFileEvent 处理会话文件事件（带防抖）
func (fw *FileWatcher) handleSessionFileEvent(fsEvent fsnotify.Event) {
	fw.debounceMu.Lock()
	defer fw.debounceMu.Unlock()

	// 取消之前的定时器
	if timer, exists := fw.debounceTimers[fsEvent.Name]; exists {
		timer.Stop()
	}

	// 创建新的防抖定时器
	fw.debounceTimers[fsEvent.Name] = time.AfterFunc(fw.config.DebounceDelay, func() {
		fw.emitSessionFileEvent(fsEvent)

		// 清理定时器
		fw.debounceMu.Lock()
		delete(fw.debounceTimers, fsEvent.Name)
		fw.debounceMu.Unlock()
	})
}

// emitSessionFileEvent 发送会话文件事件
func (fw *FileWatcher) emitSessionFileEvent(fsEvent fsnotify.Event) {
	// 解析路径获取 sessionID 和 projectKey
	sessionID, projectKey := fw.parseSessionFilePath(fsEvent.Name)
	if sessionID == "" {
		return
	}

	// 确定事件类型
	var eventType events.EventType
	switch {
	case fsEvent.Has(fsnotify.Create):
		eventType = events.SessionFileCreated
	case fsEvent.Has(fsnotify.Write):
		eventType = events.SessionFileModified
	case fsEvent.Has(fsnotify.Remove):
		eventType = events.SessionFileDeleted
	default:
		return
	}

	// 获取文件信息
	var modTime time.Time
	var fileSize int64
	if fileInfo, err := os.Stat(fsEvent.Name); err == nil {
		modTime = fileInfo.ModTime()
		fileSize = fileInfo.Size()
	}

	fw.eventBus.Publish(&events.SessionFileEvent{
		EventType:  eventType,
		SessionID:  sessionID,
		ProjectKey: projectKey,
		FilePath:   fsEvent.Name,
		ModTime:    modTime,
		FileSize:   fileSize,
		EventTime:  time.Now(),
	})

	fw.logger.Debug("Session file event emitted",
		"type", eventType,
		"session_id", sessionID,
		"project_key", projectKey,
	)
}

// parseSessionFilePath 解析会话文件路径
// 输入：/Users/.../projects/Users-xibaobao-code-cocursor/agent-transcripts/abc123.txt
// 输出：sessionID="abc123", projectKey="Users-xibaobao-code-cocursor"
func (fw *FileWatcher) parseSessionFilePath(path string) (sessionID, projectKey string) {
	// 获取文件名作为 sessionID
	fileName := filepath.Base(path)
	if !strings.HasSuffix(fileName, ".txt") {
		return "", ""
	}
	sessionID = strings.TrimSuffix(fileName, ".txt")

	// 获取 agent-transcripts 的父目录作为 projectKey
	dir := filepath.Dir(path)                     // .../agent-transcripts
	projectDir := filepath.Dir(dir)               // .../Users-xibaobao-code-cocursor
	projectKey = filepath.Base(projectDir)

	return sessionID, projectKey
}

// handleWorkspaceEvent 处理工作区事件
func (fw *FileWatcher) handleWorkspaceEvent(fsEvent fsnotify.Event) {
	if !fsEvent.Has(fsnotify.Create) {
		return
	}

	// 只处理目录创建
	info, err := os.Stat(fsEvent.Name)
	if err != nil || !info.IsDir() {
		return
	}

	workspaceID := filepath.Base(fsEvent.Name)
	projectPath := fw.getProjectPathFromWorkspace(workspaceID)

	fw.eventBus.Publish(&events.WorkspaceEvent{
		EventType:   events.WorkspaceCreated,
		WorkspaceID: workspaceID,
		ProjectPath: projectPath,
		EventTime:   time.Now(),
	})

	fw.logger.Debug("Workspace event emitted",
		"workspace_id", workspaceID,
		"project_path", projectPath,
	)
}
