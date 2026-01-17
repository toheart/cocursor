package http

import (
	"github.com/google/wire"
	"github.com/cocursor/backend/internal/interfaces/http/handler"
)

// ProviderSet HTTP 接口层 ProviderSet
var ProviderSet = wire.NewSet(
	handler.ProviderSet,
	NewServer,
)
