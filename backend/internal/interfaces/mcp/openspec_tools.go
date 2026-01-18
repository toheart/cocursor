package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	infraStorage "github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// OpenSpecListInput 列出 OpenSpec 变更和规范
type OpenSpecListInput struct {
	ProjectPath string `json:"project_path" jsonschema:"项目路径，如 D:/code/cocursor"`
	Type        string `json:"type,omitempty" jsonschema:"类型：changes|specs|all（默认all）"`
}

// OpenSpecListOutput 输出
type OpenSpecListOutput struct {
	Changes []ChangeItem `json:"changes,omitempty"`
	Specs   []SpecItem   `json:"specs,omitempty"`
}

// ChangeItem 变更项
type ChangeItem struct {
	ID          string    `json:"id"`           // 变更ID（目录名）
	Title       string    `json:"title"`        // 从 proposal.md 第一行提取
	Status      string    `json:"status"`       // active|archived
	CreatedAt   time.Time `json:"created_at"`   // 目录创建时间
	UpdatedAt   time.Time `json:"updated_at"`   // 最后修改时间
	HasProposal bool      `json:"has_proposal"` // 是否有 proposal.md
	HasTasks    bool      `json:"has_tasks"`    // 是否有 tasks.md
	HasDesign   bool      `json:"has_design"`   // 是否有 design.md
	SpecDeltas  []string  `json:"spec_deltas"`  // 涉及的 spec 能力列表
}

// SpecItem 规范项
type SpecItem struct {
	Capability string    `json:"capability"` // 能力名称（目录名）
	HasSpec    bool      `json:"has_spec"`   // 是否有 spec.md
	HasDesign  bool      `json:"has_design"` // 是否有 design.md
	UpdatedAt  time.Time `json:"updated_at"` // 最后修改时间
}

// OpenSpecValidateInput 验证变更
type OpenSpecValidateInput struct {
	ProjectPath string `json:"project_path" jsonschema:"项目路径"`
	ChangeID    string `json:"change_id" jsonschema:"变更ID"`
	Strict      bool   `json:"strict,omitempty" jsonschema:"严格模式"`
}

// OpenSpecValidateOutput 验证结果
type OpenSpecValidateOutput struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	ChangeID string   `json:"change_id"`
}

// RecordOpenSpecWorkflowInput 记录工作流状态
type RecordOpenSpecWorkflowInput struct {
	ProjectPath string                 `json:"project_path" jsonschema:"项目路径"`
	ChangeID    string                 `json:"change_id" jsonschema:"变更ID"`
	Stage       string                 `json:"stage" jsonschema:"阶段：proposal|apply（只记录这两个阶段，init 不记录）"`
	Status      string                 `json:"status" jsonschema:"状态：in_progress|completed|paused"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" jsonschema:"元数据"`
}

// RecordOpenSpecWorkflowOutput 输出
type RecordOpenSpecWorkflowOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// GenerateOpenSpecWorkflowSummaryInput 生成工作总结
type GenerateOpenSpecWorkflowSummaryInput struct {
	ProjectPath string `json:"project_path" jsonschema:"项目路径"`
	ChangeID    string `json:"change_id" jsonschema:"变更ID"`
}

// GenerateOpenSpecWorkflowSummaryOutput 输出
type GenerateOpenSpecWorkflowSummaryOutput struct {
	ChangeID       string   `json:"change_id"`
	Stage          string   `json:"stage"`
	Summary        string   `json:"summary"`
	TasksCompleted int      `json:"tasks_completed"`
	TasksTotal     int      `json:"tasks_total"`
	FilesChanged   []string `json:"files_changed"`
	TimeSpent      string   `json:"time_spent"`
}

// GetOpenSpecWorkflowStatusInput 获取工作流状态
type GetOpenSpecWorkflowStatusInput struct {
	ProjectPath string `json:"project_path,omitempty" jsonschema:"项目路径（可选）"`
	Status      string `json:"status,omitempty" jsonschema:"状态筛选（可选）：in_progress|completed|paused"`
}

// GetOpenSpecWorkflowStatusOutput 输出
type GetOpenSpecWorkflowStatusOutput struct {
	Workflows []WorkflowStatusItem `json:"workflows"`
}

// WorkflowStatusItem 工作流状态项
type WorkflowStatusItem struct {
	ChangeID  string                        `json:"change_id"`
	Stage     string                        `json:"stage"`
	Status    string                        `json:"status"`
	Progress  *WorkflowProgress             `json:"progress,omitempty"`
	UpdatedAt time.Time                     `json:"updated_at"`
	Summary   *infraStorage.WorkflowSummary `json:"summary,omitempty"`
}

// WorkflowProgress 工作流进度
type WorkflowProgress struct {
	TasksCompleted int `json:"tasks_completed"`
	TasksTotal     int `json:"tasks_total"`
}

// openspecListTool 列出 OpenSpec 变更和规范
func openspecListTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input OpenSpecListInput,
) (*mcp.CallToolResult, OpenSpecListOutput, error) {
	// 验证项目路径
	if input.ProjectPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, OpenSpecListOutput{}, fmt.Errorf("project_path is required or cannot get current directory: %w", err)
		}
		input.ProjectPath = cwd
	}

	openspecDir := filepath.Join(input.ProjectPath, "openspec")

	var changes []ChangeItem
	var specs []SpecItem

	// 扫描 changes 目录
	if input.Type == "changes" || input.Type == "all" || input.Type == "" {
		changesDir := filepath.Join(openspecDir, "changes")
		if entries, err := os.ReadDir(changesDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() || entry.Name() == "archive" {
					continue
				}

				changeID := entry.Name()
				changePath := filepath.Join(changesDir, changeID)

				// 读取 proposal.md 第一行作为标题
				title := extractTitleFromProposal(changePath)

				// 检查文件存在性
				hasProposal := fileExists(filepath.Join(changePath, "proposal.md"))
				hasTasks := fileExists(filepath.Join(changePath, "tasks.md"))
				hasDesign := fileExists(filepath.Join(changePath, "design.md"))

				// 扫描 spec deltas
				specDeltas := scanSpecDeltas(changePath)

				// 获取时间戳
				info, _ := entry.Info()
				createdAt := info.ModTime()
				updatedAt := getLatestModTime(changePath)

				changes = append(changes, ChangeItem{
					ID:          changeID,
					Title:       title,
					Status:      "active",
					CreatedAt:   createdAt,
					UpdatedAt:   updatedAt,
					HasProposal: hasProposal,
					HasTasks:    hasTasks,
					HasDesign:   hasDesign,
					SpecDeltas:  specDeltas,
				})
			}
		}
	}

	// 扫描 specs 目录
	if input.Type == "specs" || input.Type == "all" || input.Type == "" {
		specsDir := filepath.Join(openspecDir, "specs")
		if entries, err := os.ReadDir(specsDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				capability := entry.Name()
				capabilityPath := filepath.Join(specsDir, capability)

				hasSpec := fileExists(filepath.Join(capabilityPath, "spec.md"))
				hasDesign := fileExists(filepath.Join(capabilityPath, "design.md"))

				info, _ := entry.Info()
				updatedAt := getLatestModTime(capabilityPath)
				if updatedAt.IsZero() {
					updatedAt = info.ModTime()
				}

				specs = append(specs, SpecItem{
					Capability: capability,
					HasSpec:    hasSpec,
					HasDesign:  hasDesign,
					UpdatedAt:  updatedAt,
				})
			}
		}
	}

	return nil, OpenSpecListOutput{
		Changes: changes,
		Specs:   specs,
	}, nil
}

// openspecValidateTool 验证 OpenSpec 变更格式
func openspecValidateTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input OpenSpecValidateInput,
) (*mcp.CallToolResult, OpenSpecValidateOutput, error) {
	if input.ProjectPath == "" {
		cwd, _ := os.Getwd()
		input.ProjectPath = cwd
	}

	changePath := filepath.Join(input.ProjectPath, "openspec", "changes", input.ChangeID)

	var errors []string
	var warnings []string

	// 检查必需文件
	if !fileExists(filepath.Join(changePath, "proposal.md")) {
		errors = append(errors, "proposal.md is required but not found")
	}
	if !fileExists(filepath.Join(changePath, "tasks.md")) {
		errors = append(errors, "tasks.md is required but not found")
	}

	// 验证 spec delta 文件
	specsDir := filepath.Join(changePath, "specs")
	if entries, err := os.ReadDir(specsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			specPath := filepath.Join(specsDir, entry.Name(), "spec.md")
			if !fileExists(specPath) {
				warnings = append(warnings, fmt.Sprintf("spec delta %s/spec.md not found", entry.Name()))
				continue
			}

			// 验证格式
			content, err := os.ReadFile(specPath)
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to read %s/spec.md: %v", entry.Name(), err))
				continue
			}

			// 检查是否有 ADDED|MODIFIED|REMOVED Requirements
			contentStr := string(content)
			hasOperation := strings.Contains(contentStr, "## ADDED Requirements") ||
				strings.Contains(contentStr, "## MODIFIED Requirements") ||
				strings.Contains(contentStr, "## REMOVED Requirements")

			if !hasOperation {
				errors = append(errors, fmt.Sprintf("%s/spec.md must contain ## ADDED|MODIFIED|REMOVED Requirements", entry.Name()))
			}

			// 检查每个 requirement 是否有 scenario
			if !strings.Contains(contentStr, "#### Scenario:") {
				errors = append(errors, fmt.Sprintf("%s/spec.md must have at least one #### Scenario: per requirement", entry.Name()))
			}
		}
	} else if input.Strict {
		warnings = append(warnings, "no spec deltas found")
	}

	// 验证 proposal.md 格式
	proposalPath := filepath.Join(changePath, "proposal.md")
	if content, err := os.ReadFile(proposalPath); err == nil {
		contentStr := string(content)
		if !strings.Contains(contentStr, "## Why") {
			warnings = append(warnings, "proposal.md should contain ## Why section")
		}
		if !strings.Contains(contentStr, "## What Changes") {
			warnings = append(warnings, "proposal.md should contain ## What Changes section")
		}
	}

	valid := len(errors) == 0

	return nil, OpenSpecValidateOutput{
		Valid:    valid,
		Errors:   errors,
		Warnings: warnings,
		ChangeID: input.ChangeID,
	}, nil
}

// recordOpenSpecWorkflowTool 记录工作流状态
func (s *MCPServer) recordOpenSpecWorkflowTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input RecordOpenSpecWorkflowInput,
) (*mcp.CallToolResult, RecordOpenSpecWorkflowOutput, error) {
	// init 阶段不需要记录，这是初始化过程
	if input.Stage == "init" {
		return nil, RecordOpenSpecWorkflowOutput{
			Success: true,
			Message: "Init stage skipped (not recorded)",
		}, nil
	}

	// 只记录 proposal 和 apply 阶段
	if input.Stage != "proposal" && input.Stage != "apply" {
		return nil, RecordOpenSpecWorkflowOutput{}, fmt.Errorf("only proposal and apply stages are recorded, got: %s", input.Stage)
	}

	// 获取工作区 ID
	pathResolver := infraCursor.NewPathResolver()
	workspaceID, err := pathResolver.GetWorkspaceIDByPath(input.ProjectPath)
	if err != nil {
		return nil, RecordOpenSpecWorkflowOutput{}, fmt.Errorf("failed to get workspace ID: %w", err)
	}

	// 获取工作流仓储
	workflowRepo, err := infraStorage.NewOpenSpecWorkflowRepository()
	if err != nil {
		return nil, RecordOpenSpecWorkflowOutput{}, fmt.Errorf("failed to create workflow repository: %w", err)
	}

	// 检查是否已有工作流记录
	existing, _ := workflowRepo.FindByWorkspaceAndChange(workspaceID, input.ChangeID)

	// 构建工作流数据
	workflow := &infraStorage.OpenSpecWorkflow{
		WorkspaceID: workspaceID,
		ProjectPath: input.ProjectPath,
		ChangeID:    input.ChangeID,
		Stage:       input.Stage,
		Status:      input.Status,
		Metadata:    input.Metadata,
	}

	// 确保 metadata 存在
	if workflow.Metadata == nil {
		workflow.Metadata = make(map[string]interface{})
	}

	// 根据阶段添加工作流过程说明
	if input.Stage == "proposal" {
		if input.Status == "in_progress" {
			workflow.Metadata["workflow_process"] = "正在创建提案：编写 proposal.md、tasks.md 和规范变更文件"
		} else if input.Status == "completed" {
			workflow.Metadata["workflow_process"] = "提案已完成：提案已创建并通过验证，等待审批"
		}
	} else if input.Stage == "apply" {
		if input.Status == "in_progress" {
			workflow.Metadata["workflow_process"] = "正在实施变更：按照 tasks.md 逐步完成编码任务"
		} else if input.Status == "completed" {
			workflow.Metadata["workflow_process"] = "实施已完成：所有任务已完成，工作总结已生成"
		}
	}

	// 如果从 proposal 转换到 apply，记录工作流转换过程
	if existing != nil && existing.Stage == "proposal" && input.Stage == "apply" {
		// 记录阶段转换历史
		if history, ok := workflow.Metadata["stage_history"].([]interface{}); ok {
			workflow.Metadata["stage_history"] = append(history, map[string]interface{}{
				"from":      existing.Stage,
				"to":        input.Stage,
				"timestamp": time.Now().UnixMilli(),
				"note":      "从提案阶段转换到实施阶段",
			})
		} else {
			workflow.Metadata["stage_history"] = []interface{}{
				map[string]interface{}{
					"from":      existing.Stage,
					"to":        input.Stage,
					"timestamp": time.Now().UnixMilli(),
					"note":      "从提案阶段转换到实施阶段",
				},
			}
		}
		// 保留 proposal 阶段的信息
		if existing.Metadata != nil {
			if proposalInfo, ok := existing.Metadata["proposal_info"]; ok {
				workflow.Metadata["proposal_info"] = proposalInfo
			}
		}
	}

	// 如果 stage 是 "apply"，检测 tasks.md 完成状态
	if input.Stage == "apply" {
		changePath := filepath.Join(input.ProjectPath, "openspec", "changes", input.ChangeID)
		tasksPath := filepath.Join(changePath, "tasks.md")
		if content, err := os.ReadFile(tasksPath); err == nil {
			if allTasksCompleted(content) {
				// 自动触发工作总结生成
				summary, err := s.generateWorkflowSummary(input.ProjectPath, input.ChangeID, workflowRepo)
				if err == nil {
					workflow.Summary = summary
					workflow.Status = "completed"
				}
			}
		}
	}

	// 保存工作流状态
	if err := workflowRepo.Save(workflow); err != nil {
		return nil, RecordOpenSpecWorkflowOutput{}, fmt.Errorf("failed to save workflow: %w", err)
	}

	return nil, RecordOpenSpecWorkflowOutput{
		Success: true,
		Message: "Workflow state recorded",
	}, nil
}

// generateOpenSpecWorkflowSummaryTool 生成工作总结
func (s *MCPServer) generateOpenSpecWorkflowSummaryTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GenerateOpenSpecWorkflowSummaryInput,
) (*mcp.CallToolResult, GenerateOpenSpecWorkflowSummaryOutput, error) {
	// 获取工作流仓储
	workflowRepo, err := infraStorage.NewOpenSpecWorkflowRepository()
	if err != nil {
		return nil, GenerateOpenSpecWorkflowSummaryOutput{}, fmt.Errorf("failed to create workflow repository: %w", err)
	}

	// 生成总结
	summary, err := s.generateWorkflowSummary(input.ProjectPath, input.ChangeID, workflowRepo)
	if err != nil {
		return nil, GenerateOpenSpecWorkflowSummaryOutput{}, fmt.Errorf("failed to generate summary: %w", err)
	}

	// 更新工作流状态
	pathResolver := infraCursor.NewPathResolver()
	workspaceID, _ := pathResolver.GetWorkspaceIDByPath(input.ProjectPath)
	workflow, _ := workflowRepo.FindByWorkspaceAndChange(workspaceID, input.ChangeID)
	if workflow != nil {
		workflow.Summary = summary
		workflow.Status = "completed"
		workflowRepo.Save(workflow)
	}

	return nil, GenerateOpenSpecWorkflowSummaryOutput{
		ChangeID:       input.ChangeID,
		Stage:          "apply",
		Summary:        summary.Summary,
		TasksCompleted: summary.TasksCompleted,
		TasksTotal:     summary.TasksTotal,
		FilesChanged:   summary.FilesChanged,
		TimeSpent:      summary.TimeSpent,
	}, nil
}

// getOpenSpecWorkflowStatusTool 获取工作流状态
func (s *MCPServer) getOpenSpecWorkflowStatusTool(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetOpenSpecWorkflowStatusInput,
) (*mcp.CallToolResult, GetOpenSpecWorkflowStatusOutput, error) {
	// 获取工作流仓储
	workflowRepo, err := infraStorage.NewOpenSpecWorkflowRepository()
	if err != nil {
		return nil, GetOpenSpecWorkflowStatusOutput{}, fmt.Errorf("failed to create workflow repository: %w", err)
	}

	var workflows []*infraStorage.OpenSpecWorkflow

	if input.Status != "" {
		// 按状态查询
		workflows, err = workflowRepo.FindByStatus(input.Status)
	} else if input.ProjectPath != "" {
		// 按工作区查询
		pathResolver := infraCursor.NewPathResolver()
		workspaceID, err := pathResolver.GetWorkspaceIDByPath(input.ProjectPath)
		if err != nil {
			return nil, GetOpenSpecWorkflowStatusOutput{}, fmt.Errorf("failed to get workspace ID: %w", err)
		}
		workflows, err = workflowRepo.FindByWorkspace(workspaceID)
	} else {
		// 查询所有 in_progress 状态
		workflows, err = workflowRepo.FindByStatus("in_progress")
	}

	if err != nil {
		return nil, GetOpenSpecWorkflowStatusOutput{}, fmt.Errorf("failed to query workflows: %w", err)
	}

	// 转换为输出格式
	var items []WorkflowStatusItem
	for _, wf := range workflows {
		item := WorkflowStatusItem{
			ChangeID:  wf.ChangeID,
			Stage:     wf.Stage,
			Status:    wf.Status,
			UpdatedAt: wf.UpdatedAt,
			Summary:   wf.Summary,
		}

		// 从 metadata 中提取进度
		if wf.Metadata != nil {
			if tasksCompleted, ok := wf.Metadata["tasks_completed"].(float64); ok {
				if tasksTotal, ok := wf.Metadata["tasks_total"].(float64); ok {
					item.Progress = &WorkflowProgress{
						TasksCompleted: int(tasksCompleted),
						TasksTotal:     int(tasksTotal),
					}
				}
			}
		}

		items = append(items, item)
	}

	return nil, GetOpenSpecWorkflowStatusOutput{
		Workflows: items,
	}, nil
}

// 辅助函数：从 proposal.md 提取标题
func extractTitleFromProposal(changePath string) string {
	proposalPath := filepath.Join(changePath, "proposal.md")
	content, err := os.ReadFile(proposalPath)
	if err != nil {
		return ""
	}

	// 读取第一行，提取标题（去除 # 前缀）
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
		if strings.HasPrefix(line, "#") {
			return strings.TrimPrefix(line, "#")
		}
	}

	return ""
}

// 辅助函数：检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// 辅助函数：扫描 spec deltas
func scanSpecDeltas(changePath string) []string {
	specsDir := filepath.Join(changePath, "specs")
	var deltas []string

	if entries, err := os.ReadDir(specsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				deltas = append(deltas, entry.Name())
			}
		}
	}

	return deltas
}

// 辅助函数：获取目录下最新修改时间
func getLatestModTime(dirPath string) time.Time {
	var latestTime time.Time

	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
		}
		return nil
	})

	return latestTime
}

// 辅助函数：检查所有任务是否完成
func allTasksCompleted(tasksContent []byte) bool {
	content := string(tasksContent)
	lines := strings.Split(content, "\n")

	taskCount := 0
	completedCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- [") {
			taskCount++
			if strings.Contains(line, "- [x]") || strings.Contains(line, "- [X]") {
				completedCount++
			}
		}
	}

	// 如果所有任务都完成
	return taskCount > 0 && taskCount == completedCount
}

// 辅助函数：生成工作总结
func (s *MCPServer) generateWorkflowSummary(projectPath, changeID string, workflowRepo infraStorage.OpenSpecWorkflowRepository) (*infraStorage.WorkflowSummary, error) {
	changePath := filepath.Join(projectPath, "openspec", "changes", changeID)
	tasksPath := filepath.Join(changePath, "tasks.md")

	// 读取 tasks.md
	content, err := os.ReadFile(tasksPath)
	if err != nil {
		return nil, err
	}

	// 统计任务
	taskCount := 0
	completedCount := 0
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- [") {
			taskCount++
			if strings.Contains(line, "- [x]") || strings.Contains(line, "- [X]") {
				completedCount++
			}
		}
	}

	// 获取工作流记录计算时间
	pathResolver := infraCursor.NewPathResolver()
	workspaceID, _ := pathResolver.GetWorkspaceIDByPath(projectPath)
	existing, _ := workflowRepo.FindByWorkspaceAndChange(workspaceID, changeID)

	var timeSpent string
	if existing != nil {
		duration := time.Since(existing.StartedAt)
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		if hours > 0 {
			timeSpent = fmt.Sprintf("%dh %dm", hours, minutes)
		} else {
			timeSpent = fmt.Sprintf("%dm", minutes)
		}
	}

	// 扫描变更的文件（简化：通过 git diff 或文件修改时间）
	filesChanged := scanChangedFiles(changePath)

	// 生成总结文本
	summary := fmt.Sprintf("完成变更 %s：共 %d 个任务，耗时 %s，修改了 %d 个文件。", changeID, completedCount, timeSpent, len(filesChanged))

	return &infraStorage.WorkflowSummary{
		TasksCompleted: completedCount,
		TasksTotal:     taskCount,
		FilesChanged:   filesChanged,
		TimeSpent:      timeSpent,
		Summary:        summary,
	}, nil
}

// 辅助函数：扫描变更的文件
func scanChangedFiles(changePath string) []string {
	// 简化实现：扫描 change 目录下的所有文件（排除 .md 文件）
	var files []string
	filepath.Walk(changePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".md") {
			return nil
		}
		relPath, _ := filepath.Rel(changePath, path)
		files = append(files, relPath)
		return nil
	})
	return files
}
