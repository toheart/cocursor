package lifecycle

import "time"

// WindowInfo 窗口信息
type WindowInfo struct {
	// WindowID 唯一窗口标识（UUID 格式）
	WindowID string `json:"window_id"`
	// LastSeen 最后心跳时间
	LastSeen time.Time `json:"last_seen"`
	// ProjectPath 当前项目路径（可选）
	ProjectPath string `json:"project_path,omitempty"`
}

// HeartbeatRequest 心跳请求
type HeartbeatRequest struct {
	// WindowID 唯一窗口标识
	WindowID string `json:"window_id" binding:"required"`
	// ProjectPath 当前项目路径（可选）
	ProjectPath string `json:"project_path,omitempty"`
}

// HeartbeatResponse 心跳响应
type HeartbeatResponse struct {
	// Status 状态
	Status string `json:"status"`
	// ActiveWindows 当前活跃窗口数
	ActiveWindows int `json:"active_windows"`
}

// LifecycleStatus 生命周期状态（用于调试）
type LifecycleStatus struct {
	// ActiveWindows 活跃窗口列表
	ActiveWindows []*WindowInfo `json:"active_windows"`
	// IdleSince 空闲开始时间（无活跃窗口时）
	IdleSince *time.Time `json:"idle_since,omitempty"`
	// WillShutdownAt 预计关闭时间（空闲超时时）
	WillShutdownAt *time.Time `json:"will_shutdown_at,omitempty"`
}

// 常量定义
const (
	// HeartbeatInterval 心跳间隔
	HeartbeatInterval = 30 * time.Second
	// HeartbeatTimeout 心跳超时（4 次心跳失败）
	HeartbeatTimeout = 2 * time.Minute
	// IdleShutdownTimeout 空闲关闭超时
	IdleShutdownTimeout = 5 * time.Minute
	// CleanupInterval 清理间隔
	CleanupInterval = 1 * time.Minute
)
