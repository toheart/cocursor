package codeanalysis

import "time"

// AlgorithmType 调用图分析算法类型
type AlgorithmType string

const (
	// AlgorithmStatic 静态分析，最快但精度最低
	AlgorithmStatic AlgorithmType = "static"
	// AlgorithmCHA 类层次分析，保守但快速
	AlgorithmCHA AlgorithmType = "cha"
	// AlgorithmRTA 快速类型分析，推荐使用，平衡精度和性能
	AlgorithmRTA AlgorithmType = "rta"
	// AlgorithmVTA 变量类型分析，最精确但最慢
	AlgorithmVTA AlgorithmType = "vta"
)

// ChangeType 函数变更类型
type ChangeType string

const (
	// ChangeTypeModified 函数被修改
	ChangeTypeModified ChangeType = "modified"
	// ChangeTypeAdded 新增函数
	ChangeTypeAdded ChangeType = "added"
	// ChangeTypeDeleted 删除函数
	ChangeTypeDeleted ChangeType = "deleted"
)

// Project 项目配置
type Project struct {
	// ID 项目唯一标识（基于 remote URL 的 hash）
	ID string `json:"id" yaml:"id"`
	// Name 项目名称
	Name string `json:"name" yaml:"name"`
	// RemoteURL 远程仓库 URL（规范化后）
	RemoteURL string `json:"remote_url" yaml:"remote_url"`
	// LocalPaths 本地路径列表（同一项目可能有多个本地副本）
	LocalPaths []string `json:"local_paths" yaml:"local_paths"`
	// EntryPoints 入口函数列表（格式: file:func，如 cmd/server/main.go:main）
	EntryPoints []string `json:"entry_points" yaml:"entry_points"`
	// Exclude 排除路径列表
	Exclude []string `json:"exclude" yaml:"exclude"`
	// Algorithm 分析算法
	Algorithm AlgorithmType `json:"algorithm" yaml:"algorithm"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// EntryPointCandidate 入口函数候选
type EntryPointCandidate struct {
	// File 文件路径（相对于项目根目录）
	File string `json:"file"`
	// Function 函数名
	Function string `json:"function"`
	// Type 入口类型：cmd, root, all_exported
	Type string `json:"type"`
	// Priority 优先级，数字越小优先级越高
	Priority int `json:"priority"`
	// Recommended 是否推荐
	Recommended bool `json:"recommended"`
}

// FuncNode 函数节点
type FuncNode struct {
	// ID 节点 ID（数据库自增）
	ID int64 `json:"id"`
	// FullName 完整函数名（包含包路径），如 github.com/example/pkg.FuncName
	FullName string `json:"full_name"`
	// Package 包路径
	Package string `json:"package"`
	// FuncName 函数名
	FuncName string `json:"func_name"`
	// FilePath 源文件路径（相对于项目根目录）
	FilePath string `json:"file_path"`
	// LineStart 起始行号
	LineStart int `json:"line_start"`
	// LineEnd 结束行号
	LineEnd int `json:"line_end"`
	// IsExported 是否为导出函数
	IsExported bool `json:"is_exported"`
}

// FuncEdge 函数调用边
type FuncEdge struct {
	// ID 边 ID（数据库自增）
	ID int64 `json:"id"`
	// CallerID 调用者函数 ID
	CallerID int64 `json:"caller_id"`
	// CalleeID 被调用者函数 ID
	CalleeID int64 `json:"callee_id"`
	// CallSiteFile 调用位置文件
	CallSiteFile string `json:"call_site_file"`
	// CallSiteLine 调用位置行号
	CallSiteLine int `json:"call_site_line"`
}

// CallGraph 调用图元数据
type CallGraph struct {
	// Commit Git commit hash（短 hash）
	Commit string `json:"commit"`
	// FullCommit 完整 commit hash
	FullCommit string `json:"full_commit"`
	// Branch 分支名
	Branch string `json:"branch"`
	// Algorithm 使用的分析算法
	Algorithm AlgorithmType `json:"algorithm"`
	// FuncCount 函数数量
	FuncCount int `json:"func_count"`
	// EdgeCount 调用边数量
	EdgeCount int `json:"edge_count"`
	// DBPath 数据库文件路径
	DBPath string `json:"db_path"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
	// GenerationTimeMs 生成耗时（毫秒）
	GenerationTimeMs int64 `json:"generation_time_ms"`
	// ActualAlgorithm 实际使用的算法（可能因降级而与配置不同）
	ActualAlgorithm AlgorithmType `json:"actual_algorithm,omitempty"`
	// Fallback 是否发生了算法降级
	Fallback bool `json:"fallback,omitempty"`
	// FallbackReason 降级原因
	FallbackReason string `json:"fallback_reason,omitempty"`
}

// CallGraphStatus 调用图状态
type CallGraphStatus struct {
	// Exists 调用图是否存在
	Exists bool `json:"exists"`
	// UpToDate 是否为最新（与当前 HEAD 匹配）
	UpToDate bool `json:"up_to_date"`
	// CurrentCommit 当前调用图对应的 commit
	CurrentCommit string `json:"current_commit,omitempty"`
	// HeadCommit 当前 HEAD commit
	HeadCommit string `json:"head_commit,omitempty"`
	// CommitsBehind 落后的 commit 数量
	CommitsBehind int `json:"commits_behind,omitempty"`
	// ProjectRegistered 项目是否已注册
	ProjectRegistered bool `json:"project_registered"`
	// DBPath 数据库文件路径
	DBPath string `json:"db_path,omitempty"`
	// CreatedAt 调用图创建时间
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// FuncCount 函数数量
	FuncCount int `json:"func_count,omitempty"`
	// ValidGoModule 当前路径是否为有效的 Go 模块
	ValidGoModule bool `json:"valid_go_module"`
	// GoModuleError Go 模块验证错误信息
	GoModuleError string `json:"go_module_error,omitempty"`
}

// ChangedFunction 变更的函数
type ChangedFunction struct {
	// Name 函数名
	Name string `json:"name"`
	// FullName 完整函数名（包含包路径）
	FullName string `json:"full_name"`
	// Package 包路径
	Package string `json:"package"`
	// File 文件路径
	File string `json:"file"`
	// LineStart 起始行号
	LineStart int `json:"line_start"`
	// LineEnd 结束行号
	LineEnd int `json:"line_end"`
	// ChangeType 变更类型
	ChangeType ChangeType `json:"change_type"`
	// LinesAdded 新增行数
	LinesAdded int `json:"lines_added"`
	// LinesRemoved 删除行数
	LinesRemoved int `json:"lines_removed"`
}

// DiffAnalysisResult Git diff 分析结果
type DiffAnalysisResult struct {
	// CommitRange 分析的 commit 范围
	CommitRange string `json:"commit_range"`
	// ChangedFunctions 变更的函数列表
	ChangedFunctions []ChangedFunction `json:"changed_functions"`
	// ChangedFiles 变更的文件列表
	ChangedFiles []string `json:"changed_files"`
}

// CallerInfo 调用者信息
type CallerInfo struct {
	// Function 完整函数名
	Function string `json:"function"`
	// DisplayName 显示名称（短函数名）
	DisplayName string `json:"display_name"`
	// Package 包路径
	Package string `json:"package"`
	// File 文件路径
	File string `json:"file"`
	// Line 调用行号
	Line int `json:"line"`
	// Depth 调用深度（从变更函数开始计算）
	Depth int `json:"depth"`
}

// FunctionImpact 单个函数的影响分析
type FunctionImpact struct {
	// Function 完整函数名
	Function string `json:"function"`
	// DisplayName 显示名称（短函数名）
	DisplayName string `json:"display_name"`
	// File 文件路径
	File string `json:"file"`
	// Callers 调用者列表
	Callers []CallerInfo `json:"callers"`
	// TotalCallers 调用者总数
	TotalCallers int `json:"total_callers"`
	// MaxDepthReached 达到的最大深度
	MaxDepthReached int `json:"max_depth_reached"`
}

// ImpactSummary 影响分析摘要
type ImpactSummary struct {
	// FunctionsAnalyzed 分析的函数数量
	FunctionsAnalyzed int `json:"functions_analyzed"`
	// TotalAffected 受影响的函数总数
	TotalAffected int `json:"total_affected"`
	// AffectedFiles 受影响的文件列表
	AffectedFiles []string `json:"affected_files"`
}

// ImpactAnalysisResult 影响分析结果
type ImpactAnalysisResult struct {
	// AnalysisCommit 分析使用的 commit
	AnalysisCommit string `json:"analysis_commit"`
	// Impacts 各函数的影响分析
	Impacts []FunctionImpact `json:"impacts"`
	// Summary 摘要信息
	Summary ImpactSummary `json:"summary"`
}

// GenerationTask 调用图生成任务
type GenerationTask struct {
	// TaskID 任务 ID
	TaskID string `json:"task_id"`
	// ProjectID 项目 ID
	ProjectID string `json:"project_id"`
	// ProjectPath 项目路径
	ProjectPath string `json:"project_path"`
	// Commit 目标 commit
	Commit string `json:"commit"`
	// Status 任务状态：pending, running, completed, failed
	Status string `json:"status"`
	// Progress 进度（0-100）
	Progress int `json:"progress"`
	// Message 状态消息
	Message string `json:"message"`
	// Result 生成结果（完成时填充）
	Result *CallGraph `json:"result,omitempty"`
	// Error 错误信息（失败时填充）
	Error string `json:"error,omitempty"`
	// StartedAt 开始时间
	StartedAt *time.Time `json:"started_at,omitempty"`
	// CompletedAt 完成时间
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
