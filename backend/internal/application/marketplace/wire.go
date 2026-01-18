package marketplace

import (
	"github.com/google/wire"
)

// ProviderSet Marketplace 应用层 ProviderSet
var ProviderSet = wire.NewSet(
	NewPluginService,
)
