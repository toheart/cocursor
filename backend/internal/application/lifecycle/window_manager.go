package lifecycle

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/cocursor/backend/internal/domain/lifecycle"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// WindowManager 窗口管理器
// 负责跟踪活跃的 VSCode 窗口，并在所有窗口关闭后触发后端退出
type WindowManager struct {
	logger    *slog.Logger
	windows   map[string]*lifecycle.WindowInfo
	mu        sync.RWMutex
	idleSince *time.Time // 空闲开始时间

	// 用于优雅关闭
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownCh chan struct{}
}

// NewWindowManager 创建窗口管理器
func NewWindowManager() *WindowManager {
	ctx, cancel := context.WithCancel(context.Background())
	wm := &WindowManager{
		logger:     log.NewModuleLogger("lifecycle", "window_manager"),
		windows:    make(map[string]*lifecycle.WindowInfo),
		ctx:        ctx,
		cancel:     cancel,
		shutdownCh: make(chan struct{}),
	}
	return wm
}

// Start 启动窗口管理器（开始清理循环）
func (wm *WindowManager) Start() {
	wm.logger.Info("window manager started")
	go wm.cleanupLoop()
}

// Stop 停止窗口管理器
func (wm *WindowManager) Stop() {
	wm.cancel()
	wm.logger.Info("window manager stopped")
}

// Heartbeat 处理窗口心跳
func (wm *WindowManager) Heartbeat(windowID, projectPath string) *lifecycle.HeartbeatResponse {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	now := time.Now()

	// 更新或创建窗口信息
	if existing, ok := wm.windows[windowID]; ok {
		existing.LastSeen = now
		existing.ProjectPath = projectPath
		wm.logger.Debug("heartbeat received", "window_id", windowID, "project", projectPath)
	} else {
		wm.windows[windowID] = &lifecycle.WindowInfo{
			WindowID:    windowID,
			LastSeen:    now,
			ProjectPath: projectPath,
		}
		wm.logger.Info("new window registered", "window_id", windowID, "project", projectPath)
	}

	// 有活跃窗口，清除空闲状态
	wm.idleSince = nil

	return &lifecycle.HeartbeatResponse{
		Status:        "ok",
		ActiveWindows: len(wm.windows),
	}
}

// GetActiveWindows 获取活跃窗口列表
func (wm *WindowManager) GetActiveWindows() []*lifecycle.WindowInfo {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	result := make([]*lifecycle.WindowInfo, 0, len(wm.windows))
	for _, w := range wm.windows {
		result = append(result, &lifecycle.WindowInfo{
			WindowID:    w.WindowID,
			LastSeen:    w.LastSeen,
			ProjectPath: w.ProjectPath,
		})
	}
	return result
}

// GetStatus 获取生命周期状态
func (wm *WindowManager) GetStatus() *lifecycle.LifecycleStatus {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	status := &lifecycle.LifecycleStatus{
		ActiveWindows: wm.GetActiveWindows(),
	}

	if wm.idleSince != nil {
		status.IdleSince = wm.idleSince
		shutdownAt := wm.idleSince.Add(lifecycle.IdleShutdownTimeout)
		status.WillShutdownAt = &shutdownAt
	}

	return status
}

// ActiveWindowCount 获取活跃窗口数量
func (wm *WindowManager) ActiveWindowCount() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return len(wm.windows)
}

// cleanupLoop 定期清理超时窗口并检查是否需要退出
func (wm *WindowManager) cleanupLoop() {
	ticker := time.NewTicker(lifecycle.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-wm.ctx.Done():
			return
		case <-ticker.C:
			wm.cleanup()
			wm.checkIdleShutdown()
		}
	}
}

// cleanup 清理超时的窗口
func (wm *WindowManager) cleanup() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	now := time.Now()
	var removed []string

	for id, w := range wm.windows {
		if now.Sub(w.LastSeen) > lifecycle.HeartbeatTimeout {
			delete(wm.windows, id)
			removed = append(removed, id)
		}
	}

	if len(removed) > 0 {
		wm.logger.Info("cleaned up inactive windows",
			"removed", removed,
			"remaining", len(wm.windows),
		)
	}

	// 如果没有活跃窗口，开始计时
	if len(wm.windows) == 0 && wm.idleSince == nil {
		now := time.Now()
		wm.idleSince = &now
		wm.logger.Info("no active windows, starting idle timer",
			"will_shutdown_at", now.Add(lifecycle.IdleShutdownTimeout).Format(time.RFC3339),
		)
	}
}

// checkIdleShutdown 检查是否需要因空闲而退出
func (wm *WindowManager) checkIdleShutdown() {
	wm.mu.RLock()
	idleSince := wm.idleSince
	windowCount := len(wm.windows)
	wm.mu.RUnlock()

	// 如果有活跃窗口，不退出
	if windowCount > 0 {
		return
	}

	// 如果空闲时间超过阈值，退出
	if idleSince != nil && time.Since(*idleSince) > lifecycle.IdleShutdownTimeout {
		wm.logger.Info("idle timeout reached, initiating shutdown",
			"idle_duration", time.Since(*idleSince).String(),
		)
		wm.initiateShutdown()
	}
}

// initiateShutdown 触发优雅关闭
func (wm *WindowManager) initiateShutdown() {
	wm.logger.Info("backend is shutting down due to idle timeout")

	// 给一点时间让日志写入
	time.Sleep(100 * time.Millisecond)

	// 优雅退出
	os.Exit(0)
}

// ShutdownChannel 返回关闭通道（用于外部监听）
func (wm *WindowManager) ShutdownChannel() <-chan struct{} {
	return wm.shutdownCh
}
