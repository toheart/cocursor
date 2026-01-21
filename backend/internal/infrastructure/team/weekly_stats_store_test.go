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

func setupWeeklyStatsTestDir(t *testing.T) (string, func()) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "weekly-stats-test")
	require.NoError(t, err)

	// 保存原始 HOME（跨平台兼容）
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")

	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)

	cleanup := func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func createTestStats(memberID, weekStart string) *domainTeam.MemberWeeklyStats {
	return &domainTeam.MemberWeeklyStats{
		MemberID:   memberID,
		MemberName: "Test User",
		WeekStart:  weekStart,
		DailyStats: []domainTeam.MemberDailyStats{
			{
				Date: weekStart,
				GitStats: &domainTeam.GitDailyStats{
					TotalCommits: 5,
					TotalAdded:   100,
					TotalRemoved: 20,
				},
				HasReport: true,
			},
		},
		UpdatedAt: time.Now(),
	}
}

func TestWeeklyStatsStore_NewStore(t *testing.T) {
	_, cleanup := setupWeeklyStatsTestDir(t)
	defer cleanup()

	store, err := NewWeeklyStatsStore("test-team-id")
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestWeeklyStatsStore_SetAndGet(t *testing.T) {
	_, cleanup := setupWeeklyStatsTestDir(t)
	defer cleanup()

	store, err := NewWeeklyStatsStore("test-team-id")
	require.NoError(t, err)

	stats := createTestStats("member-1", "2026-01-20")

	// 设置缓存
	err = store.Set(stats)
	require.NoError(t, err)

	// 获取缓存
	got, isExpired, err := store.Get("member-1", "2026-01-20")
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.False(t, isExpired)
	assert.Equal(t, "member-1", got.MemberID)
	assert.Equal(t, "2026-01-20", got.WeekStart)
}

func TestWeeklyStatsStore_GetNotFound(t *testing.T) {
	_, cleanup := setupWeeklyStatsTestDir(t)
	defer cleanup()

	store, err := NewWeeklyStatsStore("test-team-id")
	require.NoError(t, err)

	got, _, err := store.Get("non-existent", "2026-01-20")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestWeeklyStatsStore_SetWithExpiration(t *testing.T) {
	_, cleanup := setupWeeklyStatsTestDir(t)
	defer cleanup()

	store, err := NewWeeklyStatsStore("test-team-id")
	require.NoError(t, err)

	stats := createTestStats("member-1", "2026-01-20")

	// 设置非常短的过期时间
	err = store.SetWithExpiration(stats, 1*time.Millisecond)
	require.NoError(t, err)

	// 等待过期
	time.Sleep(5 * time.Millisecond)

	// 获取缓存，应该是过期的
	got, isExpired, err := store.Get("member-1", "2026-01-20")
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.True(t, isExpired)
}

func TestWeeklyStatsStore_GetAll(t *testing.T) {
	_, cleanup := setupWeeklyStatsTestDir(t)
	defer cleanup()

	store, err := NewWeeklyStatsStore("test-team-id")
	require.NoError(t, err)

	weekStart := "2026-01-20"

	// 添加多个成员的数据
	err = store.Set(createTestStats("member-1", weekStart))
	require.NoError(t, err)
	err = store.Set(createTestStats("member-2", weekStart))
	require.NoError(t, err)
	err = store.Set(createTestStats("member-3", "2026-01-13")) // 不同的周
	require.NoError(t, err)

	// 获取指定周的所有数据
	all, err := store.GetAll(weekStart)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestWeeklyStatsStore_GetExpiredMembers(t *testing.T) {
	_, cleanup := setupWeeklyStatsTestDir(t)
	defer cleanup()

	store, err := NewWeeklyStatsStore("test-team-id")
	require.NoError(t, err)

	weekStart := "2026-01-20"

	// member-1: 未过期
	err = store.Set(createTestStats("member-1", weekStart))
	require.NoError(t, err)

	// member-2: 过期
	err = store.SetWithExpiration(createTestStats("member-2", weekStart), 1*time.Millisecond)
	require.NoError(t, err)
	time.Sleep(5 * time.Millisecond)

	// member-3: 未缓存

	memberIDs := []string{"member-1", "member-2", "member-3"}
	expired := store.GetExpiredMembers(weekStart, memberIDs)

	// member-2 (过期) 和 member-3 (未缓存) 应该在列表中
	assert.Len(t, expired, 2)
	assert.Contains(t, expired, "member-2")
	assert.Contains(t, expired, "member-3")
	assert.NotContains(t, expired, "member-1")
}

func TestWeeklyStatsStore_Delete(t *testing.T) {
	_, cleanup := setupWeeklyStatsTestDir(t)
	defer cleanup()

	store, err := NewWeeklyStatsStore("test-team-id")
	require.NoError(t, err)

	stats := createTestStats("member-1", "2026-01-20")
	err = store.Set(stats)
	require.NoError(t, err)

	// 确认存在
	got, _, err := store.Get("member-1", "2026-01-20")
	require.NoError(t, err)
	assert.NotNil(t, got)

	// 删除
	err = store.Delete("member-1", "2026-01-20")
	require.NoError(t, err)

	// 确认已删除
	got, _, err = store.Get("member-1", "2026-01-20")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestWeeklyStatsStore_Clear(t *testing.T) {
	_, cleanup := setupWeeklyStatsTestDir(t)
	defer cleanup()

	store, err := NewWeeklyStatsStore("test-team-id")
	require.NoError(t, err)

	// 添加多个缓存
	err = store.Set(createTestStats("member-1", "2026-01-20"))
	require.NoError(t, err)
	err = store.Set(createTestStats("member-2", "2026-01-20"))
	require.NoError(t, err)

	// 清空
	err = store.Clear()
	require.NoError(t, err)

	// 验证清空
	all, err := store.GetAll("2026-01-20")
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestWeeklyStatsStore_CleanExpired(t *testing.T) {
	_, cleanup := setupWeeklyStatsTestDir(t)
	defer cleanup()

	store, err := NewWeeklyStatsStore("test-team-id")
	require.NoError(t, err)

	// 添加一个未过期的缓存
	err = store.Set(createTestStats("member-1", "2026-01-20"))
	require.NoError(t, err)

	// 添加一个过期的缓存
	err = store.SetWithExpiration(createTestStats("member-2", "2026-01-20"), 1*time.Millisecond)
	require.NoError(t, err)
	time.Sleep(5 * time.Millisecond)

	// 清理过期条目
	// 注意：CleanExpired 保留过期不超过 7 天的条目（用于离线成员）
	// 所以刚刚过期的条目不会被清理
	removed, err := store.CleanExpired()
	require.NoError(t, err)
	// 刚过期的不会被删除，需要过期超过 7 天
	assert.Equal(t, 0, removed)
}

func TestWeeklyStatsStore_Persistence(t *testing.T) {
	tmpDir, cleanup := setupWeeklyStatsTestDir(t)
	defer cleanup()

	teamID := "test-team-id"

	// 创建 store 并添加数据
	store1, err := NewWeeklyStatsStore(teamID)
	require.NoError(t, err)
	err = store1.Set(createTestStats("member-1", "2026-01-20"))
	require.NoError(t, err)

	// 验证文件已创建
	filePath := filepath.Join(tmpDir, ".cocursor", "team", teamID, "weekly_stats_cache.json")
	_, err = os.Stat(filePath)
	require.NoError(t, err, "cache file should exist")

	// 创建新的 store 实例，验证数据持久化
	store2, err := NewWeeklyStatsStore(teamID)
	require.NoError(t, err)

	got, _, err := store2.Get("member-1", "2026-01-20")
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "member-1", got.MemberID)
}

func TestWeeklyStatsStore_UpdateExisting(t *testing.T) {
	_, cleanup := setupWeeklyStatsTestDir(t)
	defer cleanup()

	store, err := NewWeeklyStatsStore("test-team-id")
	require.NoError(t, err)

	// 第一次设置
	stats1 := createTestStats("member-1", "2026-01-20")
	stats1.DailyStats[0].GitStats.TotalCommits = 5
	err = store.Set(stats1)
	require.NoError(t, err)

	// 更新（使用相同的 key）
	stats2 := createTestStats("member-1", "2026-01-20")
	stats2.DailyStats[0].GitStats.TotalCommits = 10
	err = store.Set(stats2)
	require.NoError(t, err)

	// 验证更新
	got, _, err := store.Get("member-1", "2026-01-20")
	require.NoError(t, err)
	assert.Equal(t, 10, got.DailyStats[0].GitStats.TotalCommits)

	// 验证只有一个条目（不是两个）
	all, err := store.GetAll("2026-01-20")
	require.NoError(t, err)
	assert.Len(t, all, 1)
}
