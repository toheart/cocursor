package mcp

import (
	"fmt"
	"net/http"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	appRAG "github.com/cocursor/backend/internal/application/rag"
	infraStorage "github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServer MCP 服务器
type MCPServer struct {
	server         *mcp.Server
	handler        http.Handler
	projectManager *appCursor.ProjectManager
	summaryRepo    infraStorage.DailySummaryRepository
	sessionRepo    infraStorage.WorkspaceSessionRepository
	ragInitializer *appRAG.RAGInitializer
}

// NewServer 创建 MCP 服务器
func NewServer(
	projectManager *appCursor.ProjectManager,
	summaryRepo infraStorage.DailySummaryRepository,
	sessionRepo infraStorage.WorkspaceSessionRepository,
	ragInitializer *appRAG.RAGInitializer,
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
		sessionRepo:    sessionRepo,
		ragInitializer: ragInitializer,
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

	// 注册 RAG 工具：search_history
	mcp.AddTool(server, &mcp.Tool{
		Name: "search_history",
		Description: `Search through historical AI conversations in the current project to find relevant context and solutions.

Use this tool when you need to:
- Find previous discussions about similar problems or code patterns in this project
- Look up how a similar issue was solved before
- Retrieve context from past conversations about specific topics, files, or technologies

Parameters:
- query (string, required): Natural language description of what you're looking for. Be specific about the problem, technology, or concept.
- project_path (string, required): Current project path, e.g., /Users/xxx/code/myproject
- limit (int, optional): Maximum number of results to return (1-10, default: 3)

Returns: List of relevant conversations with summaries, topics, tags, and time info.

Example queries:
- "How to implement pagination in Go API"
- "Fix React useEffect memory leak"
- "Database connection pooling configuration"`,
	}, mcpServer.searchHistoryTool)

	// 注册 User Profile 工具：get_user_messages_for_profile
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_user_messages_for_profile",
		Description: `Get user messages from chat history for profile analysis. Only extracts user messages (not AI responses).
Parameters:
- scope (string, required): 'global' for all projects or 'project' for specific project
- project_path (string, optional): Project path (required when scope is 'project')
- days_back (int, optional): Number of days to analyze, defaults to 30
- recent_sessions (int, optional): Number of recent sessions to fully extract, defaults to 10
- sampling_rate (float, optional): Sampling rate for historical sessions (0-1), defaults to 0.3
- max_historical_msgs (int, optional): Maximum historical messages, defaults to 200

Returns: user messages (recent and historical), statistics (time/project distribution), existing profile, and metadata for idempotency check.`,
	}, mcpServer.getUserMessagesForProfileTool)

	// 注册 User Profile 工具：save_user_profile
	mcp.AddTool(server, &mcp.Tool{
		Name: "save_user_profile",
		Description: `Save user profile to local filesystem.
Parameters:
- scope (string, required): 'global' or 'project'
- project_path (string, optional): Project path (required when scope is 'project')
- content (string, required): Profile content in Markdown format (without YAML frontmatter)
- language (string, optional): Language for frontmatter description, use 'zh' for Chinese or 'en' for English. Should match stats.primary_language from get_user_messages_for_profile result.

For project scope, the profile is saved to {project}/.cursor/rules/user-profile.mdc with YAML frontmatter (alwaysApply: true) and .gitignore is automatically updated.
For global scope, the profile is saved to ~/.cocursor/profiles/global.md.

Returns: success status, file path, git_ignored flag, and message.`,
	}, mcpServer.saveUserProfileTool)

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
