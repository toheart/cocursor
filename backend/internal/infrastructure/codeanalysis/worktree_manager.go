package codeanalysis

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cocursor/backend/internal/infrastructure/log"
)

// WorktreeManager 管理 git worktree 的创建和清理
type WorktreeManager struct {
	logger *slog.Logger
}

// NewWorktreeManager 创建 WorktreeManager
func NewWorktreeManager() *WorktreeManager {
	return &WorktreeManager{
		logger: log.NewModuleLogger("codeanalysis", "worktree_manager"),
	}
}

// WorktreeResult 创建 worktree 的结果
type WorktreeResult struct {
	// WorktreePath worktree 目录路径
	WorktreePath string
	// ResolvedCommit 解析后的完整 commit hash
	ResolvedCommit string
}

// CreateWorktree 为指定 commit 创建临时 worktree 并确保依赖可用
func (m *WorktreeManager) CreateWorktree(ctx context.Context, projectPath string, commit string) (*WorktreeResult, error) {
	// 解析 commit 为完整 hash
	resolvedCommit, err := m.resolveCommit(ctx, projectPath, commit)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve commit '%s': %w", commit, err)
	}

	// 生成临时目录名
	shortHash := resolvedCommit
	if len(shortHash) > 8 {
		shortHash = shortHash[:8]
	}
	randomSuffix := fmt.Sprintf("%06d", rand.Intn(1000000))
	worktreeDir := filepath.Join(os.TempDir(), fmt.Sprintf("cocursor-worktree-%s-%s", shortHash, randomSuffix))

	m.logger.Info("creating worktree",
		"project", projectPath,
		"commit", commit,
		"resolved_commit", resolvedCommit,
		"worktree_path", worktreeDir,
	)

	// 创建 worktree
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "git", "-C", projectPath, "worktree", "add", "--detach", worktreeDir, resolvedCommit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %s: %w", strings.TrimSpace(string(output)), err)
	}

	m.logger.Info("worktree created, running go mod download",
		"worktree_path", worktreeDir,
	)

	// 执行 go mod download 确保依赖可用
	modCtx, modCancel := context.WithTimeout(ctx, 120*time.Second)
	defer modCancel()

	modCmd := exec.CommandContext(modCtx, "go", "mod", "download")
	modCmd.Dir = worktreeDir
	modOutput, modErr := modCmd.CombinedOutput()
	if modErr != nil {
		// go mod download 失败时清理 worktree
		m.logger.Warn("go mod download failed, cleaning up worktree",
			"error", modErr,
			"output", strings.TrimSpace(string(modOutput)),
		)
		_ = m.RemoveWorktree(ctx, projectPath, worktreeDir)
		return nil, fmt.Errorf("failed to download go modules in worktree: %s: %w", strings.TrimSpace(string(modOutput)), modErr)
	}

	m.logger.Info("worktree ready",
		"worktree_path", worktreeDir,
		"resolved_commit", resolvedCommit,
	)

	return &WorktreeResult{
		WorktreePath:   worktreeDir,
		ResolvedCommit: resolvedCommit,
	}, nil
}

// RemoveWorktree 清理 worktree 目录
func (m *WorktreeManager) RemoveWorktree(ctx context.Context, projectPath string, worktreePath string) error {
	m.logger.Info("removing worktree",
		"project", projectPath,
		"worktree_path", worktreePath,
	)

	// 使用 git worktree remove 清理
	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "git", "-C", projectPath, "worktree", "remove", "--force", worktreePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		m.logger.Warn("git worktree remove failed, trying manual cleanup",
			"error", err,
			"output", strings.TrimSpace(string(output)),
		)
		// 手动清理目录
		if removeErr := os.RemoveAll(worktreePath); removeErr != nil {
			return fmt.Errorf("failed to remove worktree directory: %w", removeErr)
		}
		// 清理 git worktree 记录
		pruneCmd := exec.CommandContext(cmdCtx, "git", "-C", projectPath, "worktree", "prune")
		_ = pruneCmd.Run()
	}

	m.logger.Info("worktree removed",
		"worktree_path", worktreePath,
	)

	return nil
}

// IsHeadCommit 判断给定的 commit 是否就是当前 HEAD
func (m *WorktreeManager) IsHeadCommit(ctx context.Context, projectPath string, commit string) (bool, error) {
	if commit == "" || commit == "HEAD" {
		return true, nil
	}

	// 解析 commit
	resolvedCommit, err := m.resolveCommit(ctx, projectPath, commit)
	if err != nil {
		return false, err
	}

	// 获取 HEAD
	headCommit, err := m.resolveCommit(ctx, projectPath, "HEAD")
	if err != nil {
		return false, err
	}

	return resolvedCommit == headCommit, nil
}

// resolveCommit 将 commit 引用（分支名、tag、短 hash）解析为完整 hash
func (m *WorktreeManager) resolveCommit(ctx context.Context, projectPath string, commit string) (string, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "git", "-C", projectPath, "rev-parse", commit)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("commit '%s' not found: %s", commit, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("failed to resolve commit '%s': %w", commit, err)
	}

	return strings.TrimSpace(string(output)), nil
}
