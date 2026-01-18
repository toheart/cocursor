package marketplace

import "fmt"

// Plugin 插件
type Plugin struct {
	// 基础信息
	ID          string `json:"id"`             // 唯一标识
	Name        string `json:"name"`           // 显示名称
	Description string `json:"description"`    // 描述
	Author      string `json:"author"`         // 作者
	Version     string `json:"version"`        // 版本号
	Icon        string `json:"icon,omitempty"` // 图标路径（可选）
	Category    string `json:"category"`       // 分类

	// 状态信息
	Installed        bool   `json:"installed"`                   // 是否已安装
	InstalledVersion string `json:"installed_version,omitempty"` // 已安装的版本

	// 插件组件（skill 必须，mcp 和 command 可选）
	Skill   SkillComponent    `json:"skill"`             // Skill 组件（必须）
	MCP     *MCPComponent     `json:"mcp,omitempty"`     // MCP 组件（可选）
	Command *CommandComponent `json:"command,omitempty"` // Command 组件（可选）
}

// SkillComponent Skill 组件
type SkillComponent struct {
	SkillName string `json:"skill_name"` // skill 目录名
}

// MCPComponent MCP 组件
type MCPComponent struct {
	ServerName string            `json:"server_name"`       // MCP 服务器名称（在 mcp.json 中的 key）
	Transport  string            `json:"transport"`         // "sse" | "streamable-http"
	URL        string            `json:"url"`               // MCP 服务器 URL
	Headers    map[string]string `json:"headers,omitempty"` // HTTP 头（支持 ${env:VAR}，原样写入）
}

// CommandComponent Command 组件
type CommandComponent struct {
	Commands []CommandItem `json:"commands,omitempty"`
}

// CommandItem 单个命令项
type CommandItem struct {
	CommandID string `json:"command_id"` // command ID（文件名）
}

// Validate 验证插件数据
func (p *Plugin) Validate() error {
	if p.ID == "" {
		return ErrInvalidPluginID
	}
	if p.Name == "" {
		return ErrInvalidPluginName
	}
	if p.Skill.SkillName == "" {
		return ErrInvalidSkillName
	}
	if p.MCP != nil {
		if err := p.MCP.Validate(); err != nil {
			return err
		}
	}
	if p.Command != nil {
		if err := p.Command.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Validate 验证 Command 组件
func (c *CommandComponent) Validate() error {
	// Command 是可选的，如果没有 Commands 也是允许的
	if len(c.Commands) == 0 {
		return nil
	}

	for _, cmd := range c.Commands {
		if cmd.CommandID == "" {
			return fmt.Errorf("command_id is required in commands array")
		}
	}
	return nil
}

// Validate 验证 MCP 组件
func (m *MCPComponent) Validate() error {
	if m.ServerName == "" {
		return ErrInvalidMCPServerName
	}
	if m.Transport != "sse" && m.Transport != "streamable-http" {
		return ErrInvalidMCPTransport
	}
	if m.URL == "" {
		return ErrInvalidMCPURL
	}
	return nil
}
