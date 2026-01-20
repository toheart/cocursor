package events

// Handler 事件处理器接口
// 订阅者需要实现此接口来处理事件
type Handler interface {
	// HandleEvent 处理事件
	// 返回 error 表示处理失败（仅用于日志记录，不会重试）
	HandleEvent(event Event) error
}

// HandlerFunc 函数类型的处理器适配器
// 方便使用匿名函数作为处理器
type HandlerFunc func(event Event) error

// HandleEvent 实现 Handler 接口
func (f HandlerFunc) HandleEvent(event Event) error {
	return f(event)
}

// EventBus 事件总线接口
// 提供事件的发布和订阅功能
type EventBus interface {
	// Subscribe 订阅特定类型的事件
	// 返回取消订阅的函数
	Subscribe(eventType EventType, handler Handler) (unsubscribe func())

	// SubscribeMultiple 订阅多个类型的事件
	// 返回取消所有订阅的函数
	SubscribeMultiple(eventTypes []EventType, handler Handler) (unsubscribe func())

	// Publish 异步发布事件
	// 事件将被分发到所有匹配的订阅者
	Publish(event Event)

	// Close 关闭事件总线
	// 停止接收新事件，等待已发布事件处理完成
	Close()
}
