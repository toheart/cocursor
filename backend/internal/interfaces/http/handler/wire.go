package handler

import "github.com/google/wire"

// ProviderSet Handler ProviderSet
var ProviderSet = wire.NewSet(
	NewNotificationHandler,
	NewStatsHandler,
	NewProjectHandler,
	NewAnalyticsHandler,
	NewWorkspaceHandler,
	NewMarketplaceHandler,
	// 可以继续添加其他 Handler
)
