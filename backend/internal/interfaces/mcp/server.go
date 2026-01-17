package mcp

import (
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServer MCP 服务器
type MCPServer struct {
	server  *mcp.Server
	handler http.Handler
}

// NewServer 创建 MCP 服务器
func NewServer() *MCPServer {
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
		Description: "获取 cocursor 守护进程的状态信息，包括运行状态、版本号和数据库路径",
	}, getDaemonStatusTool)

	// 注册工具：generate_daily_report_context
	mcp.AddTool(server, &mcp.Tool{
		Name:        "generate_daily_report_context",
		Description: "生成每日协作报告上下文，需要提供 project_path 参数（如 D:/code/cocursor）",
	}, generateDailyReportContextTool)

	// 注册工具：get_session_health
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_session_health",
		Description: "获取当前活跃会话的健康状态（熵值），返回熵值、健康状态和警告信息。可选参数 project_path（如 D:/code/cocursor），如果不提供则尝试自动检测",
	}, getSessionHealthTool)

	// 创建 SSE Handler
	handler := mcp.NewSSEHandler(
		func(r *http.Request) *mcp.Server {
			// 每个请求返回同一个服务器实例
			return server
		},
		nil, // SSEOptions，使用默认值
	)

	return &MCPServer{
		server:  server,
		handler: handler,
	}
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
