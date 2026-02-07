//go:build integration
// +build integration

// P2 测试：容错与弹性场景
// 验证 Leader 离线后 Member 行为、Daemon 重启后数据持久化

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/test/integration/framework"
)

// TestResilience_LeaderOffline Leader 停止后 Member 行为
func TestResilience_LeaderOffline(t *testing.T) {
	framework.RequireDaemonBinary(t)

	leader, err := framework.NewTestDaemon(framework.BinaryPath, "leader")
	require.NoError(t, err)
	require.NoError(t, leader.Start())

	member, err := framework.NewTestDaemon(framework.BinaryPath, "member")
	require.NoError(t, err)
	require.NoError(t, member.Start())
	defer member.Stop()

	leaderClient := framework.NewAPIClient(leader.BaseURL())
	memberClient := framework.NewAPIClient(member.BaseURL())

	// 建团并加入
	_, teamID, err := leaderClient.MustCreateIdentityAndTeam("Leader", "容错测试团队")
	require.NoError(t, err)

	leaderEndpoint := fmt.Sprintf("localhost:%d", leader.HTTPPort)
	_, err = memberClient.MustJoinTeam("Member", leaderEndpoint)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)

	// 确认加入成功
	memberTeams, err := memberClient.ListTeams()
	require.NoError(t, err)
	require.GreaterOrEqual(t, memberTeams.Data.Total, 1, "Member 应已加入团队")
	t.Log("Member is in the team")

	// === Leader 停止 ===
	t.Log("--- Stopping Leader ---")
	err = leader.Stop()
	require.NoError(t, err)
	t.Log("Leader stopped")

	time.Sleep(2 * time.Second)

	// === Member 仍然可以正常运行 ===
	t.Log("--- Checking Member after Leader offline ---")

	// Member 自己的 health check 应该正常
	require.NoError(t, memberClient.HealthCheck(), "Member health 应正常")

	// Member 仍能查询自己的团队列表（本地数据）
	memberTeamsAfter, err := memberClient.ListTeams()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, memberTeamsAfter.Data.Total, 1, "Member 本地数据应保留团队信息")
	t.Logf("Member still sees %d team(s)", memberTeamsAfter.Data.Total)

	// Member 仍能查询自己的身份
	idResp, err := memberClient.GetIdentity()
	require.NoError(t, err)
	assert.True(t, idResp.Data.Exists, "Member 身份应保留")
	t.Logf("Member identity OK: %s", idResp.Data.Identity.Name)

	// Member 尝试离开（Leader 不在线）
	leaveResp, err := memberClient.LeaveTeam(teamID)
	require.NoError(t, err)
	t.Logf("Leave team result: code=%d, message=%s", leaveResp.Code, leaveResp.Message)
}

// TestResilience_MemberShareWhenLeaderOffline Leader 离线时 Member 分享会话应返回明确错误
func TestResilience_MemberShareWhenLeaderOffline(t *testing.T) {
	framework.RequireDaemonBinary(t)

	leader, err := framework.NewTestDaemon(framework.BinaryPath, "leader")
	require.NoError(t, err)
	require.NoError(t, leader.Start())

	member, err := framework.NewTestDaemon(framework.BinaryPath, "member")
	require.NoError(t, err)
	require.NoError(t, member.Start())
	defer member.Stop()

	leaderClient := framework.NewAPIClient(leader.BaseURL())
	memberClient := framework.NewAPIClient(member.BaseURL())

	// 建团并加入
	_, teamID, err := leaderClient.MustCreateIdentityAndTeam("Leader", "离线分享测试团队")
	require.NoError(t, err)

	leaderEndpoint := fmt.Sprintf("localhost:%d", leader.HTTPPort)
	_, err = memberClient.MustJoinTeam("Member", leaderEndpoint)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)

	// === 停止 Leader ===
	t.Log("--- Stopping Leader ---")
	err = leader.Stop()
	require.NoError(t, err)
	t.Log("Leader stopped")

	// 等待 Member 检测到 Leader 离线
	time.Sleep(3 * time.Second)

	// === Member 尝试分享会话（Leader 已离线） ===
	t.Log("--- Member 尝试分享会话（Leader 离线） ---")
	shareReq := &domainTeam.ShareSessionRequest{
		SessionID: "offline-session-001",
		Title:     "离线时的分享尝试",
		Messages: framework.MakeMessages([]map[string]string{
			{"role": "user", "content": "测试内容"},
		}),
		Description: "Leader 离线时的分享测试",
	}

	shareResp, err := memberClient.ShareSession(teamID, shareReq)
	require.NoError(t, err)

	// 应返回错误（非 500），而是明确的业务错误
	assert.NotEqual(t, 0, shareResp.Code, "Leader 离线时分享应返回错误码")
	t.Logf("Share when leader offline: code=%d, message=%s", shareResp.Code, shareResp.Message)

	// Member 自身应该仍然健康
	require.NoError(t, memberClient.HealthCheck(), "Member health 应正常")
}

// TestResilience_DaemonRestart Daemon 重启后数据持久化
func TestResilience_DaemonRestart(t *testing.T) {
	framework.RequireDaemonBinary(t)

	daemon, err := framework.NewTestDaemon(framework.BinaryPath, "restart-test")
	require.NoError(t, err)
	require.NoError(t, daemon.Start())

	client := framework.NewAPIClient(daemon.BaseURL())

	// 创建身份和团队
	_, teamID, err := client.MustCreateIdentityAndTeam("重启测试", "持久化团队")
	require.NoError(t, err)
	t.Logf("Created team: %s", teamID)

	// 记录配置
	dataDir := daemon.DataDir
	httpPort := daemon.HTTPPort
	mcpPort := daemon.MCPPort

	// 停止 daemon（不清理 dataDir）
	err = daemon.StopWithCleanup(false)
	require.NoError(t, err)
	t.Log("Daemon stopped (data preserved)")

	time.Sleep(1 * time.Second)

	// 用相同的 dataDir 和端口重启
	daemon2, err := framework.NewTestDaemonWithConfig(framework.BinaryPath, "restart-test-2", dataDir, httpPort, mcpPort)
	require.NoError(t, err)
	require.NoError(t, daemon2.Start())
	defer daemon2.Stop()

	client2 := framework.NewAPIClient(daemon2.BaseURL())

	// 验证身份持久化
	idResp, err := client2.GetIdentity()
	require.NoError(t, err)
	assert.True(t, idResp.Data.Exists, "重启后身份应保留")
	assert.Equal(t, "重启测试", idResp.Data.Identity.Name)
	t.Logf("Identity preserved: %s", idResp.Data.Identity.Name)

	// 验证团队持久化
	teams, err := client2.ListTeams()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, teams.Data.Total, 1, "重启后团队应保留")
	t.Logf("Teams preserved: %d", teams.Data.Total)
}
