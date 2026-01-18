package marketplace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMCPConfigManager_ReadWriteConfig(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")

	// 设置临时目录为用户主目录
	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
		if originalUserProfile != "" {
			os.Setenv("USERPROFILE", originalUserProfile)
		}
	}()

	manager := NewMCPConfigManager()

	// 测试读取不存在的配置文件（应该返回空配置）
	config, err := manager.ReadMCPConfig()
	if err != nil {
		t.Fatalf("读取不存在的配置文件失败: %v", err)
	}
	if config.MCPServers == nil {
		t.Fatal("MCPServers 应该不为 nil")
	}
	if len(config.MCPServers) != 0 {
		t.Errorf("新配置应该为空，但得到 %d 个服务器", len(config.MCPServers))
	}

	// 测试写入配置
	testConfig := &MCPConfig{
		MCPServers: map[string]interface{}{
			"test-server": map[string]interface{}{
				"type": "sse",
				"url":  "http://localhost:8080/mcp",
			},
		},
	}

	if err := manager.WriteMCPConfig(testConfig); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}

	// 验证文件是否存在
	configPath := filepath.Join(tempDir, ".cursor", "mcp.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("配置文件应该已创建")
	}

	// 测试读取配置
	readConfig, err := manager.ReadMCPConfig()
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}

	if len(readConfig.MCPServers) != 1 {
		t.Errorf("应该读取到 1 个服务器，但得到 %d", len(readConfig.MCPServers))
	}

	server, exists := readConfig.MCPServers["test-server"]
	if !exists {
		t.Fatal("应该存在 test-server")
	}

	serverMap, ok := server.(map[string]interface{})
	if !ok {
		t.Fatal("服务器配置应该是 map")
	}

	if serverMap["type"] != "sse" {
		t.Errorf("type 应该是 'sse'，但得到 '%v'", serverMap["type"])
	}
	if serverMap["url"] != "http://localhost:8080/mcp" {
		t.Errorf("url 应该是 'http://localhost:8080/mcp'，但得到 '%v'", serverMap["url"])
	}
}

func TestMCPConfigManager_AddRemoveServer(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")

	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
		if originalUserProfile != "" {
			os.Setenv("USERPROFILE", originalUserProfile)
		}
	}()

	manager := NewMCPConfigManager()

	// 添加服务器
	headers := map[string]string{
		"Authorization": "Bearer ${env:TOKEN}",
	}
	if err := manager.AddMCPServer("test-server", "sse", "http://localhost:8080/mcp", headers); err != nil {
		t.Fatalf("添加服务器失败: %v", err)
	}

	// 验证配置
	config, err := manager.ReadMCPConfig()
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}

	if len(config.MCPServers) != 1 {
		t.Errorf("应该有 1 个服务器，但得到 %d", len(config.MCPServers))
	}

	server, exists := config.MCPServers["test-server"]
	if !exists {
		t.Fatal("应该存在 test-server")
	}

	serverMap := server.(map[string]interface{})
	if serverMap["type"] != "sse" {
		t.Errorf("type 应该是 'sse'")
	}

	// 测试覆盖（添加同名服务器）
	if err := manager.AddMCPServer("test-server", "streamable-http", "http://localhost:9090/mcp", nil); err != nil {
		t.Fatalf("覆盖服务器失败: %v", err)
	}

	config, _ = manager.ReadMCPConfig()
	server = config.MCPServers["test-server"]
	serverMap = server.(map[string]interface{})
	if serverMap["type"] != "streamable-http" {
		t.Errorf("type 应该是 'streamable-http'，但得到 '%v'", serverMap["type"])
	}

	// 移除服务器
	if err := manager.RemoveMCPServer("test-server"); err != nil {
		t.Fatalf("移除服务器失败: %v", err)
	}

	config, _ = manager.ReadMCPConfig()
	if len(config.MCPServers) != 0 {
		t.Errorf("移除后应该有 0 个服务器，但得到 %d", len(config.MCPServers))
	}
}

func TestMCPConfigManager_JSONCComments(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")

	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
		if originalUserProfile != "" {
			os.Setenv("USERPROFILE", originalUserProfile)
		}
	}()

	// 创建包含注释的 JSONC 文件
	configPath := filepath.Join(tempDir, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	jsoncContent := `{
  // 这是单行注释
  "mcpServers": {
    "test-server": {
      "type": "sse",
      /* 这是多行注释 */
      "url": "http://localhost:8080/mcp"
    }
  }
}`

	if err := os.WriteFile(configPath, []byte(jsoncContent), 0644); err != nil {
		t.Fatalf("写入 JSONC 文件失败: %v", err)
	}

	manager := NewMCPConfigManager()

	// 读取配置（应该能正确解析，忽略注释）
	config, err := manager.ReadMCPConfig()
	if err != nil {
		t.Fatalf("读取 JSONC 配置失败: %v", err)
	}

	if len(config.MCPServers) != 1 {
		t.Errorf("应该读取到 1 个服务器，但得到 %d", len(config.MCPServers))
	}

	server := config.MCPServers["test-server"].(map[string]interface{})
	if server["type"] != "sse" {
		t.Errorf("type 应该是 'sse'")
	}
	if server["url"] != "http://localhost:8080/mcp" {
		t.Errorf("url 应该是 'http://localhost:8080/mcp'")
	}
}

func TestRemoveJSONCComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "单行注释",
			input:    "{\"key\": \"value\" // 这是注释\n}",
			expected: "{\"key\": \"value\"\n}",
		},
		{
			name:     "多行注释",
			input:    `{"key": "value" /* 这是多行注释 */}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "字符串中的注释符号",
			input:    `{"key": "value // 这不是注释"}`,
			expected: `{"key": "value // 这不是注释"}`,
		},
		{
			name:     "混合注释",
			input:    "{\n  // 单行注释\n  \"key\": \"value\" /* 多行注释 */\n}",
			expected: "{\n  \"key\": \"value\"\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeJSONCComments(tt.input)
			// 验证结果可以解析为有效的 JSON
			var jsonObj interface{}
			if err := json.Unmarshal([]byte(result), &jsonObj); err != nil {
				t.Errorf("结果不是有效的 JSON: %v, 输入: %s, 输出: %s", err, tt.input, result)
			}
		})
	}
}
