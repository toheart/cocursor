package cursor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadGitRemoteURL(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "git_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建 .git 目录
	gitDir := filepath.Join(tmpDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	// 创建 git config 文件
	configPath := filepath.Join(gitDir, "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = false
	bare = false

[remote "origin"]
	url = https://github.com/user/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	reader := NewGitReader()
	url, err := reader.ReadGitRemoteURL(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/user/repo", url)
}

func TestReadGitRemoteURL_SSHFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(gitDir, "config")
	configContent := `[remote "origin"]
	url = git@github.com:user/repo.git
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	reader := NewGitReader()
	url, err := reader.ReadGitRemoteURL(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/user/repo", url)
}

func TestReadGitRemoteURL_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	reader := NewGitReader()
	_, err = reader.ReadGitRemoteURL(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git repository not found")
}

func TestReadGitRemoteURL_NoOrigin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(gitDir, "config")
	configContent := `[core]
	repositoryformatversion = 0
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	reader := NewGitReader()
	_, err = reader.ReadGitRemoteURL(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remote \"origin\" not found")
}

func TestReadGitBranch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	// 创建 HEAD 文件
	headPath := filepath.Join(gitDir, "HEAD")
	headContent := "ref: refs/heads/main\n"
	err = os.WriteFile(headPath, []byte(headContent), 0644)
	require.NoError(t, err)

	reader := NewGitReader()
	branch, err := reader.ReadGitBranch(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "main", branch)
}

func TestReadGitBranch_DetachedHEAD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	headPath := filepath.Join(gitDir, "HEAD")
	headContent := "abc123def456\n" // detached HEAD
	err = os.WriteFile(headPath, []byte(headContent), 0644)
	require.NoError(t, err)

	reader := NewGitReader()
	_, err = reader.ReadGitBranch(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "detached HEAD")
}

func TestNormalizeGitURL(t *testing.T) {
	reader := NewGitReader()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTTPS with .git",
			input:    "https://github.com/user/repo.git",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "HTTPS without .git",
			input:    "https://github.com/user/repo",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "SSH git@ format",
			input:    "git@github.com:user/repo.git",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "SSH ssh:// format",
			input:    "ssh://git@github.com/user/repo.git",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "Mixed case",
			input:    "HTTPS://GITHUB.COM/USER/REPO.GIT",
			expected: "https://github.com/user/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reader.normalizeGitURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
