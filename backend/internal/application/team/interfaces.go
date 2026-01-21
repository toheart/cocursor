package team

import (
	"net/http"
	"time"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

// TeamServiceInterface 团队服务接口
// 用于依赖注入和测试 mock
//
//go:generate mockery --name=TeamServiceInterface --output=./mocks --outpkg=mocks --case=underscore
type TeamServiceInterface interface {
	// GetTeam 获取团队信息
	GetTeam(teamID string) (*domainTeam.Team, error)
	// GetTeamMembers 获取团队成员列表
	GetTeamMembers(teamID string) ([]*domainTeam.TeamMember, error)
	// GetOnlineMembers 获取在线成员
	GetOnlineMembers(teamID string) ([]*domainTeam.TeamMember, error)
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
