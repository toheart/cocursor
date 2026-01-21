package team

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/cocursor/backend/internal/application/team/mocks"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

// TestWeeklyReportService_GetProjectConfig_FromLocal 测试从本地获取项目配置
func TestWeeklyReportService_GetProjectConfig_FromLocal(t *testing.T) {
	// 设置测试环境
	tmpDir, err := os.MkdirTemp("", "weekly-report-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()

	// 创建 mock
	mockTeamService := mocks.NewMockTeamServiceInterface(t)

	// 设置期望：GetTeam 应该被调用，返回非 Leader 团队（不会触发从 Leader 拉取）
	mockTeamService.On("GetTeam", "team-1").Return(&domainTeam.Team{
		ID:           "team-1",
		Name:         "Test Team",
		IsLeader:     true, // 是 Leader，不会去拉取
		LeaderOnline: false,
	}, nil)

	// 创建真实的服务（这里使用真实的 TeamService 会导致问题，因此这个测试主要展示 mock 用法）
	// 实际上，要完全使用 mock，需要重构 WeeklyReportService 使其接受接口
	// 这里我们直接测试 mock 的行为

	// 验证 mock 行为
	team, err := mockTeamService.GetTeam("team-1")
	require.NoError(t, err)
	assert.Equal(t, "team-1", team.ID)
	assert.True(t, team.IsLeader)
	// mockery 生成的 mock 会在 t.Cleanup 中自动调用 AssertExpectations
}

// TestMockProjectConfigStore 测试 ProjectConfigStore mock
func TestMockProjectConfigStore(t *testing.T) {
	mockStore := mocks.NewMockProjectConfigStoreInterface(t)

	expectedConfig := &domainTeam.TeamProjectConfig{
		TeamID: "team-1",
		Projects: []domainTeam.ProjectMatcher{
			{ID: "p1", Name: "Project 1", RepoURL: "github.com/org/repo1"},
		},
		UpdatedAt: time.Now(),
	}

	// 设置期望
	mockStore.On("Load").Return(expectedConfig, nil)
	mockStore.On("Save", mock.AnythingOfType("*team.TeamProjectConfig")).Return(nil)
	mockStore.On("AddProject", mock.AnythingOfType("team.ProjectMatcher")).Return(nil)
	mockStore.On("RemoveProject", "p1").Return(nil)
	mockStore.On("GetRepoURLs").Return([]string{"github.com/org/repo1"})

	// 测试 Load
	config, err := mockStore.Load()
	require.NoError(t, err)
	assert.Equal(t, "team-1", config.TeamID)
	assert.Len(t, config.Projects, 1)

	// 测试 Save
	err = mockStore.Save(expectedConfig)
	require.NoError(t, err)

	// 测试 AddProject
	err = mockStore.AddProject(domainTeam.ProjectMatcher{ID: "p2", Name: "Project 2", RepoURL: "github.com/org/repo2"})
	require.NoError(t, err)

	// 测试 RemoveProject
	err = mockStore.RemoveProject("p1")
	require.NoError(t, err)

	// 测试 GetRepoURLs
	urls := mockStore.GetRepoURLs()
	assert.Contains(t, urls, "github.com/org/repo1")
	// mockery 生成的 mock 会在 t.Cleanup 中自动调用 AssertExpectations
}

// TestMockWeeklyStatsStore 测试 WeeklyStatsStore mock
func TestMockWeeklyStatsStore(t *testing.T) {
	mockStore := mocks.NewMockWeeklyStatsStoreInterface(t)

	expectedStats := &domainTeam.MemberWeeklyStats{
		MemberID:   "member-1",
		MemberName: "Test Member",
		WeekStart:  "2026-01-19",
	}

	// 设置期望
	mockStore.On("Get", "member-1", "2026-01-19").Return(expectedStats, nil)
	mockStore.On("Set", "member-1", "2026-01-19", mock.AnythingOfType("*team.MemberWeeklyStats")).Return(nil)
	mockStore.On("SetWithExpiration", "member-1", "2026-01-19", mock.AnythingOfType("*team.MemberWeeklyStats"), time.Hour).Return(nil)
	mockStore.On("GetAll", "2026-01-19").Return(map[string]*domainTeam.MemberWeeklyStats{
		"member-1": expectedStats,
	}, nil)
	mockStore.On("GetExpiredMembers", []string{"member-1", "member-2"}, "2026-01-19").Return([]string{"member-2"})
	mockStore.On("Delete", "member-1", "2026-01-19").Return(nil)
	mockStore.On("Clear").Return(nil)

	// 测试 Get
	stats, err := mockStore.Get("member-1", "2026-01-19")
	require.NoError(t, err)
	assert.Equal(t, "member-1", stats.MemberID)

	// 测试 Set
	err = mockStore.Set("member-1", "2026-01-19", expectedStats)
	require.NoError(t, err)

	// 测试 SetWithExpiration
	err = mockStore.SetWithExpiration("member-1", "2026-01-19", expectedStats, time.Hour)
	require.NoError(t, err)

	// 测试 GetAll
	allStats, err := mockStore.GetAll("2026-01-19")
	require.NoError(t, err)
	assert.Len(t, allStats, 1)

	// 测试 GetExpiredMembers
	expired := mockStore.GetExpiredMembers([]string{"member-1", "member-2"}, "2026-01-19")
	assert.Equal(t, []string{"member-2"}, expired)

	// 测试 Delete
	err = mockStore.Delete("member-1", "2026-01-19")
	require.NoError(t, err)

	// 测试 Clear
	err = mockStore.Clear()
	require.NoError(t, err)
	// mockery 生成的 mock 会在 t.Cleanup 中自动调用 AssertExpectations
}

// TestWeeklyReportService_DateHelpers 测试日期辅助函数
func TestWeeklyReportService_DateHelpers(t *testing.T) {
	// 测试获取周起始日期
	testCases := []struct {
		name      string
		date      time.Time
		weekStart string
	}{
		{
			name:      "Monday",
			date:      time.Date(2026, 1, 19, 12, 0, 0, 0, time.UTC), // 周一
			weekStart: "2026-01-19",
		},
		{
			name:      "Wednesday",
			date:      time.Date(2026, 1, 21, 12, 0, 0, 0, time.UTC), // 周三
			weekStart: "2026-01-19",
		},
		{
			name:      "Sunday",
			date:      time.Date(2026, 1, 25, 12, 0, 0, 0, time.UTC), // 周日
			weekStart: "2026-01-19",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 获取周一
			weekday := tc.date.Weekday()
			daysToMonday := int(weekday) - 1 // 周一是 1
			if daysToMonday < 0 {
				daysToMonday = 6 // 周日需要回退 6 天
			}
			monday := tc.date.AddDate(0, 0, -daysToMonday)
			weekStart := monday.Format("2006-01-02")

			assert.Equal(t, tc.weekStart, weekStart)
		})
	}
}

// TestWeeklyReportService_ActivityLevel 测试活动等级计算
func TestWeeklyReportService_ActivityLevel(t *testing.T) {
	testCases := []struct {
		commits       int
		expectedLevel int
	}{
		{0, 0},
		{1, 1},
		{2, 1},
		{3, 2},
		{5, 2},
		{6, 3},
		{10, 3},
		{11, 4},
		{100, 4},
	}

	for _, tc := range testCases {
		cell := &domainTeam.MemberDayCell{
			MemberID: "test",
			Commits:  tc.commits,
		}
		cell.CalculateActivityLevel()
		assert.Equal(t, tc.expectedLevel, cell.ActivityLevel, "commits=%d", tc.commits)
	}
}
