//go:build integration
// +build integration

// P0 测试：团队完整生命周期（黑盒）
// 验证 Leader 创建团队 → Member 加入 → Member 离开 → Leader 解散团队

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cocursor/backend/test/integration/framework"
)

// TestTeamLifecycle_CreateJoinLeaveDissolve 完整团队生命周期
func TestTeamLifecycle_CreateJoinLeaveDissolve(t *testing.T) {
	framework.RequireDaemonBinary(t)

	// === 阶段 1: 启动两个独立 daemon ===
	leader, err := framework.NewTestDaemon(framework.BinaryPath, "leader")
	require.NoError(t, err, "创建 leader daemon 失败")

	member, err := framework.NewTestDaemon(framework.BinaryPath, "member")
	require.NoError(t, err, "创建 member daemon 失败")

	// 启动 leader
	err = leader.Start()
	require.NoError(t, err, "启动 leader daemon 失败")
	defer leader.Stop()

	// 启动 member
	err = member.Start()
	require.NoError(t, err, "启动 member daemon 失败")
	defer member.Stop()

	leaderClient := framework.NewAPIClient(leader.BaseURL())
	memberClient := framework.NewAPIClient(member.BaseURL())

	// 验证健康检查
	require.NoError(t, leaderClient.HealthCheck(), "leader health check 失败")
	require.NoError(t, memberClient.HealthCheck(), "member health check 失败")

	// === 阶段 2: 创建身份 ===
	t.Log("--- 阶段 2: 创建身份 ---")

	leaderIdentity, err := leaderClient.CreateIdentity("Leader-测试")
	require.NoError(t, err, "创建 leader 身份失败")
	require.Equal(t, 0, leaderIdentity.Code, "创建 leader 身份应成功, message: %s", leaderIdentity.Message)
	require.NotNil(t, leaderIdentity.Data.Identity, "leader 身份不应为空")
	assert.NotEmpty(t, leaderIdentity.Data.Identity.ID, "leader 身份 ID 不应为空")
	assert.Equal(t, "Leader-测试", leaderIdentity.Data.Identity.Name)
	t.Logf("Leader identity created: %s (%s)", leaderIdentity.Data.Identity.Name, leaderIdentity.Data.Identity.ID)

	memberIdentity, err := memberClient.CreateIdentity("Member-测试")
	require.NoError(t, err, "创建 member 身份失败")
	require.Equal(t, 0, memberIdentity.Code, "创建 member 身份应成功, message: %s", memberIdentity.Message)
	require.NotNil(t, memberIdentity.Data.Identity, "member 身份不应为空")
	assert.NotEmpty(t, memberIdentity.Data.Identity.ID, "member 身份 ID 不应为空")
	t.Logf("Member identity created: %s (%s)", memberIdentity.Data.Identity.Name, memberIdentity.Data.Identity.ID)

	// 验证身份可以通过 GET 查询到
	leaderIdentityGet, err := leaderClient.GetIdentity()
	require.NoError(t, err, "获取 leader 身份失败")
	assert.True(t, leaderIdentityGet.Data.Exists, "leader 身份应存在")
	assert.Equal(t, leaderIdentity.Data.Identity.ID, leaderIdentityGet.Data.Identity.ID, "GET 返回的身份 ID 应与创建时一致")

	// === 阶段 3: Leader 创建团队 ===
	t.Log("--- 阶段 3: Leader 创建团队 ---")

	teamResp, err := leaderClient.CreateTeam("测试团队-集成")
	require.NoError(t, err, "创建团队失败")
	require.Equal(t, 0, teamResp.Code, "创建团队应返回成功状态码, message: %s", teamResp.Message)
	require.NotNil(t, teamResp.Data.Team, "团队信息不应为空")
	assert.NotEmpty(t, teamResp.Data.Team.ID, "团队 ID 不应为空")
	assert.Equal(t, "测试团队-集成", teamResp.Data.Team.Name)
	assert.True(t, teamResp.Data.Team.IsLeader, "创建者应是 Leader")
	teamID := teamResp.Data.Team.ID
	t.Logf("Team created: %s (%s)", teamResp.Data.Team.Name, teamID)

	// 验证 Leader 的团队列表
	leaderTeams, err := leaderClient.ListTeams()
	require.NoError(t, err, "获取 leader 团队列表失败")
	assert.Equal(t, 1, leaderTeams.Data.Total, "Leader 应有 1 个团队")

	// === 阶段 4: Member 加入团队 ===
	t.Log("--- 阶段 4: Member 加入团队 ---")

	leaderEndpoint := fmt.Sprintf("localhost:%d", leader.HTTPPort)
	joinResp, err := memberClient.JoinTeam(leaderEndpoint)
	require.NoError(t, err, "加入团队失败")
	require.Equal(t, 0, joinResp.Code, "加入团队应返回成功状态码, message: %s", joinResp.Message)
	if joinResp.Data.Team != nil {
		t.Logf("Member joined team: %s (%s), leader_endpoint: %s",
			joinResp.Data.Team.Name, joinResp.Data.Team.ID, joinResp.Data.Team.LeaderEndpoint)
	} else {
		t.Log("Member join returned nil team data")
	}

	// 等待 P2P 成员同步（重试机制，最多等 10 秒）
	var memberCount int
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		members, err := leaderClient.GetTeamMembers(teamID)
		if err == nil {
			memberCount = len(members.Data.Members)
			t.Logf("  [retry %d] Leader sees %d members", i+1, memberCount)
			if memberCount >= 2 {
				break
			}
		}
	}
	assert.GreaterOrEqual(t, memberCount, 2, "团队应至少有 2 个成员（Leader + Member）")

	// 验证 Member 的团队列表
	memberTeams, err := memberClient.ListTeams()
	require.NoError(t, err, "获取 member 团队列表失败")
	t.Logf("Member team count: %d", memberTeams.Data.Total)
	assert.GreaterOrEqual(t, memberTeams.Data.Total, 1, "Member 应有至少 1 个团队")

	// === 阶段 5: Member 离开团队 ===
	t.Log("--- 阶段 5: Member 离开团队 ---")

	leaveResp, err := memberClient.LeaveTeam(teamID)
	require.NoError(t, err, "离开团队失败")
	require.Equal(t, 0, leaveResp.Code, "离开团队应返回成功状态码, message: %s", leaveResp.Message)
	t.Log("Member left the team successfully")

	// 等待同步
	time.Sleep(1 * time.Second)

	// 验证 Member 不再有团队
	memberTeamsAfterLeave, err := memberClient.ListTeams()
	require.NoError(t, err, "获取 member 离开后团队列表失败")
	assert.Equal(t, 0, memberTeamsAfterLeave.Data.Total, "Member 离开后不应有团队")

	// === 阶段 6: Leader 解散团队 ===
	t.Log("--- 阶段 6: Leader 解散团队 ---")

	dissolveResp, err := leaderClient.DissolveTeam(teamID)
	require.NoError(t, err, "解散团队失败")
	require.Equal(t, 0, dissolveResp.Code, "解散团队应返回成功状态码, message: %s", dissolveResp.Message)
	t.Log("Team dissolved successfully")

	// 验证 Leader 不再有团队
	leaderTeamsAfterDissolve, err := leaderClient.ListTeams()
	require.NoError(t, err, "获取 leader 解散后团队列表失败")
	assert.Equal(t, 0, leaderTeamsAfterDissolve.Data.Total, "Leader 解散后不应有团队")

	t.Log("=== P0 团队完整生命周期测试通过 ===")
}

// TestTeamLifecycle_IdentityPersistence 身份持久化测试
func TestTeamLifecycle_IdentityPersistence(t *testing.T) {
	framework.RequireDaemonBinary(t)

	// 启动 daemon
	daemon, err := framework.NewTestDaemon(framework.BinaryPath, "identity-test")
	require.NoError(t, err)
	require.NoError(t, daemon.Start())
	defer daemon.Stop()

	client := framework.NewAPIClient(daemon.BaseURL())

	// 创建身份
	created, err := client.CreateIdentity("持久化测试")
	require.NoError(t, err)
	require.Equal(t, 0, created.Code, "创建身份应成功, message: %s", created.Message)
	require.NotNil(t, created.Data.Identity)
	createdID := created.Data.Identity.ID

	// 重新获取身份
	retrieved, err := client.GetIdentity()
	require.NoError(t, err)
	require.True(t, retrieved.Data.Exists, "身份应存在")
	assert.Equal(t, createdID, retrieved.Data.Identity.ID, "身份 ID 应保持一致")
	assert.Equal(t, "持久化测试", retrieved.Data.Identity.Name)
}

// TestTeamLifecycle_CreateTeamWithoutIdentity 未创建身份就创建团队
func TestTeamLifecycle_CreateTeamWithoutIdentity(t *testing.T) {
	framework.RequireDaemonBinary(t)

	daemon, err := framework.NewTestDaemon(framework.BinaryPath, "no-identity-test")
	require.NoError(t, err)
	require.NoError(t, daemon.Start())
	defer daemon.Stop()

	client := framework.NewAPIClient(daemon.BaseURL())

	// 尝试在没有身份的情况下创建团队，应返回错误
	teamResp, err := client.CreateTeam("无身份团队")
	require.NoError(t, err, "HTTP 请求本身不应失败")
	assert.NotEqual(t, 0, teamResp.Code, "未创建身份就创建团队应返回错误")
	t.Logf("Create team without identity: code=%d, message=%s", teamResp.Code, teamResp.Message)
}
