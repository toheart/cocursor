package todo

// Repository 待办事项仓储接口
type Repository interface {
	// Save 保存待办事项（创建或更新）
	Save(item *TodoItem) error

	// FindByID 根据 ID 查找待办事项
	FindByID(id string) (*TodoItem, error)

	// FindAll 获取所有待办事项
	FindAll() ([]*TodoItem, error)

	// Delete 删除待办事项
	Delete(id string) error

	// DeleteCompleted 删除所有已完成的待办事项
	DeleteCompleted() (int64, error)

	// FindCompletedByDateRange 查询日期范围内完成的待办
	// startTime 和 endTime 为 Unix 毫秒时间戳
	FindCompletedByDateRange(startTime, endTime int64) ([]*TodoItem, error)
}
