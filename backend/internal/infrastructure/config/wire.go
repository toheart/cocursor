package config

import "github.com/google/wire"

// ProviderSet 配置 ProviderSet
var ProviderSet = wire.NewSet(
	NewConfig,
	NewDatabaseConfig,
	NewServerConfig,
)
