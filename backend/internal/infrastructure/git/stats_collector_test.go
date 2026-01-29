package git

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

func TestStatsCollector_NormalizeRepoURL(t *testing.T) {
	collector := NewStatsCollector()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTTPS URL",
			input:    "https://github.com/org/repo.git",
			expected: "github.com/org/repo",
		},
		{
			name:     "HTTPS URL without .git",
			input:    "https://github.com/org/repo",
			expected: "github.com/org/repo",
		},
		{
			name:     "SSH URL with git@",
			input:    "git@github.com:org/repo.git",
			expected: "github.com/org/repo",
		},
		{
			name:     "SSH URL without .git",
			input:    "git@github.com:org/repo",
			expected: "github.com/org/repo",
		},
		{
			name:     "HTTP URL",
			input:    "http://github.com/org/repo.git",
			expected: "github.com/org/repo",
		},
		{
			name:     "SSH protocol URL",
			input:    "ssh://git@github.com/org/repo.git",
			expected: "github.com/org/repo", // ssh:// 被移除后，git@ 也被处理
		},
		{
			name:     "Already normalized",
			input:    "github.com/org/repo",
			expected: "github.com/org/repo",
		},
		{
			name:     "Mixed case",
			input:    "https://GitHub.com/Org/Repo.git",
			expected: "github.com/org/repo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := collector.normalizeRepoURL(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStatsCollector_ParseGitLogOutput(t *testing.T) {
	collector := NewStatsCollector()

	testCases := []struct {
		name           string
		input          string
		expectedStats  *domainTeam.ProjectGitStats
		expectedCommit int
	}{
		{
			name:  "Empty output",
			input: "",
			expectedStats: &domainTeam.ProjectGitStats{
				Commits:        0,
				LinesAdded:     0,
				LinesRemoved:   0,
				CommitMessages: []domainTeam.CommitSummary{},
			},
			expectedCommit: 0,
		},
		{
			name:  "Whitespace only",
			input: "   \n\t\n   ",
			expectedStats: &domainTeam.ProjectGitStats{
				Commits:        0,
				LinesAdded:     0,
				LinesRemoved:   0,
				CommitMessages: []domainTeam.CommitSummary{},
			},
			expectedCommit: 0,
		},
		{
			name: "Single commit with insertions only",
			input: `abc1234 Add new feature
 src/main.go | 10 ++++++++++
 1 file changed, 10 insertions(+)`,
			expectedStats: &domainTeam.ProjectGitStats{
				Commits:      1,
				LinesAdded:   10,
				LinesRemoved: 0,
			},
			expectedCommit: 1,
		},
		{
			name: "Single commit with deletions only",
			input: `abc1234 Remove deprecated code
 src/old.go | 5 -----
 1 file changed, 5 deletions(-)`,
			expectedStats: &domainTeam.ProjectGitStats{
				Commits:      1,
				LinesAdded:   0,
				LinesRemoved: 5,
			},
			expectedCommit: 1,
		},
		{
			name: "Single commit with both insertions and deletions",
			input: `abc1234 Refactor code
 src/main.go | 15 +++++++++------
 2 files changed, 9 insertions(+), 6 deletions(-)`,
			expectedStats: &domainTeam.ProjectGitStats{
				Commits:      1,
				LinesAdded:   9,
				LinesRemoved: 6,
			},
			expectedCommit: 1,
		},
		{
			name: "Multiple commits",
			input: `abc1234 First commit
 src/a.go | 5 +++++
 1 file changed, 5 insertions(+)

def5678 Second commit
 src/b.go | 10 +++++-----
 1 file changed, 5 insertions(+), 5 deletions(-)

aed9012 Third commit
 src/c.go | 3 ---
 1 file changed, 3 deletions(-)`,
			expectedStats: &domainTeam.ProjectGitStats{
				Commits:      3,
				LinesAdded:   10, // 5 + 5 + 0
				LinesRemoved: 8,  // 0 + 5 + 3
			},
			expectedCommit: 3,
		},
		{
			name: "Multiple files changed",
			input: `abc1234 Update multiple files
 src/a.go | 10 ++++++++++
 src/b.go | 5 ++---
 src/c.go | 2 --
 3 files changed, 12 insertions(+), 5 deletions(-)`,
			expectedStats: &domainTeam.ProjectGitStats{
				Commits:      1,
				LinesAdded:   12,
				LinesRemoved: 5,
			},
			expectedCommit: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stats, err := collector.parseGitLogOutput(tc.input)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedStats.Commits, stats.Commits, "commits count mismatch")
			assert.Equal(t, tc.expectedStats.LinesAdded, stats.LinesAdded, "lines added mismatch")
			assert.Equal(t, tc.expectedStats.LinesRemoved, stats.LinesRemoved, "lines removed mismatch")
			assert.Len(t, stats.CommitMessages, tc.expectedCommit, "commit messages count mismatch")
		})
	}
}

func TestStatsCollector_ParseGitLogOutput_CommitDetails(t *testing.T) {
	collector := NewStatsCollector()

	input := `abc1234 Fix bug in login
 src/auth.go | 5 +++++
 1 file changed, 5 insertions(+)

def5678 Add unit tests
 src/auth_test.go | 20 ++++++++++++++++++++
 1 file changed, 20 insertions(+)`

	stats, err := collector.parseGitLogOutput(input)
	require.NoError(t, err)

	assert.Equal(t, 2, stats.Commits)
	require.Len(t, stats.CommitMessages, 2)

	// 验证第一个 commit
	assert.Equal(t, "abc1234", stats.CommitMessages[0].Hash)
	assert.Equal(t, "Fix bug in login", stats.CommitMessages[0].Message)
	assert.Equal(t, 1, stats.CommitMessages[0].FilesCount)

	// 验证第二个 commit
	assert.Equal(t, "def5678", stats.CommitMessages[1].Hash)
	assert.Equal(t, "Add unit tests", stats.CommitMessages[1].Message)
	assert.Equal(t, 1, stats.CommitMessages[1].FilesCount)
}

func TestStatsCollector_ParseGitLogOutput_MaxCommits(t *testing.T) {
	collector := NewStatsCollector()

	// 生成超过 10 个 commits 的输出
	// 使用有效的十六进制字符 0-9a-f 来构建 hash
	hexChars := "0123456789abcdef"
	input := ""
	for i := 0; i < 15; i++ {
		// 生成有效的 7 位 hash（如 abc0001, abc0002, ...）
		// 生成有效的 7 位 hash（如 a0c0000, a1c0001, ...）
		hash := "a" + string(hexChars[i%16]) + "c" + fmt.Sprintf("%04x", i)
		input += hash[:7] + " Commit message " + fmt.Sprintf("%d", i) + "\n"
		input += " file.go | 1 +\n"
		input += " 1 file changed, 1 insertion(+)\n\n"
	}

	stats, err := collector.parseGitLogOutput(input)
	require.NoError(t, err)

	assert.Equal(t, 15, stats.Commits)
	// CommitMessages 应该被限制为 10 条
	assert.Len(t, stats.CommitMessages, 10)
}

func TestStatsCollector_ReadEmailFromGitConfig(t *testing.T) {
	// 创建临时目录模拟 HOME
	tmpDir, err := os.MkdirTemp("", "git-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 保存原始 HOME
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()

	// 设置 HOME（跨平台兼容）
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)

	// 创建 .gitconfig 文件
	gitConfigContent := `[user]
	name = Test User
	email = test@example.com
[core]
	autocrlf = input
`
	gitConfigPath := filepath.Join(tmpDir, ".gitconfig")
	err = os.WriteFile(gitConfigPath, []byte(gitConfigContent), 0644)
	require.NoError(t, err)

	collector := NewStatsCollector()
	email, err := collector.readEmailFromGitConfig()
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", email)
}

func TestStatsCollector_ReadEmailFromGitConfig_NotFound(t *testing.T) {
	// 创建临时目录模拟 HOME
	tmpDir, err := os.MkdirTemp("", "git-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 保存原始 HOME
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()

	// 设置 HOME（跨平台兼容）
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)

	// 创建没有 email 的 .gitconfig 文件
	gitConfigContent := `[user]
	name = Test User
[core]
	autocrlf = input
`
	gitConfigPath := filepath.Join(tmpDir, ".gitconfig")
	err = os.WriteFile(gitConfigPath, []byte(gitConfigContent), 0644)
	require.NoError(t, err)

	collector := NewStatsCollector()
	_, err = collector.readEmailFromGitConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email not found")
}

func TestStatsCollector_GetRemoteURL(t *testing.T) {
	// 创建临时仓库目录
	tmpDir, err := os.MkdirTemp("", "git-repo-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建 .git/config 文件
	gitDir := filepath.Join(tmpDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	gitConfigContent := `[core]
	repositoryformatversion = 0
	filemode = true
[remote "origin"]
	url = https://github.com/test/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
[branch "main"]
	remote = origin
	merge = refs/heads/main
`
	err = os.WriteFile(filepath.Join(gitDir, "config"), []byte(gitConfigContent), 0644)
	require.NoError(t, err)

	collector := NewStatsCollector()
	url, err := collector.getRemoteURL(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/test/repo.git", url)
}

func TestStatsCollector_GetRemoteURL_NotFound(t *testing.T) {
	// 创建临时仓库目录
	tmpDir, err := os.MkdirTemp("", "git-repo-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建没有 remote 的 .git/config 文件
	gitDir := filepath.Join(tmpDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	gitConfigContent := `[core]
	repositoryformatversion = 0
[branch "main"]
	merge = refs/heads/main
`
	err = os.WriteFile(filepath.Join(gitDir, "config"), []byte(gitConfigContent), 0644)
	require.NoError(t, err)

	collector := NewStatsCollector()
	_, err = collector.getRemoteURL(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remote origin not found")
}

func TestStatsCollector_CollectDailyStats_InvalidDate(t *testing.T) {
	collector := NewStatsCollector()

	// 使用无效的日期格式
	_, err := collector.CollectDailyStats("/some/path", "invalid-date", "test@example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date format")
}
