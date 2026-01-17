package interfaces

import (
	"github.com/cocursor/backend/internal/interfaces/http"
	"github.com/cocursor/backend/internal/interfaces/mcp"
)

// HTTPServer HTTP 服务器类型别名
type HTTPServer = http.HTTPServer

// MCPServer MCP 服务器类型别名
type MCPServer = mcp.MCPServer
