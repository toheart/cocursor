package workspace

import "github.com/google/wire"

// ProviderSet Workspace ProviderSet
var ProviderSet = wire.NewSet(
	GetInstance,
)
