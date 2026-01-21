package team

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTeamProjectConfig_FindProject(t *testing.T) {
	config := &TeamProjectConfig{
		TeamID: "team-1",
		Projects: []ProjectMatcher{
			{ID: "id-1", Name: "Project 1", RepoURL: "github.com/org/repo1"},
			{ID: "id-2", Name: "Project 2", RepoURL: "github.com/org/repo2"},
		},
	}

	// 查找存在的项目
	project := config.FindProject("github.com/org/repo1")
	assert.NotNil(t, project)
	assert.Equal(t, "id-1", project.ID)
	assert.Equal(t, "Project 1", project.Name)

	// 查找不存在的项目
	project = config.FindProject("github.com/org/repo3")
	assert.Nil(t, project)
}

func TestTeamProjectConfig_AddProject(t *testing.T) {
	config := &TeamProjectConfig{
		TeamID:   "team-1",
		Projects: []ProjectMatcher{},
	}

	// 添加新项目
	project := ProjectMatcher{ID: "id-1", Name: "Project 1", RepoURL: "github.com/org/repo1"}
	config.AddProject(project)

	assert.Len(t, config.Projects, 1)
	assert.Equal(t, "id-1", config.Projects[0].ID)

	// 添加另一个项目
	project2 := ProjectMatcher{ID: "id-2", Name: "Project 2", RepoURL: "github.com/org/repo2"}
	config.AddProject(project2)

	assert.Len(t, config.Projects, 2)

	// 更新已存在的项目（相同 ID）
	updatedProject := ProjectMatcher{ID: "id-1", Name: "Updated Project 1", RepoURL: "github.com/org/repo1-new"}
	config.AddProject(updatedProject)

	assert.Len(t, config.Projects, 2)
	assert.Equal(t, "Updated Project 1", config.Projects[0].Name)
	assert.Equal(t, "github.com/org/repo1-new", config.Projects[0].RepoURL)

	// 更新已存在的项目（相同 RepoURL）
	anotherProject := ProjectMatcher{ID: "id-3", Name: "Another Project", RepoURL: "github.com/org/repo2"}
	config.AddProject(anotherProject)

	assert.Len(t, config.Projects, 2) // 应该替换而不是新增
}

func TestTeamProjectConfig_RemoveProject(t *testing.T) {
	config := &TeamProjectConfig{
		TeamID: "team-1",
		Projects: []ProjectMatcher{
			{ID: "id-1", Name: "Project 1", RepoURL: "github.com/org/repo1"},
			{ID: "id-2", Name: "Project 2", RepoURL: "github.com/org/repo2"},
		},
	}

	// 移除存在的项目
	removed := config.RemoveProject("id-1")
	assert.True(t, removed)
	assert.Len(t, config.Projects, 1)
	assert.Equal(t, "id-2", config.Projects[0].ID)

	// 移除不存在的项目
	removed = config.RemoveProject("id-3")
	assert.False(t, removed)
	assert.Len(t, config.Projects, 1)
}

func TestTeamProjectConfig_GetRepoURLs(t *testing.T) {
	config := &TeamProjectConfig{
		TeamID: "team-1",
		Projects: []ProjectMatcher{
			{ID: "id-1", Name: "Project 1", RepoURL: "github.com/org/repo1"},
			{ID: "id-2", Name: "Project 2", RepoURL: "github.com/org/repo2"},
		},
	}

	urls := config.GetRepoURLs()
	assert.Len(t, urls, 2)
	assert.Contains(t, urls, "github.com/org/repo1")
	assert.Contains(t, urls, "github.com/org/repo2")
}

func TestMemberDayCell_CalculateActivityLevel(t *testing.T) {
	testCases := []struct {
		name          string
		commits       int
		expectedLevel int
	}{
		{"zero commits", 0, 0},
		{"one commit", 1, 1},
		{"two commits", 2, 1},
		{"three commits", 3, 2},
		{"five commits", 5, 2},
		{"six commits", 6, 3},
		{"ten commits", 10, 3},
		{"eleven commits", 11, 4},
		{"many commits", 50, 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cell := &MemberDayCell{
				MemberID: "member-1",
				Commits:  tc.commits,
			}
			cell.CalculateActivityLevel()
			assert.Equal(t, tc.expectedLevel, cell.ActivityLevel)
		})
	}
}

func TestWeeklyStatsCacheEntry_IsExpired(t *testing.T) {
	now := time.Now()

	// 未过期的条目
	notExpired := WeeklyStatsCacheEntry{
		MemberID:  "member-1",
		WeekStart: "2026-01-20",
		ExpireAt:  now.Add(1 * time.Hour),
	}
	assert.False(t, notExpired.IsExpired())

	// 已过期的条目
	expired := WeeklyStatsCacheEntry{
		MemberID:  "member-1",
		WeekStart: "2026-01-20",
		ExpireAt:  now.Add(-1 * time.Hour),
	}
	assert.True(t, expired.IsExpired())
}

func TestWeeklyStatsCache_GetEntry(t *testing.T) {
	cache := &WeeklyStatsCache{
		TeamID: "team-1",
		Entries: []WeeklyStatsCacheEntry{
			{MemberID: "member-1", WeekStart: "2026-01-20"},
			{MemberID: "member-2", WeekStart: "2026-01-20"},
			{MemberID: "member-1", WeekStart: "2026-01-13"},
		},
	}

	// 查找存在的条目
	entry := cache.GetEntry("member-1", "2026-01-20")
	assert.NotNil(t, entry)
	assert.Equal(t, "member-1", entry.MemberID)
	assert.Equal(t, "2026-01-20", entry.WeekStart)

	// 查找不存在的条目
	entry = cache.GetEntry("member-3", "2026-01-20")
	assert.Nil(t, entry)

	// 查找不同周的条目
	entry = cache.GetEntry("member-1", "2026-01-13")
	assert.NotNil(t, entry)
}

func TestWeeklyStatsCache_SetEntry(t *testing.T) {
	cache := &WeeklyStatsCache{
		TeamID:  "team-1",
		Entries: []WeeklyStatsCacheEntry{},
	}

	// 添加新条目
	entry1 := WeeklyStatsCacheEntry{
		MemberID:  "member-1",
		WeekStart: "2026-01-20",
	}
	cache.SetEntry(entry1)
	assert.Len(t, cache.Entries, 1)

	// 添加另一个条目
	entry2 := WeeklyStatsCacheEntry{
		MemberID:  "member-2",
		WeekStart: "2026-01-20",
	}
	cache.SetEntry(entry2)
	assert.Len(t, cache.Entries, 2)

	// 更新已存在的条目
	updatedEntry := WeeklyStatsCacheEntry{
		MemberID:  "member-1",
		WeekStart: "2026-01-20",
		Stats: MemberWeeklyStats{
			MemberName: "Updated Name",
		},
	}
	cache.SetEntry(updatedEntry)
	assert.Len(t, cache.Entries, 2) // 数量不变

	// 验证已更新
	got := cache.GetEntry("member-1", "2026-01-20")
	assert.Equal(t, "Updated Name", got.Stats.MemberName)
}

func TestWeeklyStatsCache_CleanExpired(t *testing.T) {
	now := time.Now()

	cache := &WeeklyStatsCache{
		TeamID: "team-1",
		Entries: []WeeklyStatsCacheEntry{
			{
				MemberID:  "member-1",
				WeekStart: "2026-01-20",
				ExpireAt:  now.Add(1 * time.Hour), // 未过期
			},
			{
				MemberID:  "member-2",
				WeekStart: "2026-01-20",
				ExpireAt:  now.Add(-2 * time.Hour), // 刚过期（不超过 7 天，会保留）
			},
			{
				MemberID:  "member-3",
				WeekStart: "2026-01-20",
				ExpireAt:  now.Add(-8 * 24 * time.Hour), // 过期超过 7 天
			},
		},
	}

	removed := cache.CleanExpired()
	assert.Equal(t, 1, removed) // 只有 member-3 被清理

	// 验证保留了正确的条目
	assert.Len(t, cache.Entries, 2)
	assert.NotNil(t, cache.GetEntry("member-1", "2026-01-20"))
	assert.NotNil(t, cache.GetEntry("member-2", "2026-01-20"))
	assert.Nil(t, cache.GetEntry("member-3", "2026-01-20"))
}
