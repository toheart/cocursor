package codeanalysis

import "github.com/google/wire"

// ProviderSet 应用层提供者集合
var ProviderSet = wire.NewSet(
	NewProjectService,
	NewCallGraphService,
	NewImpactService,
)
