package watcher

import (
	"github.com/cocursor/backend/internal/domain/events"
)

// ProvideEventBus 提供事件总线实例
func ProvideEventBus() events.EventBus {
	return NewEventBus()
}

// ProvideFileWatcher 提供文件监听器实例
func ProvideFileWatcher(eventBus events.EventBus, workspaceDir string) (*FileWatcher, error) {
	config := DefaultWatchConfig()
	config.WorkspaceDir = workspaceDir

	return NewFileWatcher(config, eventBus)
}
