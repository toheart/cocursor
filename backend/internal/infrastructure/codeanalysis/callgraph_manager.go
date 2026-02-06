package codeanalysis

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cocursor/backend/internal/domain/codeanalysis"
	"github.com/cocursor/backend/internal/infrastructure/config"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"gopkg.in/yaml.v3"
)

// CallGraphManager 调用图管理器实现
type CallGraphManager struct {
	baseDir    string
	logger     *slog.Logger
	repository *CallGraphRepository
}

// callGraphMeta 调用图元信息（用于 meta.yaml）
type callGraphMeta struct {
	ProjectID   string         `yaml:"project_id"`
	ProjectName string         `yaml:"project_name"`
	RemoteURL   string         `yaml:"remote_url"`
	CallGraphs  []callGraphRef `yaml:"callgraphs"`
	Latest      *latestRef     `yaml:"latest,omitempty"`
	Retention   retentionRef   `yaml:"retention"`
}

type callGraphRef struct {
	Commit           string    `yaml:"commit"`
	Branch           string    `yaml:"branch"`
	CreatedAt        time.Time `yaml:"created_at"`
	DBFile           string    `yaml:"db_file"`
	FuncCount        int       `yaml:"func_count"`
	EdgeCount        int       `yaml:"edge_count"`
	GenerationTimeMs int64     `yaml:"generation_time_ms"`
}

type latestRef struct {
	Commit string `yaml:"commit"`
	DBFile string `yaml:"db_file"`
}

type retentionRef struct {
	MaxCount   int `yaml:"max_count"`
	MaxAgeDays int `yaml:"max_age_days"`
}

// NewCallGraphManager 创建调用图管理器
func NewCallGraphManager(repository *CallGraphRepository) (*CallGraphManager, error) {
	baseDir := filepath.Join(config.GetDataDir(), "analysis", "callgraphs")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create callgraphs directory: %w", err)
	}

	return &CallGraphManager{
		baseDir:    baseDir,
		logger:     log.NewModuleLogger("codeanalysis", "callgraph_manager"),
		repository: repository,
	}, nil
}

// GetProjectDir 获取项目的调用图存储目录
func (m *CallGraphManager) GetProjectDir(projectID string) string {
	return filepath.Join(m.baseDir, projectID)
}

// GetCommitDBPath 获取指定 commit 的数据库文件路径
func (m *CallGraphManager) GetCommitDBPath(projectID string, commit string) string {
	// 使用短 commit（前 7 位）
	shortCommit := commit
	if len(commit) > 7 {
		shortCommit = commit[:7]
	}
	return filepath.Join(m.GetProjectDir(projectID), "commits", shortCommit+".db")
}

// ListCommits 列出项目的所有 commit 版本
func (m *CallGraphManager) ListCommits(_ context.Context, projectID string) ([]codeanalysis.CallGraph, error) {
	meta, err := m.loadMeta(projectID)
	if err != nil {
		return nil, err
	}

	var result []codeanalysis.CallGraph
	for _, ref := range meta.CallGraphs {
		result = append(result, codeanalysis.CallGraph{
			Commit:           ref.Commit,
			Branch:           ref.Branch,
			FuncCount:        ref.FuncCount,
			EdgeCount:        ref.EdgeCount,
			DBPath:           filepath.Join(m.GetProjectDir(projectID), ref.DBFile),
			CreatedAt:        ref.CreatedAt,
			GenerationTimeMs: ref.GenerationTimeMs,
		})
	}

	return result, nil
}

// GetLatest 获取最新的调用图
func (m *CallGraphManager) GetLatest(ctx context.Context, projectID string) (*codeanalysis.CallGraph, error) {
	meta, err := m.loadMeta(projectID)
	if err != nil {
		return nil, err
	}

	if meta.Latest == nil {
		return nil, fmt.Errorf("no latest call graph found for project: %s", projectID)
	}

	// 从列表中找到对应的调用图
	for _, ref := range meta.CallGraphs {
		if ref.Commit == meta.Latest.Commit {
			dbPath := filepath.Join(m.GetProjectDir(projectID), ref.DBFile)

			// 获取实际的函数和边数量
			funcCount, _ := m.repository.GetFuncCount(ctx, dbPath)
			edgeCount, _ := m.repository.GetEdgeCount(ctx, dbPath)

			return &codeanalysis.CallGraph{
				Commit:           ref.Commit,
				Branch:           ref.Branch,
				FuncCount:        funcCount,
				EdgeCount:        edgeCount,
				DBPath:           dbPath,
				CreatedAt:        ref.CreatedAt,
				GenerationTimeMs: ref.GenerationTimeMs,
			}, nil
		}
	}

	return nil, fmt.Errorf("latest commit not found in call graph list: %s", meta.Latest.Commit)
}

// SetLatest 设置最新的调用图
func (m *CallGraphManager) SetLatest(_ context.Context, projectID string, commit string) error {
	meta, err := m.loadMeta(projectID)
	if err != nil {
		// 如果 meta 不存在，创建新的
		meta = &callGraphMeta{
			ProjectID: projectID,
			Retention: retentionRef{
				MaxCount:   10,
				MaxAgeDays: 30,
			},
		}
	}

	// 找到对应的调用图
	var found bool
	for _, ref := range meta.CallGraphs {
		if ref.Commit == commit || strings.HasPrefix(ref.Commit, commit) {
			meta.Latest = &latestRef{
				Commit: ref.Commit,
				DBFile: ref.DBFile,
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("commit not found: %s", commit)
	}

	return m.saveMeta(projectID, meta)
}

// DeleteCommit 删除指定 commit 的调用图
func (m *CallGraphManager) DeleteCommit(_ context.Context, projectID string, commit string) error {
	meta, err := m.loadMeta(projectID)
	if err != nil {
		return err
	}

	// 找到并删除
	var newCallGraphs []callGraphRef
	var dbFileToDelete string
	for _, ref := range meta.CallGraphs {
		if ref.Commit == commit || strings.HasPrefix(ref.Commit, commit) {
			dbFileToDelete = ref.DBFile
		} else {
			newCallGraphs = append(newCallGraphs, ref)
		}
	}

	if dbFileToDelete != "" {
		dbPath := filepath.Join(m.GetProjectDir(projectID), dbFileToDelete)
		os.Remove(dbPath)
	}

	meta.CallGraphs = newCallGraphs

	// 如果删除的是 latest，更新 latest
	if meta.Latest != nil && (meta.Latest.Commit == commit || strings.HasPrefix(meta.Latest.Commit, commit)) {
		if len(newCallGraphs) > 0 {
			// 选择最新的
			sort.Slice(newCallGraphs, func(i, j int) bool {
				return newCallGraphs[i].CreatedAt.After(newCallGraphs[j].CreatedAt)
			})
			meta.Latest = &latestRef{
				Commit: newCallGraphs[0].Commit,
				DBFile: newCallGraphs[0].DBFile,
			}
		} else {
			meta.Latest = nil
		}
	}

	return m.saveMeta(projectID, meta)
}

// CleanOldVersions 清理旧版本
func (m *CallGraphManager) CleanOldVersions(_ context.Context, projectID string, maxCount int, maxAgeDays int) error {
	meta, err := m.loadMeta(projectID)
	if err != nil {
		return err
	}

	if len(meta.CallGraphs) <= maxCount {
		return nil
	}

	// 按时间排序（最新的在前）
	sort.Slice(meta.CallGraphs, func(i, j int) bool {
		return meta.CallGraphs[i].CreatedAt.After(meta.CallGraphs[j].CreatedAt)
	})

	cutoffTime := time.Now().AddDate(0, 0, -maxAgeDays)

	var toKeep []callGraphRef
	var toDelete []callGraphRef

	for i, ref := range meta.CallGraphs {
		// 保留最新的 maxCount 个，或者在时间范围内的
		if i < maxCount && ref.CreatedAt.After(cutoffTime) {
			toKeep = append(toKeep, ref)
		} else if ref.CreatedAt.After(cutoffTime) && len(toKeep) < maxCount {
			toKeep = append(toKeep, ref)
		} else {
			toDelete = append(toDelete, ref)
		}
	}

	// 删除文件
	for _, ref := range toDelete {
		dbPath := filepath.Join(m.GetProjectDir(projectID), ref.DBFile)
		os.Remove(dbPath)
		m.logger.Info("deleted old call graph", "project", projectID, "commit", ref.Commit)
	}

	meta.CallGraphs = toKeep
	return m.saveMeta(projectID, meta)
}

// GetCallGraphStatus 获取调用图状态
func (m *CallGraphManager) GetCallGraphStatus(ctx context.Context, projectID string, projectPath string, targetCommit string) (*codeanalysis.CallGraphStatus, error) {
	status := &codeanalysis.CallGraphStatus{
		ProjectRegistered: projectID != "",
	}

	if projectID == "" {
		return status, nil
	}

	// 获取当前 HEAD commit
	headCommit, err := m.getCurrentCommit(projectPath)
	if err != nil {
		m.logger.Warn("failed to get current commit", "error", err)
	}
	status.HeadCommit = headCommit

	// 如果没有指定 target，使用 HEAD
	if targetCommit == "" || targetCommit == "HEAD" {
		targetCommit = headCommit
	}

	// 检查是否有调用图
	meta, err := m.loadMeta(projectID)
	if err != nil {
		status.Exists = false
		return status, nil
	}

	// 检查是否有匹配的调用图
	for _, ref := range meta.CallGraphs {
		if ref.Commit == targetCommit || strings.HasPrefix(ref.Commit, targetCommit) || strings.HasPrefix(targetCommit, ref.Commit) {
			dbPath := filepath.Join(m.GetProjectDir(projectID), ref.DBFile)

			// 验证文件存在
			if _, err := os.Stat(dbPath); err == nil {
				status.Exists = true
				status.CurrentCommit = ref.Commit
				status.DBPath = dbPath
				status.CreatedAt = &ref.CreatedAt
				status.FuncCount = ref.FuncCount

				// 判断是否最新
				status.UpToDate = (ref.Commit == headCommit || strings.HasPrefix(headCommit, ref.Commit))

				if !status.UpToDate && headCommit != "" {
					// 计算落后多少 commit
					commitsBehind, _ := m.getCommitsBetween(projectPath, ref.Commit, headCommit)
					status.CommitsBehind = commitsBehind
				}

				return status, nil
			}
		}
	}

	status.Exists = false
	return status, nil
}

// SaveCallGraph 保存调用图信息
func (m *CallGraphManager) SaveCallGraph(ctx context.Context, projectID string, projectName string, remoteURL string, cg *codeanalysis.CallGraph) error {
	// 确保目录存在
	projectDir := m.GetProjectDir(projectID)
	commitsDir := filepath.Join(projectDir, "commits")
	if err := os.MkdirAll(commitsDir, 0755); err != nil {
		return err
	}

	// 加载或创建 meta
	meta, err := m.loadMeta(projectID)
	if err != nil {
		meta = &callGraphMeta{
			ProjectID:   projectID,
			ProjectName: projectName,
			RemoteURL:   remoteURL,
			Retention: retentionRef{
				MaxCount:   10,
				MaxAgeDays: 30,
			},
		}
	}

	// 更新或添加调用图记录
	shortCommit := cg.Commit
	if len(shortCommit) > 7 {
		shortCommit = shortCommit[:7]
	}

	ref := callGraphRef{
		Commit:           cg.Commit,
		Branch:           cg.Branch,
		CreatedAt:        cg.CreatedAt,
		DBFile:           filepath.Join("commits", shortCommit+".db"),
		FuncCount:        cg.FuncCount,
		EdgeCount:        cg.EdgeCount,
		GenerationTimeMs: cg.GenerationTimeMs,
	}

	// 检查是否已存在
	found := false
	for i, existing := range meta.CallGraphs {
		if existing.Commit == cg.Commit {
			meta.CallGraphs[i] = ref
			found = true
			break
		}
	}
	if !found {
		meta.CallGraphs = append(meta.CallGraphs, ref)
	}

	// 更新 latest
	meta.Latest = &latestRef{
		Commit: cg.Commit,
		DBFile: ref.DBFile,
	}

	return m.saveMeta(projectID, meta)
}

// loadMeta 加载项目元信息
func (m *CallGraphManager) loadMeta(projectID string) (*callGraphMeta, error) {
	metaPath := filepath.Join(m.GetProjectDir(projectID), "meta.yaml")

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var meta callGraphMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// saveMeta 保存项目元信息
func (m *CallGraphManager) saveMeta(projectID string, meta *callGraphMeta) error {
	projectDir := m.GetProjectDir(projectID)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return err
	}

	metaPath := filepath.Join(projectDir, "meta.yaml")

	data, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}

	return os.WriteFile(metaPath, data, 0644)
}

// getCurrentCommit 获取当前 HEAD commit
func (m *CallGraphManager) getCurrentCommit(projectPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", projectPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// getCommitsBetween 获取两个 commit 之间的 commit 数量
func (m *CallGraphManager) getCommitsBetween(projectPath string, fromCommit string, toCommit string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", projectPath, "rev-list", "--count", fromCommit+".."+toCommit)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var count int
	fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count)
	return count, nil
}

// 确保实现接口
var _ codeanalysis.CallGraphManager = (*CallGraphManager)(nil)
