package marketplace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MCPConfigManager MCP 配置管理器
type MCPConfigManager struct{}

// NewMCPConfigManager 创建 MCP 配置管理器
func NewMCPConfigManager() *MCPConfigManager {
	return &MCPConfigManager{}
}

// MCPConfig MCP 配置文件结构
type MCPConfig struct {
	MCPServers map[string]interface{} `json:"mcpServers"`
}

// ReadMCPConfig 读取 MCP 配置文件
// 支持 JSONC 格式（移除注释）
func (m *MCPConfigManager) ReadMCPConfig() (*MCPConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".cursor", "mcp.json")

	// 如果文件不存在，返回空配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &MCPConfig{
			MCPServers: make(map[string]interface{}),
		}, nil
	}

	// 读取文件内容
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mcp.json: %w", err)
	}

	// 移除 JSONC 注释
	cleanedData := removeJSONCComments(string(data))

	// 解析 JSON
	var config MCPConfig
	if err := json.Unmarshal([]byte(cleanedData), &config); err != nil {
		return nil, fmt.Errorf("failed to parse mcp.json: %w", err)
	}

	// 确保 mcpServers 不为 nil
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]interface{})
	}

	return &config, nil
}

// WriteMCPConfig 写入 MCP 配置文件
func (m *MCPConfigManager) WriteMCPConfig(config *MCPConfig) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".cursor", "mcp.json")

	// 确保目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 确保 mcpServers 不为 nil
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]interface{})
	}

	// 格式化 JSON（2 空格缩进）
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write mcp.json: %w", err)
	}

	return nil
}

// AddMCPServer 添加 MCP 服务器配置
func (m *MCPConfigManager) AddMCPServer(serverName string, transport string, url string, headers map[string]string) error {
	// 读取现有配置
	config, err := m.ReadMCPConfig()
	if err != nil {
		return err
	}

	// 构建 MCP 配置对象
	mcpServerConfig := map[string]interface{}{
		"type": transport,
		"url":  url,
	}

	// 如果有 headers，添加到配置中（原样写入，不处理变量插值）
	if len(headers) > 0 {
		mcpServerConfig["headers"] = headers
	}

	// 添加到 mcpServers（如果已存在，覆盖）
	config.MCPServers[serverName] = mcpServerConfig

	// 写入配置
	return m.WriteMCPConfig(config)
}

// RemoveMCPServer 移除 MCP 服务器配置
func (m *MCPConfigManager) RemoveMCPServer(serverName string) error {
	// 读取现有配置
	config, err := m.ReadMCPConfig()
	if err != nil {
		return err
	}

	// 从 mcpServers 中删除
	delete(config.MCPServers, serverName)

	// 如果 mcpServers 为空，保留空对象（不删除整个键）
	// 写入配置
	return m.WriteMCPConfig(config)
}

// removeJSONCComments 移除 JSONC 注释
// 支持单行注释（//）和多行注释（/* */）
func removeJSONCComments(content string) string {
	// 使用状态机方式处理，更准确地处理字符串中的注释符号
	var result strings.Builder
	inString := false
	escapeNext := false
	inSingleLineComment := false
	inMultiLineComment := false

	for i := 0; i < len(content); i++ {
		char := content[i]
		nextChar := byte(0)
		if i+1 < len(content) {
			nextChar = content[i+1]
		}

		if escapeNext {
			if !inSingleLineComment && !inMultiLineComment {
				result.WriteByte(char)
			}
			escapeNext = false
			continue
		}

		if char == '\\' && inString {
			if !inSingleLineComment && !inMultiLineComment {
				result.WriteByte(char)
			}
			escapeNext = true
			continue
		}

		if char == '"' && !inSingleLineComment && !inMultiLineComment {
			inString = !inString
			result.WriteByte(char)
			continue
		}

		if inString {
			if !inSingleLineComment && !inMultiLineComment {
				result.WriteByte(char)
			}
			continue
		}

		// 检查单行注释
		if char == '/' && nextChar == '/' && !inMultiLineComment {
			inSingleLineComment = true
			i++ // 跳过下一个字符
			continue
		}

		if inSingleLineComment && char == '\n' {
			inSingleLineComment = false
			result.WriteByte(char)
			continue
		}

		if inSingleLineComment {
			continue
		}

		// 检查多行注释
		if char == '/' && nextChar == '*' && !inSingleLineComment {
			inMultiLineComment = true
			i++ // 跳过下一个字符
			continue
		}

		if inMultiLineComment && char == '*' && nextChar == '/' {
			inMultiLineComment = false
			i++ // 跳过下一个字符
			continue
		}

		if inMultiLineComment {
			continue
		}

		// 正常字符
		result.WriteByte(char)
	}

	return result.String()
}
