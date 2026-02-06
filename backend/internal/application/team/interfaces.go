package team

import (
	"net/http"
	"time"

	"github.com/cocursor/backend/internal/domain/p2p"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

// TeamServiceInterface 团队服务接口
// 用于依赖注入和测试 mock
//
//go:generate mockery --name=TeamServiceInterface --output=./mocks --outpkg=mocks --case=underscore
type TeamServiceInterface interface {
	// GetTeam 获取团队信息
	GetTeam(teamID string) (*domainTeam.Team, error)
	// GetTeamList 获取已加入团队列表
	GetTeamList() []*domainTeam.Team
	// GetTeamMembers 获取团队成员列表
	GetTeamMembers(teamID string) ([]*domainTeam.TeamMember, error)
	// GetOnlineMembers 获取在线成员
	GetOnlineMembers(teamID string) ([]*domainTeam.TeamMember, error)
	// GetSkillIndex 获取团队技能目录
	GetSkillIndex(teamID string) (*domainTeam.TeamSkillIndex, error)
	// UpdateLastSync 更新最后同步时间
	UpdateLastSync(teamID string)
	// UpdateLeaderOnline 更新 Leader 在线状态
	UpdateLeaderOnline(teamID string, online bool)
}

// IdentityProvider 身份提供者接口
// 用于获取和管理本机身份信息
//
//go:generate mockery --name=IdentityProvider --output=./mocks --outpkg=mocks --case=underscore
type IdentityProvider interface {
	// GetIdentity 获取本机身份
	GetIdentity() (*domainTeam.Identity, error)
	// CreateIdentity 创建本机身份
	CreateIdentity(name string) (*domainTeam.Identity, error)
	// UpdateIdentity 更新本机身份
	UpdateIdentity(name string) (*domainTeam.Identity, error)
	// EnsureIdentity 确保身份存在，不存在则创建
	EnsureIdentity(name string) (*domainTeam.Identity, error)
}

// NetworkConfigProvider 网络配置提供者接口
// 用于获取和管理网络配置信息
//
//go:generate mockery --name=NetworkConfigProvider --output=./mocks --outpkg=mocks --case=underscore
type NetworkConfigProvider interface {
	// GetNetworkInterfaces 获取可用网卡列表
	GetNetworkInterfaces() ([]domainTeam.NetworkInterface, error)
	// GetNetworkConfig 获取网卡配置
	GetNetworkConfig() *domainTeam.NetworkConfig
	// SetNetworkConfig 设置网卡配置
	SetNetworkConfig(preferredInterface, preferredIP string) error
	// GetCurrentEndpoint 获取当前端点地址
	GetCurrentEndpoint() string
}

// EventBroadcaster 事件广播者接口
// 用于向团队成员广播事件
//
//go:generate mockery --name=EventBroadcaster --output=./mocks --outpkg=mocks --case=underscore
type EventBroadcaster interface {
	// BroadcastEvent 广播事件到团队所有成员
	BroadcastEvent(teamID string, eventType p2p.EventType, payload interface{}) error
}

// SkillIndexProvider 技能目录提供者接口
// 用于获取和管理团队技能目录
//
//go:generate mockery --name=SkillIndexProvider --output=./mocks --outpkg=mocks --case=underscore
type SkillIndexProvider interface {
	// GetSkillIndex 获取团队技能目录
	GetSkillIndex(teamID string) (*domainTeam.TeamSkillIndex, error)
	// AddSkillToIndex 添加技能到索引
	AddSkillToIndex(teamID string, entry *domainTeam.TeamSkillEntry) error
	// RemoveSkillFromIndex 从索引中移除技能
	RemoveSkillFromIndex(teamID, pluginID string) error
}

// ProjectConfigStoreInterface 项目配置存储接口
//
//go:generate mockery --name=ProjectConfigStoreInterface --output=./mocks --outpkg=mocks --case=underscore
type ProjectConfigStoreInterface interface {
	// Load 加载配置
	Load() (*domainTeam.TeamProjectConfig, error)
	// Save 保存配置
	Save(config *domainTeam.TeamProjectConfig) error
	// AddProject 添加项目
	AddProject(project domainTeam.ProjectMatcher) error
	// RemoveProject 移除项目
	RemoveProject(projectID string) error
	// UpdateProject 更新项目
	UpdateProject(project domainTeam.ProjectMatcher) error
	// GetRepoURLs 获取所有项目的 Repo URL
	GetRepoURLs() []string
}

// WeeklyStatsStoreInterface 周统计缓存存储接口
//
//go:generate mockery --name=WeeklyStatsStoreInterface --output=./mocks --outpkg=mocks --case=underscore
type WeeklyStatsStoreInterface interface {
	// Get 获取成员周统计
	Get(memberID, weekStart string) (*domainTeam.MemberWeeklyStats, error)
	// Set 设置成员周统计
	Set(memberID, weekStart string, stats *domainTeam.MemberWeeklyStats) error
	// SetWithExpiration 设置带过期时间的成员周统计
	SetWithExpiration(memberID, weekStart string, stats *domainTeam.MemberWeeklyStats, expiration time.Duration) error
	// GetAll 获取指定周的所有成员统计
	GetAll(weekStart string) (map[string]*domainTeam.MemberWeeklyStats, error)
	// GetExpiredMembers 获取过期或未缓存的成员 ID
	GetExpiredMembers(memberIDs []string, weekStart string) []string
	// Delete 删除成员周统计
	Delete(memberID, weekStart string) error
	// Clear 清空所有缓存
	Clear() error
}

// HTTPClientInterface HTTP 客户端接口
//
//go:generate mockery --name=HTTPClientInterface --output=./mocks --outpkg=mocks --case=underscore
type HTTPClientInterface interface {
	// Do 执行 HTTP 请求
	Do(req *http.Request) (*http.Response, error)
}

// TeamServiceConfig 团队服务配置
// 用于共享配置和资源，避免重复创建
type TeamServiceConfig struct {
	// Port P2P 服务端口
	Port int
	// Version 应用版本号
	Version string
	// HTTPClient 共享的 HTTP 客户端
	HTTPClient *http.Client
	// RequestTimeout 请求超时时间
	RequestTimeout time.Duration
	// SyncTimeout 同步操作超时时间
	SyncTimeout time.Duration
}

// DefaultTeamServiceConfig 返回默认配置
func DefaultTeamServiceConfig() *TeamServiceConfig {
	return &TeamServiceConfig{
		Port:           19960,
		Version:        "unknown",
		RequestTimeout: 10 * time.Second,
		SyncTimeout:    30 * time.Second,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewTeamServiceConfig 创建团队服务配置
func NewTeamServiceConfig(port int, version string) *TeamServiceConfig {
	config := DefaultTeamServiceConfig()
	config.Port = port
	config.Version = version
	return config
}

// CacheManager 缓存管理器接口
// 用于统一管理服务间的缓存一致性
//
//go:generate mockery --name=CacheManager --output=./mocks --outpkg=mocks --case=underscore
type CacheManager interface {
	// InvalidateMemberCache 使成员相关缓存失效
	InvalidateMemberCache(teamID, memberID string)
	// InvalidateTeamCache 使团队相关缓存失效
	InvalidateTeamCache(teamID string)
}
