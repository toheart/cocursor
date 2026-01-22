//go:build integration
// +build integration

// 团队加入功能集成测试
// 测试多节点场景下的团队创建、加入、离开流程

package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appTeam "github.com/cocursor/backend/internal/application/team"
	infraP2P "github.com/cocursor/backend/internal/infrastructure/p2p"
	httpHandler "github.com/cocursor/backend/internal/interfaces/http/handler"
	"github.com/cocursor/backend/internal/interfaces/p2p/handler"
)

// TestNode 测试节点，封装一个完整的团队服务节点
type TestNode struct {
	Service              *appTeam.TeamService
	CollaborationService *appTeam.CollaborationService
	WeeklyReportService  *appTeam.WeeklyReportService
	P2PHandler           *handler.P2PTeamHandler
	Server               *http.Server
	Port                 int
	HomeDir              string
	Name                 string

	// 原始 HOME 环境变量，用于恢复
	originalHome string
}

// StartTestNode 启动一个测试节点
// 创建隔离的 HOME 目录，启动 TeamService 和完整的 HTTP Server
func StartTestNode(t *testing.T, port int, name string) *TestNode {
	t.Helper()

	// 创建临时 HOME 目录
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("team-test-%s-", name))
	require.NoError(t, err)

	// 保存原始 HOME 并设置新 HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	// 创建 TeamService
	service, err := appTeam.NewTeamService(port, "1.0.0-test")
	if err != nil {
		os.Setenv("HOME", originalHome)
		os.RemoveAll(tmpDir)
		require.NoError(t, err)
	}

	// 创建身份
	_, err = service.EnsureIdentity(name)
	if err != nil {
		service.Close()
		os.Setenv("HOME", originalHome)
		os.RemoveAll(tmpDir)
		require.NoError(t, err)
	}

	// 创建协作服务和周报服务
	collaborationService := appTeam.NewCollaborationService(service, nil)
	weeklyReportService := appTeam.NewWeeklyReportService(service)

	// 获取 WebSocket Server（用于 P2P Handler）
	wsServer := service.GetWebSocketServer()
	var wsServerTyped *infraP2P.WebSocketServer
	if wsServer != nil {
		wsServerTyped = wsServer.(*infraP2P.WebSocketServer)
	}

	// 创建 P2P Handler
	p2pHandler := handler.NewP2PTeamHandler(service, wsServerTyped)

	// 创建 gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// 注册 P2P 路由（用于团队加入等）
	p2pHandler.RegisterRoutes(router)

	// 注册 API 路由（用于协作功能）
	api := router.Group("/api/v1")
	team := api.Group("/team")
	{
		// 协作功能
		collaborationHandler := httpHandler.NewTeamCollaborationHandler(service, collaborationService)
		team.POST("/:id/share-code", collaborationHandler.ShareCode)
		team.POST("/:id/status", collaborationHandler.UpdateWorkStatus)
		team.POST("/:id/daily-summaries/share", collaborationHandler.ShareDailySummary)
		team.GET("/:id/daily-summaries", collaborationHandler.GetDailySummaries)
		team.GET("/:id/daily-summaries/:member_id", collaborationHandler.GetDailySummaryDetail)

		// 周报功能
		weeklyReportHandler := httpHandler.NewTeamWeeklyReportHandler(weeklyReportService)
		team.GET("/:id/project-config", weeklyReportHandler.GetProjectConfig)
		team.POST("/:id/project-config", weeklyReportHandler.UpdateProjectConfig)
		team.POST("/:id/project-config/add", weeklyReportHandler.AddProject)
		team.POST("/:id/project-config/remove", weeklyReportHandler.RemoveProject)
	}

	// 创建 HTTP Server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	// 启动 HTTP Server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("HTTP server error: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 恢复原始 HOME（让其他代码使用正常 HOME）
	// 但保留 tmpDir 用于清理
	os.Setenv("HOME", originalHome)

	return &TestNode{
		Service:              service,
		CollaborationService: collaborationService,
		WeeklyReportService:  weeklyReportService,
		P2PHandler:           p2pHandler,
		Server:               server,
		Port:                 port,
		HomeDir:              tmpDir,
		Name:                 name,
		originalHome:         originalHome,
	}
}

// Endpoint 返回节点的网络端点
func (n *TestNode) Endpoint() string {
	return fmt.Sprintf("127.0.0.1:%d", n.Port)
}

// Stop 停止节点并清理资源
func (n *TestNode) Stop() {
	// 关闭 HTTP Server
	if n.Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		n.Server.Shutdown(ctx)
	}

	// 关闭 TeamService
	if n.Service != nil {
		n.Service.Close()
	}

	// 删除临时目录
	if n.HomeDir != "" {
		os.RemoveAll(n.HomeDir)
	}
}

// TestTeamJoin_FullFlow 测试完整的团队加入流程
// Leader 创建团队 -> Member 通过 endpoint 加入 -> 验证双方状态
func TestTeamJoin_FullFlow(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 19980, "Leader")
	defer leader.Stop()

	// Leader 创建团队
	// 需要临时切换 HOME 到 leader 的目录
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("TestTeam", "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, team.ID)
	assert.Equal(t, "TestTeam", team.Name)
	assert.True(t, team.IsLeader)

	t.Logf("Leader created team: %s (ID: %s)", team.Name, team.ID)
	t.Logf("Leader endpoint: %s", leader.Endpoint())

	// 启动 Member 节点
	member := StartTestNode(t, 19981, "Member")
	defer member.Stop()

	// Member 通过 endpoint 加入团队
	os.Setenv("HOME", member.HomeDir)
	joinedTeam, err := member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)
	assert.Equal(t, team.ID, joinedTeam.ID)
	assert.Equal(t, "TestTeam", joinedTeam.Name)
	assert.False(t, joinedTeam.IsLeader)

	t.Logf("Member joined team: %s", joinedTeam.Name)

	// 等待状态同步
	time.Sleep(200 * time.Millisecond)

	// 验证 Leader 端成员列表包含 Member
	os.Setenv("HOME", leader.HomeDir)
	members, err := leader.Service.GetTeamMembers(team.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2, "Leader should see 2 members")

	// 验证成员角色
	var hasLeader, hasMember bool
	for _, m := range members {
		t.Logf("Member: %s (Leader: %v)", m.Name, m.IsLeader)
		if m.IsLeader {
			hasLeader = true
			assert.Equal(t, "Leader", m.Name)
		} else {
			hasMember = true
			assert.Equal(t, "Member", m.Name)
		}
	}
	assert.True(t, hasLeader, "Should have leader in members")
	assert.True(t, hasMember, "Should have member in members")

	// 验证 Member 端团队列表
	os.Setenv("HOME", member.HomeDir)
	memberTeams := member.Service.GetTeamList()
	assert.Len(t, memberTeams, 1, "Member should have 1 team")
	assert.Equal(t, team.ID, memberTeams[0].ID)
}

// TestTeamJoin_LeaveFlow 测试成员离开团队流程
func TestTeamJoin_LeaveFlow(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 19982, "LeaderForLeave")
	defer leader.Stop()

	// Leader 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("LeaveTestTeam", "", "")
	require.NoError(t, err)

	// 启动 Member 节点
	member := StartTestNode(t, 19983, "MemberForLeave")
	defer member.Stop()

	// Member 加入团队
	os.Setenv("HOME", member.HomeDir)
	_, err = member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)

	// 等待同步
	time.Sleep(200 * time.Millisecond)

	// 验证加入成功
	os.Setenv("HOME", leader.HomeDir)
	members, _ := leader.Service.GetTeamMembers(team.ID)
	assert.Len(t, members, 2)

	// Member 离开团队
	os.Setenv("HOME", member.HomeDir)
	err = member.Service.LeaveTeam(ctx, team.ID)
	require.NoError(t, err)

	t.Log("Member left the team")

	// 等待同步
	time.Sleep(200 * time.Millisecond)

	// 验证 Leader 端成员列表只剩 Leader
	os.Setenv("HOME", leader.HomeDir)
	members, err = leader.Service.GetTeamMembers(team.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1, "Should only have leader after member left")
	assert.True(t, members[0].IsLeader)

	// 验证 Member 端团队列表为空
	os.Setenv("HOME", member.HomeDir)
	memberTeams := member.Service.GetTeamList()
	assert.Len(t, memberTeams, 0, "Member should have no teams after leaving")
}

// TestTeamJoin_MultipleMembers 测试多成员加入
func TestTeamJoin_MultipleMembers(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 19984, "MultiLeader")
	defer leader.Stop()

	// Leader 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("MultiMemberTeam", "", "")
	require.NoError(t, err)

	// 启动多个 Member 节点
	member1 := StartTestNode(t, 19985, "Member1")
	defer member1.Stop()

	member2 := StartTestNode(t, 19986, "Member2")
	defer member2.Stop()

	// Member1 加入
	os.Setenv("HOME", member1.HomeDir)
	_, err = member1.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)

	// Member2 加入
	os.Setenv("HOME", member2.HomeDir)
	_, err = member2.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)

	// 等待同步
	time.Sleep(300 * time.Millisecond)

	// 验证 Leader 端成员列表包含 3 人
	os.Setenv("HOME", leader.HomeDir)
	members, err := leader.Service.GetTeamMembers(team.ID)
	require.NoError(t, err)
	assert.Len(t, members, 3, "Should have 3 members (leader + 2 members)")

	// 记录成员信息
	for _, m := range members {
		t.Logf("Team member: %s (Leader: %v)", m.Name, m.IsLeader)
	}
}

// TestTeamDissolve 测试 Leader 解散团队
// Leader 解散团队后，Member 的团队列表应该清空
func TestTeamDissolve(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 19987, "DissolveLeader")
	defer leader.Stop()

	// Leader 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("DissolveTeam", "", "")
	require.NoError(t, err)

	// 启动 Member 节点
	member := StartTestNode(t, 19988, "DissolveMember")
	defer member.Stop()

	// Member 加入团队
	os.Setenv("HOME", member.HomeDir)
	_, err = member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)

	// 等待同步
	time.Sleep(200 * time.Millisecond)

	// 验证加入成功
	os.Setenv("HOME", leader.HomeDir)
	members, _ := leader.Service.GetTeamMembers(team.ID)
	assert.Len(t, members, 2)

	// Leader 解散团队
	err = leader.Service.DissolveTeam(ctx, team.ID)
	require.NoError(t, err)

	t.Log("Leader dissolved the team")

	// 验证 Leader 端团队列表为空
	leaderTeams := leader.Service.GetTeamList()
	assert.Len(t, leaderTeams, 0, "Leader should have no teams after dissolving")

	// 注意：Member 端需要通过 WebSocket 事件收到解散通知
	// 在没有 WebSocket 连接的测试环境下，Member 可能不会立即感知到解散
	// 这里只验证 Leader 端状态
}

// TestTeamJoin_Rejoin 测试成员离开后重新加入
func TestTeamJoin_Rejoin(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 19989, "RejoinLeader")
	defer leader.Stop()

	// Leader 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("RejoinTeam", "", "")
	require.NoError(t, err)

	// 启动 Member 节点
	member := StartTestNode(t, 19990, "RejoinMember")
	defer member.Stop()

	// 第一次加入
	os.Setenv("HOME", member.HomeDir)
	_, err = member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)
	t.Log("Member joined for the first time")

	time.Sleep(200 * time.Millisecond)

	// 验证加入成功
	os.Setenv("HOME", leader.HomeDir)
	members, _ := leader.Service.GetTeamMembers(team.ID)
	assert.Len(t, members, 2)

	// 离开团队
	os.Setenv("HOME", member.HomeDir)
	err = member.Service.LeaveTeam(ctx, team.ID)
	require.NoError(t, err)
	t.Log("Member left the team")

	time.Sleep(200 * time.Millisecond)

	// 验证离开成功
	os.Setenv("HOME", leader.HomeDir)
	members, _ = leader.Service.GetTeamMembers(team.ID)
	assert.Len(t, members, 1)

	// 重新加入
	os.Setenv("HOME", member.HomeDir)
	rejoinedTeam, err := member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)
	assert.Equal(t, team.ID, rejoinedTeam.ID)
	t.Log("Member rejoined the team")

	time.Sleep(200 * time.Millisecond)

	// 验证重新加入成功
	os.Setenv("HOME", leader.HomeDir)
	members, err = leader.Service.GetTeamMembers(team.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2, "Should have 2 members after rejoin")
}

// TestTeamJoin_InvalidEndpoint 测试加入不存在的团队端点
func TestTeamJoin_InvalidEndpoint(t *testing.T) {
	ctx := context.Background()

	// 启动 Member 节点
	member := StartTestNode(t, 19991, "InvalidMember")
	defer member.Stop()

	// 尝试加入不存在的端点
	os.Setenv("HOME", member.HomeDir)
	_, err := member.Service.JoinTeam(ctx, "127.0.0.1:19999")

	// 应该返回错误
	assert.Error(t, err, "Should fail to join non-existent endpoint")
	t.Logf("Expected error: %v", err)

	// 验证 Member 端团队列表仍然为空
	memberTeams := member.Service.GetTeamList()
	assert.Len(t, memberTeams, 0, "Member should have no teams after failed join")
}

// TestTeamJoin_DuplicateJoin 测试重复加入同一团队
func TestTeamJoin_DuplicateJoin(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 19992, "DupLeader")
	defer leader.Stop()

	// Leader 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("DuplicateTeam", "", "")
	require.NoError(t, err)

	// 启动 Member 节点
	member := StartTestNode(t, 19993, "DupMember")
	defer member.Stop()

	// 第一次加入
	os.Setenv("HOME", member.HomeDir)
	_, err = member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// 第二次加入同一团队（应该成功或幂等处理）
	joinedTeam, err := member.Service.JoinTeam(ctx, leader.Endpoint())
	// 根据实现，可能成功（幂等）或返回 ErrAlreadyTeamMember
	if err == nil {
		assert.Equal(t, team.ID, joinedTeam.ID)
		t.Log("Duplicate join succeeded (idempotent)")
	} else {
		t.Logf("Duplicate join returned error (expected): %v", err)
	}

	// 无论如何，成员列表应该只有 2 人
	os.Setenv("HOME", leader.HomeDir)
	members, _ := leader.Service.GetTeamMembers(team.ID)
	assert.Len(t, members, 2, "Should still have 2 members after duplicate join")
}

// TestTeamIdentity 测试身份创建和更新
func TestTeamIdentity(t *testing.T) {
	// 创建临时 HOME 目录
	tmpDir, err := os.MkdirTemp("", "identity-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// 创建 TeamService
	service, err := appTeam.NewTeamService(19994, "1.0.0-test")
	require.NoError(t, err)
	defer service.Close()

	// 初始无身份
	_, err = service.GetIdentity()
	assert.Error(t, err, "Should have no identity initially")

	// 创建身份
	identity, err := service.CreateIdentity("TestUser")
	require.NoError(t, err)
	assert.NotEmpty(t, identity.ID)
	assert.Equal(t, "TestUser", identity.Name)
	t.Logf("Created identity: %s (ID: %s)", identity.Name, identity.ID)

	// 获取身份
	got, err := service.GetIdentity()
	require.NoError(t, err)
	assert.Equal(t, identity.ID, got.ID)
	assert.Equal(t, "TestUser", got.Name)

	// 更新身份名称
	updated, err := service.UpdateIdentity("NewUserName")
	require.NoError(t, err)
	assert.Equal(t, identity.ID, updated.ID, "ID should not change")
	assert.Equal(t, "NewUserName", updated.Name)
	t.Logf("Updated identity name to: %s", updated.Name)

	// 验证更新后的身份
	got, err = service.GetIdentity()
	require.NoError(t, err)
	assert.Equal(t, "NewUserName", got.Name)
}

// TestTeamCreate_OnlyOneTeam 测试只能创建一个团队
func TestTeamCreate_OnlyOneTeam(t *testing.T) {
	// 启动节点
	node := StartTestNode(t, 19995, "SingleTeamLeader")
	defer node.Stop()

	// 创建第一个团队
	os.Setenv("HOME", node.HomeDir)
	team1, err := node.Service.CreateTeam("FirstTeam", "", "")
	require.NoError(t, err)
	t.Logf("Created first team: %s", team1.Name)

	// 尝试创建第二个团队（应该失败）
	_, err = node.Service.CreateTeam("SecondTeam", "", "")
	assert.Error(t, err, "Should not be able to create second team")
	t.Logf("Expected error when creating second team: %v", err)

	// 验证只有一个团队
	teams := node.Service.GetTeamList()
	assert.Len(t, teams, 1)
	assert.Equal(t, "FirstTeam", teams[0].Name)
}
