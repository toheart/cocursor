package team

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

func setupTestEnv(t *testing.T) func() {
	tmpDir, err := os.MkdirTemp("", "team-service-test")
	require.NoError(t, err)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	return func() {
		os.Setenv("HOME", oldHome)
		os.RemoveAll(tmpDir)
	}
}

func TestTeamService_Identity(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	service, err := NewTeamService(19960, "1.0.0")
	require.NoError(t, err)
	defer service.Close()

	// 初始无身份
	_, err = service.GetIdentity()
	assert.Error(t, err)

	// 创建身份
	identity, err := service.CreateIdentity("TestUser")
	require.NoError(t, err)
	assert.NotEmpty(t, identity.ID)
	assert.Equal(t, "TestUser", identity.Name)

	// 获取身份
	got, err := service.GetIdentity()
	require.NoError(t, err)
	assert.Equal(t, identity.ID, got.ID)

	// 更新身份
	updated, err := service.UpdateIdentity("NewName")
	require.NoError(t, err)
	assert.Equal(t, "NewName", updated.Name)

	// EnsureIdentity - 已存在时会更新名称
	ensured, err := service.EnsureIdentity("AnotherName")
	require.NoError(t, err)
	assert.Equal(t, "AnotherName", ensured.Name) // 应该更新为新名称
}

func TestTeamService_NetworkInterfaces(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	service, err := NewTeamService(19960, "1.0.0")
	require.NoError(t, err)
	defer service.Close()

	interfaces, err := service.GetNetworkInterfaces()
	require.NoError(t, err)

	t.Logf("Found %d interfaces", len(interfaces))
	for _, iface := range interfaces {
		t.Logf("  %s: %v", iface.Name, iface.Addresses)
	}

	// 设置网卡配置
	if len(interfaces) > 0 && len(interfaces[0].Addresses) > 0 {
		err = service.SetNetworkConfig(interfaces[0].Name, interfaces[0].Addresses[0])
		require.NoError(t, err)

		config := service.GetNetworkConfig()
		require.NotNil(t, config)
		assert.Equal(t, interfaces[0].Name, config.PreferredInterface)
		assert.Equal(t, interfaces[0].Addresses[0], config.PreferredIP)
	}
}

func TestTeamService_CreateTeam(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	service, err := NewTeamService(19960, "1.0.0")
	require.NoError(t, err)
	defer service.Close()

	// 先创建身份
	_, err = service.CreateIdentity("TeamLeader")
	require.NoError(t, err)

	// 获取网卡
	interfaces, _ := service.GetNetworkInterfaces()
	var preferredIF, preferredIP string
	if len(interfaces) > 0 && len(interfaces[0].Addresses) > 0 {
		preferredIF = interfaces[0].Name
		preferredIP = interfaces[0].Addresses[0]
	}

	// 创建团队
	team, err := service.CreateTeam("TestTeam", preferredIF, preferredIP)
	require.NoError(t, err)
	assert.NotEmpty(t, team.ID)
	assert.Equal(t, "TestTeam", team.Name)
	assert.True(t, team.IsLeader)
	assert.Equal(t, 1, team.MemberCount)

	// 获取团队列表
	teams := service.GetTeamList()
	assert.Len(t, teams, 1)
	assert.Equal(t, team.ID, teams[0].ID)

	// 获取成员列表
	members, err := service.GetTeamMembers(team.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.True(t, members[0].IsLeader)
	assert.Equal(t, "TeamLeader", members[0].Name)

	// 不能创建第二个团队
	_, err = service.CreateTeam("SecondTeam", "", "")
	assert.Error(t, err)

	// 解散团队
	err = service.DissolveTeam(context.TODO(), team.ID)
	require.NoError(t, err)

	teams = service.GetTeamList()
	assert.Len(t, teams, 0)
}

func TestTeamService_GetTeam(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	service, err := NewTeamService(19960, "1.0.0")
	require.NoError(t, err)
	defer service.Close()

	// 不存在的团队
	_, err = service.GetTeam("non-existent")
	assert.Error(t, err)

	// 创建身份和团队
	_, _ = service.CreateIdentity("Leader")
	team, _ := service.CreateTeam("MyTeam", "", "")
	defer service.DissolveTeam(context.TODO(), team.ID)

	// 获取存在的团队
	got, err := service.GetTeam(team.ID)
	require.NoError(t, err)
	assert.Equal(t, "MyTeam", got.Name)
}

func TestTeamService_HandleJoinRequest(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	service, err := NewTeamService(19960, "1.0.0")
	require.NoError(t, err)
	defer service.Close()

	// 创建身份和团队
	_, _ = service.CreateIdentity("Leader")
	team, _ := service.CreateTeam("MyTeam", "", "")
	defer service.DissolveTeam(context.TODO(), team.ID)

	// 模拟加入请求
	joinReq := &domainTeam.JoinRequest{
		MemberID:   "new-member-id",
		MemberName: "NewMember",
		Endpoint:   "192.168.1.50:19960",
	}

	resp, err := service.HandleJoinRequest(team.ID, joinReq)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Team)
	assert.Len(t, resp.Members, 2) // Leader + new member

	// 验证成员已添加
	members, _ := service.GetTeamMembers(team.ID)
	assert.Len(t, members, 2)

	// 处理离开请求
	err = service.HandleLeaveRequest(team.ID, "new-member-id")
	require.NoError(t, err)

	members, _ = service.GetTeamMembers(team.ID)
	assert.Len(t, members, 1)
}
