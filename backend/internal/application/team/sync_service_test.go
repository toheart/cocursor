package team

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cocursor/backend/internal/domain/p2p"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
	infraTeam "github.com/cocursor/backend/internal/infrastructure/team"
)

func TestSyncService_HandleSkillEvents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sync-service-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// 创建存储
	teamStore, err := infraTeam.NewTeamStore()
	require.NoError(t, err)

	teamID := "test-team"
	skillIndexStore, err := infraTeam.NewSkillIndexStore(teamID)
	require.NoError(t, err)

	skillIndexStores := map[string]*infraTeam.SkillIndexStore{
		teamID: skillIndexStore,
	}

	// 创建同步服务
	syncService := NewSyncService(teamStore, skillIndexStores)

	// 设置回调
	var updatedSkill *domainTeam.TeamSkillEntry
	var deletedPluginID string
	syncService.SetEventCallbacks(
		func(tid string, entry *domainTeam.TeamSkillEntry) {
			updatedSkill = entry
		},
		func(tid, pluginID string) {
			deletedPluginID = pluginID
		},
		nil,
		nil,
	)

	// 测试技能发布事件
	publishEvent, err := p2p.NewEvent(p2p.EventSkillPublished, teamID, p2p.SkillPublishedPayload{
		PluginID:       "skill-1",
		Name:           "Test Skill",
		Description:    "A test skill",
		Version:        "1.0.0",
		AuthorID:       "author-1",
		AuthorName:     "Author",
		AuthorEndpoint: "192.168.1.100:19960",
		FileCount:      5,
		TotalSize:      1024,
		Checksum:       "abc123",
		PublishedAt:    time.Now(),
	})
	require.NoError(t, err)

	err = syncService.HandleWebSocketEvent(publishEvent)
	require.NoError(t, err)

	// 验证技能已添加
	skill := skillIndexStore.GetSkill("skill-1")
	require.NotNil(t, skill)
	assert.Equal(t, "Test Skill", skill.Name)
	assert.Equal(t, "1.0.0", skill.Version)

	// 验证回调被调用
	require.NotNil(t, updatedSkill)
	assert.Equal(t, "skill-1", updatedSkill.PluginID)

	// 测试技能更新事件
	updateEvent, _ := p2p.NewEvent(p2p.EventSkillUpdated, teamID, p2p.SkillUpdatedPayload{
		PluginID:  "skill-1",
		Version:   "2.0.0",
		Checksum:  "def456",
		UpdatedAt: time.Now(),
	})

	err = syncService.HandleWebSocketEvent(updateEvent)
	require.NoError(t, err)

	skill = skillIndexStore.GetSkill("skill-1")
	assert.Equal(t, "2.0.0", skill.Version)

	// 测试技能删除事件
	deleteEvent, _ := p2p.NewEvent(p2p.EventSkillDeleted, teamID, p2p.SkillDeletedPayload{
		PluginID:  "skill-1",
		DeletedBy: "author-1",
		DeletedAt: time.Now(),
	})

	err = syncService.HandleWebSocketEvent(deleteEvent)
	require.NoError(t, err)

	skill = skillIndexStore.GetSkill("skill-1")
	assert.Nil(t, skill)

	// 验证删除回调
	assert.Equal(t, "skill-1", deletedPluginID)
}

func TestSyncService_HandleMemberEvents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sync-service-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// 创建存储
	teamStore, err := infraTeam.NewTeamStore()
	require.NoError(t, err)

	teamID := "test-team"

	// 添加一个团队
	team := &domainTeam.Team{
		ID:          teamID,
		Name:        "Test Team",
		MemberCount: 2,
		IsLeader:    false,
		JoinedAt:    time.Now(),
		CreatedAt:   time.Now(),
	}
	teamStore.Add(team)

	// 创建同步服务
	syncService := NewSyncService(teamStore, nil)

	// 测试成员加入事件
	joinEvent, _ := p2p.NewEvent(p2p.EventMemberJoined, teamID, p2p.MemberJoinedPayload{
		MemberID:   "new-member",
		MemberName: "New Member",
		Endpoint:   "192.168.1.50:19960",
		JoinedAt:   time.Now(),
	})

	err = syncService.HandleWebSocketEvent(joinEvent)
	require.NoError(t, err)

	// 验证成员数量增加
	updatedTeam, _ := teamStore.Get(teamID)
	assert.Equal(t, 3, updatedTeam.MemberCount)

	// 测试成员离开事件
	leftEvent, _ := p2p.NewEvent(p2p.EventMemberLeft, teamID, p2p.MemberLeftPayload{
		MemberID:   "new-member",
		MemberName: "New Member",
		LeftAt:     time.Now(),
	})

	err = syncService.HandleWebSocketEvent(leftEvent)
	require.NoError(t, err)

	updatedTeam, _ = teamStore.Get(teamID)
	assert.Equal(t, 2, updatedTeam.MemberCount)
}

func TestSyncService_HandleTeamDissolved(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sync-service-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// 创建存储
	teamStore, err := infraTeam.NewTeamStore()
	require.NoError(t, err)

	teamID := "test-team"

	// 添加团队
	team := &domainTeam.Team{
		ID:       teamID,
		Name:     "Test Team",
		IsLeader: false,
		JoinedAt: time.Now(),
	}
	teamStore.Add(team)

	skillIndexStore, _ := infraTeam.NewSkillIndexStore(teamID)
	skillIndexStores := map[string]*infraTeam.SkillIndexStore{
		teamID: skillIndexStore,
	}

	// 创建同步服务
	syncService := NewSyncService(teamStore, skillIndexStores)

	// 设置回调
	var dissolvedTeamID string
	syncService.SetEventCallbacks(nil, nil, nil, func(tid string) {
		dissolvedTeamID = tid
	})

	// 发送解散事件
	dissolveEvent, _ := p2p.NewEvent(p2p.EventTeamDissolved, teamID, p2p.TeamDissolvedPayload{
		TeamID:      teamID,
		TeamName:    "Test Team",
		DissolvedBy: "leader",
		DissolvedAt: time.Now(),
	})

	err = syncService.HandleWebSocketEvent(dissolveEvent)
	require.NoError(t, err)

	// 验证团队已移除
	_, err = teamStore.Get(teamID)
	assert.Error(t, err)

	// 验证回调被调用
	assert.Equal(t, teamID, dissolvedTeamID)
}

func TestEventListener(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "event-listener-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	teamStore, _ := infraTeam.NewTeamStore()

	teamID := "test-team"
	team := &domainTeam.Team{
		ID:           teamID,
		Name:         "Test Team",
		IsLeader:     false,
		LeaderOnline: false,
		JoinedAt:     time.Now(),
	}
	teamStore.Add(team)

	syncService := NewSyncService(teamStore, nil)
	listener := NewEventListener(syncService, teamStore)

	// 测试连接回调
	listener.OnConnect(teamID)
	updatedTeam, _ := teamStore.Get(teamID)
	assert.True(t, updatedTeam.LeaderOnline)

	// 测试断开回调
	listener.OnDisconnect(teamID, nil)
	updatedTeam, _ = teamStore.Get(teamID)
	assert.False(t, updatedTeam.LeaderOnline)
}
