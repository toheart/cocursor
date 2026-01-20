// Package watcher 提供文件监听和事件分发功能
package watcher

import (
	"log/slog"
	"sync"

	"github.com/cocursor/backend/internal/domain/events"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// eventBusImpl EventBus 的实现
type eventBusImpl struct {
	// handlers 按事件类型存储的处理器列表
	handlers map[events.EventType][]events.Handler
	// mu 保护 handlers 的互斥锁
	mu sync.RWMutex
	// logger 日志记录器
	logger *slog.Logger
	// closed 是否已关闭
	closed bool
	// wg 等待所有事件处理完成
	wg sync.WaitGroup
}

// NewEventBus 创建新的事件总线实例
func NewEventBus() events.EventBus {
	return &eventBusImpl{
		handlers: make(map[events.EventType][]events.Handler),
		logger:   log.NewModuleLogger("watcher", "event_bus"),
	}
}

// Subscribe 订阅特定类型的事件
func (b *eventBusImpl) Subscribe(eventType events.EventType, handler events.Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)

	// 返回取消订阅函数
	return func() {
		b.unsubscribe(eventType, handler)
	}
}

// SubscribeMultiple 订阅多个类型的事件
func (b *eventBusImpl) SubscribeMultiple(eventTypes []events.EventType, handler events.Handler) func() {
	unsubscribers := make([]func(), 0, len(eventTypes))

	for _, eventType := range eventTypes {
		unsub := b.Subscribe(eventType, handler)
		unsubscribers = append(unsubscribers, unsub)
	}

	// 返回取消所有订阅的函数
	return func() {
		for _, unsub := range unsubscribers {
			unsub()
		}
	}
}

// unsubscribe 取消订阅
func (b *eventBusImpl) unsubscribe(eventType events.EventType, handler events.Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	handlers := b.handlers[eventType]
	for i, h := range handlers {
		// 使用指针比较来识别处理器
		if &h == &handler {
			b.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// Publish 异步发布事件
func (b *eventBusImpl) Publish(event events.Event) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return
	}

	// 复制处理器列表，避免长时间持有锁
	handlers := make([]events.Handler, len(b.handlers[event.Type()]))
	copy(handlers, b.handlers[event.Type()])
	b.mu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	b.logger.Debug("Publishing event",
		"type", event.Type(),
		"handlers_count", len(handlers),
	)

	// 异步分发到所有处理器
	for _, handler := range handlers {
		b.wg.Add(1)
		go b.dispatchToHandler(event, handler)
	}
}

// dispatchToHandler 分发事件到单个处理器
func (b *eventBusImpl) dispatchToHandler(event events.Event, handler events.Handler) {
	defer b.wg.Done()

	// 捕获 panic，防止单个处理器崩溃影响其他处理器
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("Handler panicked",
				"type", event.Type(),
				"panic", r,
			)
		}
	}()

	if err := handler.HandleEvent(event); err != nil {
		b.logger.Error("Handler returned error",
			"type", event.Type(),
			"error", err,
		)
	}
}

// Close 关闭事件总线
func (b *eventBusImpl) Close() {
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()

	// 等待所有正在处理的事件完成
	b.wg.Wait()

	b.logger.Info("Event bus closed")
}
