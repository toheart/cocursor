package cursor

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
)

// WorkspaceState 工作区运行时状态
type WorkspaceState struct {
	WorkspaceID   string    // 工作区 ID
	Path          string    // 项目路径
	LastHeartbeat time.Time // 最后心跳时间
	LastFocus     time.Time // 最后获得焦点时间
}

// WorkspaceChangeCallback 工作区变化回调函数类型
// action: "added" - 新工作区, "updated" - 工作区更新, "deleted" - 工作区删除
type WorkspaceChangeCallback func(workspaceID string, action string)

// ProjectManager 项目管理器（内存缓存）
type ProjectManager struct {
	mu                       sync.RWMutex
	projects                 map[string]*domainCursor.ProjectInfo // projectKey (Git URL 或目录名) -> *ProjectInfo
	displayNameMap           map[string]string                    // displayName -> projectKey (用于通过显示名称查找)
	pathMap                  map[string]string                    // normalized path -> projectKey
	workspaceStates          map[string]*WorkspaceState           // workspaceID -> 运行时状态
	activeWorkspaceID        string                               // 当前活跃工作区 ID
	discovery                *ProjectDiscovery
	matcher                  *infraCursor.PathMatcher
	pathResolver             *infraCursor.PathResolver
	workspaceChangeCallbacks []WorkspaceChangeCallback // 工作区变化回调列表
	callbackMu               sync.RWMutex              // 回调列表的锁
}

// NewProjectManager 创建项目管理器实例
func NewProjectManager() *ProjectManager {
	return &ProjectManager{
		projects:                 make(map[string]*domainCursor.ProjectInfo),
		displayNameMap:           make(map[string]string),
		pathMap:                  make(map[string]string),
		workspaceStates:          make(map[string]*WorkspaceState),
		workspaceChangeCallbacks: make([]WorkspaceChangeCallback, 0),
		discovery:                NewProjectDiscovery(),
		matcher:                  infraCursor.NewPathMatcher(),
		pathResolver:             infraCursor.NewPathResolver(),
	}
}

// Start 启动项目管理器（扫描所有工作区并分组）
func (pm *ProjectManager) Start() error {
	log.Println("开始扫描 Cursor 工作区...")

	// 扫描所有工作区
	workspaces, err := pm.discovery.ScanAllWorkspaces()
	if err != nil {
		return fmt.Errorf("failed to scan workspaces: %w", err)
	}

	log.Printf("发现 %d 个工作区", len(workspaces))

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 按"同一项目"规则分组
	projectGroups := pm.groupBySameProject(workspaces)

	// 保存到内存
	for projectKey, projectInfo := range projectGroups {
		pm.projects[projectKey] = projectInfo

		// 更新显示名称映射（支持通过显示名称查找）
		pm.displayNameMap[projectInfo.ProjectName] = projectKey

		// 更新路径映射（路径 -> projectKey）
		// 初始化工作区运行时状态
		for _, ws := range projectInfo.Workspaces {
			normalizedPath, _ := pm.normalizePath(ws.Path)
			pm.pathMap[normalizedPath] = projectKey

			// 初始化工作区运行时状态
			now := time.Now()
			pm.workspaceStates[ws.WorkspaceID] = &WorkspaceState{
				WorkspaceID:   ws.WorkspaceID,
				Path:          ws.Path,
				LastHeartbeat: now,
				LastFocus:     now,
			}
		}
	}

	// 设置初始活跃工作区（选择第一个项目的主工作区）
	if len(projectGroups) > 0 {
		for _, project := range projectGroups {
			for _, ws := range project.Workspaces {
				if ws.IsPrimary {
					pm.activeWorkspaceID = ws.WorkspaceID
					break
				}
			}
			if pm.activeWorkspaceID != "" {
				break
			}
		}
		// 如果没有主工作区，选择第一个工作区
		if pm.activeWorkspaceID == "" {
			for _, project := range projectGroups {
				if len(project.Workspaces) > 0 {
					pm.activeWorkspaceID = project.Workspaces[0].WorkspaceID
					break
				}
			}
		}
	}

	log.Printf("项目管理器启动完成: 发现 %d 个项目, %d 个工作区", len(pm.projects), len(workspaces))

	return nil
}

// groupBySameProject 按"同一项目"规则分组
func (pm *ProjectManager) groupBySameProject(workspaces []*DiscoveredWorkspace) map[string]*domainCursor.ProjectInfo {
	groups := make(map[string]*domainCursor.ProjectInfo)
	processed := make(map[string]bool)

	for _, ws := range workspaces {
		if processed[ws.WorkspaceID] {
			continue
		}

		// 查找所有属于同一项目的工作区
		sameProject := pm.findSameProject(ws, workspaces)

		// 生成项目唯一标识符（用于分组和查找）
		projectKey := pm.generateProjectKey(sameProject)
		// 生成项目显示名称（用于展示，更友好）
		projectDisplayName := pm.generateProjectDisplayName(sameProject)

		// 创建或更新 ProjectInfo
		if existing, exists := groups[projectKey]; exists {
			// 已存在，添加新的工作区
			for _, sp := range sameProject {
				existing.Workspaces = append(existing.Workspaces, pm.toWorkspaceInfo(sp, projectDisplayName))
			}
			existing.LastUpdatedAt = time.Now()

			// 重新判断哪个是主工作区（最新的）
			pm.updatePrimaryWorkspace(existing)
		} else {
			// 新项目，创建 ProjectInfo
			workspaceInfos := make([]*domainCursor.WorkspaceInfo, 0, len(sameProject))
			for _, sp := range sameProject {
				workspaceInfos = append(workspaceInfos, pm.toWorkspaceInfo(sp, projectDisplayName))
			}

			groups[projectKey] = &domainCursor.ProjectInfo{
				ProjectName:   projectDisplayName, // 使用友好的显示名称
				ProjectID:     projectKey,         // 使用唯一标识符作为 ID
				Workspaces:    workspaceInfos,
				GitRemoteURL:  sameProject[0].GitRemoteURL,
				GitBranch:     sameProject[0].GitBranch,
				CreatedAt:     time.Now(),
				LastUpdatedAt: time.Now(),
			}

			// 设置主工作区
			pm.updatePrimaryWorkspace(groups[projectKey])
		}

		// 标记已处理
		for _, s := range sameProject {
			processed[s.WorkspaceID] = true
		}
	}

	return groups
}

// findSameProject 查找所有属于同一项目的工作区
func (pm *ProjectManager) findSameProject(ws *DiscoveredWorkspace, all []*DiscoveredWorkspace) []*DiscoveredWorkspace {
	var sameProject []*DiscoveredWorkspace
	sameProject = append(sameProject, ws)

	for _, other := range all {
		if other.WorkspaceID == ws.WorkspaceID {
			continue
		}

		if pm.isSameProject(ws, other) {
			sameProject = append(sameProject, other)
		}
	}

	return sameProject
}

// isSameProject 判断两个工作区是否属于同一项目
// 优先级：P0 (Git URL) > P1 (物理路径) > P2 (项目名 + 路径相似度)
func (pm *ProjectManager) isSameProject(ws1, ws2 *DiscoveredWorkspace) bool {
	// P0: Git 远程 URL 相同
	if ws1.GitRemoteURL != "" && ws2.GitRemoteURL != "" {
		if ws1.GitRemoteURL == ws2.GitRemoteURL {
			return true
		}
	}

	// P1: 物理路径完全相同（解析符号链接）
	realPath1, err1 := filepath.EvalSymlinks(ws1.Path)
	realPath2, err2 := filepath.EvalSymlinks(ws2.Path)
	if err1 == nil && err2 == nil {
		norm1, _ := pm.normalizePath(realPath1)
		norm2, _ := pm.normalizePath(realPath2)
		if norm1 == norm2 {
			return true
		}
	}

	// P2: 项目名相同 + 路径相似度 > 90%
	if ws1.ProjectName == ws2.ProjectName {
		similarity := pm.matcher.CalculatePathSimilarity(ws1.Path, ws2.Path)
		if similarity > 0.9 {
			return true
		}
	}

	return false
}

// generateProjectKey 生成项目唯一标识符（用于分组和查找）
// 优先级：Git URL > 项目名
func (pm *ProjectManager) generateProjectKey(workspaces []*DiscoveredWorkspace) string {
	primary := workspaces[0]

	// 优先级：Git URL > 项目名
	if primary.GitRemoteURL != "" {
		return primary.GitRemoteURL
	}

	return primary.ProjectName
}

// generateProjectDisplayName 生成项目显示名称（用于展示）
// 优先级：从 Git URL 提取仓库名 > 目录名
func (pm *ProjectManager) generateProjectDisplayName(workspaces []*DiscoveredWorkspace) string {
	primary := workspaces[0]

	// 如果有 Git URL，尝试提取仓库名
	if primary.GitRemoteURL != "" {
		repoName := pm.extractRepoNameFromURL(primary.GitRemoteURL)
		if repoName != "" {
			return repoName
		}
	}

	// 否则使用目录名
	return primary.ProjectName
}

// extractRepoNameFromURL 从 Git URL 提取仓库名
// 例如：https://github.com/toheart/cocursor -> cocursor
//
//	https://github.com/nsqio/nsq -> nsq
func (pm *ProjectManager) extractRepoNameFromURL(gitURL string) string {
	// 移除协议前缀
	url := gitURL
	if strings.Contains(url, "://") {
		parts := strings.SplitN(url, "://", 2)
		if len(parts) == 2 {
			url = parts[1]
		}
	}

	// 移除域名部分，获取路径
	// https://github.com/user/repo -> user/repo
	// git@github.com:user/repo -> user/repo
	if strings.Contains(url, "/") {
		// 找到第一个 / 或 : 之后的部分
		idx := strings.Index(url, "/")
		if idx == -1 {
			idx = strings.Index(url, ":")
		}
		if idx >= 0 && idx < len(url)-1 {
			path := url[idx+1:]
			// 移除末尾的 .git
			path = strings.TrimSuffix(path, ".git")
			// 获取最后一部分（仓库名）
			parts := strings.Split(path, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
		}
	}

	return ""
}

// toWorkspaceInfo 转换为 WorkspaceInfo
// projectName 是项目的统一名称（可能是 Git URL 或项目名）
func (pm *ProjectManager) toWorkspaceInfo(ws *DiscoveredWorkspace, projectName string) *domainCursor.WorkspaceInfo {
	return &domainCursor.WorkspaceInfo{
		WorkspaceID:  ws.WorkspaceID,
		Path:         ws.Path,
		ProjectName:  projectName, // 使用项目的统一名称，而不是工作区的本地项目名
		GitRemoteURL: ws.GitRemoteURL,
		GitBranch:    ws.GitBranch,
		IsActive:     false,
		IsPrimary:    false,
	}
}

// updatePrimaryWorkspace 更新主工作区（最新的）
func (pm *ProjectManager) updatePrimaryWorkspace(project *domainCursor.ProjectInfo) {
	if len(project.Workspaces) == 0 {
		return
	}

	// 重置所有主工作区标记
	for _, ws := range project.Workspaces {
		ws.IsPrimary = false
	}

	// 设置第一个为主工作区（可以后续优化为按时间戳选择最新的）
	if len(project.Workspaces) > 0 {
		project.Workspaces[0].IsPrimary = true
	}
}

// normalizePath 规范化路径
func (pm *ProjectManager) normalizePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	normalized := filepath.ToSlash(absPath)
	normalized = filepath.Clean(normalized)

	return normalized, nil
}

// GetProject 根据项目名获取项目信息
// 支持通过显示名称（如 "cocursor"）或 Git URL（如 "https://github.com/toheart/cocursor"）查找
func (pm *ProjectManager) GetProject(projectName string) *domainCursor.ProjectInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 先尝试直接查找（可能是 Git URL 或目录名）
	if project, exists := pm.projects[projectName]; exists {
		return project
	}

	// 如果不存在，尝试通过显示名称查找
	if projectKey, exists := pm.displayNameMap[projectName]; exists {
		return pm.projects[projectKey]
	}

	return nil
}

// FindByPath 根据路径查找项目
// 返回：项目显示名称和工作区信息
func (pm *ProjectManager) FindByPath(path string) (string, *domainCursor.WorkspaceInfo) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	normalizedPath, err := pm.normalizePath(path)
	if err != nil {
		return "", nil
	}

	// 查找路径映射（pathMap 存储的是 projectKey）
	projectKey, exists := pm.pathMap[normalizedPath]
	if !exists {
		return "", nil
	}

	project := pm.projects[projectKey]
	if project == nil {
		return "", nil
	}

	// 查找匹配的工作区
	for _, ws := range project.Workspaces {
		wsNorm, _ := pm.normalizePath(ws.Path)
		if wsNorm == normalizedPath {
			// 返回项目显示名称（而不是 projectKey）
			return project.ProjectName, ws
		}
	}

	// 返回项目显示名称（而不是 projectKey）
	return project.ProjectName, nil
}

// MarkWorkspaceActive 标记工作区为活跃
func (pm *ProjectManager) MarkWorkspaceActive(workspaceID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 重置所有工作区的活跃状态
	for _, project := range pm.projects {
		for _, ws := range project.Workspaces {
			ws.IsActive = (ws.WorkspaceID == workspaceID)
		}
	}
}

// ListAllProjects 列出所有项目
func (pm *ProjectManager) ListAllProjects() []*domainCursor.ProjectInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	projects := make([]*domainCursor.ProjectInfo, 0, len(pm.projects))
	for _, project := range pm.projects {
		projects = append(projects, project)
	}

	return projects
}

// RegisterWorkspace 注册工作区并更新项目分组
// 如果工作区已存在，只更新运行时状态；如果不存在，触发增量扫描和分组
func (pm *ProjectManager) RegisterWorkspace(path string) (*WorkspaceState, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 通过路径计算 WorkspaceID
	workspaceID, err := pm.pathResolver.GetWorkspaceIDByPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace ID from path: %w", err)
	}

	// 检查工作区是否已存在
	state, exists := pm.workspaceStates[workspaceID]
	now := time.Now()

	if exists {
		// 已存在，只更新心跳时间
		state.LastHeartbeat = now
		if state.Path != path {
			state.Path = path
		}
		return state, nil
	}

	// 工作区不存在，需要增量扫描
	// 先创建临时状态
	state = &WorkspaceState{
		WorkspaceID:   workspaceID,
		Path:          path,
		LastHeartbeat: now,
		LastFocus:     now,
	}
	pm.workspaceStates[workspaceID] = state

	// 如果这是第一个工作区，设置为活跃
	if pm.activeWorkspaceID == "" {
		pm.activeWorkspaceID = workspaceID
	}

	// 触发增量扫描（重新扫描所有工作区并更新分组）
	// 注意：这里在锁内执行，但 refreshWorkspaceUnlocked 会重新获取锁，所以需要先解锁
	pm.mu.Unlock()
	err = pm.refreshWorkspaceUnlocked(path, workspaceID)
	pm.mu.Lock()

	if err != nil {
		// 扫描失败，移除临时状态
		delete(pm.workspaceStates, workspaceID)
		return nil, fmt.Errorf("failed to refresh workspace: %w", err)
	}

	// 触发回调通知新工作区（异步执行，不阻塞）
	go pm.notifyWorkspaceChange(workspaceID, "added")

	return state, nil
}

// UpdateWorkspaceFocus 更新工作区焦点
// workspaceID: 工作区 ID，如果为空则通过 path 查找
// path: 项目路径（可选，如果提供了 workspaceID 则不需要）
func (pm *ProjectManager) UpdateWorkspaceFocus(workspaceID string, path string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var targetWorkspaceID string
	var state *WorkspaceState

	if workspaceID != "" {
		// 直接使用提供的 workspaceID
		targetWorkspaceID = workspaceID
		state, _ = pm.workspaceStates[targetWorkspaceID]
	} else if path != "" {
		// 通过路径查找 workspaceID
		id, err := pm.pathResolver.GetWorkspaceIDByPath(path)
		if err != nil {
			// 如果找不到工作区 ID，尝试注册（可能是新工作区）
			log.Printf("[ProjectManager.UpdateWorkspaceFocus] 通过路径查找工作区 ID 失败，尝试注册: path=%s, error=%v", path, err)
			pm.mu.Unlock()
			registeredState, registerErr := pm.RegisterWorkspace(path)
			pm.mu.Lock()
			if registerErr != nil {
				return fmt.Errorf("failed to get workspace ID from path and register failed: %w (register error: %v)", err, registerErr)
			}
			targetWorkspaceID = registeredState.WorkspaceID
			state = registeredState
		} else {
			targetWorkspaceID = id
			state, _ = pm.workspaceStates[targetWorkspaceID]
		}
	} else {
		return fmt.Errorf("either workspaceID or path must be provided")
	}

	// 如果工作区不存在，尝试注册
	if state == nil {
		if path == "" {
			return fmt.Errorf("workspace not found (workspaceID=%s) and path not provided", targetWorkspaceID)
		}
		pm.mu.Unlock()
		registeredState, err := pm.RegisterWorkspace(path)
		pm.mu.Lock()
		if err != nil {
			return fmt.Errorf("failed to register workspace: %w", err)
		}
		targetWorkspaceID = registeredState.WorkspaceID
		state = registeredState
	}

	// 更新焦点时间
	now := time.Now()
	state.LastFocus = now
	state.LastHeartbeat = now

	// 更新活跃工作区
	pm.activeWorkspaceID = targetWorkspaceID

	// 更新项目中的活跃状态
	pm.updateActiveWorkspaceInProjects(targetWorkspaceID)

	return nil
}

// updateActiveWorkspaceInProjects 更新项目中的活跃工作区标记
func (pm *ProjectManager) updateActiveWorkspaceInProjects(workspaceID string) {
	for _, project := range pm.projects {
		for _, ws := range project.Workspaces {
			ws.IsActive = (ws.WorkspaceID == workspaceID)
		}
	}
}

// GetActiveWorkspace 获取当前活跃工作区
func (pm *ProjectManager) GetActiveWorkspace() *WorkspaceState {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.activeWorkspaceID == "" {
		return nil
	}

	state, exists := pm.workspaceStates[pm.activeWorkspaceID]
	if !exists {
		return nil
	}

	return state
}

// RefreshWorkspace 刷新单个工作区（重新扫描并更新分组）
func (pm *ProjectManager) RefreshWorkspace(path string) error {
	workspaceID, err := pm.pathResolver.GetWorkspaceIDByPath(path)
	if err != nil {
		return fmt.Errorf("failed to get workspace ID from path: %w", err)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	return pm.refreshWorkspaceUnlocked(path, workspaceID)
}

// refreshWorkspaceUnlocked 刷新工作区（在锁外执行，内部会重新加锁）
func (pm *ProjectManager) refreshWorkspaceUnlocked(path string, workspaceID string) error {
	// 重新扫描所有工作区（简单实现，后续可优化为增量扫描）
	workspaces, err := pm.discovery.ScanAllWorkspaces()
	if err != nil {
		return fmt.Errorf("failed to scan workspaces: %w", err)
	}

	// 重新分组
	projectGroups := pm.groupBySameProject(workspaces)

	// 更新项目映射（需要加锁）
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for projectKey, projectInfo := range projectGroups {
		pm.projects[projectKey] = projectInfo
		pm.displayNameMap[projectInfo.ProjectName] = projectKey

		// 更新路径映射和工作区状态
		for _, ws := range projectInfo.Workspaces {
			normalizedPath, _ := pm.normalizePath(ws.Path)
			pm.pathMap[normalizedPath] = projectKey

			// 更新或创建工作区状态
			now := time.Now()
			if state, exists := pm.workspaceStates[ws.WorkspaceID]; exists {
				// 更新路径（如果变化）
				if state.Path != ws.Path {
					state.Path = ws.Path
				}
			} else {
				// 创建新状态
				pm.workspaceStates[ws.WorkspaceID] = &WorkspaceState{
					WorkspaceID:   ws.WorkspaceID,
					Path:          ws.Path,
					LastHeartbeat: now,
					LastFocus:     now,
				}
			}
		}
	}

	return nil
}

// RefreshAllWorkspaces 重新扫描所有工作区（用于手动刷新）
func (pm *ProjectManager) RefreshAllWorkspaces() error {
	return pm.Start()
}

// RegisterWorkspaceChangeCallback 注册工作区变化回调
func (pm *ProjectManager) RegisterWorkspaceChangeCallback(callback WorkspaceChangeCallback) {
	pm.callbackMu.Lock()
	defer pm.callbackMu.Unlock()
	pm.workspaceChangeCallbacks = append(pm.workspaceChangeCallbacks, callback)
}

// notifyWorkspaceChange 触发工作区变化回调（异步执行）
func (pm *ProjectManager) notifyWorkspaceChange(workspaceID string, action string) {
	pm.callbackMu.RLock()
	callbacks := make([]WorkspaceChangeCallback, len(pm.workspaceChangeCallbacks))
	copy(callbacks, pm.workspaceChangeCallbacks)
	pm.callbackMu.RUnlock()

	// 异步执行，避免阻塞
	for _, callback := range callbacks {
		go callback(workspaceID, action)
	}
}

// HasWorkspace 检查工作区是否存在
func (pm *ProjectManager) HasWorkspace(workspaceID string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	_, exists := pm.workspaceStates[workspaceID]
	return exists
}
