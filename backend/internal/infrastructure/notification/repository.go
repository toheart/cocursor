package notification

import (
	"sync"

	"github.com/cocursor/backend/internal/domain/notification"
)

// MemoryRepository 内存仓储实现
type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]*notification.Notification
}

// NewMemoryRepository 创建内存仓储
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		items: make(map[string]*notification.Notification),
	}
}

// Save 保存通知
func (r *MemoryRepository) Save(n *notification.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[n.ID] = n
	return nil
}

// FindByTeamCode 根据团队码查找通知
func (r *MemoryRepository) FindByTeamCode(teamCode string) ([]*notification.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*notification.Notification
	for _, n := range r.items {
		if n.TeamCode == teamCode {
			result = append(result, n)
		}
	}
	return result, nil
}

// 编译时检查接口实现
var _ notification.Repository = (*MemoryRepository)(nil)
