package notification

import (
	"time"

	"github.com/cocursor/backend/internal/domain/notification"
	"github.com/google/uuid"
)

// Service 应用服务（用例编排）
type Service struct {
	domainRepo notification.Repository
	domainSvc  *notification.Service
	pusher     Pusher
}

// NewService 创建应用服务
func NewService(
	domainRepo notification.Repository,
	domainSvc *notification.Service,
	pusher Pusher,
) *Service {
	return &Service{
		domainRepo: domainRepo,
		domainSvc:  domainSvc,
		pusher:     pusher,
	}
}

// CreateAndPush 创建并推送通知（用例）
func (s *Service) CreateAndPush(dto *CreateNotificationDTO) (*NotificationDTO, error) {
	// 1. 创建领域实体
	notif := &notification.Notification{
		ID:        uuid.New().String(),
		TeamCode:  dto.TeamCode,
		Title:     dto.Title,
		Message:   dto.Message,
		Type:      notification.Type(dto.Type),
		CreatedAt: time.Now(),
	}

	// 2. 使用领域服务验证
	if err := s.domainSvc.Validate(notif); err != nil {
		return nil, err
	}

	// 3. 保存到仓储
	if err := s.domainRepo.Save(notif); err != nil {
		return nil, err
	}

	// 4. 推送到团队（技术能力）
	if err := s.pusher.PushToTeam(dto.TeamCode, notif); err != nil {
		// 推送失败不影响保存
	}

	// 5. 返回 DTO
	return toDTO(notif), nil
}

// toDTO 转换为 DTO
func toDTO(n *notification.Notification) *NotificationDTO {
	return &NotificationDTO{
		ID:        n.ID,
		TeamCode:  n.TeamCode,
		Title:     n.Title,
		Message:   n.Message,
		Type:      int(n.Type),
		CreatedAt: n.CreatedAt.Format(time.RFC3339),
	}
}
