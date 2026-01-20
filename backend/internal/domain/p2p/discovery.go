package p2p

import (
	"context"
	"time"
)

// ServiceType mDNS 服务类型
const ServiceType = "_cocursor._tcp"

// DefaultPort 默认端口
const DefaultPort = 19960

// 心跳配置
const (
	HeartbeatInterval = 30 * time.Second // 心跳间隔
	HeartbeatTimeout  = 60 * time.Second // 心跳超时
)

// 重连配置
const (
	ReconnectMinInterval = 1 * time.Second  // 最小重连间隔
	ReconnectMaxInterval = 30 * time.Second // 最大重连间隔
)

// ServiceInfo mDNS 服务信息
type ServiceInfo struct {
	InstanceName string            // 服务实例名（团队名称）
	ServiceType  string            // 服务类型
	Domain       string            // 域（默认 "local."）
	HostName     string            // 主机名
	Port         int               // 端口
	IPs          []string          // IP 地址列表
	TxtRecords   map[string]string // TXT 记录
}

// GetTeamID 从 TXT 记录获取团队 ID
func (s *ServiceInfo) GetTeamID() string {
	return s.TxtRecords["team_id"]
}

// GetLeaderName 从 TXT 记录获取 Leader 名称
func (s *ServiceInfo) GetLeaderName() string {
	return s.TxtRecords["leader_name"]
}

// GetMemberCount 从 TXT 记录获取成员数量
func (s *ServiceInfo) GetMemberCount() string {
	return s.TxtRecords["member_count"]
}

// GetVersion 从 TXT 记录获取版本
func (s *ServiceInfo) GetVersion() string {
	return s.TxtRecords["version"]
}

// Discovery 服务发现接口
type Discovery interface {
	// Discover 发现服务
	Discover(ctx context.Context, timeout time.Duration) ([]ServiceInfo, error)
	// Close 关闭发现器
	Close() error
}

// Advertiser 服务广播接口
type Advertiser interface {
	// Start 开始广播服务
	Start(info ServiceInfo) error
	// Stop 停止广播
	Stop() error
	// UpdateTxtRecords 更新 TXT 记录
	UpdateTxtRecords(records map[string]string) error
	// IsRunning 是否正在广播
	IsRunning() bool
}

// ConnectionState 连接状态
type ConnectionState int

const (
	// StateDisconnected 未连接
	StateDisconnected ConnectionState = iota
	// StateConnecting 正在连接
	StateConnecting
	// StateConnected 已连接
	StateConnected
	// StateReconnecting 正在重连
	StateReconnecting
)

// String 返回状态字符串
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	default:
		return "unknown"
	}
}

// ConnectionInfo 连接信息
type ConnectionInfo struct {
	TeamID    string          // 团队 ID
	Endpoint  string          // 端点
	State     ConnectionState // 连接状态
	LastPing  time.Time       // 最后心跳时间
	RetryCount int            // 重试次数
}

// EventHandler 事件处理器
type EventHandler interface {
	// HandleEvent 处理事件
	HandleEvent(event *Event) error
}

// EventListener 事件监听器
type EventListener interface {
	// OnEvent 事件回调
	OnEvent(event *Event)
	// OnConnect 连接成功回调
	OnConnect(teamID string)
	// OnDisconnect 断开连接回调
	OnDisconnect(teamID string, err error)
}
