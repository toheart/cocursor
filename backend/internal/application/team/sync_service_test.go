package team

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/cocursor/backend/internal/application/team/mocks"
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

	teamID := "test-team"

	// 创建 mockery mock 服务
	mockService := mocks.NewMockTeamServiceInterface(t)

	skillIndexStore, err := infraTeam.NewSkillIndexStore(teamID)
	require.NoError(t, err)

	skillIndexStores := map[string]*infraTeam.SkillIndexStore{
		teamID: skillIndexStore,
	}

	// 创建同步服务
	syncService := NewSyncServiceLegacy(mockService, skillIndexStores)

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

	teamID := "test-team"

	// 创建 mockery mock 服务
	mockService := mocks.NewMockTeamServiceInterface(t)

	// 创建同步服务
	syncService := NewSyncService(mockService, nil)

	// 设置回调
	var memberJoined, memberLeft bool
	syncService.SetEventCallbacks(nil, nil, func(tid, memberID string, joined bool) {
		if joined {
			memberJoined = true
		} else {
			memberLeft = true
		}
	}, nil)

	// 测试成员加入事件
	joinEvent, _ := p2p.NewEvent(p2p.EventMemberJoined, teamID, p2p.MemberJoinedPayload{
		MemberID:   "new-member",
		MemberName: "New Member",
		Endpoint:   "192.168.1.50:19960",
		JoinedAt:   time.Now(),
	})

	err = syncService.HandleWebSocketEvent(joinEvent)
	require.NoError(t, err)

	// 验证回调被调用
	assert.True(t, memberJoined)

	// 测试成员离开事件
	leftEvent, _ := p2p.NewEvent(p2p.EventMemberLeft, teamID, p2p.MemberLeftPayload{
		MemberID:   "new-member",
		MemberName: "New Member",
		LeftAt:     time.Now(),
	})

	err = syncService.HandleWebSocketEvent(leftEvent)
	require.NoError(t, err)

	// 验证回调被调用
	assert.True(t, memberLeft)
}

func TestSyncService_HandleTeamDissolved(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sync-service-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	teamID := "test-team"

	// 创建 mockery mock 服务
	mockService := mocks.NewMockTeamServiceInterface(t)

	skillIndexStore, _ := infraTeam.NewSkillIndexStore(teamID)
	skillIndexStores := map[string]*infraTeam.SkillIndexStore{
		teamID: skillIndexStore,
	}

	// 创建同步服务
	syncService := NewSyncServiceLegacy(mockService, skillIndexStores)

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

	teamID := "test-team"

	// 创建 mockery mock 服务
	mockService := mocks.NewMockTeamServiceInterface(t)

	// OnConnect 会调用 UpdateLeaderOnline(teamID, true)
	// 然后启动 goroutine 调用 SyncSkillIndex -> GetTeam -> 返回 IsLeader=true，跳过同步
	// OnDisconnect 会调用 UpdateLeaderOnline(teamID, false)
	mockService.On("UpdateLeaderOnline", teamID, true).Return().Once()
	mockService.On("GetTeam", teamID).Return(&domainTeam.Team{
		ID:       teamID,
		Name:     "Test Team",
		IsLeader: true, // 使 SyncSkillIndex 跳过 HTTP 请求
	}, nil).Maybe()
	mockService.On("UpdateLeaderOnline", teamID, false).Return().Once()

	syncService := NewSyncService(mockService, nil)
	listener := NewEventListener(syncService, mockService)

	// 测试连接回调
	listener.OnConnect(teamID)

	// 等待 OnConnect 启动的 goroutine（SyncSkillIndex）完成
	time.Sleep(100 * time.Millisecond)

	// 验证 UpdateLeaderOnline(true) 被调用
	mockService.AssertCalled(t, "UpdateLeaderOnline", teamID, true)

	// 测试断开回调
	listener.OnDisconnect(teamID, nil)

	// 验证 UpdateLeaderOnline(false) 被调用
	mockService.AssertCalled(t, "UpdateLeaderOnline", teamID, false)

	// 验证所有期望都满足（由 Cleanup 自动检查）
	mock.AssertExpectationsForObjects(t, mockService)
}
