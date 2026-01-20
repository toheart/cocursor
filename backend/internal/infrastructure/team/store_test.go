package team

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

func setupTestDir(t *testing.T) (string, func()) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "team-store-test")
	require.NoError(t, err)

	// 设置 HOME 环境变量
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	cleanup := func() {
		os.Setenv("HOME", oldHome)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestIdentityStore(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// 创建存储
	store, err := NewIdentityStore()
	require.NoError(t, err)

	// 初始状态：无身份
	assert.False(t, store.Exists())
	_, err = store.Get()
	assert.ErrorIs(t, err, domainTeam.ErrIdentityNotFound)

	// 创建身份
	identity, err := store.Create("TestUser")
	require.NoError(t, err)
	assert.NotEmpty(t, identity.ID)
	assert.Equal(t, "TestUser", identity.Name)
	assert.True(t, store.Exists())

	// 获取身份
	got, err := store.Get()
	require.NoError(t, err)
	assert.Equal(t, identity.ID, got.ID)
	assert.Equal(t, identity.Name, got.Name)

	// 更新名称
	updated, err := store.UpdateName("NewName")
	require.NoError(t, err)
	assert.Equal(t, "NewName", updated.Name)
	assert.Equal(t, identity.ID, updated.ID)

	// 删除
	err = store.Delete()
	require.NoError(t, err)
	assert.False(t, store.Exists())
}

func TestTeamStore(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// 创建存储
	store, err := NewTeamStore()
	require.NoError(t, err)

	// 初始状态：无团队
	assert.Equal(t, 0, store.Count())
	assert.False(t, store.HasLeaderTeam())

	// 添加团队
	team1 := &domainTeam.Team{
		ID:             "team-1",
		Name:           "Test Team 1",
		LeaderID:       "leader-1",
		LeaderName:     "Leader One",
		LeaderEndpoint: "192.168.1.100:19960",
		IsLeader:       true,
		JoinedAt:       time.Now(),
		CreatedAt:      time.Now(),
	}
	err = store.Add(team1)
	require.NoError(t, err)

	// 获取团队
	got, err := store.Get("team-1")
	require.NoError(t, err)
	assert.Equal(t, team1.ID, got.ID)
	assert.Equal(t, team1.Name, got.Name)
	assert.True(t, got.IsLeader)

	// 检查 Leader 团队
	assert.True(t, store.HasLeaderTeam())
	leaderTeam := store.GetLeaderTeam()
	assert.NotNil(t, leaderTeam)
	assert.Equal(t, "team-1", leaderTeam.ID)

	// 添加另一个团队（非 Leader）
	team2 := &domainTeam.Team{
		ID:             "team-2",
		Name:           "Test Team 2",
		LeaderID:       "leader-2",
		LeaderName:     "Leader Two",
		LeaderEndpoint: "192.168.1.101:19960",
		IsLeader:       false,
		JoinedAt:       time.Now().Add(time.Hour),
		CreatedAt:      time.Now(),
	}
	err = store.Add(team2)
	require.NoError(t, err)

	// 列表（Leader 团队应在前）
	teams := store.List()
	assert.Len(t, teams, 2)
	assert.Equal(t, "team-1", teams[0].ID) // Leader 团队在前

	// 移除团队
	err = store.Remove("team-1")
	require.NoError(t, err)
	assert.Equal(t, 1, store.Count())
	assert.False(t, store.HasLeaderTeam())
}

func TestMemberStore(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	teamID := "test-team"

	// 确保目录存在
	teamDir := filepath.Join(tmpDir, ".cocursor", "team", teamID)
	err := os.MkdirAll(teamDir, 0755)
	require.NoError(t, err)

	// 创建存储
	store, err := NewMemberStore(teamID)
	require.NoError(t, err)

	// 初始状态
	assert.Equal(t, 0, store.Count())

	// 添加 Leader
	err = store.AddLeader("leader-1", "Leader", "192.168.1.100:19960")
	require.NoError(t, err)

	// 添加成员
	member := &domainTeam.TeamMember{
		ID:       "member-1",
		Name:     "Member One",
		Endpoint: "192.168.1.101:19960",
		IsLeader: false,
		IsOnline: true,
		JoinedAt: time.Now(),
	}
	err = store.Add(member)
	require.NoError(t, err)

	assert.Equal(t, 2, store.Count())

	// 获取成员
	got, err := store.Get("member-1")
	require.NoError(t, err)
	assert.Equal(t, "Member One", got.Name)

	// 列表（Leader 在前）
	members := store.List()
	assert.Len(t, members, 2)
	assert.True(t, members[0].IsLeader)

	// 设置离线
	err = store.SetOnline("member-1", false)
	require.NoError(t, err)

	got, _ = store.Get("member-1")
	assert.False(t, got.IsOnline)

	// 在线数量
	assert.Equal(t, 1, store.OnlineCount()) // 只有 Leader 在线

	// 移除成员
	err = store.Remove("member-1")
	require.NoError(t, err)
	assert.Equal(t, 1, store.Count())
}

func TestSkillIndexStore(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	teamID := "test-team"

	// 确保目录存在
	teamDir := filepath.Join(tmpDir, ".cocursor", "team", teamID)
	err := os.MkdirAll(teamDir, 0755)
	require.NoError(t, err)

	// 创建存储
	store, err := NewSkillIndexStore(teamID)
	require.NoError(t, err)

	// 初始状态
	assert.Equal(t, 0, store.Count())

	// 添加技能
	skill1 := domainTeam.TeamSkillEntry{
		PluginID:       "skill-1",
		Name:           "Skill One",
		Description:    "Description one",
		Version:        "1.0.0",
		AuthorID:       "author-1",
		AuthorName:     "Author One",
		AuthorEndpoint: "192.168.1.100:19960",
		PublishedAt:    time.Now(),
	}
	err = store.AddOrUpdate(skill1)
	require.NoError(t, err)

	assert.Equal(t, 1, store.Count())

	// 获取技能
	got := store.GetSkill("skill-1")
	require.NotNil(t, got)
	assert.Equal(t, "Skill One", got.Name)

	// 更新技能
	skill1.Version = "2.0.0"
	err = store.AddOrUpdate(skill1)
	require.NoError(t, err)

	got = store.GetSkill("skill-1")
	assert.Equal(t, "2.0.0", got.Version)

	// 添加第二个技能
	skill2 := domainTeam.TeamSkillEntry{
		PluginID:       "skill-2",
		Name:           "Skill Two",
		AuthorID:       "author-1",
		AuthorName:     "Author One",
		AuthorEndpoint: "192.168.1.100:19960",
		PublishedAt:    time.Now(),
	}
	err = store.AddOrUpdate(skill2)
	require.NoError(t, err)

	// 按作者查找
	authorSkills := store.FindByAuthor("author-1")
	assert.Len(t, authorSkills, 2)

	// 移除技能
	err = store.Remove("skill-1")
	require.NoError(t, err)
	assert.Equal(t, 1, store.Count())
	assert.Nil(t, store.GetSkill("skill-1"))
}

func TestNetworkConfigStore(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// 创建存储
	store, err := NewNetworkConfigStore()
	require.NoError(t, err)

	// 初始状态：无配置
	assert.Nil(t, store.Get())
	assert.Empty(t, store.GetPreferredInterface())
	assert.Empty(t, store.GetPreferredIP())

	// 设置配置
	err = store.Set("en0", "192.168.1.100")
	require.NoError(t, err)

	// 获取配置
	config := store.Get()
	require.NotNil(t, config)
	assert.Equal(t, "en0", config.PreferredInterface)
	assert.Equal(t, "192.168.1.100", config.PreferredIP)

	// 获取单独字段
	assert.Equal(t, "en0", store.GetPreferredInterface())
	assert.Equal(t, "192.168.1.100", store.GetPreferredIP())

	// 清除配置
	err = store.Clear()
	require.NoError(t, err)
	assert.Nil(t, store.Get())
}
