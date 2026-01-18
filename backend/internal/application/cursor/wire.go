package cursor

import "github.com/google/wire"

// ProviderSet Cursor 应用服务 ProviderSet
var ProviderSet = wire.NewSet(
	NewStatsService,
)
