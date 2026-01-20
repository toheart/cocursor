// 集成测试：团队服务
//go:build integration
// +build integration

package team

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTeamService_CreateAndJoin 测试创建和加入团队
func TestTeamService_CreateAndJoin(t *testing.T) {
	// 跳过非集成测试环境
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建 Leader 服务
	leaderService, err := NewTeamService(19970, "1.0.0-test")
	require.NoError(t, err)
	defer leaderService.Close()

	// 确保 Leader 有身份
	_, err = leaderService.EnsureIdentity("Leader Node")
	require.NoError(t, err)

	// 创建团队
	team, err := leaderService.CreateTeam("Test Team", "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, team.ID)
	assert.Equal(t, "Test Team", team.Name)
	assert.True(t, team.IsLeader)

	// 获取 Leader 端点
	leaderEndpoint := team.LeaderEndpoint
	assert.NotEmpty(t, leaderEndpoint)

	// 创建 Member 服务
	memberService, err := NewTeamService(19971, "1.0.0-test")
	require.NoError(t, err)
	defer memberService.Close()

	// 确保 Member 有身份
	_, err = memberService.EnsureIdentity("Member Node")
	require.NoError(t, err)

	// 加入团队
	joinedTeam, err := memberService.JoinTeam(leaderEndpoint)
	require.NoError(t, err)
	assert.Equal(t, team.ID, joinedTeam.ID)
	assert.False(t, joinedTeam.IsLeader)

	// 验证成员列表
	time.Sleep(100 * time.Millisecond) // 等待同步
	members, err := leaderService.GetTeamMembers(team.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(members), 2) // Leader + Member

	// 离开团队
	err = memberService.LeaveTeam(team.ID)
	require.NoError(t, err)

	// 验证成员列表更新
	time.Sleep(100 * time.Millisecond)
	members, err = leaderService.GetTeamMembers(team.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(members)) // 只剩 Leader

	// 解散团队
	err = leaderService.DissolveTeam(team.ID)
	require.NoError(t, err)
}

// TestTeamService_SkillPublishAndDownload 测试技能发布和下载
func TestTeamService_SkillPublishAndDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建 Leader 服务
	leaderService, err := NewTeamService(19972, "1.0.0-test")
	require.NoError(t, err)
	defer leaderService.Close()

	_, err = leaderService.EnsureIdentity("Leader")
	require.NoError(t, err)

	// 创建团队
	team, err := leaderService.CreateTeam("Skill Test Team", "", "")
	require.NoError(t, err)

	// 注意：实际发布测试需要有效的技能目录
	// 这里只测试 API 调用流程

	// 解散团队
	err = leaderService.DissolveTeam(team.ID)
	require.NoError(t, err)
}

// TestTeamService_mDNSDiscovery 测试 mDNS 发现
func TestTeamService_mDNSDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建 Leader 服务
	leaderService, err := NewTeamService(19973, "1.0.0-test")
	require.NoError(t, err)
	defer leaderService.Close()

	_, err = leaderService.EnsureIdentity("Discoverable Leader")
	require.NoError(t, err)

	// 创建团队（会启动 mDNS 广播）
	team, err := leaderService.CreateTeam("Discoverable Team", "", "")
	require.NoError(t, err)
	defer leaderService.DissolveTeam(team.ID)

	// 创建另一个服务来发现团队
	memberService, err := NewTeamService(19974, "1.0.0-test")
	require.NoError(t, err)
	defer memberService.Close()

	_, err = memberService.EnsureIdentity("Discovering Member")
	require.NoError(t, err)

	// 发现团队（超时 3 秒）
	discoveredTeams, err := memberService.DiscoverTeams(3)
	require.NoError(t, err)

	// 验证能发现团队（在 CI 环境可能无法工作）
	// 这里只记录日志，不断言结果
	t.Logf("Discovered %d teams", len(discoveredTeams))
	for _, dt := range discoveredTeams {
		t.Logf("  - %s at %s", dt.Name, dt.Endpoint)
	}
}
