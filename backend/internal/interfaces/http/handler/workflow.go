package handler

import (
	"net/http"

	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	infraStorage "github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// WorkflowHandler 工作流处理器
type WorkflowHandler struct{}

// NewWorkflowHandler 创建 WorkflowHandler
func NewWorkflowHandler() *WorkflowHandler {
	return &WorkflowHandler{}
}

// WorkflowItem 工作流列表项
type WorkflowItem struct {
	ID          int64                  `json:"id"`
	WorkspaceID string                 `json:"workspace_id"`
	ProjectPath string                 `json:"project_path"`
	ChangeID    string                 `json:"change_id"`
	Stage       string                 `json:"stage"`      // init|proposal|apply|archive
	Status      string                 `json:"status"`     // in_progress|completed|paused
	StartedAt   int64                  `json:"started_at"` // Unix 毫秒时间戳
	UpdatedAt   int64                  `json:"updated_at"` // Unix 毫秒时间戳
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Summary     *WorkflowSummary       `json:"summary,omitempty"`
}

// WorkflowSummary 工作总结
type WorkflowSummary struct {
	TasksCompleted int      `json:"tasks_completed"`
	TasksTotal     int      `json:"tasks_total"`
	FilesChanged   []string `json:"files_changed"`
	TimeSpent      string   `json:"time_spent"`
	Summary        string   `json:"summary"`
}

// ListWorkflows 获取工作流列表
// @Summary 获取工作流列表
// @Tags 工作流
// @Accept json
// @Produce json
// @Param project_path query string false "项目路径，如 D:/code/cocursor"
// @Param status query string false "状态筛选：in_progress|completed|paused"
// @Success 200 {object} response.Response{data=[]WorkflowItem}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /workflows [get]
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	// 获取项目路径参数
	projectPath := c.Query("project_path")
	status := c.Query("status")

	// 获取工作流仓储
	workflowRepo, err := infraStorage.NewOpenSpecWorkflowRepository()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 700001, "创建工作流仓储失败: "+err.Error())
		return
	}

	var workflows []*infraStorage.OpenSpecWorkflow

	// 如果提供了项目路径，根据工作区查询
	if projectPath != "" {
		pathResolver := infraCursor.NewPathResolver()
		workspaceID, err := pathResolver.GetWorkspaceIDByPath(projectPath)
		if err != nil {
			// 如果找不到工作区，尝试直接通过 project_path 查询（备用方案）
			workflows, err = workflowRepo.FindByProjectPath(projectPath)
			if err != nil {
				response.Error(c, http.StatusInternalServerError, 700001, "查询工作流失败: "+err.Error())
				return
			}
		} else {
			// 优先通过 workspace_id 查询
			workflows, err = workflowRepo.FindByWorkspace(workspaceID)
			if err != nil {
				response.Error(c, http.StatusInternalServerError, 700001, "查询工作流失败: "+err.Error())
				return
			}
		}
	} else if status != "" {
		// 如果提供了状态，根据状态查询
		workflows, err = workflowRepo.FindByStatus(status)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 700001, "查询工作流失败: "+err.Error())
			return
		}
	} else {
		// 如果没有提供参数，返回空列表
		response.Success(c, []WorkflowItem{})
		return
	}

	// 转换为响应格式
	items := make([]WorkflowItem, 0, len(workflows))
	for _, wf := range workflows {
		// 如果提供了状态筛选，进行过滤
		if status != "" && wf.Status != status {
			continue
		}

		item := WorkflowItem{
			ID:          wf.ID,
			WorkspaceID: wf.WorkspaceID,
			ProjectPath: wf.ProjectPath,
			ChangeID:    wf.ChangeID,
			Stage:       wf.Stage,
			Status:      wf.Status,
			StartedAt:   wf.StartedAt.UnixMilli(),
			UpdatedAt:   wf.UpdatedAt.UnixMilli(),
			Metadata:    wf.Metadata,
		}

		if wf.Summary != nil {
			item.Summary = &WorkflowSummary{
				TasksCompleted: wf.Summary.TasksCompleted,
				TasksTotal:     wf.Summary.TasksTotal,
				FilesChanged:   wf.Summary.FilesChanged,
				TimeSpent:      wf.Summary.TimeSpent,
				Summary:        wf.Summary.Summary,
			}
		}

		items = append(items, item)
	}

	response.Success(c, items)
}

// GetWorkflowDetail 获取工作流详情
// @Summary 获取工作流详情
// @Tags 工作流
// @Accept json
// @Produce json
// @Param project_path query string true "项目路径，如 D:/code/cocursor"
// @Param change_id path string true "变更 ID"
// @Success 200 {object} response.Response{data=WorkflowItem}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /workflows/{change_id} [get]
func (h *WorkflowHandler) GetWorkflowDetail(c *gin.Context) {
	changeID := c.Param("change_id")
	if changeID == "" {
		response.Error(c, http.StatusBadRequest, 100001, "change_id 参数不能为空")
		return
	}

	projectPath := c.Query("project_path")
	if projectPath == "" {
		response.Error(c, http.StatusBadRequest, 100001, "project_path 参数不能为空")
		return
	}

	// 获取工作流仓储
	workflowRepo, err := infraStorage.NewOpenSpecWorkflowRepository()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 700001, "创建工作流仓储失败: "+err.Error())
		return
	}

	// 获取工作区 ID（可选）
	pathResolver := infraCursor.NewPathResolver()
	workspaceID, err := pathResolver.GetWorkspaceIDByPath(projectPath)

	var workflow *infraStorage.OpenSpecWorkflow
	if err == nil && workspaceID != "" {
		// 优先通过 workspace_id 查询
		workflow, err = workflowRepo.FindByWorkspaceAndChange(workspaceID, changeID)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 700001, "查询工作流失败: "+err.Error())
			return
		}
	}

	// 如果通过 workspace_id 没找到，尝试通过 project_path 查询所有工作流，然后筛选
	if workflow == nil {
		workflows, err := workflowRepo.FindByProjectPath(projectPath)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 700001, "查询工作流失败: "+err.Error())
			return
		}

		// 从结果中查找匹配的 change_id
		for _, wf := range workflows {
			if wf.ChangeID == changeID {
				workflow = wf
				break
			}
		}
	}

	if workflow == nil {
		response.Error(c, http.StatusNotFound, 100004, "工作流不存在")
		return
	}

	// 转换为响应格式
	item := WorkflowItem{
		ID:          workflow.ID,
		WorkspaceID: workflow.WorkspaceID,
		ProjectPath: workflow.ProjectPath,
		ChangeID:    workflow.ChangeID,
		Stage:       workflow.Stage,
		Status:      workflow.Status,
		StartedAt:   workflow.StartedAt.UnixMilli(),
		UpdatedAt:   workflow.UpdatedAt.UnixMilli(),
		Metadata:    workflow.Metadata,
	}

	if workflow.Summary != nil {
		item.Summary = &WorkflowSummary{
			TasksCompleted: workflow.Summary.TasksCompleted,
			TasksTotal:     workflow.Summary.TasksTotal,
			FilesChanged:   workflow.Summary.FilesChanged,
			TimeSpent:      workflow.Summary.TimeSpent,
			Summary:        workflow.Summary.Summary,
		}
	}

	response.Success(c, item)
}

// GetWorkflowStatus 获取工作流状态（用于快速查询）
// @Summary 获取工作流状态
// @Tags 工作流
// @Accept json
// @Produce json
// @Param project_path query string false "项目路径"
// @Param status query string false "状态筛选：in_progress|completed|paused"
// @Success 200 {object} response.Response{data=[]WorkflowItem}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /workflows/status [get]
func (h *WorkflowHandler) GetWorkflowStatus(c *gin.Context) {
	// 这个接口与 ListWorkflows 功能相同，但用于快速查询状态
	// 直接调用 ListWorkflows 的逻辑
	h.ListWorkflows(c)
}
