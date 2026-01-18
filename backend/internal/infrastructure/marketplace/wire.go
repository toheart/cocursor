package marketplace

import "github.com/google/wire"

// ProviderSet Marketplace 基础设施层 ProviderSet
var ProviderSet = wire.NewSet(
	NewStateManager,
	NewPluginLoader,
	NewAgentsUpdater,
	NewMCPConfigManager,
	NewMCPInstaller,
	NewMCPInitializer,
	NewCommandInstaller,
	NewSkillInstaller,
)
