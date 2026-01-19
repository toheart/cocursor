package log

import "github.com/google/wire"

// ProviderSet 日志基础设施 ProviderSet
var ProviderSet = wire.NewSet(
	NewConfigFromEnv,
)
