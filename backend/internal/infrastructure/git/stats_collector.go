package git

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// 默认 git 命令超时时间
const DefaultGitTimeout = 10 * time.Second

// StatsCollector Git 统计收集器
type StatsCollector struct {
	logger *slog.Logger
}

// NewStatsCollector 创建 Git 统计收集器
func NewStatsCollector() *StatsCollector {
	return &StatsCollector{
		logger: log.NewModuleLogger("git", "stats_collector"),
	}
}

// GetUserEmail 获取 Git 用户邮箱
// 优先读取全局配置，如果未配置则返回空
func (c *StatsCollector) GetUserEmail() (string, error) {
	// 尝试从 git config 获取
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "config", "--global", "user.email")
	output, err := cmd.Output()
	if err != nil {
		// 尝试读取 ~/.gitconfig 文件
		return c.readEmailFromGitConfig()
	}

	email := strings.TrimSpace(string(output))
	if email != "" {
		return email, nil
	}

	return c.readEmailFromGitConfig()
}

// readEmailFromGitConfig 从 ~/.gitconfig 文件读取邮箱
func (c *StatsCollector) readEmailFromGitConfig() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".gitconfig")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read .gitconfig: %w", err)
	}

	// 解析 [user] 段的 email
	lines := strings.Split(string(content), "\n")
	inUserSection := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[user]") {
			inUserSection = true
			continue
		}
		if strings.HasPrefix(line, "[") && inUserSection {
			break
		}
		if inUserSection && strings.HasPrefix(line, "email") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "", fmt.Errorf("email not found in .gitconfig")
}

// GetRepoUserEmail 获取仓库级别的 Git 用户邮箱
func (c *StatsCollector) GetRepoUserEmail(repoPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "config", "user.email")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get repo user.email: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// FindRepoByRemoteURL 在本地查找匹配指定远程 URL 的仓库
// 搜索用户常见项目目录
func (c *StatsCollector) FindRepoByRemoteURL(repoURL string) (string, error) {
	// 规范化目标 URL
	targetURL := c.normalizeRepoURL(repoURL)

	// 搜索常见目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	searchDirs := []string{
		filepath.Join(homeDir, "code"),
		filepath.Join(homeDir, "Code"),
		filepath.Join(homeDir, "projects"),
		filepath.Join(homeDir, "Projects"),
		filepath.Join(homeDir, "workspace"),
		filepath.Join(homeDir, "Workspace"),
		filepath.Join(homeDir, "dev"),
		filepath.Join(homeDir, "Dev"),
		filepath.Join(homeDir, "src"),
		filepath.Join(homeDir, "go", "src"),
	}

	for _, dir := range searchDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		// 遍历目录
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			repoPath := filepath.Join(dir, entry.Name())
			url, err := c.getRemoteURL(repoPath)
			if err != nil {
				continue
			}

			if c.normalizeRepoURL(url) == targetURL {
				return repoPath, nil
			}
		}
	}

	return "", fmt.Errorf("repository not found for URL: %s", repoURL)
}

// getRemoteURL 获取仓库的 remote origin URL
func (c *StatsCollector) getRemoteURL(repoPath string) (string, error) {
	gitConfigPath := filepath.Join(repoPath, ".git", "config")
	if _, err := os.Stat(gitConfigPath); err != nil {
		return "", err
	}

	content, err := os.ReadFile(gitConfigPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	inRemoteOrigin := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "[remote \"origin\"]") {
			inRemoteOrigin = true
			continue
		}
		if strings.HasPrefix(line, "[") && inRemoteOrigin {
			break
		}
		if inRemoteOrigin && strings.HasPrefix(line, "url") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "", fmt.Errorf("remote origin not found")
}

// normalizeRepoURL 规范化仓库 URL
func (c *StatsCollector) normalizeRepoURL(url string) string {
	normalized := strings.ToLower(url)
	normalized = strings.TrimSuffix(normalized, ".git")

	// 移除协议前缀
	normalized = strings.TrimPrefix(normalized, "https://")
	normalized = strings.TrimPrefix(normalized, "http://")
	normalized = strings.TrimPrefix(normalized, "ssh://")

	// 处理 git@ 格式
	if strings.HasPrefix(normalized, "git@") {
		normalized = strings.TrimPrefix(normalized, "git@")
		normalized = strings.Replace(normalized, ":", "/", 1)
	}

	return normalized
}

// CollectDailyStats 收集指定日期的 Git 统计
func (c *StatsCollector) CollectDailyStats(repoPath, date, authorEmail string) (*domainTeam.ProjectGitStats, error) {
	// 验证日期格式
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// 获取项目名称
	projectName := filepath.Base(repoPath)

	// 获取 remote URL
	remoteURL, _ := c.getRemoteURL(repoPath)

	// 执行 git log
	stats, err := c.runGitLog(repoPath, date, authorEmail)
	if err != nil {
		return nil, err
	}

	stats.ProjectName = projectName
	stats.RepoURL = c.normalizeRepoURL(remoteURL)

	return stats, nil
}

// runGitLog 执行 git log 命令并解析结果
func (c *StatsCollector) runGitLog(repoPath, date, authorEmail string) (*domainTeam.ProjectGitStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultGitTimeout)
	defer cancel()

	// 构建时间范围
	since := date + " 00:00:00"
	until := date + " 23:59:59"

	// git log --author=email --since=date --until=date --stat --oneline
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "log",
		"--author="+authorEmail,
		"--since="+since,
		"--until="+until,
		"--stat",
		"--oneline",
		"--no-merges",
	)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// git log 没有结果时返回空
			if len(exitErr.Stderr) == 0 {
				return &domainTeam.ProjectGitStats{
					Commits:        0,
					LinesAdded:     0,
					LinesRemoved:   0,
					CommitMessages: []domainTeam.CommitSummary{},
				}, nil
			}
		}
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	return c.parseGitLogOutput(string(output))
}

// parseGitLogOutput 解析 git log 输出
func (c *StatsCollector) parseGitLogOutput(output string) (*domainTeam.ProjectGitStats, error) {
	stats := &domainTeam.ProjectGitStats{
		Commits:        0,
		LinesAdded:     0,
		LinesRemoved:   0,
		CommitMessages: []domainTeam.CommitSummary{},
	}

	if strings.TrimSpace(output) == "" {
		return stats, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(output))

	// 正则表达式
	// commit 行格式: abc1234 commit message here
	commitRegex := regexp.MustCompile(`^([a-f0-9]{7,40})\s+(.*)$`)
	// stat 总结行格式: 3 files changed, 10 insertions(+), 5 deletions(-)
	statRegex := regexp.MustCompile(`(\d+)\s+files?\s+changed(?:,\s+(\d+)\s+insertions?\(\+\))?(?:,\s+(\d+)\s+deletions?\(-\))?`)

	var currentCommit *domainTeam.CommitSummary
	var currentFilesCount int

	for scanner.Scan() {
		line := scanner.Text()

		// 检查是否是 commit 行
		if matches := commitRegex.FindStringSubmatch(line); matches != nil {
			// 保存前一个 commit
			if currentCommit != nil {
				currentCommit.FilesCount = currentFilesCount
				stats.CommitMessages = append(stats.CommitMessages, *currentCommit)
			}

			stats.Commits++
			currentCommit = &domainTeam.CommitSummary{
				Hash:    matches[1],
				Message: matches[2],
				Time:    "", // git log --oneline 不包含时间
			}
			currentFilesCount = 0
			continue
		}

		// 检查是否是 stat 总结行
		if matches := statRegex.FindStringSubmatch(line); matches != nil {
			filesChanged, _ := strconv.Atoi(matches[1])
			insertions := 0
			deletions := 0
			if matches[2] != "" {
				insertions, _ = strconv.Atoi(matches[2])
			}
			if matches[3] != "" {
				deletions, _ = strconv.Atoi(matches[3])
			}

			stats.LinesAdded += insertions
			stats.LinesRemoved += deletions
			currentFilesCount = filesChanged
		}
	}

	// 保存最后一个 commit
	if currentCommit != nil {
		currentCommit.FilesCount = currentFilesCount
		stats.CommitMessages = append(stats.CommitMessages, *currentCommit)
	}

	// 限制 commit messages 数量（最多 10 条）
	if len(stats.CommitMessages) > 10 {
		stats.CommitMessages = stats.CommitMessages[:10]
	}

	return stats, nil
}

// CollectWeeklyStats 收集一周的 Git 统计
func (c *StatsCollector) CollectWeeklyStats(repoPath, weekStart, authorEmail string) (map[string]*domainTeam.ProjectGitStats, error) {
	// 解析周起始日期
	startDate, err := time.Parse("2006-01-02", weekStart)
	if err != nil {
		return nil, fmt.Errorf("invalid week start date: %w", err)
	}

	result := make(map[string]*domainTeam.ProjectGitStats)

	// 收集 7 天的数据
	for i := 0; i < 7; i++ {
		date := startDate.AddDate(0, 0, i).Format("2006-01-02")
		stats, err := c.CollectDailyStats(repoPath, date, authorEmail)
		if err != nil {
			c.logger.Warn("failed to collect daily stats",
				"repoPath", repoPath,
				"date", date,
				"error", err,
			)
			// 继续收集其他日期
			stats = &domainTeam.ProjectGitStats{
				Commits:        0,
				LinesAdded:     0,
				LinesRemoved:   0,
				CommitMessages: []domainTeam.CommitSummary{},
			}
		}
		result[date] = stats
	}

	return result, nil
}
