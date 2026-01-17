package interfaces

import (
	"github.com/cocursor/backend/internal/interfaces/http"
	"github.com/cocursor/backend/internal/interfaces/mcp"
	"github.com/google/wire"
)

// ProviderSet Interfaces 层总 ProviderSet
var ProviderSet = wire.NewSet(
	http.ProviderSet,
	mcp.ProviderSet,
)
