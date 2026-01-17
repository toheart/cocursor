package websocket

import "github.com/google/wire"

// ProviderSet WebSocket ProviderSet
var ProviderSet = wire.NewSet(
	NewHub,
)
