package cursor

import (
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/google/wire"
)

// ProviderSet Cursor 基础设施层 ProviderSet
var ProviderSet = wire.NewSet(
	NewPathResolver,
	NewDBReader,
	ProvideGlobalDBReader,
)

// ProvideGlobalDBReader 提供 GlobalDBReader 实例（用于依赖注入）
func ProvideGlobalDBReader(pathResolver *PathResolver) (domainCursor.GlobalDBReader, error) {
	return NewGlobalDBReader(pathResolver)
}
