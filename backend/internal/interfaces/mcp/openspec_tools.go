package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// OpenSpecListInput 列出 OpenSpec 变更和规范
type OpenSpecListInput struct {
	ProjectPath string `json:"project_path" jsonschema:"Project path, e.g., D:/code/cocursor"`
	Type        string `json:"type,omitempty" jsonschema:"Type: changes|specs|all (default: all)"`
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
	ProjectPath string `json:"project_path" jsonschema:"Project path"`
	ChangeID    string `json:"change_id" jsonschema:"Change ID"`
	Strict      bool   `json:"strict,omitempty" jsonschema:"Strict mode"`
}

// OpenSpecValidateOutput 验证结果
type OpenSpecValidateOutput struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	ChangeID string   `json:"change_id"`
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
		specs = scanSpecsDir(filepath.Join(openspecDir, "specs"))
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
	specErrors, specWarnings := validateSpecDeltaFiles(filepath.Join(changePath, "specs"), input.Strict)
	errors = append(errors, specErrors...)
	warnings = append(warnings, specWarnings...)

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

// scanSpecsDir 扫描 specs 目录并返回规范列表
func scanSpecsDir(specsDir string) []SpecItem {
	var specs []SpecItem

	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return specs
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		capability := entry.Name()
		capabilityPath := filepath.Join(specsDir, capability)

		info, _ := entry.Info()
		updatedAt := getLatestModTime(capabilityPath)
		if updatedAt.IsZero() && info != nil {
			updatedAt = info.ModTime()
		}

		specs = append(specs, SpecItem{
			Capability: capability,
			HasSpec:    fileExists(filepath.Join(capabilityPath, "spec.md")),
			HasDesign:  fileExists(filepath.Join(capabilityPath, "design.md")),
			UpdatedAt:  updatedAt,
		})
	}
	return specs
}

// validateSpecDeltaFiles 验证 spec delta 文件
func validateSpecDeltaFiles(specsDir string, strict bool) (errors, warnings []string) {
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
		errs, warns := validateSpecDelta(specsDir, entry.Name())
		errors = append(errors, errs...)
		warnings = append(warnings, warns...)
	}
	return
}

// validateSpecDelta 验证单个 spec delta
func validateSpecDelta(specsDir, name string) (errors, warnings []string) {
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
