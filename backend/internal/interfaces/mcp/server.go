package mcp

import (
	"fmt"
	"net/http"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	infraStorage "github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServer MCP 服务器
type MCPServer struct {
	server         *mcp.Server
	handler        http.Handler
	projectManager *appCursor.ProjectManager
	summaryRepo    infraStorage.DailySummaryRepository
	workflowRepo   infraStorage.OpenSpecWorkflowRepository
	sessionRepo    infraStorage.WorkspaceSessionRepository
}

// NewServer 创建 MCP 服务器
func NewServer(
	projectManager *appCursor.ProjectManager,
	summaryRepo infraStorage.DailySummaryRepository,
	workflowRepo infraStorage.OpenSpecWorkflowRepository,
	sessionRepo infraStorage.WorkspaceSessionRepository,
) *MCPServer {
	// 创建 MCP 服务器实例
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "cocursor-daemon",
			Version: "0.1.0",
		},
		nil, // 使用默认能力
	)

	// 注册工具：get_daemon_status
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_daemon_status",
		Description: "Get the status information of the cocursor daemon, including running status, version number, and database path. No parameters required.",
	}, getDaemonStatusTool)

	// 注册工具：generate_daily_report_context
	mcp.AddTool(server, &mcp.Tool{
		Name:        "generate_daily_report_context",
		Description: "Generate daily collaboration report context. Parameters: project_path (string, required) - project path, e.g., D:/code/cocursor. Returns: date, total chats, active users list, and summary.",
	}, generateDailyReportContextTool)

	// 注册工具：get_session_health
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_session_health",
		Description: "Get the health status (entropy) of the current active session. Parameters: project_path (string, optional) - project path, e.g., D:/code/cocursor, if not provided will attempt auto-detection. Returns: entropy value, health status (healthy/sub_healthy/dangerous), warning message, and suggestion message.",
	}, getSessionHealthTool)

	// 创建服务器实例（用于闭包捕获依赖）
	mcpServer := &MCPServer{
		server:         server,
		projectManager: projectManager,
		summaryRepo:    summaryRepo,
		workflowRepo:   workflowRepo,
		sessionRepo:    sessionRepo,
	}

	// 注册新工具：get_daily_sessions
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_daily_sessions",
		Description: "Query the list of sessions created or updated on the specified date, grouped by project. Parameters: date (string, optional) - date format YYYY-MM-DD, defaults to today. Returns: date, sessions grouped by project, and total session count.",
	}, mcpServer.getDailySessionsTool)

	// 注册新工具：get_session_content
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_session_content",
		Description: "Read the plain text conversation content of the specified session, filtering out tool calls and code blocks. Parameters: session_id (string, required) - session ID. Returns: session ID, name, project name, plain text messages list, and total message count.",
	}, mcpServer.getSessionContentTool)

	// 注册新工具：save_daily_summary
	// 使用自动 schema 推断，指针类型会自动生成包含 null 和 object 的 schema
	mcp.AddTool(server, &mcp.Tool{
		Name: "save_daily_summary",
		Description: `Save daily summary to database. 
Parameters:
- date (string, required): Date in YYYY-MM-DD format
- summary (string, required): Summary content in Markdown format
- language (string, optional): Language code, either "zh" or "en", defaults to "zh"
- projects (array, optional): Array of project summary objects (not strings). Each object contains: project_name, project_path, workspace_id, work_items (array), sessions (array), session_count (int)
- categories (object, optional): Work category statistics object. Must be a JSON object (not a string) with integer fields: requirements_discussion, coding, problem_solving, refactoring, code_review, documentation, testing, other. Example: {"requirements_discussion": 3, "coding": 8, "problem_solving": 4, "refactoring": 3, "code_review": 0, "documentation": 0, "testing": 2, "other": 1}
- total_sessions (int, required): Total number of sessions

Returns: success status, summary ID, and message.`,
	}, mcpServer.saveDailySummaryTool)

	// 注册新工具：get_daily_summary
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_daily_summary",
		Description: "Query daily summary for the specified date. Parameters: date (string, required) - date in YYYY-MM-DD format. Returns: summary object (if found) and found flag.",
	}, mcpServer.getDailySummaryTool)

	// 注册新工具：get_daily_conversations（一次性返回所有项目的所有对话内容）
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_daily_conversations",
		Description: "Get all conversation content for the specified date, grouped by project. Returns all sessions with their text messages in a single call. Parameters: date (string, optional) - date format YYYY-MM-DD, defaults to today. Returns: date, projects with sessions and messages, and total session count.",
	}, mcpServer.getDailyConversationsTool)

	// 注册 OpenSpec 工具：openspec_list
	mcp.AddTool(server, &mcp.Tool{
		Name:        "openspec_list",
		Description: "List OpenSpec changes and specifications. Parameters: project_path (string, required) - project path, e.g., D:/code/cocursor; type (string, optional) - type: changes|specs|all, defaults to all. Returns: changes list and specs list.",
	}, openspecListTool)

	// 注册 OpenSpec 工具：openspec_validate
	mcp.AddTool(server, &mcp.Tool{
		Name:        "openspec_validate",
		Description: "Validate OpenSpec change format. Parameters: project_path (string, required) - project path; change_id (string, required) - change ID; strict (bool, optional) - strict mode. Returns: valid status, errors list, warnings list, and change ID.",
	}, openspecValidateTool)

	// 注册 OpenSpec 工具：record_openspec_workflow
	mcp.AddTool(server, &mcp.Tool{
		Name: "record_openspec_workflow",
		Description: `Record OpenSpec workflow status. Only records proposal and apply stages (init stage is skipped).
Parameters:
- project_path (string, required): Project path, e.g., D:/code/cocursor
- change_id (string, required): Change ID
- stage (string, required): Stage, must be one of: "proposal" or "apply" (init and archive are not recorded)
- status (string, required): Status, must be one of: "in_progress", "completed", or "paused"
- metadata (object, optional): Metadata object (not a string) including task progress. Must be a JSON object with string keys and any value types. Example: {"tasks_completed": 5, "tasks_total": 10, "progress": 0.5}

Workflow transitions from proposal to apply are tracked. If stage is apply and tasks.md is completed, will automatically generate work summary.

Returns: success status and message.`,
	}, mcpServer.recordOpenSpecWorkflowTool)

	// 注册 OpenSpec 工具：generate_openspec_workflow_summary
	mcp.AddTool(server, &mcp.Tool{
		Name:        "generate_openspec_workflow_summary",
		Description: "Generate OpenSpec workflow work summary. Parameters: project_path (string, required) - project path; change_id (string, required) - change ID. Returns: change ID, stage, summary, completed tasks count, total tasks count, changed files list, and time spent.",
	}, mcpServer.generateOpenSpecWorkflowSummaryTool)

	// 注册 OpenSpec 工具：get_openspec_workflow_status
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_openspec_workflow_status",
		Description: "Get OpenSpec workflow status. Parameters: project_path (string, optional) - project path; status (string, optional) - status filter: in_progress|completed|paused. Returns: workflow status items list, including change ID, stage, status, progress, updated timestamp, and summary.",
	}, mcpServer.getOpenSpecWorkflowStatusTool)

	// 创建 SSE Handler
	handler := mcp.NewSSEHandler(
		func(r *http.Request) *mcp.Server {
			// 每个请求返回同一个服务器实例
			return server
		},
		nil, // SSEOptions，使用默认值
	)

	mcpServer.handler = handler
	return mcpServer
}

// GetHandler 获取 HTTP Handler（用于集成到 HTTP 服务器）
func (s *MCPServer) GetHandler() http.Handler {
	return s.handler
}

// Start 启动服务器（HTTP/SSE 模式）
// 注意：MCP 服务器通过 HTTP Handler 提供服务，不需要单独启动
func (s *MCPServer) Start() error {
	// HTTP/SSE 模式下，服务器通过 HTTP Handler 提供服务
	// 不需要单独启动，由 HTTP 服务器统一管理
	fmt.Println("MCP 服务器已就绪（HTTP/SSE 模式）")
	return nil
}

// Stop 停止服务器
func (s *MCPServer) Stop() error {
	// HTTP/SSE 模式下，由 HTTP 服务器统一管理生命周期
	return nil
}
