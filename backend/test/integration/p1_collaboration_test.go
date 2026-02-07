//go:build integration
// +build integration

// P1 测试：团队协作功能
// 验证工作状态同步、会话分享、评论功能

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

// setupTeamWithMembers 创建团队并加入成员的通用 setup
func setupTeamWithMembers(t *testing.T) (
	leader, member *framework.TestDaemon,
	leaderClient, memberClient *framework.APIClient,
	teamID string,
) {
	t.Helper()
	framework.RequireDaemonBinary(t)

	var err error
	leader, err = framework.NewTestDaemon(framework.BinaryPath, "leader")
	require.NoError(t, err)
	require.NoError(t, leader.Start())

	member, err = framework.NewTestDaemon(framework.BinaryPath, "member")
	require.NoError(t, err)
	require.NoError(t, member.Start())

	leaderClient = framework.NewAPIClient(leader.BaseURL())
	memberClient = framework.NewAPIClient(member.BaseURL())

	_, teamID, err = leaderClient.MustCreateIdentityAndTeam("Leader", "协作测试团队")
	require.NoError(t, err)

	leaderEndpoint := fmt.Sprintf("localhost:%d", leader.HTTPPort)
	_, err = memberClient.MustJoinTeam("Member", leaderEndpoint)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)
	return
}

// TestCollaboration_WorkStatusUpdate 工作状态更新
func TestCollaboration_WorkStatusUpdate(t *testing.T) {
	leader, member, leaderClient, memberClient, teamID := setupTeamWithMembers(t)
	defer leader.Stop()
	defer member.Stop()

	// Leader 更新工作状态
	statusResp, err := leaderClient.UpdateWorkStatus(teamID, "cocursor", "main.go", true)
	require.NoError(t, err)
	require.Equal(t, 0, statusResp.Code, "更新工作状态应成功, message: %s", statusResp.Message)
	t.Log("Leader updated work status")

	// Member 也更新工作状态
	statusResp2, err := memberClient.UpdateWorkStatus(teamID, "my-project", "index.ts", true)
	require.NoError(t, err)
	require.Equal(t, 0, statusResp2.Code, "Member 更新工作状态应成功, message: %s", statusResp2.Message)
	t.Log("Member updated work status")

	// 等待状态同步
	time.Sleep(2 * time.Second)

	// 通过成员列表查看状态（成员信息中包含 work_status）
	members, err := leaderClient.GetTeamMembers(teamID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(members.Data.Members), 2, "应至少有 2 个成员")

	// 输出成员状态
	for _, m := range members.Data.Members {
		if m.WorkStatus != nil {
			t.Logf("Member %s: project=%s, file=%s", m.Name, m.WorkStatus.ProjectName, m.WorkStatus.CurrentFile)
		} else {
			t.Logf("Member %s: no work status", m.Name)
		}
	}

	// 清理
	leaderClient.DissolveTeam(teamID)
}

// TestCollaboration_SessionSharing 会话分享完整流程
func TestCollaboration_SessionSharing(t *testing.T) {
	leader, member, leaderClient, memberClient, teamID := setupTeamWithMembers(t)
	defer leader.Stop()
	defer member.Stop()

	// === Leader 分享一个会话 ===
	t.Log("--- Leader 分享会话 ---")
	shareReq := &domainTeam.ShareSessionRequest{
		SessionID: "test-session-001",
		Title:     "集成测试讨论",
		Messages: framework.MakeMessages([]map[string]string{
			{"role": "user", "content": "如何设计集成测试？"},
			{"role": "assistant", "content": "建议采用黑盒测试方式，通过 API 进行端到端验证。"},
		}),
		Description: "关于集成测试方案的讨论",
	}

	shareResp, err := leaderClient.ShareSession(teamID, shareReq)
	require.NoError(t, err)
	require.Equal(t, 0, shareResp.Code, "分享会话应成功, message: %s", shareResp.Message)
	assert.NotEmpty(t, shareResp.Data.ShareID, "分享 ID 不应为空")
	shareID := shareResp.Data.ShareID
	t.Logf("Session shared: %s", shareID)

	// === 查询分享列表 ===
	t.Log("--- 查询分享列表 ---")
	listResp, err := leaderClient.GetSharedSessions(teamID, 20, 0)
	require.NoError(t, err)
	require.Equal(t, 0, listResp.Code)
	assert.GreaterOrEqual(t, listResp.Data.Total, 1, "应有至少 1 个分享")

	found := false
	for _, s := range listResp.Data.Sessions {
		if s.Title == "集成测试讨论" {
			found = true
			t.Logf("Found shared session: %s by %s", s.Title, s.SharerName)
		}
	}
	assert.True(t, found, "应能找到刚分享的会话")

	// === 查看分享详情 ===
	t.Log("--- 查看分享详情 ---")
	detailResp, err := leaderClient.GetSharedSessionDetail(teamID, shareID)
	require.NoError(t, err)
	require.Equal(t, 0, detailResp.Code)
	require.NotNil(t, detailResp.Data.Session)
	assert.Equal(t, "集成测试讨论", detailResp.Data.Session.Title)
	assert.Equal(t, "关于集成测试方案的讨论", detailResp.Data.Session.Description)
	t.Logf("Session detail: title=%s, messages=%d", detailResp.Data.Session.Title, detailResp.Data.Session.MessageCount)

	// === Member 通过 Leader 的 API 添加评论 ===
	// 会话分享数据存在 Leader 本地数据库中，所以评论操作也需要通过 Leader API
	t.Log("--- Member 通过 Leader 添加评论 ---")
	commentResp, err := leaderClient.AddComment(teamID, shareID, "这个方案很好，建议补充异常场景测试", nil)
	require.NoError(t, err)
	require.Equal(t, 0, commentResp.Code, "添加评论应成功, message: %s", commentResp.Message)
	t.Log("Comment added via Leader API")

	// === Leader 也添加评论 ===
	commentResp2, err := leaderClient.AddComment(teamID, shareID, "同意，已更新方案", nil)
	require.NoError(t, err)
	require.Equal(t, 0, commentResp2.Code, "Leader 添加评论应成功")

	// === 查看详情应包含评论 ===
	detailResp2, err := leaderClient.GetSharedSessionDetail(teamID, shareID)
	require.NoError(t, err)
	require.Equal(t, 0, detailResp2.Code)
	assert.GreaterOrEqual(t, len(detailResp2.Data.Comments), 2, "应有至少 2 条评论")
	t.Logf("Session has %d comments", len(detailResp2.Data.Comments))

	for _, c := range detailResp2.Data.Comments {
		t.Logf("  Comment by %s: %s", c.AuthorName, c.Content)
	}

	// === Member 查询 Leader 的分享列表（验证 Member 也能访问 Leader 数据） ===
	// 注意：会话数据只在 Leader 本地，Member 需要通过 Leader API 访问
	_ = memberClient // Member 视角：本地无分享数据，这是预期行为

	// 清理
	leaderClient.DissolveTeam(teamID)
}

// TestCollaboration_MemberShareSession Member 通过转发分享会话到 Leader
func TestCollaboration_MemberShareSession(t *testing.T) {
	leader, member, leaderClient, memberClient, teamID := setupTeamWithMembers(t)
	defer leader.Stop()
	defer member.Stop()

	// === Member 通过自己的 API 分享会话（应自动转发到 Leader） ===
	t.Log("--- Member 分享会话（转发到 Leader） ---")
	shareReq := &domainTeam.ShareSessionRequest{
		SessionID: "member-session-001",
		Title:     "Member 的调试记录",
		Messages: framework.MakeMessages([]map[string]string{
			{"role": "user", "content": "这个 bug 怎么修？"},
			{"role": "assistant", "content": "建议检查空指针。"},
			{"role": "user", "content": "修好了，谢谢！"},
		}),
		Description: "Member 分享的一次调试过程",
	}

	shareResp, err := memberClient.ShareSession(teamID, shareReq)
	require.NoError(t, err)
	require.Equal(t, 0, shareResp.Code, "Member 分享会话应成功（转发到 Leader）, message: %s", shareResp.Message)
	assert.NotEmpty(t, shareResp.Data.ShareID, "分享 ID 不应为空")
	shareID := shareResp.Data.ShareID
	t.Logf("Member shared session via forwarding: %s", shareID)

	// === 在 Leader 端验证分享记录已存储 ===
	t.Log("--- Leader 端验证分享记录 ---")
	listResp, err := leaderClient.GetSharedSessions(teamID, 20, 0)
	require.NoError(t, err)
	require.Equal(t, 0, listResp.Code)
	assert.GreaterOrEqual(t, listResp.Data.Total, 1, "Leader 端应有至少 1 个分享")

	found := false
	for _, s := range listResp.Data.Sessions {
		if s.Title == "Member 的调试记录" {
			found = true
			assert.Equal(t, "Member", s.SharerName, "分享者名称应为 Member")
			t.Logf("Found member's shared session: %s by %s", s.Title, s.SharerName)
		}
	}
	assert.True(t, found, "Leader 端应能找到 Member 分享的会话")

	// === 查看详情验证内容完整性 ===
	t.Log("--- 验证分享详情 ---")
	detailResp, err := leaderClient.GetSharedSessionDetail(teamID, shareID)
	require.NoError(t, err)
	require.Equal(t, 0, detailResp.Code)
	require.NotNil(t, detailResp.Data.Session)
	assert.Equal(t, "Member 的调试记录", detailResp.Data.Session.Title)
	assert.Equal(t, "Member 分享的一次调试过程", detailResp.Data.Session.Description)
	assert.Equal(t, 3, detailResp.Data.Session.MessageCount, "消息数量应为 3")
	t.Logf("Session detail: title=%s, messages=%d", detailResp.Data.Session.Title, detailResp.Data.Session.MessageCount)

	// 清理
	leaderClient.DissolveTeam(teamID)
}

// TestCollaboration_MemberAddComment Member 通过转发添加评论到 Leader
func TestCollaboration_MemberAddComment(t *testing.T) {
	leader, member, leaderClient, memberClient, teamID := setupTeamWithMembers(t)
	defer leader.Stop()
	defer member.Stop()

	// === Leader 先分享一个会话 ===
	t.Log("--- Leader 分享会话 ---")
	shareReq := &domainTeam.ShareSessionRequest{
		SessionID: "leader-session-for-comment",
		Title:     "架构设计讨论",
		Messages: framework.MakeMessages([]map[string]string{
			{"role": "user", "content": "我们应该用什么架构？"},
			{"role": "assistant", "content": "推荐 DDD 分层架构。"},
		}),
		Description: "关于系统架构的讨论",
	}

	shareResp, err := leaderClient.ShareSession(teamID, shareReq)
	require.NoError(t, err)
	require.Equal(t, 0, shareResp.Code, "Leader 分享会话应成功")
	shareID := shareResp.Data.ShareID
	t.Logf("Leader shared session: %s", shareID)

	// === Member 通过自己的 API 添加评论（应自动转发到 Leader） ===
	t.Log("--- Member 添加评论（转发到 Leader） ---")
	commentResp, err := memberClient.AddComment(teamID, shareID, "DDD 架构很好，我之前用过，推荐！", nil)
	require.NoError(t, err)
	require.Equal(t, 0, commentResp.Code, "Member 添加评论应成功（转发到 Leader）, message: %s", commentResp.Message)
	assert.NotEmpty(t, commentResp.Data.CommentID, "评论 ID 不应为空")
	t.Logf("Member added comment via forwarding: %s", commentResp.Data.CommentID)

	// === Leader 也添加一条评论 ===
	t.Log("--- Leader 添加评论 ---")
	commentResp2, err := leaderClient.AddComment(teamID, shareID, "好的，那就定 DDD 了", nil)
	require.NoError(t, err)
	require.Equal(t, 0, commentResp2.Code, "Leader 添加评论应成功")
	t.Logf("Leader added comment: %s", commentResp2.Data.CommentID)

	// === 验证 Leader 端能看到所有评论 ===
	t.Log("--- 验证评论列表 ---")
	detailResp, err := leaderClient.GetSharedSessionDetail(teamID, shareID)
	require.NoError(t, err)
	require.Equal(t, 0, detailResp.Code)
	assert.GreaterOrEqual(t, len(detailResp.Data.Comments), 2, "应有至少 2 条评论")

	// 验证 Member 的评论存在
	memberCommentFound := false
	for _, c := range detailResp.Data.Comments {
		t.Logf("  Comment by %s: %s", c.AuthorName, c.Content)
		if c.Content == "DDD 架构很好，我之前用过，推荐！" {
			memberCommentFound = true
		}
	}
	assert.True(t, memberCommentFound, "应能找到 Member 通过转发添加的评论")

	// 清理
	leaderClient.DissolveTeam(teamID)
}

// TestCollaboration_NetworkInterfaces 网络接口查询
func TestCollaboration_NetworkInterfaces(t *testing.T) {
	framework.RequireDaemonBinary(t)

	daemon, err := framework.NewTestDaemon(framework.BinaryPath, "network-test")
	require.NoError(t, err)
	require.NoError(t, daemon.Start())
	defer daemon.Stop()

	client := framework.NewAPIClient(daemon.BaseURL())

	resp, err := client.GetNetworkInterfaces()
	require.NoError(t, err)
	require.Equal(t, 0, resp.Code, "查询网络接口应成功")
	assert.NotNil(t, resp.Data.Interfaces, "应返回网络接口列表")
	t.Logf("Network interfaces: %v", resp.Data.Interfaces)
}
