package marketplace

import (
	"os"
	"testing"

	domainMarketplace "github.com/cocursor/backend/internal/domain/marketplace"
)

func TestMCPInstaller_InstallMCP(t *testing.T) {
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

	mcpConfigManager := NewMCPConfigManager()
	installer := NewMCPInstaller(mcpConfigManager)

	// 测试安装 MCP
	mcp := &domainMarketplace.MCPComponent{
		ServerName: "test-server",
		Transport:  "sse",
		URL:        "http://localhost:8080/mcp",
		Headers: map[string]string{
			"Authorization": "Bearer ${env:TOKEN}",
		},
	}

	if err := installer.InstallMCP(mcp); err != nil {
		t.Fatalf("安装 MCP 失败: %v", err)
	}

	// 验证配置
	config, err := mcpConfigManager.ReadMCPConfig()
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}

	if len(config.MCPServers) != 1 {
		t.Errorf("应该有 1 个服务器，但得到 %d", len(config.MCPServers))
	}

	server := config.MCPServers["test-server"].(map[string]interface{})
	if server["type"] != "sse" {
		t.Errorf("type 应该是 'sse'")
	}
	if server["url"] != "http://localhost:8080/mcp" {
		t.Errorf("url 应该是 'http://localhost:8080/mcp'")
	}

	headers := server["headers"].(map[string]interface{})
	if headers["Authorization"] != "Bearer ${env:TOKEN}" {
		t.Errorf("headers 应该包含 Authorization")
	}
}

func TestMCPInstaller_InstallMCP_Validation(t *testing.T) {
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

	mcpConfigManager := NewMCPConfigManager()
	installer := NewMCPInstaller(mcpConfigManager)

	tests := []struct {
		name    string
		mcp     *domainMarketplace.MCPComponent
		wantErr bool
	}{
		{
			name:    "nil MCP（应该成功，因为 MCP 是可选的）",
			mcp:     nil,
			wantErr: false,
		},
		{
			name: "缺少 server name",
			mcp: &domainMarketplace.MCPComponent{
				Transport: "sse",
				URL:       "http://localhost:8080/mcp",
			},
			wantErr: true,
		},
		{
			name: "缺少 transport",
			mcp: &domainMarketplace.MCPComponent{
				ServerName: "test-server",
				URL:        "http://localhost:8080/mcp",
			},
			wantErr: true,
		},
		{
			name: "缺少 URL",
			mcp: &domainMarketplace.MCPComponent{
				ServerName: "test-server",
				Transport:  "sse",
			},
			wantErr: true,
		},
		{
			name: "无效的 transport 类型",
			mcp: &domainMarketplace.MCPComponent{
				ServerName: "test-server",
				Transport:  "invalid",
				URL:        "http://localhost:8080/mcp",
			},
			wantErr: true,
		},
		{
			name: "有效的 sse transport",
			mcp: &domainMarketplace.MCPComponent{
				ServerName: "test-server",
				Transport:  "sse",
				URL:        "http://localhost:8080/mcp",
			},
			wantErr: false,
		},
		{
			name: "有效的 streamable-http transport",
			mcp: &domainMarketplace.MCPComponent{
				ServerName: "test-server",
				Transport:  "streamable-http",
				URL:        "http://localhost:8080/mcp",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := installer.InstallMCP(tt.mcp)
			if (err != nil) != tt.wantErr {
				t.Errorf("InstallMCP() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMCPInstaller_UninstallMCP(t *testing.T) {
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

	mcpConfigManager := NewMCPConfigManager()
	installer := NewMCPInstaller(mcpConfigManager)

	// 先安装一个服务器
	mcp := &domainMarketplace.MCPComponent{
		ServerName: "test-server",
		Transport:  "sse",
		URL:        "http://localhost:8080/mcp",
	}

	if err := installer.InstallMCP(mcp); err != nil {
		t.Fatalf("安装 MCP 失败: %v", err)
	}

	// 验证已安装
	config, _ := mcpConfigManager.ReadMCPConfig()
	if len(config.MCPServers) != 1 {
		t.Fatal("应该已安装 1 个服务器")
	}

	// 卸载
	if err := installer.UninstallMCP("test-server"); err != nil {
		t.Fatalf("卸载 MCP 失败: %v", err)
	}

	// 验证已卸载
	config, _ = mcpConfigManager.ReadMCPConfig()
	if len(config.MCPServers) != 0 {
		t.Errorf("卸载后应该有 0 个服务器，但得到 %d", len(config.MCPServers))
	}

	// 测试卸载不存在的服务器（应该成功，不报错）
	if err := installer.UninstallMCP("non-existent"); err != nil {
		t.Errorf("卸载不存在的服务器不应该报错: %v", err)
	}

	// 测试卸载空字符串（应该成功，不报错）
	if err := installer.UninstallMCP(""); err != nil {
		t.Errorf("卸载空字符串不应该报错: %v", err)
	}
}
