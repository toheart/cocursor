package notification

// CreateNotificationDTO 创建通知请求
type CreateNotificationDTO struct {
	TeamCode string `json:"teamCode" binding:"required"`
	Title    string `json:"title" binding:"required"`
	Message  string `json:"message" binding:"required"`
	Type     int    `json:"type"`
}

// NotificationDTO 通知响应
type NotificationDTO struct {
	ID        string `json:"id"`
	TeamCode  string `json:"teamCode"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Type      int    `json:"type"`
	CreatedAt string `json:"createdAt"`
}
