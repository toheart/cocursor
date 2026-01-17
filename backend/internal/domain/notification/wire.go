package notification

import "github.com/google/wire"

// ProviderSet 通知领域层 ProviderSet
var ProviderSet = wire.NewSet(
	NewService,
)
