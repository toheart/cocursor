package marketplace

import "errors"

var (
	// ErrInvalidPluginID 无效的插件 ID
	ErrInvalidPluginID = errors.New("invalid plugin id")
	// ErrInvalidPluginName 无效的插件名称
	ErrInvalidPluginName = errors.New("invalid plugin name")
	// ErrInvalidSkillName 无效的 Skill 名称
	ErrInvalidSkillName = errors.New("invalid skill name")
	// ErrInvalidMCPServerName 无效的 MCP 服务器名称
	ErrInvalidMCPServerName = errors.New("invalid mcp server name")
	// ErrInvalidMCPTransport 无效的 MCP 传输方式
	ErrInvalidMCPTransport = errors.New("invalid mcp transport, must be 'sse' or 'streamable-http'")
	// ErrInvalidMCPURL 无效的 MCP URL
	ErrInvalidMCPURL = errors.New("invalid mcp url")
)
