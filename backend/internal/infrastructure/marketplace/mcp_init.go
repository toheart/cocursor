package marketplace

import (
	"fmt"
	"log"

	"github.com/cocursor/backend/internal/infrastructure/config"
)

// MCPInitializer MCP 初始化器
type MCPInitializer struct {
	mcpConfigManager *MCPConfigManager
	serverConfig     *config.ServerConfig
}

// NewMCPInitializer 创建 MCP 初始化器
func NewMCPInitializer(mcpConfigManager *MCPConfigManager, serverConfig *config.ServerConfig) *MCPInitializer {
	return &MCPInitializer{
		mcpConfigManager: mcpConfigManager,
		serverConfig:     serverConfig,
	}
}

// InitializeDefaultMCP 初始化默认 MCP 服务器配置
// 在服务器启动时自动配置 CoCursor 的 MCP 服务器
func (m *MCPInitializer) InitializeDefaultMCP() error {
	// 构建 MCP URL（使用 HTTP 端口）
	httpPort := m.serverConfig.HTTPPort
	// 确保端口格式正确（如果没有 : 前缀，添加它）
	if httpPort != "" && httpPort[0] != ':' {
		httpPort = ":" + httpPort
	}
	if httpPort == "" {
		httpPort = ":19960" // 默认端口
	}
	mcpURL := fmt.Sprintf("http://localhost%s/mcp/sse", httpPort)
	serverName := "cocursor"

	// 检查是否已配置
	config, err := m.mcpConfigManager.ReadMCPConfig()
	if err != nil {
		return fmt.Errorf("failed to read MCP config: %w", err)
	}

	// 检查是否已存在相同配置
	if servers, ok := config.MCPServers[serverName].(map[string]interface{}); ok {
		if url, ok := servers["url"].(string); ok && url == mcpURL {
			log.Printf("Default MCP server '%s' already configured, skipping", serverName)
			return nil
		}
	}

	// 配置默认 MCP 服务器
	if err := m.mcpConfigManager.AddMCPServer(serverName, "sse", mcpURL, nil); err != nil {
		return fmt.Errorf("failed to configure default MCP server: %w", err)
	}

	log.Printf("Default MCP server '%s' configured successfully: %s", serverName, mcpURL)
	return nil
}
