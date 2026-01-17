package notification

import "github.com/google/wire"

// ProviderSet 通知应用层 ProviderSet
var ProviderSet = wire.NewSet(
	NewService,
	// 注意：Pusher 接口绑定在顶层 wire.go 中处理
)
