package codeanalysis

import "context"

// ProjectRepository 项目配置仓库接口
type ProjectRepository interface {
	// GetByID 根据 ID 获取项目配置
	GetByID(ctx context.Context, id string) (*Project, error)

	// GetByPath 根据本地路径获取项目配置
	GetByPath(ctx context.Context, path string) (*Project, error)

	// GetByRemoteURL 根据远程 URL 获取项目配置
	GetByRemoteURL(ctx context.Context, remoteURL string) (*Project, error)

	// List 获取所有项目配置
	List(ctx context.Context) ([]*Project, error)

	// Save 保存项目配置（新增或更新）
	Save(ctx context.Context, project *Project) error

	// Delete 删除项目配置
	Delete(ctx context.Context, id string) error

	// AddLocalPath 为项目添加本地路径
	AddLocalPath(ctx context.Context, id string, path string) error

	// RemoveLocalPath 移除项目的本地路径
	RemoveLocalPath(ctx context.Context, id string, path string) error
}

// CallGraphRepository 调用图存储仓库接口
type CallGraphRepository interface {
	// Init 初始化数据库表结构
	Init(ctx context.Context, dbPath string) error

	// SaveFuncNode 保存函数节点
	SaveFuncNode(ctx context.Context, dbPath string, node *FuncNode) (int64, error)

	// SaveFuncNodes 批量保存函数节点
	SaveFuncNodes(ctx context.Context, dbPath string, nodes []*FuncNode) error

	// SaveFuncEdge 保存调用边
	SaveFuncEdge(ctx context.Context, dbPath string, edge *FuncEdge) error

	// SaveFuncEdges 批量保存调用边
	SaveFuncEdges(ctx context.Context, dbPath string, edges []*FuncEdge) error

	// SaveMetadata 保存元数据
	SaveMetadata(ctx context.Context, dbPath string, key string, value string) error

	// GetMetadata 获取元数据
	GetMetadata(ctx context.Context, dbPath string, key string) (string, error)

	// GetFuncNodeByFullName 根据完整函数名获取节点
	GetFuncNodeByFullName(ctx context.Context, dbPath string, fullName string) (*FuncNode, error)

	// GetFuncNodesByFullNames 批量获取函数节点
	GetFuncNodesByFullNames(ctx context.Context, dbPath string, fullNames []string) ([]*FuncNode, error)

	// GetFuncNodeByFile 获取文件中的所有函数节点
	GetFuncNodeByFile(ctx context.Context, dbPath string, filePath string) ([]*FuncNode, error)

	// GetCallers 获取函数的所有调用者
	GetCallers(ctx context.Context, dbPath string, funcID int64) ([]*FuncNode, error)

	// GetCallersWithDepth 递归获取函数的调用者（带深度限制）
	GetCallersWithDepth(ctx context.Context, dbPath string, funcIDs []int64, maxDepth int) ([]CallerInfo, error)

	// GetFuncCount 获取函数数量
	GetFuncCount(ctx context.Context, dbPath string) (int, error)

	// GetEdgeCount 获取调用边数量
	GetEdgeCount(ctx context.Context, dbPath string) (int, error)
}

// CallGraphManager 调用图管理器接口
type CallGraphManager interface {
	// GetProjectDir 获取项目的调用图存储目录
	GetProjectDir(projectID string) string

	// GetCommitDBPath 获取指定 commit 的数据库文件路径
	GetCommitDBPath(projectID string, commit string) string

	// ListCommits 列出项目的所有 commit 版本
	ListCommits(ctx context.Context, projectID string) ([]CallGraph, error)

	// GetLatest 获取最新的调用图
	GetLatest(ctx context.Context, projectID string) (*CallGraph, error)

	// SetLatest 设置最新的调用图
	SetLatest(ctx context.Context, projectID string, commit string) error

	// DeleteCommit 删除指定 commit 的调用图
	DeleteCommit(ctx context.Context, projectID string, commit string) error

	// CleanOldVersions 清理旧版本（根据保留策略）
	CleanOldVersions(ctx context.Context, projectID string, maxCount int, maxAgeDays int) error

	// GetCallGraphStatus 获取调用图状态
	GetCallGraphStatus(ctx context.Context, projectID string, projectPath string, targetCommit string) (*CallGraphStatus, error)
}

// EntryPointScanner 入口函数扫描器接口
type EntryPointScanner interface {
	// ScanEntryPoints 扫描项目中的入口函数候选
	ScanEntryPoints(ctx context.Context, projectPath string) ([]EntryPointCandidate, error)

	// GetModulePath 获取项目的 go.mod 模块路径
	GetModulePath(ctx context.Context, projectPath string) (string, error)

	// GetRemoteURL 获取项目的 remote URL
	GetRemoteURL(ctx context.Context, projectPath string) (string, error)
}

// SSAAnalyzer SSA 分析器接口
type SSAAnalyzer interface {
	// Analyze 分析项目，生成调用图
	Analyze(ctx context.Context, projectPath string, entryPoints []string, algorithm AlgorithmType) (*AnalysisResult, error)
}

// AnalysisResult SSA 分析结果
type AnalysisResult struct {
	// ModulePath 模块路径
	ModulePath string
	// FuncNodes 函数节点列表
	FuncNodes []*FuncNode
	// FuncEdges 调用边列表
	FuncEdges []*FuncEdge
	// ActualAlgorithm 实际使用的算法（可能因降级而与请求的算法不同）
	ActualAlgorithm AlgorithmType
	// Fallback 是否发生了算法降级
	Fallback bool
	// FallbackReason 降级原因（如果发生了降级）
	FallbackReason string
}

// DiffAnalyzer Git diff 分析器接口
type DiffAnalyzer interface {
	// AnalyzeDiff 分析 Git diff，返回变更的函数列表
	AnalyzeDiff(ctx context.Context, projectPath string, commitRange string) (*DiffAnalysisResult, error)

	// GetCurrentCommit 获取当前 HEAD commit
	GetCurrentCommit(ctx context.Context, projectPath string) (string, error)

	// GetCommitsBetween 获取两个 commit 之间的 commit 数量
	GetCommitsBetween(ctx context.Context, projectPath string, fromCommit string, toCommit string) (int, error)
}

// ImpactAnalyzer 影响面分析器接口
type ImpactAnalyzer interface {
	// AnalyzeImpact 分析函数变更的影响面
	AnalyzeImpact(ctx context.Context, dbPath string, functions []string, maxDepth int) (*ImpactAnalysisResult, error)
}
