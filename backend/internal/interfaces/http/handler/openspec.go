package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// OpenSpecHandler OpenSpec 处理器
type OpenSpecHandler struct{}

// NewOpenSpecHandler 创建 OpenSpec 处理器
func NewOpenSpecHandler() *OpenSpecHandler {
	return &OpenSpecHandler{}
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

// OpenSpecListResponse 列表响应
type OpenSpecListResponse struct {
	Changes []ChangeItem `json:"changes,omitempty"` // 变更列表
	Specs   []SpecItem   `json:"specs,omitempty"`   // 规范列表
}

// ValidateRequest 验证请求
type ValidateRequest struct {
	ProjectPath string `json:"project_path"` // 项目路径
	ChangeID    string `json:"change_id"`    // 变更 ID
	Strict      bool   `json:"strict"`       // 严格模式
}

// ValidateResponse 验证响应
type ValidateResponse struct {
	Valid    bool     `json:"valid"`              // 是否有效
	Errors   []string `json:"errors,omitempty"`   // 错误列表
	Warnings []string `json:"warnings,omitempty"` // 警告列表
	ChangeID string   `json:"change_id"`          // 变更 ID
}

// List 列出 OpenSpec 变更和规范
// @Summary 列出 OpenSpec 变更和规范
// @Tags OpenSpec
// @Accept json
// @Produce json
// @Param project_path query string true "项目路径"
// @Param type query string false "类型：changes|specs|all"
// @Success 200 {object} response.Response{data=OpenSpecListResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /openspec/list [get]
func (h *OpenSpecHandler) List(c *gin.Context) {
	projectPath := c.Query("project_path")
	listType := c.DefaultQuery("type", "all")

	// 验证项目路径
	if projectPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			response.Error(c, http.StatusBadRequest, 900001, "project_path is required or cannot get current directory")
			return
		}
		projectPath = cwd
	}

	openspecDir := filepath.Join(projectPath, "openspec")

	var changes []ChangeItem
	var specs []SpecItem

	// 扫描 changes 目录
	if listType == "changes" || listType == "all" {
		changesDir := filepath.Join(openspecDir, "changes")
		if entries, err := os.ReadDir(changesDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() || entry.Name() == "archive" {
					continue
				}

				changeID := entry.Name()
				changePath := filepath.Join(changesDir, changeID)

				title := extractTitleFromProposal(changePath)
				hasProposal := fileExists(filepath.Join(changePath, "proposal.md"))
				hasTasks := fileExists(filepath.Join(changePath, "tasks.md"))
				hasDesign := fileExists(filepath.Join(changePath, "design.md"))
				specDeltas := scanSpecDeltas(changePath)

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
	if listType == "specs" || listType == "all" {
		specs = scanSpecsDirectory(filepath.Join(openspecDir, "specs"))
	}

	response.Success(c, OpenSpecListResponse{
		Changes: changes,
		Specs:   specs,
	})
}

// Validate 验证 OpenSpec 变更格式
// @Summary 验证 OpenSpec 变更格式
// @Tags OpenSpec
// @Accept json
// @Produce json
// @Param body body ValidateRequest true "验证请求"
// @Success 200 {object} response.Response{data=ValidateResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /openspec/validate [post]
func (h *OpenSpecHandler) Validate(c *gin.Context) {
	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 900001, "request parameter error: "+err.Error())
		return
	}

	if req.ProjectPath == "" {
		cwd, _ := os.Getwd()
		req.ProjectPath = cwd
	}

	if req.ChangeID == "" {
		response.Error(c, http.StatusBadRequest, 900002, "change_id is required")
		return
	}

	changePath := filepath.Join(req.ProjectPath, "openspec", "changes", req.ChangeID)

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
	specErrors, specWarnings := validateSpecDeltas(filepath.Join(changePath, "specs"), req.Strict)
	errors = append(errors, specErrors...)
	warnings = append(warnings, specWarnings...)

	// 验证 proposal.md 格式
	proposalPath := filepath.Join(changePath, "proposal.md")
	if content, err := os.ReadFile(proposalPath); err == nil {
		contentStr := string(content)
		if !strings.Contains(contentStr, "## Why") && !strings.Contains(contentStr, "## 问题背景") {
			warnings = append(warnings, "proposal.md should contain ## Why or ## 问题背景 section")
		}
		if !strings.Contains(contentStr, "## What Changes") && !strings.Contains(contentStr, "## 解决方案") && !strings.Contains(contentStr, "## 变更范围") {
			warnings = append(warnings, "proposal.md should contain ## What Changes or ## 解决方案 section")
		}
	}

	valid := len(errors) == 0

	response.Success(c, ValidateResponse{
		Valid:    valid,
		Errors:   errors,
		Warnings: warnings,
		ChangeID: req.ChangeID,
	})
}

// 辅助函数：从 proposal.md 提取标题
func extractTitleFromProposal(changePath string) string {
	proposalPath := filepath.Join(changePath, "proposal.md")
	content, err := os.ReadFile(proposalPath)
	if err != nil {
		return ""
	}

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

// scanSpecsDirectory 扫描 specs 目录并返回规范列表
func scanSpecsDirectory(specsDir string) []SpecItem {
	var specs []SpecItem

	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return specs
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if spec := buildSpecItem(specsDir, entry); spec != nil {
			specs = append(specs, *spec)
		}
	}
	return specs
}

// buildSpecItem 构建单个规范项
func buildSpecItem(specsDir string, entry os.DirEntry) *SpecItem {
	capability := entry.Name()
	capabilityPath := filepath.Join(specsDir, capability)

	info, _ := entry.Info()
	updatedAt := getLatestModTime(capabilityPath)
	if updatedAt.IsZero() && info != nil {
		updatedAt = info.ModTime()
	}

	return &SpecItem{
		Capability: capability,
		HasSpec:    fileExists(filepath.Join(capabilityPath, "spec.md")),
		HasDesign:  fileExists(filepath.Join(capabilityPath, "design.md")),
		UpdatedAt:  updatedAt,
	}
}

// validateSpecDeltas 验证 spec delta 文件
func validateSpecDeltas(specsDir string, strict bool) (errors, warnings []string) {
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		if strict {
			warnings = append(warnings, "no spec deltas found")
		}
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		errs, warns := validateSingleSpecDelta(specsDir, entry.Name())
		errors = append(errors, errs...)
		warnings = append(warnings, warns...)
	}
	return
}

// validateSingleSpecDelta 验证单个 spec delta
func validateSingleSpecDelta(specsDir, name string) (errors, warnings []string) {
	specPath := filepath.Join(specsDir, name, "spec.md")
	if !fileExists(specPath) {
		warnings = append(warnings, fmt.Sprintf("spec delta %s/spec.md not found", name))
		return
	}

	content, err := os.ReadFile(specPath)
	if err != nil {
		errors = append(errors, fmt.Sprintf("failed to read %s/spec.md: %v", name, err))
		return
	}

	contentStr := string(content)
	hasOperation := strings.Contains(contentStr, "## ADDED Requirements") ||
		strings.Contains(contentStr, "## MODIFIED Requirements") ||
		strings.Contains(contentStr, "## REMOVED Requirements")

	if !hasOperation {
		errors = append(errors, fmt.Sprintf("%s/spec.md must contain ## ADDED|MODIFIED|REMOVED Requirements", name))
	}

	if !strings.Contains(contentStr, "#### Scenario:") {
		errors = append(errors, fmt.Sprintf("%s/spec.md must have at least one #### Scenario: per requirement", name))
	}
	return
}
