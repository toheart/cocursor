package workspace

import (
	"fmt"
	"sync"
	"time"

	"github.com/cocursor/backend/internal/infrastructure/cursor"
)

// Workspace 工作区信息
type Workspace struct {
	ID           string    // 工作区 ID（哈希值）
	Path         string    // 项目路径
	LastHeartbeat time.Time // 最后心跳时间
	LastFocus    time.Time // 最后获得焦点时间
}

// Manager 工作区管理器（单例）
type Manager struct {
	mu                sync.RWMutex
	workspaces        map[string]*Workspace // workspaceID -> Workspace
	activeWorkspaceID string                 // 当前活跃工作区 ID
	pathResolver      *cursor.PathResolver
}

var (
	instance *Manager
	once     sync.Once
)

// GetInstance 获取 Manager 单例实例
func GetInstance() *Manager {
	once.Do(func() {
		instance = &Manager{
			workspaces:   make(map[string]*Workspace),
			pathResolver: cursor.NewPathResolver(),
		}
	})
	return instance
}

// Register 注册工作区
// path: 项目路径，如 "D:/code/cocursor"
// 返回: Workspace 实例和错误
func (m *Manager) Register(path string) (*Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 通过路径计算 WorkspaceID
	workspaceID, err := m.pathResolver.GetWorkspaceIDByPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace ID: %w", err)
	}

	// 创建或更新 Workspace 记录
	now := time.Now()
	ws, exists := m.workspaces[workspaceID]
	if !exists {
		ws = &Workspace{
			ID:            workspaceID,
			Path:          path,
			LastHeartbeat: now,
		}
		m.workspaces[workspaceID] = ws
	} else {
		// 更新心跳时间
		ws.LastHeartbeat = now
		// 如果路径发生变化，更新路径
		if ws.Path != path {
			ws.Path = path
		}
	}

	// 如果这是第一个工作区，或者当前没有活跃工作区，设置为活跃
	if m.activeWorkspaceID == "" {
		m.activeWorkspaceID = workspaceID
	}

	return ws, nil
}

// GetActive 获取当前活跃工作区
// 返回: 最后上报心跳或焦点的项目，如果没有则返回 nil
func (m *Manager) GetActive() *Workspace {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.activeWorkspaceID == "" {
		return nil
	}

	ws, exists := m.workspaces[m.activeWorkspaceID]
	if !exists {
		return nil
	}

	return ws
}

// UpdateFocus 更新活跃工作区（窗口获得焦点时调用）
// workspaceID: 工作区 ID，如果为空则通过 path 查找
// path: 项目路径（可选，如果提供了 workspaceID 则不需要）
func (m *Manager) UpdateFocus(workspaceID string, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var targetWorkspaceID string

	if workspaceID != "" {
		// 直接使用提供的 workspaceID
		targetWorkspaceID = workspaceID
	} else if path != "" {
		// 通过路径查找 workspaceID
		id, err := m.pathResolver.GetWorkspaceIDByPath(path)
		if err != nil {
			return fmt.Errorf("failed to get workspace ID from path: %w", err)
		}
		targetWorkspaceID = id
	} else {
		return fmt.Errorf("either workspaceID or path must be provided")
	}

	// 检查工作区是否存在
	ws, exists := m.workspaces[targetWorkspaceID]
	if !exists {
		// 如果工作区不存在，尝试注册
		if path == "" {
			return fmt.Errorf("workspace not found and path not provided")
		}
		var err error
		ws, err = m.Register(path)
		if err != nil {
			return fmt.Errorf("failed to register workspace: %w", err)
		}
	}

	// 更新焦点时间
	now := time.Now()
	ws.LastFocus = now
	ws.LastHeartbeat = now

	// 更新活跃工作区
	m.activeWorkspaceID = targetWorkspaceID

	return nil
}

// GetWorkspace 根据 workspaceID 获取工作区
func (m *Manager) GetWorkspace(workspaceID string) (*Workspace, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ws, exists := m.workspaces[workspaceID]
	return ws, exists
}
