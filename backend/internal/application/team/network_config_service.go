package team

import (
	"log/slog"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/infrastructure/log"
	infraP2P "github.com/cocursor/backend/internal/infrastructure/p2p"
	infraTeam "github.com/cocursor/backend/internal/infrastructure/team"
)

// NetworkConfigService 网络配置服务
// 负责管理网络配置和端点信息
type NetworkConfigService struct {
	configStore    *infraTeam.NetworkConfigStore
	networkManager *infraP2P.NetworkManager
	port           int
	logger         *slog.Logger
}

// NewNetworkConfigService 创建网络配置服务
func NewNetworkConfigService(port int) (*NetworkConfigService, error) {
	configStore, err := infraTeam.NewNetworkConfigStore()
	if err != nil {
		return nil, err
	}

	return &NetworkConfigService{
		configStore:    configStore,
		networkManager: infraP2P.NewNetworkManager(),
		port:           port,
		logger:         log.NewModuleLogger("team", "network_config"),
	}, nil
}

// NewNetworkConfigServiceWithDeps 使用指定依赖创建网络配置服务（用于测试）
func NewNetworkConfigServiceWithDeps(
	configStore *infraTeam.NetworkConfigStore,
	networkManager *infraP2P.NetworkManager,
	port int,
) *NetworkConfigService {
	return &NetworkConfigService{
		configStore:    configStore,
		networkManager: networkManager,
		port:           port,
		logger:         log.NewModuleLogger("team", "network_config"),
	}
}

// GetNetworkInterfaces 获取可用网卡列表
func (s *NetworkConfigService) GetNetworkInterfaces() ([]domainTeam.NetworkInterface, error) {
	return s.networkManager.GetAvailableInterfaces()
}

// GetNetworkConfig 获取网卡配置
func (s *NetworkConfigService) GetNetworkConfig() *domainTeam.NetworkConfig {
	return s.configStore.Get()
}

// SetNetworkConfig 设置网卡配置
func (s *NetworkConfigService) SetNetworkConfig(preferredInterface, preferredIP string) error {
	err := s.configStore.Set(preferredInterface, preferredIP)
	if err != nil {
		return err
	}

	s.logger.Info("network config updated",
		"preferred_interface", preferredInterface,
		"preferred_ip", preferredIP,
	)

	return nil
}

// GetCurrentEndpoint 获取当前端点地址
func (s *NetworkConfigService) GetCurrentEndpoint() string {
	endpoint, err := s.networkManager.BuildMemberEndpoint(s.configStore.Get(), s.port)
	if err != nil {
		s.logger.Warn("failed to build member endpoint", "error", err)
		return ""
	}
	return endpoint.GetAddress()
}

// BuildMemberEndpoint 构建成员端点
func (s *NetworkConfigService) BuildMemberEndpoint() (*domainTeam.MemberEndpoint, error) {
	return s.networkManager.BuildMemberEndpoint(s.configStore.Get(), s.port)
}

// SelectMatchingLocalIP 选择匹配的本地 IP
func (s *NetworkConfigService) SelectMatchingLocalIP(targetHost string) (string, error) {
	return s.networkManager.SelectMatchingLocalIP(targetHost)
}

// GetPort 获取端口
func (s *NetworkConfigService) GetPort() int {
	return s.port
}

// 确保 NetworkConfigService 实现 NetworkConfigProvider 接口
var _ NetworkConfigProvider = (*NetworkConfigService)(nil)
