package notification

import "errors"

var (
	// ErrInvalidTeamCode 无效的团队码
	ErrInvalidTeamCode = errors.New("invalid team code")
	// ErrInvalidTitle 无效的标题
	ErrInvalidTitle = errors.New("invalid title")
)

// Service 领域服务（纯业务逻辑）
type Service struct {
	// 不依赖任何基础设施，只依赖领域概念
}

// NewService 创建领域服务
func NewService() *Service {
	return &Service{}
}

// Validate 验证通知内容（领域规则）
func (s *Service) Validate(n *Notification) error {
	if n.TeamCode == "" {
		return ErrInvalidTeamCode
	}
	if n.Title == "" {
		return ErrInvalidTitle
	}
	return nil
}

// CalculatePriority 计算通知优先级（领域逻辑）
func (s *Service) CalculatePriority(n *Notification) int {
	switch n.Type {
	case TypeError:
		return 3
	case TypeWarning:
		return 2
	default:
		return 1
	}
}
