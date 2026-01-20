package cursor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GitReader Git 信息读取器
type GitReader struct{}

// NewGitReader 创建 Git 信息读取器实例
func NewGitReader() *GitReader {
	return &GitReader{}
}

// ReadGitRemoteURL 读取 Git 远程仓库 URL
// projectPath: 项目路径
// 返回: Git 远程 URL（已规范化），如果不存在返回空字符串和错误
func (r *GitReader) ReadGitRemoteURL(projectPath string) (string, error) {
	gitConfigPath := filepath.Join(projectPath, ".git", "config")

	if _, err := os.Stat(gitConfigPath); err != nil {
		return "", fmt.Errorf("git repository not found: %w", err)
	}

	file, err := os.Open(gitConfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to read git config: %w", err)
	}
	defer file.Close()

	// 读取文件内容，查找 [remote "origin"] 部分
	scanner := bufio.NewScanner(file)
	inRemoteOrigin := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 检测是否进入 [remote "origin"] 部分
		if strings.Contains(line, "[remote \"origin\"]") {
			inRemoteOrigin = true
			continue
		}

		// 如果遇到新的 section，退出 origin 部分
		if strings.HasPrefix(line, "[") && inRemoteOrigin {
			break
		}

		// 在 origin 部分查找 url
		if inRemoteOrigin && strings.HasPrefix(line, "url =") {
			url := strings.TrimSpace(strings.TrimPrefix(line, "url ="))
			if url != "" {
				// 规范化 URL
				normalizedURL := r.normalizeGitURL(url)
				return normalizedURL, nil
			}
		}
	}

	return "", fmt.Errorf("remote \"origin\" not found in git config")
}

// ReadGitBranch 读取 Git 当前分支
// projectPath: 项目路径
// 返回: 当前分支名，如果不存在返回空字符串和错误
func (r *GitReader) ReadGitBranch(projectPath string) (string, error) {
	headPath := filepath.Join(projectPath, ".git", "HEAD")

	if _, err := os.Stat(headPath); err != nil {
		return "", fmt.Errorf("git repository not found: %w", err)
	}

	content, err := os.ReadFile(headPath)
	if err != nil {
		return "", fmt.Errorf("failed to read git HEAD: %w", err)
	}

	// 解析分支引用：ref: refs/heads/main
	line := strings.TrimSpace(string(content))
	if strings.HasPrefix(line, "ref: refs/heads/") {
		branch := strings.TrimPrefix(line, "ref: refs/heads/")
		return branch, nil
	}

	// 如果是 detached HEAD（直接是 commit hash），返回空
	return "", fmt.Errorf("git branch not found (detached HEAD)")
}

// normalizeGitURL 规范化 Git URL
// 统一协议、大小写、移除 .git 后缀
func (r *GitReader) normalizeGitURL(url string) string {
	// 1. 先转小写（确保后续比较是大小写不敏感的）
	normalized := strings.ToLower(url)

	// 2. 移除 .git 后缀（现在是小写了）
	normalized = strings.TrimSuffix(normalized, ".git")

	// 3. 统一协议：git@github.com: -> https://github.com/
	// 注意：先处理 ssh:// 格式，再处理 git@ 格式
	if strings.HasPrefix(normalized, "ssh://git@") {
		normalized = strings.Replace(normalized, "ssh://git@github.com/", "https://github.com/", 1)
		normalized = strings.Replace(normalized, "ssh://git@gitlab.com/", "https://gitlab.com/", 1)
	} else if strings.HasPrefix(normalized, "git@") {
		normalized = strings.Replace(normalized, "git@github.com:", "https://github.com/", 1)
		normalized = strings.Replace(normalized, "git@gitlab.com:", "https://gitlab.com/", 1)
	}

	return normalized
}
