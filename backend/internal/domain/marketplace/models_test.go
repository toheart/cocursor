package marketplace

import (
	"encoding/json"
	"testing"
)

func TestPlugin_JSONSerialization(t *testing.T) {
	plugin := &Plugin{
		ID:          "test-plugin",
		Name:        "测试插件",
		Description: "这是一个测试插件",
		Author:      "Test Author",
		Version:     "1.0.0",
		Category:    "工具",
		Installed:   false,
		Skill: SkillComponent{
			SkillName: "test-skill",
		},
		MCP: &MCPComponent{
			ServerName: "test-mcp",
			Transport:  "sse",
			URL:        "http://localhost:19961/mcp/sse",
			Headers: map[string]string{
				"Authorization": "Bearer ${env:TOKEN}",
			},
		},
		Command: &CommandComponent{
			Commands: []CommandItem{
				{
					CommandID: "test-command",
				},
			},
		},
	}

	// 测试序列化
	jsonData, err := json.Marshal(plugin)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	// 测试反序列化
	var decoded Plugin
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	// 验证数据
	if decoded.ID != plugin.ID {
		t.Errorf("ID 不匹配: 期望 %s, 得到 %s", plugin.ID, decoded.ID)
	}
	if decoded.Skill.SkillName != plugin.Skill.SkillName {
		t.Errorf("SkillName 不匹配: 期望 %s, 得到 %s", plugin.Skill.SkillName, decoded.Skill.SkillName)
	}
}

func TestPlugin_Validate(t *testing.T) {
	tests := []struct {
		name    string
		plugin  *Plugin
		wantErr bool
	}{
		{
			name: "有效插件",
			plugin: &Plugin{
				ID:      "valid-plugin",
				Name:    "有效插件",
				Version: "1.0.0",
				Skill: SkillComponent{
					SkillName: "valid-skill",
				},
			},
			wantErr: false,
		},
		{
			name: "缺少 ID",
			plugin: &Plugin{
				Name:    "测试插件",
				Version: "1.0.0",
				Skill: SkillComponent{
					SkillName: "test-skill",
				},
			},
			wantErr: true,
		},
		{
			name: "缺少 SkillName",
			plugin: &Plugin{
				ID:      "test-plugin",
				Name:    "测试插件",
				Version: "1.0.0",
				Skill: SkillComponent{
					SkillName: "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plugin.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMCPComponent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mcp     *MCPComponent
		wantErr bool
	}{
		{
			name: "有效的 SSE MCP",
			mcp: &MCPComponent{
				ServerName: "test-mcp",
				Transport:  "sse",
				URL:        "http://localhost:19961/mcp/sse",
			},
			wantErr: false,
		},
		{
			name: "有效的 streamable-http MCP",
			mcp: &MCPComponent{
				ServerName: "test-mcp",
				Transport:  "streamable-http",
				URL:        "https://api.example.com/mcp",
			},
			wantErr: false,
		},
		{
			name: "无效的传输方式",
			mcp: &MCPComponent{
				ServerName: "test-mcp",
				Transport:  "stdio",
				URL:        "http://localhost:19961/mcp/sse",
			},
			wantErr: true,
		},
		{
			name: "缺少 URL",
			mcp: &MCPComponent{
				ServerName: "test-mcp",
				Transport:  "sse",
				URL:        "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mcp.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
