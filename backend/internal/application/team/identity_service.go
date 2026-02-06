package team

import (
	"log/slog"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/infrastructure/log"
	infraTeam "github.com/cocursor/backend/internal/infrastructure/team"
)

// IdentityService 身份服务
// 负责管理本机身份信息
type IdentityService struct {
	store  *infraTeam.IdentityStore
	logger *slog.Logger
}

// NewIdentityService 创建身份服务
func NewIdentityService() (*IdentityService, error) {
	store, err := infraTeam.NewIdentityStore()
	if err != nil {
		return nil, err
	}

	return &IdentityService{
		store:  store,
		logger: log.NewModuleLogger("team", "identity"),
	}, nil
}

// NewIdentityServiceWithStore 使用指定存储创建身份服务（用于测试）
func NewIdentityServiceWithStore(store *infraTeam.IdentityStore) *IdentityService {
	return &IdentityService{
		store:  store,
		logger: log.NewModuleLogger("team", "identity"),
	}
}

// GetIdentity 获取本机身份
func (s *IdentityService) GetIdentity() (*domainTeam.Identity, error) {
	return s.store.Get()
}

// CreateIdentity 创建本机身份
func (s *IdentityService) CreateIdentity(name string) (*domainTeam.Identity, error) {
	identity, err := s.store.Create(name)
	if err != nil {
		return nil, err
	}

	s.logger.Info("identity created",
		"id", identity.ID,
		"name", identity.Name,
	)

	return identity, nil
}

// UpdateIdentity 更新本机身份
func (s *IdentityService) UpdateIdentity(name string) (*domainTeam.Identity, error) {
	identity, err := s.store.UpdateName(name)
	if err != nil {
		return nil, err
	}

	s.logger.Info("identity updated",
		"id", identity.ID,
		"name", identity.Name,
	)

	return identity, nil
}

// EnsureIdentity 确保身份存在，不存在则创建，存在则更新名称
func (s *IdentityService) EnsureIdentity(name string) (*domainTeam.Identity, error) {
	identity, err := s.store.Get()
	if err == nil {
		// 身份已存在，检查名称是否需要更新
		if identity.Name != name {
			return s.store.UpdateName(name)
		}
		return identity, nil
	}

	if err == domainTeam.ErrIdentityNotFound {
		return s.store.Create(name)
	}

	return nil, err
}

// 确保 IdentityService 实现 IdentityProvider 接口
var _ IdentityProvider = (*IdentityService)(nil)
