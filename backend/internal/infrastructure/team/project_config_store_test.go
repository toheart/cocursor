package team

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

func setupProjectConfigTestDir(t *testing.T) (string, func()) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "project-config-test")
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

func TestProjectConfigStore_NewStore(t *testing.T) {
	_, cleanup := setupProjectConfigTestDir(t)
	defer cleanup()

	store, err := NewProjectConfigStore("test-team-id")
	require.NoError(t, err)
	assert.NotNil(t, store)

	// 验证初始配置
	config, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, "test-team-id", config.TeamID)
	assert.Empty(t, config.Projects)
}

func TestProjectConfigStore_AddProject(t *testing.T) {
	_, cleanup := setupProjectConfigTestDir(t)
	defer cleanup()

	store, err := NewProjectConfigStore("test-team-id")
	require.NoError(t, err)

	// 添加项目
	project, err := store.AddProject("My Project", "github.com/org/repo")
	require.NoError(t, err)
	assert.NotEmpty(t, project.ID)
	assert.Equal(t, "My Project", project.Name)
	assert.Equal(t, "github.com/org/repo", project.RepoURL)

	// 验证已添加
	config, err := store.Load()
	require.NoError(t, err)
	assert.Len(t, config.Projects, 1)
	assert.Equal(t, project.ID, config.Projects[0].ID)
}

func TestProjectConfigStore_AddProject_Duplicate(t *testing.T) {
	_, cleanup := setupProjectConfigTestDir(t)
	defer cleanup()

	store, err := NewProjectConfigStore("test-team-id")
	require.NoError(t, err)

	// 添加第一个项目
	_, err = store.AddProject("Project 1", "github.com/org/repo")
	require.NoError(t, err)

	// 尝试添加相同 URL 的项目
	_, err = store.AddProject("Project 2", "github.com/org/repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestProjectConfigStore_AddProject_ValidationErrors(t *testing.T) {
	_, cleanup := setupProjectConfigTestDir(t)
	defer cleanup()

	store, err := NewProjectConfigStore("test-team-id")
	require.NoError(t, err)

	// 空名称
	_, err = store.AddProject("", "github.com/org/repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")

	// 空 URL
	_, err = store.AddProject("My Project", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "URL is required")
}

func TestProjectConfigStore_RemoveProject(t *testing.T) {
	_, cleanup := setupProjectConfigTestDir(t)
	defer cleanup()

	store, err := NewProjectConfigStore("test-team-id")
	require.NoError(t, err)

	// 添加项目
	project1, err := store.AddProject("Project 1", "github.com/org/repo1")
	require.NoError(t, err)
	project2, err := store.AddProject("Project 2", "github.com/org/repo2")
	require.NoError(t, err)

	// 验证有 2 个项目
	config, err := store.Load()
	require.NoError(t, err)
	assert.Len(t, config.Projects, 2)

	// 移除第一个项目
	err = store.RemoveProject(project1.ID)
	require.NoError(t, err)

	// 验证只剩 1 个项目
	config, err = store.Load()
	require.NoError(t, err)
	assert.Len(t, config.Projects, 1)
	assert.Equal(t, project2.ID, config.Projects[0].ID)
}

func TestProjectConfigStore_RemoveProject_NotFound(t *testing.T) {
	_, cleanup := setupProjectConfigTestDir(t)
	defer cleanup()

	store, err := NewProjectConfigStore("test-team-id")
	require.NoError(t, err)

	// 尝试移除不存在的项目
	err = store.RemoveProject("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProjectConfigStore_UpdateProject(t *testing.T) {
	_, cleanup := setupProjectConfigTestDir(t)
	defer cleanup()

	store, err := NewProjectConfigStore("test-team-id")
	require.NoError(t, err)

	// 添加项目
	project, err := store.AddProject("Original Name", "github.com/org/original")
	require.NoError(t, err)

	// 更新项目
	updated, err := store.UpdateProject(project.ID, "New Name", "github.com/org/new")
	require.NoError(t, err)
	assert.Equal(t, project.ID, updated.ID)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "github.com/org/new", updated.RepoURL)

	// 验证更新已持久化
	config, err := store.Load()
	require.NoError(t, err)
	assert.Len(t, config.Projects, 1)
	assert.Equal(t, "New Name", config.Projects[0].Name)
}

func TestProjectConfigStore_UpdateProject_NotFound(t *testing.T) {
	_, cleanup := setupProjectConfigTestDir(t)
	defer cleanup()

	store, err := NewProjectConfigStore("test-team-id")
	require.NoError(t, err)

	_, err = store.UpdateProject("non-existent-id", "Name", "github.com/org/repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProjectConfigStore_UpdateProject_DuplicateURL(t *testing.T) {
	_, cleanup := setupProjectConfigTestDir(t)
	defer cleanup()

	store, err := NewProjectConfigStore("test-team-id")
	require.NoError(t, err)

	// 添加两个项目
	project1, err := store.AddProject("Project 1", "github.com/org/repo1")
	require.NoError(t, err)
	_, err = store.AddProject("Project 2", "github.com/org/repo2")
	require.NoError(t, err)

	// 尝试将项目 1 的 URL 改为项目 2 的 URL
	_, err = store.UpdateProject(project1.ID, "Project 1 Updated", "github.com/org/repo2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestProjectConfigStore_Save(t *testing.T) {
	_, cleanup := setupProjectConfigTestDir(t)
	defer cleanup()

	store, err := NewProjectConfigStore("test-team-id")
	require.NoError(t, err)

	// 保存完整配置
	config := &domainTeam.TeamProjectConfig{
		TeamID: "test-team-id",
		Projects: []domainTeam.ProjectMatcher{
			{ID: "id-1", Name: "Project 1", RepoURL: "github.com/org/repo1"},
			{ID: "id-2", Name: "Project 2", RepoURL: "github.com/org/repo2"},
		},
	}

	err = store.Save(config)
	require.NoError(t, err)

	// 验证保存成功
	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, "test-team-id", loaded.TeamID)
	assert.Len(t, loaded.Projects, 2)
}

func TestProjectConfigStore_GetRepoURLs(t *testing.T) {
	_, cleanup := setupProjectConfigTestDir(t)
	defer cleanup()

	store, err := NewProjectConfigStore("test-team-id")
	require.NoError(t, err)

	// 添加项目
	_, err = store.AddProject("Project 1", "github.com/org/repo1")
	require.NoError(t, err)
	_, err = store.AddProject("Project 2", "github.com/org/repo2")
	require.NoError(t, err)

	// 获取 URL 列表
	urls := store.GetRepoURLs()
	assert.Len(t, urls, 2)
	assert.Contains(t, urls, "github.com/org/repo1")
	assert.Contains(t, urls, "github.com/org/repo2")
}

func TestProjectConfigStore_Persistence(t *testing.T) {
	tmpDir, cleanup := setupProjectConfigTestDir(t)
	defer cleanup()

	teamID := "test-team-id"

	// 创建 store 并添加项目
	store1, err := NewProjectConfigStore(teamID)
	require.NoError(t, err)
	project, err := store1.AddProject("My Project", "github.com/org/repo")
	require.NoError(t, err)

	// 验证文件已创建
	filePath := filepath.Join(tmpDir, ".cocursor", "team", teamID, "project_config.json")
	_, err = os.Stat(filePath)
	require.NoError(t, err, "config file should exist")

	// 创建新的 store 实例，验证数据持久化
	store2, err := NewProjectConfigStore(teamID)
	require.NoError(t, err)

	config, err := store2.Load()
	require.NoError(t, err)
	assert.Len(t, config.Projects, 1)
	assert.Equal(t, project.ID, config.Projects[0].ID)
	assert.Equal(t, "My Project", config.Projects[0].Name)
}

func TestProjectConfigStore_LoadReturnsCopy(t *testing.T) {
	_, cleanup := setupProjectConfigTestDir(t)
	defer cleanup()

	store, err := NewProjectConfigStore("test-team-id")
	require.NoError(t, err)

	_, err = store.AddProject("Original", "github.com/org/repo")
	require.NoError(t, err)

	// 获取配置
	config1, err := store.Load()
	require.NoError(t, err)

	// 修改返回的配置
	config1.Projects[0].Name = "Modified"

	// 再次获取，验证原始数据未被修改
	config2, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, "Original", config2.Projects[0].Name)
}
