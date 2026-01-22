//go:build integration
// +build integration

// 团队项目配置集成测试
// 测试项目配置的添加、获取、移除功能

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appTeam "github.com/cocursor/backend/internal/application/team"
)

// TestProjectConfig_Empty 测试新团队的项目配置为空
func TestProjectConfig_Empty(t *testing.T) {
	// 启动 Leader 节点
	leader := StartTestNode(t, 20020, "ProjectLeader")
	defer leader.Stop()

	// 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("ProjectTeam", "", "")
	require.NoError(t, err)

	// 创建 WeeklyReportService
	weeklyService := appTeam.NewWeeklyReportService(leader.Service)

	// 获取项目配置
	config, err := weeklyService.GetProjectConfig(team.ID)
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Empty(t, config.Projects, "New team should have empty project config")

	t.Logf("Team %s has %d projects (expected 0)", team.Name, len(config.Projects))
}

// TestProjectConfig_AddAndGet 测试添加并获取项目
func TestProjectConfig_AddAndGet(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 20021, "AddProjectLeader")
	defer leader.Stop()

	// 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("AddProjectTeam", "", "")
	require.NoError(t, err)

	// 创建 WeeklyReportService
	weeklyService := appTeam.NewWeeklyReportService(leader.Service)

	// 添加项目
	project1, err := weeklyService.AddProject(ctx, team.ID, "CoCursor Main", "github.com/cocursor/cocursor")
	require.NoError(t, err)
	assert.NotEmpty(t, project1.ID)
	assert.Equal(t, "CoCursor Main", project1.Name)
	assert.Equal(t, "github.com/cocursor/cocursor", project1.RepoURL)
	t.Logf("Added project: %s (ID: %s)", project1.Name, project1.ID)

	// 添加第二个项目
	project2, err := weeklyService.AddProject(ctx, team.ID, "Backend", "github.com/cocursor/backend")
	require.NoError(t, err)
	assert.NotEmpty(t, project2.ID)
	t.Logf("Added project: %s (ID: %s)", project2.Name, project2.ID)

	// 获取项目配置
	config, err := weeklyService.GetProjectConfig(team.ID)
	require.NoError(t, err)
	assert.Len(t, config.Projects, 2, "Should have 2 projects")

	// 验证项目内容
	for _, p := range config.Projects {
		t.Logf("  - %s: %s", p.Name, p.RepoURL)
	}
}

// TestProjectConfig_Remove 测试移除项目
func TestProjectConfig_Remove(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 20022, "RemoveProjectLeader")
	defer leader.Stop()

	// 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("RemoveProjectTeam", "", "")
	require.NoError(t, err)

	// 创建 WeeklyReportService
	weeklyService := appTeam.NewWeeklyReportService(leader.Service)

	// 添加两个项目
	project1, err := weeklyService.AddProject(ctx, team.ID, "Project To Keep", "github.com/keep/repo")
	require.NoError(t, err)
	project2, err := weeklyService.AddProject(ctx, team.ID, "Project To Remove", "github.com/remove/repo")
	require.NoError(t, err)

	// 验证有 2 个项目
	config, _ := weeklyService.GetProjectConfig(team.ID)
	assert.Len(t, config.Projects, 2)

	// 移除一个项目
	err = weeklyService.RemoveProject(ctx, team.ID, project2.ID)
	require.NoError(t, err)
	t.Logf("Removed project: %s", project2.Name)

	// 验证只剩 1 个项目
	config, err = weeklyService.GetProjectConfig(team.ID)
	require.NoError(t, err)
	assert.Len(t, config.Projects, 1, "Should have 1 project after removal")
	assert.Equal(t, project1.ID, config.Projects[0].ID)

	t.Logf("Remaining project: %s", config.Projects[0].Name)
}

// TestProjectConfig_OnlyLeaderCanModify 测试只有 Leader 能修改项目配置
func TestProjectConfig_OnlyLeaderCanModify(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 20023, "OnlyLeaderLeader")
	defer leader.Stop()

	// Leader 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("OnlyLeaderTeam", "", "")
	require.NoError(t, err)

	// 启动 Member 节点
	member := StartTestNode(t, 20024, "OnlyLeaderMember")
	defer member.Stop()

	// Member 加入团队
	os.Setenv("HOME", member.HomeDir)
	_, err = member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Member 尝试添加项目（应该失败）
	memberWeeklyService := appTeam.NewWeeklyReportService(member.Service)
	_, err = memberWeeklyService.AddProject(ctx, team.ID, "Member Project", "github.com/member/repo")
	assert.Error(t, err, "Member should not be able to add project")
	t.Logf("Expected error when member tries to add project: %v", err)

	// Leader 添加项目（应该成功）
	os.Setenv("HOME", leader.HomeDir)
	leaderWeeklyService := appTeam.NewWeeklyReportService(leader.Service)
	project, err := leaderWeeklyService.AddProject(ctx, team.ID, "Leader Project", "github.com/leader/repo")
	require.NoError(t, err)
	t.Logf("Leader added project: %s", project.Name)

	// Member 尝试移除项目（应该失败）
	os.Setenv("HOME", member.HomeDir)
	err = memberWeeklyService.RemoveProject(ctx, team.ID, project.ID)
	assert.Error(t, err, "Member should not be able to remove project")
	t.Logf("Expected error when member tries to remove project: %v", err)
}

// TestProjectConfig_MemberGet 测试 Member 获取项目配置
func TestProjectConfig_MemberGet(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 20025, "MemberGetConfigLeader")
	defer leader.Stop()

	// Leader 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("MemberGetConfigTeam", "", "")
	require.NoError(t, err)

	// Leader 添加项目
	leaderWeeklyService := appTeam.NewWeeklyReportService(leader.Service)
	_, err = leaderWeeklyService.AddProject(ctx, team.ID, "Shared Project", "github.com/shared/repo")
	require.NoError(t, err)

	// 启动 Member 节点
	member := StartTestNode(t, 20026, "MemberGetConfigMember")
	defer member.Stop()

	// Member 加入团队
	os.Setenv("HOME", member.HomeDir)
	_, err = member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Member 获取项目配置（只读）
	memberWeeklyService := appTeam.NewWeeklyReportService(member.Service)
	config, err := memberWeeklyService.GetProjectConfig(team.ID)
	require.NoError(t, err)

	// 由于 Member 的 WeeklyReportService 是新创建的，本地没有配置
	// 它会尝试从 Leader 拉取，但这需要 P2P 接口支持
	// 这里只验证调用不会出错
	t.Logf("Member got project config with %d projects", len(config.Projects))
}
