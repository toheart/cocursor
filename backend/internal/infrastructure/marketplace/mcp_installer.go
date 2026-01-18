package marketplace

import (
	"fmt"

	domainMarketplace "github.com/cocursor/backend/internal/domain/marketplace"
)

// MCPInstaller MCP 安装器
type MCPInstaller struct {
	mcpConfigManager *MCPConfigManager
}

// NewMCPInstaller 创建 MCP 安装器
func NewMCPInstaller(mcpConfigManager *MCPConfigManager) *MCPInstaller {
	return &MCPInstaller{
		mcpConfigManager: mcpConfigManager,
	}
}

// InstallMCP 安装 MCP 配置
func (m *MCPInstaller) InstallMCP(mcp *domainMarketplace.MCPComponent) error {
	if mcp == nil {
		return nil // MCP 是可选的，如果没有则跳过
	}

	// 验证必需字段
	if mcp.ServerName == "" {
		return fmt.Errorf("MCP server name is required")
	}
	if mcp.Transport == "" {
		return fmt.Errorf("MCP transport is required")
	}
	if mcp.URL == "" {
		return fmt.Errorf("MCP URL is required")
	}

	// 验证 transport 类型
	if mcp.Transport != "sse" && mcp.Transport != "streamable-http" {
		return fmt.Errorf("invalid MCP transport type: %s (must be 'sse' or 'streamable-http')", mcp.Transport)
	}

	// 添加到 MCP 配置
	return m.mcpConfigManager.AddMCPServer(
		mcp.ServerName,
		mcp.Transport,
		mcp.URL,
		mcp.Headers,
	)
}

// UninstallMCP 卸载 MCP 配置
func (m *MCPInstaller) UninstallMCP(serverName string) error {
	if serverName == "" {
		return nil // 如果没有 server name，跳过
	}

	return m.mcpConfigManager.RemoveMCPServer(serverName)
}
