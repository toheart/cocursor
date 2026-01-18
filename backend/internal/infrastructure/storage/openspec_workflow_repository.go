package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// OpenSpecWorkflow 工作流模型
type OpenSpecWorkflow struct {
	ID          int64                  `json:"id"`
	WorkspaceID string                 `json:"workspace_id"`
	ProjectPath string                 `json:"project_path"`
	ChangeID    string                 `json:"change_id"`
	Stage       string                 `json:"stage"`  // init|proposal|apply|archive
	Status      string                 `json:"status"` // in_progress|completed|paused
	StartedAt   time.Time              `json:"started_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
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

// OpenSpecWorkflowRepository 工作流仓储接口
type OpenSpecWorkflowRepository interface {
	Save(workflow *OpenSpecWorkflow) error
	FindByWorkspaceAndChange(workspaceID, changeID string) (*OpenSpecWorkflow, error)
	FindByWorkspace(workspaceID string) ([]*OpenSpecWorkflow, error)
	FindByProjectPath(projectPath string) ([]*OpenSpecWorkflow, error)
	FindByStatus(status string) ([]*OpenSpecWorkflow, error)
}

// openSpecWorkflowRepository 工作流仓储实现
type openSpecWorkflowRepository struct {
	db *sql.DB
}

// NewOpenSpecWorkflowRepository 创建工作流仓储实例
func NewOpenSpecWorkflowRepository() (OpenSpecWorkflowRepository, error) {
	// 确保数据库已初始化
	if err := InitDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	db, err := OpenDB()
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &openSpecWorkflowRepository{
		db: db,
	}, nil
}

// Save 保存工作流（upsert）
func (r *openSpecWorkflowRepository) Save(workflow *OpenSpecWorkflow) error {
	// 检查是否已存在
	existing, _ := r.FindByWorkspaceAndChange(workflow.WorkspaceID, workflow.ChangeID)

	now := time.Now()
	if existing == nil {
		// 新记录
		workflow.StartedAt = now
		workflow.UpdatedAt = now
	} else {
		// 更新记录，保持原有的 StartedAt
		workflow.StartedAt = existing.StartedAt
		workflow.UpdatedAt = now
		workflow.ID = existing.ID
	}

	metadataJSON, _ := json.Marshal(workflow.Metadata)
	summaryJSON, _ := json.Marshal(workflow.Summary)

	query := `
		INSERT OR REPLACE INTO openspec_workflows 
		(id, workspace_id, project_path, change_id, stage, status, started_at, updated_at, metadata, summary)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		workflow.ID,
		workflow.WorkspaceID,
		workflow.ProjectPath,
		workflow.ChangeID,
		workflow.Stage,
		workflow.Status,
		workflow.StartedAt.UnixMilli(),
		workflow.UpdatedAt.UnixMilli(),
		string(metadataJSON),
		string(summaryJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to save workflow: %w", err)
	}

	return nil
}

// FindByWorkspaceAndChange 查询指定工作区和变更的工作流
func (r *openSpecWorkflowRepository) FindByWorkspaceAndChange(workspaceID, changeID string) (*OpenSpecWorkflow, error) {
	query := `
		SELECT id, workspace_id, project_path, change_id, stage, status, started_at, updated_at, metadata, summary
		FROM openspec_workflows
		WHERE workspace_id = ? AND change_id = ?`

	var workflow OpenSpecWorkflow
	var metadataJSON, summaryJSON sql.NullString
	var startedAt, updatedAt int64

	err := r.db.QueryRow(query, workspaceID, changeID).Scan(
		&workflow.ID,
		&workflow.WorkspaceID,
		&workflow.ProjectPath,
		&workflow.ChangeID,
		&workflow.Stage,
		&workflow.Status,
		&startedAt,
		&updatedAt,
		&metadataJSON,
		&summaryJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query workflow: %w", err)
	}

	workflow.StartedAt = time.UnixMilli(startedAt)
	workflow.UpdatedAt = time.UnixMilli(updatedAt)

	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &workflow.Metadata)
	}

	if summaryJSON.Valid {
		json.Unmarshal([]byte(summaryJSON.String), &workflow.Summary)
	}

	return &workflow, nil
}

// FindByWorkspace 查询指定工作区的所有工作流
func (r *openSpecWorkflowRepository) FindByWorkspace(workspaceID string) ([]*OpenSpecWorkflow, error) {
	query := `
		SELECT id, workspace_id, project_path, change_id, stage, status, started_at, updated_at, metadata, summary
		FROM openspec_workflows
		WHERE workspace_id = ?
		ORDER BY updated_at DESC`

	rows, err := r.db.Query(query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflows: %w", err)
	}
	defer rows.Close()

	var workflows []*OpenSpecWorkflow
	for rows.Next() {
		var workflow OpenSpecWorkflow
		var metadataJSON, summaryJSON sql.NullString
		var startedAt, updatedAt int64

		if err := rows.Scan(
			&workflow.ID,
			&workflow.WorkspaceID,
			&workflow.ProjectPath,
			&workflow.ChangeID,
			&workflow.Stage,
			&workflow.Status,
			&startedAt,
			&updatedAt,
			&metadataJSON,
			&summaryJSON,
		); err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}

		workflow.StartedAt = time.UnixMilli(startedAt)
		workflow.UpdatedAt = time.UnixMilli(updatedAt)

		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &workflow.Metadata)
		}

		if summaryJSON.Valid {
			json.Unmarshal([]byte(summaryJSON.String), &workflow.Summary)
		}

		workflows = append(workflows, &workflow)
	}

	return workflows, nil
}

// FindByProjectPath 直接通过项目路径查询工作流（不依赖 workspace_id）
func (r *openSpecWorkflowRepository) FindByProjectPath(projectPath string) ([]*OpenSpecWorkflow, error) {
	// 规范化路径以便匹配
	normalizedPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize path: %w", err)
	}
	normalizedPathSlash := filepath.ToSlash(normalizedPath)
	normalizedPathSlash = strings.TrimSuffix(normalizedPathSlash, "/")
	normalizedPathBackslash := strings.ReplaceAll(normalizedPathSlash, "/", "\\")
	normalizedPathBackslash = strings.TrimSuffix(normalizedPathBackslash, "\\")

	// 查询所有可能的路径格式（支持不同的路径分隔符和大小写）
	// SQLite 的 LIKE 操作符不区分大小写（取决于 collation），但为了保险，我们使用多个条件
	query := `
		SELECT id, workspace_id, project_path, change_id, stage, status, started_at, updated_at, metadata, summary
		FROM openspec_workflows
		WHERE LOWER(REPLACE(project_path, '\', '/')) = LOWER(?)
		   OR LOWER(REPLACE(project_path, '/', '\')) = LOWER(?)
		   OR LOWER(project_path) = LOWER(?)
		   OR LOWER(project_path) = LOWER(?)
		ORDER BY updated_at DESC`

	rows, err := r.db.Query(query, normalizedPathSlash, normalizedPathBackslash, normalizedPath, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflows by project path: %w", err)
	}
	defer rows.Close()

	var workflows []*OpenSpecWorkflow
	for rows.Next() {
		var workflow OpenSpecWorkflow
		var metadataJSON, summaryJSON sql.NullString
		var startedAt, updatedAt int64

		if err := rows.Scan(
			&workflow.ID,
			&workflow.WorkspaceID,
			&workflow.ProjectPath,
			&workflow.ChangeID,
			&workflow.Stage,
			&workflow.Status,
			&startedAt,
			&updatedAt,
			&metadataJSON,
			&summaryJSON,
		); err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}

		workflow.StartedAt = time.UnixMilli(startedAt)
		workflow.UpdatedAt = time.UnixMilli(updatedAt)

		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &workflow.Metadata)
		}

		if summaryJSON.Valid {
			json.Unmarshal([]byte(summaryJSON.String), &workflow.Summary)
		}

		workflows = append(workflows, &workflow)
	}

	return workflows, nil
}

// FindByStatus 查询指定状态的工作流
func (r *openSpecWorkflowRepository) FindByStatus(status string) ([]*OpenSpecWorkflow, error) {
	query := `
		SELECT id, workspace_id, project_path, change_id, stage, status, started_at, updated_at, metadata, summary
		FROM openspec_workflows
		WHERE status = ?
		ORDER BY updated_at DESC`

	rows, err := r.db.Query(query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflows: %w", err)
	}
	defer rows.Close()

	var workflows []*OpenSpecWorkflow
	for rows.Next() {
		var workflow OpenSpecWorkflow
		var metadataJSON, summaryJSON sql.NullString
		var startedAt, updatedAt int64

		if err := rows.Scan(
			&workflow.ID,
			&workflow.WorkspaceID,
			&workflow.ProjectPath,
			&workflow.ChangeID,
			&workflow.Stage,
			&workflow.Status,
			&startedAt,
			&updatedAt,
			&metadataJSON,
			&summaryJSON,
		); err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}

		workflow.StartedAt = time.UnixMilli(startedAt)
		workflow.UpdatedAt = time.UnixMilli(updatedAt)

		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &workflow.Metadata)
		}

		if summaryJSON.Valid {
			json.Unmarshal([]byte(summaryJSON.String), &workflow.Summary)
		}

		workflows = append(workflows, &workflow)
	}

	return workflows, nil
}
