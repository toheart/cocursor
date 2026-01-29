package lifecycle

import "github.com/google/wire"

// ProviderSet Lifecycle ProviderSet
var ProviderSet = wire.NewSet(
	NewWindowManager,
)
