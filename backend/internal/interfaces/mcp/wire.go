package mcp

import "github.com/google/wire"

// ProviderSet MCP 接口层 ProviderSet
var ProviderSet = wire.NewSet(
	NewServer,
)
