package notification

// Repository 通知仓储接口
type Repository interface {
	Save(notification *Notification) error
	FindByTeamCode(teamCode string) ([]*Notification, error)
}
