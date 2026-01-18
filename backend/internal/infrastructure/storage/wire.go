package storage

import "github.com/google/wire"

// ProviderSet Storage 基础设施层 ProviderSet
var ProviderSet = wire.NewSet(
	NewDailySummaryRepository,
	NewOpenSpecWorkflowRepository,
)
