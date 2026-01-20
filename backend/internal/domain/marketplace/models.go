package marketplace

import (
	"fmt"
	"time"
)

// PluginSource 插件来源
type PluginSource string

const (
	// SourceBuiltin 内建技能，随 cocursor 发布
	SourceBuiltin PluginSource = "builtin"
	// SourceProject 项目级技能，存储在项目 .cocursor/skills/
	SourceProject PluginSource = "project"
	// SourceTeamGlobal 团队全局技能，所有团队成员可见
	SourceTeamGlobal PluginSource = "team_global"
	// SourceTeamProject 团队项目级技能，所有团队成员可见
	SourceTeamProject PluginSource = "team_project"
)

// LocaleField 多语言字段
type LocaleField struct {
	ZhCN string `json:"zh-CN,omitempty"` // 中文
	En   string `json:"en,omitempty"`    // 英文
}

// Get 获取指定语言的文本，默认返回英文
func (f *LocaleField) Get(lang string) string {
	if f == nil {
		return ""
	}
	if lang == "zh-CN" && f.ZhCN != "" {
		return f.ZhCN
	}
	if f.En != "" {
		return f.En
	}
	if f.ZhCN != "" {
		return f.ZhCN
	}
	return ""
}

// Plugin 插件
type Plugin struct {
	// 基础信息
	ID              string      `json:"id"`                         // 唯一标识
	Name            string      `json:"name"`                       // 显示名称（向后兼容，用于解析单语言格式）
	NameI18n        LocaleField `json:"name_i18n,omitempty"`        // 多语言名称
	Description     string      `json:"description"`                // 描述（向后兼容，用于解析单语言格式）
	DescI18n        LocaleField `json:"description_i18n,omitempty"` // 多语言描述
	Author          string      `json:"author"`                     // 作者
	Version         string      `json:"version"`                    // 版本号
	Icon            string      `json:"icon,omitempty"`             // 图标路径（可选）
	Category        string      `json:"category"`                   // 分类（英文代码，用于筛选）
	CategoryI18n    LocaleField `json:"category_i18n,omitempty"`    // 多语言分类显示
	CategoryDisplay string      `json:"category_display,omitempty"` // 本地化后的分类显示名称（返回给前端）

	// 状态信息
	Installed        bool   `json:"installed"`                   // 是否已安装
	InstalledVersion string `json:"installed_version,omitempty"` // 已安装的版本

	// 插件组件（skill 必须，mcp 和 command 可选）
	Skill   SkillComponent    `json:"skill"`             // Skill 组件（必须）
	MCP     *MCPComponent     `json:"mcp,omitempty"`     // MCP 组件（可选）
	Command *CommandComponent `json:"command,omitempty"` // Command 组件（可选）

	// 来源信息（团队技能扩展）
	Source   PluginSource `json:"source,omitempty"`    // 来源类型
	FullID   string       `json:"full_id,omitempty"`   // 唯一标识（格式：{team_id}:{plugin_id}）
	TeamID   string       `json:"team_id,omitempty"`   // 所属团队 ID（仅团队技能）
	TeamName string       `json:"team_name,omitempty"` // 所属团队名称（仅团队技能）

	// 团队技能作者信息
	AuthorID       string `json:"author_id,omitempty"`       // 作者成员 ID
	AuthorName     string `json:"author_name,omitempty"`     // 作者名称
	AuthorEndpoint string `json:"author_endpoint,omitempty"` // 作者端点（用于 P2P 下载）
	AuthorOnline   bool   `json:"author_online,omitempty"`   // 作者是否在线

	// 团队技能时间信息
	PublishedAt *time.Time `json:"published_at,omitempty"` // 发布时间

	// 团队技能下载状态
	IsDownloaded bool       `json:"is_downloaded,omitempty"` // 是否已下载到本地
	DownloadedAt *time.Time `json:"downloaded_at,omitempty"` // 下载时间
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

// Localize 根据语言返回本地化的插件信息
func (p *Plugin) Localize(lang string) *Plugin {
	localized := *p

	// 使用多语言字段
	if localized.NameI18n.Get(lang) != "" {
		localized.Name = localized.NameI18n.Get(lang)
	}
	if localized.DescI18n.Get(lang) != "" {
		localized.Description = localized.DescI18n.Get(lang)
	}
	if localized.CategoryI18n.Get(lang) != "" {
		localized.CategoryDisplay = localized.CategoryI18n.Get(lang)
		// 保留原始的 category（英文代码），用于前端筛选
		// 将本地化的分类名称放入 CategoryDisplay
	}

	return &localized
}

// GetFullID 获取完整的唯一标识
// 对于团队技能，格式为 {team_id}:{plugin_id}
// 对于其他来源，直接返回 ID
func (p *Plugin) GetFullID() string {
	if p.FullID != "" {
		return p.FullID
	}
	if p.TeamID != "" {
		return fmt.Sprintf("%s:%s", p.TeamID, p.ID)
	}
	return p.ID
}

// IsTeamSkill 判断是否为团队技能
func (p *Plugin) IsTeamSkill() bool {
	return p.Source == SourceTeamGlobal || p.Source == SourceTeamProject
}

// GetInstallDirName 获取安装目录名称
// 对于团队技能，格式为 {team_id}-{skill_name}，避免与内建技能冲突
func (p *Plugin) GetInstallDirName() string {
	if p.IsTeamSkill() && p.TeamID != "" {
		return fmt.Sprintf("%s-%s", p.TeamID, p.Skill.SkillName)
	}
	return p.Skill.SkillName
}
