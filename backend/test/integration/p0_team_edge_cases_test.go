//go:build integration
// +build integration

// P0 补充测试：团队边界场景
// 验证异常情况、重复操作、多成员等场景

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cocursor/backend/test/integration/framework"
)

// TestTeamEdge_JoinInvalidEndpoint 加入不存在的端点
func TestTeamEdge_JoinInvalidEndpoint(t *testing.T) {
	framework.RequireDaemonBinary(t)

	daemon, err := framework.NewTestDaemon(framework.BinaryPath, "invalid-ep")
	require.NoError(t, err)
	require.NoError(t, daemon.Start())
	defer daemon.Stop()

	client := framework.NewAPIClient(daemon.BaseURL())
	_, err = client.CreateIdentity("测试用户")
	require.NoError(t, err)

	// 加入一个不存在的端点（服务端可能超时，所以用较长超时）
	joinResp, err := client.JoinTeam("127.0.0.1:1")
	if err != nil {
		// HTTP 请求超时也算测试通过（说明连接不可达时有保护）
		t.Logf("Join invalid endpoint got HTTP error (expected): %v", err)
		return
	}
	// 如果请求成功返回了 JSON 响应，应返回错误 code
	assert.NotEqual(t, 0, joinResp.Code, "加入不存在端点应返回错误")
	t.Logf("Join invalid endpoint: code=%d, message=%s", joinResp.Code, joinResp.Message)
}

// TestTeamEdge_DuplicateJoin 重复加入同一团队
func TestTeamEdge_DuplicateJoin(t *testing.T) {
	framework.RequireDaemonBinary(t)

	leader, err := framework.NewTestDaemon(framework.BinaryPath, "leader")
	require.NoError(t, err)
	require.NoError(t, leader.Start())
	defer leader.Stop()

	member, err := framework.NewTestDaemon(framework.BinaryPath, "member")
	require.NoError(t, err)
	require.NoError(t, member.Start())
	defer member.Stop()

	leaderClient := framework.NewAPIClient(leader.BaseURL())
	memberClient := framework.NewAPIClient(member.BaseURL())

	// Leader 创建身份和团队
	_, teamID, err := leaderClient.MustCreateIdentityAndTeam("Leader", "重复加入测试")
	require.NoError(t, err)

	// Member 创建身份并加入
	leaderEndpoint := fmt.Sprintf("localhost:%d", leader.HTTPPort)
	_, err = memberClient.MustJoinTeam("Member", leaderEndpoint)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	// 再次加入同一团队（应该不报错，幂等处理）
	joinResp2, err := memberClient.JoinTeam(leaderEndpoint)
	require.NoError(t, err)
	t.Logf("Duplicate join: code=%d, message=%s", joinResp2.Code, joinResp2.Message)

	// 验证 Leader 端成员数不会翻倍
	members, err := leaderClient.GetTeamMembers(teamID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(members.Data.Members), "成员数不应因重复加入而增加")

	// 清理
	leaderClient.DissolveTeam(teamID)
}

// TestTeamEdge_Rejoin 离开后重新加入
func TestTeamEdge_Rejoin(t *testing.T) {
	framework.RequireDaemonBinary(t)

	leader, err := framework.NewTestDaemon(framework.BinaryPath, "leader")
	require.NoError(t, err)
	require.NoError(t, leader.Start())
	defer leader.Stop()

	member, err := framework.NewTestDaemon(framework.BinaryPath, "member")
	require.NoError(t, err)
	require.NoError(t, member.Start())
	defer member.Stop()

	leaderClient := framework.NewAPIClient(leader.BaseURL())
	memberClient := framework.NewAPIClient(member.BaseURL())

	_, teamID, err := leaderClient.MustCreateIdentityAndTeam("Leader", "重新加入测试")
	require.NoError(t, err)

	leaderEndpoint := fmt.Sprintf("localhost:%d", leader.HTTPPort)
	_, err = memberClient.MustJoinTeam("Member", leaderEndpoint)
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	// 离开
	leaveResp, err := memberClient.LeaveTeam(teamID)
	require.NoError(t, err)
	require.Equal(t, 0, leaveResp.Code, "离开应成功")
	time.Sleep(1 * time.Second)

	// 验证已离开
	memberTeams, err := memberClient.ListTeams()
	require.NoError(t, err)
	assert.Equal(t, 0, memberTeams.Data.Total, "离开后应无团队")

	// 重新加入
	rejoinResp, err := memberClient.JoinTeam(leaderEndpoint)
	require.NoError(t, err)
	require.Equal(t, 0, rejoinResp.Code, "重新加入应成功, message: %s", rejoinResp.Message)
	time.Sleep(1 * time.Second)

	// 验证重新加入后状态
	memberTeamsAfter, err := memberClient.ListTeams()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, memberTeamsAfter.Data.Total, 1, "重新加入后应有团队")

	members, err := leaderClient.GetTeamMembers(teamID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(members.Data.Members), "重新加入后应有 2 个成员")

	// 清理
	leaderClient.DissolveTeam(teamID)
}

// TestTeamEdge_MultipleMembers 多成员加入（1 Leader + 2 Member）
func TestTeamEdge_MultipleMembers(t *testing.T) {
	framework.RequireDaemonBinary(t)

	leader, err := framework.NewTestDaemon(framework.BinaryPath, "leader")
	require.NoError(t, err)
	require.NoError(t, leader.Start())
	defer leader.Stop()

	member1, err := framework.NewTestDaemon(framework.BinaryPath, "member1")
	require.NoError(t, err)
	require.NoError(t, member1.Start())
	defer member1.Stop()

	member2, err := framework.NewTestDaemon(framework.BinaryPath, "member2")
	require.NoError(t, err)
	require.NoError(t, member2.Start())
	defer member2.Stop()

	leaderClient := framework.NewAPIClient(leader.BaseURL())
	member1Client := framework.NewAPIClient(member1.BaseURL())
	member2Client := framework.NewAPIClient(member2.BaseURL())

	_, teamID, err := leaderClient.MustCreateIdentityAndTeam("Leader", "多成员测试")
	require.NoError(t, err)
	t.Logf("Team created: %s", teamID)

	leaderEndpoint := fmt.Sprintf("localhost:%d", leader.HTTPPort)

	// 两个 Member 加入
	_, err = member1Client.MustJoinTeam("Member-1", leaderEndpoint)
	require.NoError(t, err)
	t.Log("Member-1 joined")

	_, err = member2Client.MustJoinTeam("Member-2", leaderEndpoint)
	require.NoError(t, err)
	t.Log("Member-2 joined")

	// 等待同步
	time.Sleep(2 * time.Second)

	// 验证成员数
	members, err := leaderClient.GetTeamMembers(teamID)
	require.NoError(t, err)
	assert.Equal(t, 3, len(members.Data.Members), "应有 3 个成员（Leader + 2 Member）")
	t.Logf("Team has %d members", len(members.Data.Members))

	// Member-1 离开
	member1Client.LeaveTeam(teamID)
	time.Sleep(1 * time.Second)

	members2, err := leaderClient.GetTeamMembers(teamID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(members2.Data.Members), "Member-1 离开后应有 2 个成员")

	// 清理
	leaderClient.DissolveTeam(teamID)
}

// TestTeamEdge_DiscoverTeams mDNS 发现团队
func TestTeamEdge_DiscoverTeams(t *testing.T) {
	framework.RequireDaemonBinary(t)

	leader, err := framework.NewTestDaemon(framework.BinaryPath, "leader")
	require.NoError(t, err)
	require.NoError(t, leader.Start())
	defer leader.Stop()

	member, err := framework.NewTestDaemon(framework.BinaryPath, "discoverer")
	require.NoError(t, err)
	require.NoError(t, member.Start())
	defer member.Stop()

	leaderClient := framework.NewAPIClient(leader.BaseURL())
	memberClient := framework.NewAPIClient(member.BaseURL())

	// Leader 创建团队（使用 ASCII 团队名避免 mDNS TXT record 编码问题）
	_, teamID, err := leaderClient.MustCreateIdentityAndTeam("Leader", "DiscoverableTeam")
	require.NoError(t, err)

	// 等待 mDNS 广播就绪
	time.Sleep(3 * time.Second)

	// Member 发现团队
	discoverResp, err := memberClient.DiscoverTeams(5)
	require.NoError(t, err)
	require.Equal(t, 0, discoverResp.Code)
	t.Logf("Discovered %d teams", len(discoverResp.Data.Teams))

	// 查看发现的团队
	found := false
	for _, team := range discoverResp.Data.Teams {
		t.Logf("Discovered team: name=%q, leader=%s, id=%s", team.Name, team.LeaderName, team.TeamID)
		if team.Name == "DiscoverableTeam" {
			found = true
		}
	}
	// mDNS 发现依赖网络环境
	assert.True(t, found, "应能通过 mDNS 发现 Leader 创建的团队")

	// 清理
	leaderClient.DissolveTeam(teamID)
}
