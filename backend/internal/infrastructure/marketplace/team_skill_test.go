package marketplace

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cocursor/backend/internal/domain/marketplace"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

func setupTestHome(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "team-skill-test")
	require.NoError(t, err)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	return tmpDir, func() {
		os.Setenv("HOME", oldHome)
		os.RemoveAll(tmpDir)
	}
}

func createTestSkillDir(t *testing.T, basePath, name, description string) string {
	skillDir := filepath.Join(basePath, name)
	err := os.MkdirAll(skillDir, 0755)
	require.NoError(t, err)

	// 创建 SKILL.md
	skillMD := `---
name: "` + name + `"
description: "` + description + `"
version: "1.0.0"
---

# ` + name + `

This is a test skill.
`
	err = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644)
	require.NoError(t, err)

	return skillDir
}

func TestTeamSkillValidator_ValidateDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestHome(t)
	defer cleanup()

	validator := NewTeamSkillValidator()

	// 测试有效目录
	skillDir := createTestSkillDir(t, tmpDir, "test-skill", "A test skill")

	result, err := validator.ValidateDirectory(skillDir)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, "test-skill", result.Name)
	assert.Equal(t, "A test skill", result.Description)
	assert.Equal(t, "1.0.0", result.Version)
	assert.Contains(t, result.Files, "SKILL.md")

	// 测试无效目录（不存在）
	result, err = validator.ValidateDirectory(filepath.Join(tmpDir, "nonexistent"))
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Error, "does not exist")

	// 测试无效目录（无 SKILL.md）
	emptyDir := filepath.Join(tmpDir, "empty")
	os.MkdirAll(emptyDir, 0755)

	result, err = validator.ValidateDirectory(emptyDir)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Error, "SKILL.md not found")
}

func TestTeamSkillValidator_BuildSkillEntry(t *testing.T) {
	tmpDir, cleanup := setupTestHome(t)
	defer cleanup()

	validator := NewTeamSkillValidator()
	skillDir := createTestSkillDir(t, tmpDir, "my-skill", "My awesome skill")

	result, err := validator.ValidateDirectory(skillDir)
	require.NoError(t, err)
	require.True(t, result.Valid)

	entry, err := validator.BuildSkillEntry(
		result,
		"my-skill",
		"author-123",
		"Test Author",
		"192.168.1.100:19960",
	)
	require.NoError(t, err)

	assert.Equal(t, "my-skill", entry.PluginID)
	assert.Equal(t, "my-skill", entry.Name)
	assert.Equal(t, "My awesome skill", entry.Description)
	assert.Equal(t, "1.0.0", entry.Version)
	assert.Equal(t, "author-123", entry.AuthorID)
	assert.Equal(t, "Test Author", entry.AuthorName)
	assert.Equal(t, "192.168.1.100:19960", entry.AuthorEndpoint)
	assert.NotEmpty(t, entry.Checksum)
	assert.Greater(t, entry.FileCount, 0)
}

func TestTeamSkillPublisher_PublishLocal(t *testing.T) {
	tmpDir, cleanup := setupTestHome(t)
	defer cleanup()

	publisher := NewTeamSkillPublisher()
	skillDir := createTestSkillDir(t, tmpDir, "publish-test", "Skill to publish")

	req := &PublishRequest{
		TeamID:     "team-123",
		PluginID:   "publish-test",
		LocalPath:  skillDir,
		AuthorID:   "author-456",
		AuthorName: "Publisher",
		Endpoint:   "192.168.1.50:19960",
	}

	entry, err := publisher.PublishLocal(req)
	require.NoError(t, err)

	assert.Equal(t, "publish-test", entry.PluginID)
	assert.Equal(t, "Skill to publish", entry.Description)
	assert.NotZero(t, entry.PublishedAt)

	// 验证路径映射
	path, exists := publisher.GetSkillPath("publish-test")
	assert.True(t, exists)
	assert.Equal(t, skillDir, path)

	// 验证可以获取元数据
	meta, err := publisher.GetSkillMeta("publish-test")
	require.NoError(t, err)
	assert.Equal(t, "publish-test", meta.PluginID)

	// 验证可以获取打包
	archive, err := publisher.GetSkillArchive("publish-test")
	require.NoError(t, err)
	assert.NotEmpty(t, archive)
}

func TestTeamSkillLoader_LoadTeamSkills(t *testing.T) {
	_, cleanup := setupTestHome(t)
	defer cleanup()

	loader := NewTeamSkillLoader()

	now := time.Now()
	index := &domainTeam.TeamSkillIndex{
		TeamID:    "team-abc",
		UpdatedAt: now,
		Skills: []domainTeam.TeamSkillEntry{
			{
				PluginID:       "skill-1",
				Name:           "Skill One",
				Description:    "First skill",
				Version:        "1.0.0",
				AuthorID:       "author-1",
				AuthorName:     "Author One",
				AuthorEndpoint: "192.168.1.100:19960",
				PublishedAt:    now,
			},
			{
				PluginID:       "skill-2",
				Name:           "Skill Two",
				Description:    "Second skill",
				Version:        "2.0.0",
				AuthorID:       "author-2",
				AuthorName:     "Author Two",
				AuthorEndpoint: "192.168.1.101:19960",
				PublishedAt:    now,
			},
		},
	}

	team := &domainTeam.Team{
		ID:   "team-abc",
		Name: "Test Team",
	}

	plugins := loader.LoadTeamSkills("team-abc", index, team)

	assert.Len(t, plugins, 2)

	// 验证第一个插件
	assert.Equal(t, "skill-1", plugins[0].ID)
	assert.Equal(t, "team-abc:skill-1", plugins[0].FullID)
	assert.Equal(t, "Skill One", plugins[0].Name)
	assert.Equal(t, marketplace.SourceTeamGlobal, plugins[0].Source)
	assert.Equal(t, "team-abc", plugins[0].TeamID)
	assert.Equal(t, "Test Team", plugins[0].TeamName)
	assert.Equal(t, "Author One", plugins[0].AuthorName)
}

func TestFilterBySource(t *testing.T) {
	plugins := []*marketplace.Plugin{
		{ID: "p1", Source: marketplace.SourceBuiltin},
		{ID: "p2", Source: marketplace.SourceTeamGlobal},
		{ID: "p3", Source: marketplace.SourceBuiltin},
		{ID: "p4", Source: marketplace.SourceTeamGlobal},
	}

	// 筛选内建
	builtin := FilterBySource(plugins, marketplace.SourceBuiltin)
	assert.Len(t, builtin, 2)

	// 筛选团队
	team := FilterBySource(plugins, marketplace.SourceTeamGlobal)
	assert.Len(t, team, 2)

	// 不筛选
	all := FilterBySource(plugins, "")
	assert.Len(t, all, 4)
}

func TestFilterByTeam(t *testing.T) {
	plugins := []*marketplace.Plugin{
		{ID: "p1", TeamID: "team-1"},
		{ID: "p2", TeamID: "team-2"},
		{ID: "p3", TeamID: "team-1"},
		{ID: "p4", TeamID: ""},
	}

	// 筛选团队 1
	team1 := FilterByTeam(plugins, "team-1")
	assert.Len(t, team1, 2)

	// 筛选团队 2
	team2 := FilterByTeam(plugins, "team-2")
	assert.Len(t, team2, 1)

	// 不筛选
	all := FilterByTeam(plugins, "")
	assert.Len(t, all, 4)
}

func TestMergePlugins(t *testing.T) {
	builtinPlugins := []*marketplace.Plugin{
		{ID: "p1", Source: marketplace.SourceBuiltin, FullID: "p1"},
		{ID: "p2", Source: marketplace.SourceBuiltin, FullID: "p2"},
	}

	teamPlugins := []*marketplace.Plugin{
		{ID: "p1", Source: marketplace.SourceTeamGlobal, TeamID: "team-1", FullID: "team-1:p1"},
		{ID: "p3", Source: marketplace.SourceTeamGlobal, TeamID: "team-1", FullID: "team-1:p3"},
	}

	merged := MergePlugins(builtinPlugins, teamPlugins)

	// 应该有 4 个（p1 内建, p2, team-1:p1, team-1:p3）
	assert.Len(t, merged, 4)
}
