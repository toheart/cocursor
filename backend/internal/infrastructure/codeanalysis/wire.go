package codeanalysis

import "github.com/google/wire"

// ProviderSet 基础设施层提供者集合
var ProviderSet = wire.NewSet(
	NewProjectStore,
	NewEntryPointScanner,
	NewSSAAnalyzer,
	NewCallGraphRepository,
	NewCallGraphManager,
	NewDiffAnalyzer,
	NewImpactAnalyzer,
)
