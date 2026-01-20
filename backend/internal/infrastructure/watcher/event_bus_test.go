package watcher

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cocursor/backend/internal/domain/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventBus_Subscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var received atomic.Bool

	unsub := bus.Subscribe(events.SessionFileCreated, events.HandlerFunc(func(event events.Event) error {
		received.Store(true)
		return nil
	}))
	defer unsub()

	// 发布事件
	bus.Publish(&events.SessionFileEvent{
		EventType: events.SessionFileCreated,
		SessionID: "test-session",
		EventTime: time.Now(),
	})

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	assert.True(t, received.Load(), "handler should have received the event")
}

func TestEventBus_MultipleHandlers(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var count atomic.Int32

	// 注册多个处理器
	for i := 0; i < 3; i++ {
		unsub := bus.Subscribe(events.SessionFileModified, events.HandlerFunc(func(event events.Event) error {
			count.Add(1)
			return nil
		}))
		defer unsub()
	}

	// 发布事件
	bus.Publish(&events.SessionFileEvent{
		EventType: events.SessionFileModified,
		SessionID: "test-session",
		EventTime: time.Now(),
	})

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(3), count.Load(), "all 3 handlers should have received the event")
}

func TestEventBus_SubscribeMultiple(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var count atomic.Int32

	// 订阅多个事件类型
	unsub := bus.SubscribeMultiple(
		[]events.EventType{events.SessionFileCreated, events.SessionFileModified},
		events.HandlerFunc(func(event events.Event) error {
			count.Add(1)
			return nil
		}),
	)
	defer unsub()

	// 发布两种类型的事件
	bus.Publish(&events.SessionFileEvent{
		EventType: events.SessionFileCreated,
		EventTime: time.Now(),
	})
	bus.Publish(&events.SessionFileEvent{
		EventType: events.SessionFileModified,
		EventTime: time.Now(),
	})

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(2), count.Load(), "handler should have received both events")
}

func TestEventBus_ErrorIsolation(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var successCount atomic.Int32

	// 注册一个会失败的处理器
	bus.Subscribe(events.SessionFileCreated, events.HandlerFunc(func(event events.Event) error {
		return errors.New("handler error")
	}))

	// 注册一个正常的处理器
	bus.Subscribe(events.SessionFileCreated, events.HandlerFunc(func(event events.Event) error {
		successCount.Add(1)
		return nil
	}))

	// 发布事件
	bus.Publish(&events.SessionFileEvent{
		EventType: events.SessionFileCreated,
		EventTime: time.Now(),
	})

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), successCount.Load(), "second handler should still receive the event")
}

func TestEventBus_PanicRecovery(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var successCount atomic.Int32

	// 注册一个会 panic 的处理器
	bus.Subscribe(events.SessionFileCreated, events.HandlerFunc(func(event events.Event) error {
		panic("handler panic")
	}))

	// 注册一个正常的处理器
	bus.Subscribe(events.SessionFileCreated, events.HandlerFunc(func(event events.Event) error {
		successCount.Add(1)
		return nil
	}))

	// 发布事件（不应该 panic）
	require.NotPanics(t, func() {
		bus.Publish(&events.SessionFileEvent{
			EventType: events.SessionFileCreated,
			EventTime: time.Now(),
		})
	})

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), successCount.Load(), "second handler should still receive the event")
}

func TestEventBus_NoHandlers(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	// 发布没有订阅者的事件（不应该 panic）
	require.NotPanics(t, func() {
		bus.Publish(&events.SessionFileEvent{
			EventType: events.SessionFileCreated,
			EventTime: time.Now(),
		})
	})
}

func TestEventBus_CloseWaitsForHandlers(t *testing.T) {
	bus := NewEventBus().(*eventBusImpl)

	var wg sync.WaitGroup
	wg.Add(1)

	handlerStarted := make(chan struct{})
	handlerDone := make(chan struct{})

	bus.Subscribe(events.SessionFileCreated, events.HandlerFunc(func(event events.Event) error {
		close(handlerStarted)
		time.Sleep(200 * time.Millisecond) // 模拟耗时处理
		close(handlerDone)
		return nil
	}))

	// 发布事件
	bus.Publish(&events.SessionFileEvent{
		EventType: events.SessionFileCreated,
		EventTime: time.Now(),
	})

	// 等待处理器开始
	<-handlerStarted

	// 在另一个 goroutine 中关闭
	go func() {
		bus.Close()
		wg.Done()
	}()

	// Close 应该等待处理器完成
	select {
	case <-handlerDone:
		// 好，处理器完成了
	case <-time.After(500 * time.Millisecond):
		t.Fatal("handler should have completed")
	}

	wg.Wait()
}
