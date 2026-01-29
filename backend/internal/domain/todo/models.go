package todo

import "time"

// TodoItem 待办事项实体
type TodoItem struct {
	ID          string     // 唯一标识
	Content     string     // 待办内容
	Completed   bool       // 是否完成
	CreatedAt   time.Time  // 创建时间
	CompletedAt *time.Time // 完成时间（可选）
}

// MarkComplete 标记为完成
func (t *TodoItem) MarkComplete() {
	t.Completed = true
	now := time.Now()
	t.CompletedAt = &now
}

// MarkIncomplete 标记为未完成
func (t *TodoItem) MarkIncomplete() {
	t.Completed = false
	t.CompletedAt = nil
}

// Toggle 切换完成状态
func (t *TodoItem) Toggle() {
	if t.Completed {
		t.MarkIncomplete()
	} else {
		t.MarkComplete()
	}
}
