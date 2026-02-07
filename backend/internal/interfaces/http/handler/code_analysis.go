package handler

import (
	"net/http"

	"github.com/cocursor/backend/internal/application/codeanalysis"
	domainCodeanalysis "github.com/cocursor/backend/internal/domain/codeanalysis"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// CodeAnalysisHandler 代码分析 API 处理器
type CodeAnalysisHandler struct {
	projectService   *codeanalysis.ProjectService
	callGraphService *codeanalysis.CallGraphService
	impactService    *codeanalysis.ImpactService
}

// NewCodeAnalysisHandler 创建代码分析处理器
func NewCodeAnalysisHandler(
	projectService *codeanalysis.ProjectService,
	callGraphService *codeanalysis.CallGraphService,
	impactService *codeanalysis.ImpactService,
) *CodeAnalysisHandler {
	return &CodeAnalysisHandler{
		projectService:   projectService,
		callGraphService: callGraphService,
		impactService:    impactService,
	}
}

// ScanEntryPointsRequest 扫描入口函数请求
type ScanEntryPointsRequest struct {
	ProjectPath string `json:"project_path" binding:"required"`
}

// ScanEntryPoints 扫描入口函数
// @Summary 扫描入口函数
// @Description 扫描项目中的入口函数候选
// @Tags 代码分析
// @Accept json
// @Produce json
// @Param request body ScanEntryPointsRequest true "请求参数"
// @Success 200 {object} response.Response{data=codeanalysis.ScanEntryPointsResponse}
// @Router /api/v1/analysis/scan-entry-points [post]
func (h *CodeAnalysisHandler) ScanEntryPoints(c *gin.Context) {
	var req ScanEntryPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	result, err := h.projectService.ScanEntryPoints(c.Request.Context(), &codeanalysis.ScanEntryPointsRequest{
		ProjectPath: req.ProjectPath,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	response.Success(c, result)
}

// RegisterProjectRequest 注册项目请求
type RegisterProjectRequest struct {
	ProjectPath string   `json:"project_path" binding:"required"`
	EntryPoints []string `json:"entry_points" binding:"required"`
	Exclude     []string `json:"exclude"`
	Algorithm   string   `json:"algorithm"`
}

// GetProjectConfigRequest 获取项目配置请求
type GetProjectConfigRequest struct {
	ProjectPath string `json:"project_path" binding:"required"`
}

// RegisterProject 注册项目
// @Summary 注册项目
// @Description 注册或更新项目配置（Deprecated: 请使用 /callgraph/generate-with-config）
// @Tags 代码分析
// @Accept json
// @Produce json
// @Param request body RegisterProjectRequest true "请求参数"
// @Success 200 {object} response.Response{data=codeanalysis.RegisterProjectResponse}
// @Router /api/v1/analysis/projects [post]
func (h *CodeAnalysisHandler) RegisterProject(c *gin.Context) {
	var req RegisterProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	result, err := h.projectService.RegisterProject(c.Request.Context(), &codeanalysis.RegisterProjectRequest{
		ProjectPath: req.ProjectPath,
		EntryPoints: req.EntryPoints,
		Exclude:     req.Exclude,
		Algorithm:   parseAlgorithm(req.Algorithm),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	response.Success(c, result)
}

// GetProjectConfig 获取项目配置
// @Summary 获取项目配置
// @Description 获取项目已保存的入口函数、算法配置
// @Tags 代码分析
// @Accept json
// @Produce json
// @Param request body GetProjectConfigRequest true "请求参数"
// @Success 200 {object} response.Response{data=domainCodeanalysis.Project}
// @Router /api/v1/analysis/projects/config [post]
func (h *CodeAnalysisHandler) GetProjectConfig(c *gin.Context) {
	var req GetProjectConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	result, err := h.projectService.GetProject(c.Request.Context(), req.ProjectPath)
	if err != nil {
		response.Error(c, http.StatusNotFound, 404, err.Error())
		return
	}

	response.Success(c, result)
}

// CheckCallGraphStatusRequest 检查调用图状态请求
type CheckCallGraphStatusRequest struct {
	ProjectPath string `json:"project_path" binding:"required"`
	Commit      string `json:"commit"`
}

// CheckCallGraphStatus 检查调用图状态
// @Summary 检查调用图状态
// @Description 检查项目的调用图是否存在及是否最新
// @Tags 代码分析
// @Accept json
// @Produce json
// @Param request body CheckCallGraphStatusRequest true "请求参数"
// @Success 200 {object} response.Response{data=codeanalysis.CallGraphStatus}
// @Router /api/v1/analysis/callgraph/status [post]
func (h *CodeAnalysisHandler) CheckCallGraphStatus(c *gin.Context) {
	var req CheckCallGraphStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	result, err := h.callGraphService.CheckStatus(c.Request.Context(), &codeanalysis.CheckStatusRequest{
		ProjectPath: req.ProjectPath,
		Commit:      req.Commit,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	response.Success(c, result)
}

// GenerateCallGraphRequest 生成调用图请求
type GenerateCallGraphRequest struct {
	ProjectPath string `json:"project_path" binding:"required"`
	Commit      string `json:"commit"`
}

// GenerateCallGraphWithConfigRequest 生成调用图请求（包含配置）
type GenerateCallGraphWithConfigRequest struct {
	ProjectPath        string   `json:"project_path" binding:"required"`
	EntryPoints        []string `json:"entry_points" binding:"required"`
	Exclude            []string `json:"exclude"`
	Algorithm          string   `json:"algorithm"`
	Commit             string   `json:"commit"`
	IntegrationTestDir string   `json:"integration_test_dir"`
	IntegrationTestTag string   `json:"integration_test_tag"`
}

// GenerateCallGraph 生成调用图
// @Summary 生成调用图
// @Description 为项目生成调用图（同步，Deprecated: 请使用 /callgraph/generate-with-config）
// @Tags 代码分析
// @Accept json
// @Produce json
// @Param request body GenerateCallGraphRequest true "请求参数"
// @Success 200 {object} response.Response{data=codeanalysis.GenerateResponse}
// @Router /api/v1/analysis/callgraph/generate [post]
func (h *CodeAnalysisHandler) GenerateCallGraph(c *gin.Context) {
	var req GenerateCallGraphRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	result, err := h.callGraphService.Generate(c.Request.Context(), &codeanalysis.GenerateRequest{
		ProjectPath: req.ProjectPath,
		Commit:      req.Commit,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	response.Success(c, result)
}

// GenerateCallGraphAsync 生成调用图（异步）
// @Summary 生成调用图（异步）
// @Description 为项目生成调用图（异步执行，Deprecated: 请使用 /callgraph/generate-with-config）
// @Tags 代码分析
// @Accept json
// @Produce json
// @Param request body GenerateCallGraphRequest true "请求参数"
// @Success 200 {object} response.Response{data=codeanalysis.GenerateAsyncResponse}
// @Router /api/v1/analysis/callgraph/generate-async [post]
func (h *CodeAnalysisHandler) GenerateCallGraphAsync(c *gin.Context) {
	var req GenerateCallGraphRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	result, err := h.callGraphService.GenerateAsync(c.Request.Context(), &codeanalysis.GenerateRequest{
		ProjectPath: req.ProjectPath,
		Commit:      req.Commit,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	response.Success(c, result)
}

// GenerateCallGraphWithConfig 生成调用图（异步，包含配置）
// @Summary 生成调用图（异步，包含配置）
// @Description 为项目生成调用图（异步执行，包含入口函数和算法配置）
// @Tags 代码分析
// @Accept json
// @Produce json
// @Param request body GenerateCallGraphWithConfigRequest true "请求参数"
// @Success 200 {object} response.Response{data=codeanalysis.GenerateAsyncResponse}
// @Router /api/v1/analysis/callgraph/generate-with-config [post]
func (h *CodeAnalysisHandler) GenerateCallGraphWithConfig(c *gin.Context) {
	var req GenerateCallGraphWithConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	result, err := h.callGraphService.GenerateWithConfigAsync(c.Request.Context(), &codeanalysis.GenerateWithConfigRequest{
		ProjectPath:        req.ProjectPath,
		EntryPoints:        req.EntryPoints,
		Exclude:            req.Exclude,
		Algorithm:          parseAlgorithm(req.Algorithm),
		Commit:             req.Commit,
		IntegrationTestDir: req.IntegrationTestDir,
		IntegrationTestTag: req.IntegrationTestTag,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	response.Success(c, result)
}

// GetGenerationProgress 获取生成进度
// @Summary 获取生成进度
// @Description 获取异步生成任务的进度
// @Tags 代码分析
// @Produce json
// @Param task_id path string true "任务 ID"
// @Success 200 {object} response.Response{data=codeanalysis.GenerationTask}
// @Router /api/v1/analysis/callgraph/progress/{task_id} [get]
func (h *CodeAnalysisHandler) GetGenerationProgress(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		response.Error(c, http.StatusBadRequest, 400, "task_id is required")
		return
	}

	result, err := h.callGraphService.GetTaskProgress(taskID)
	if err != nil {
		response.Error(c, http.StatusNotFound, 404, err.Error())
		return
	}

	response.Success(c, result)
}

// AnalyzeDiffRequest 分析 diff 请求
type AnalyzeDiffRequest struct {
	ProjectPath string `json:"project_path" binding:"required"`
	CommitRange string `json:"commit_range"`
}

// AnalyzeDiff 分析 Git diff
// @Summary 分析 Git diff
// @Description 分析 Git diff，返回变更的函数列表
// @Tags 代码分析
// @Accept json
// @Produce json
// @Param request body AnalyzeDiffRequest true "请求参数"
// @Success 200 {object} response.Response{data=codeanalysis.DiffAnalysisResult}
// @Router /api/v1/analysis/diff [post]
func (h *CodeAnalysisHandler) AnalyzeDiff(c *gin.Context) {
	var req AnalyzeDiffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	result, err := h.impactService.AnalyzeDiff(c.Request.Context(), &codeanalysis.AnalyzeDiffRequest{
		ProjectPath: req.ProjectPath,
		CommitRange: req.CommitRange,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	response.Success(c, result)
}

// QueryImpactRequest 查询影响面请求
type QueryImpactRequest struct {
	ProjectPath string   `json:"project_path" binding:"required"`
	Functions   []string `json:"functions" binding:"required"`
	Depth       int      `json:"depth"`
	Commit      string   `json:"commit"`
}

// QueryImpact 查询影响面
// @Summary 查询影响面
// @Description 查询函数变更的影响面
// @Tags 代码分析
// @Accept json
// @Produce json
// @Param request body QueryImpactRequest true "请求参数"
// @Success 200 {object} response.Response{data=codeanalysis.ImpactAnalysisResult}
// @Router /api/v1/analysis/impact [post]
func (h *CodeAnalysisHandler) QueryImpact(c *gin.Context) {
	var req QueryImpactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	result, err := h.impactService.QueryImpact(c.Request.Context(), &codeanalysis.QueryImpactRequest{
		ProjectPath: req.ProjectPath,
		Functions:   req.Functions,
		Depth:       req.Depth,
		Commit:      req.Commit,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	response.Success(c, result)
}

// parseAlgorithm 解析算法类型
func parseAlgorithm(s string) domainCodeanalysis.AlgorithmType {
	switch s {
	case "static":
		return domainCodeanalysis.AlgorithmStatic
	case "cha":
		return domainCodeanalysis.AlgorithmCHA
	case "rta":
		return domainCodeanalysis.AlgorithmRTA
	case "vta":
		return domainCodeanalysis.AlgorithmVTA
	default:
		return domainCodeanalysis.AlgorithmRTA
	}
}
