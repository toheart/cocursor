package p2p

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/cocursor/backend/internal/domain/team"
)

// 虚拟网卡名称前缀列表
var virtualInterfacePrefixes = []string{
	"vmnet",     // VMware
	"vboxnet",   // VirtualBox
	"veth",      // Docker/容器
	"docker",    // Docker
	"br-",       // Docker bridge
	"virbr",     // libvirt/KVM
	"lxc",       // LXC
	"lxd",       // LXD
	"flannel",   // Kubernetes flannel
	"cni",       // Kubernetes CNI
	"calico",    // Kubernetes calico
	"weave",     // Kubernetes weave
	"tun",       // VPN tunnel
	"tap",       // VPN tap
	"utun",      // macOS VPN
	"awdl",      // Apple Wireless Direct Link
	"llw",       // Low Latency WLAN
	"bridge",    // Bridge
	"Parallels", // Parallels Desktop
}

// NetworkManager 网络管理器
type NetworkManager struct{}

// NewNetworkManager 创建网络管理器
func NewNetworkManager() *NetworkManager {
	return &NetworkManager{}
}

// GetAvailableInterfaces 获取可用的网络接口
// 过滤规则：
// - 排除未启用的接口
// - 排除回环接口（127.x.x.x）
// - 排除链路本地地址（169.254.x.x）
// - 只保留私有地址段（10.x、172.16-31.x、192.168.x）
// - 标记虚拟网卡（VMware、VirtualBox、Docker等）
func (m *NetworkManager) GetAvailableInterfaces() ([]team.NetworkInterface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	var result []team.NetworkInterface
	for _, iface := range interfaces {
		// 跳过未启用的接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		// 跳过回环接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		var ipv4Addrs []string
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip4 := ipnet.IP.To4()
			if ip4 == nil {
				continue
			}

			// 过滤特殊地址
			if !isValidLANAddress(ip4) {
				continue
			}

			ipv4Addrs = append(ipv4Addrs, ip4.String())
		}

		if len(ipv4Addrs) > 0 {
			result = append(result, team.NetworkInterface{
				Name:       iface.Name,
				Addresses:  ipv4Addrs,
				IsUp:       true,
				IsLoopback: false,
				IsVirtual:  isVirtualInterface(iface.Name),
			})
		}
	}

	// 排序：物理网卡在前，虚拟网卡在后；同类按名称排序
	sort.Slice(result, func(i, j int) bool {
		// 优先按虚拟/物理分类
		if result[i].IsVirtual != result[j].IsVirtual {
			return !result[i].IsVirtual // 物理网卡在前
		}
		// 同类按名称排序
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// isValidLANAddress 判断是否为有效的局域网地址
func isValidLANAddress(ip net.IP) bool {
	// 排除回环地址
	if ip.IsLoopback() {
		return false
	}

	// 排除链路本地地址 (169.254.x.x)
	if ip[0] == 169 && ip[1] == 254 {
		return false
	}

	// 只保留私有地址段
	return ip.IsPrivate()
}

// isVirtualInterface 判断是否为虚拟网卡
func isVirtualInterface(name string) bool {
	lowerName := strings.ToLower(name)
	for _, prefix := range virtualInterfacePrefixes {
		if strings.HasPrefix(lowerName, strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}

// SelectMatchingLocalIP 选择与目标地址同网段的本地 IP
// 如果没有找到同网段的，返回第一个可用地址
func (m *NetworkManager) SelectMatchingLocalIP(targetIP string) (string, error) {
	target := net.ParseIP(targetIP)
	if target == nil {
		return "", fmt.Errorf("invalid target IP: %s", targetIP)
	}

	interfaces, err := m.GetAvailableInterfaces()
	if err != nil {
		return "", err
	}

	// 首先尝试找同网段的 IP
	for _, iface := range interfaces {
		for _, addr := range iface.Addresses {
			localIP := net.ParseIP(addr)
			if localIP == nil {
				continue
			}
			if isSameSubnet(localIP.To4(), target.To4()) {
				return addr, nil
			}
		}
	}

	// 没有匹配的网段，返回第一个可用地址
	if len(interfaces) > 0 && len(interfaces[0].Addresses) > 0 {
		return interfaces[0].Addresses[0], nil
	}

	return "", team.ErrNoValidInterface
}

// isSameSubnet 判断是否同网段（简化版，假设 /24）
// 比较前 3 个字节
func isSameSubnet(a, b net.IP) bool {
	if a == nil || b == nil {
		return false
	}
	a4 := a.To4()
	b4 := b.To4()
	if a4 == nil || b4 == nil {
		return false
	}
	return a4[0] == b4[0] && a4[1] == b4[1] && a4[2] == b4[2]
}

// GetPrimaryIP 获取主要 IP 地址
// 优先使用配置的偏好，否则返回第一个可用地址
func (m *NetworkManager) GetPrimaryIP(config *team.NetworkConfig) (string, error) {
	// 如果有配置的偏好 IP，检查是否仍然有效
	if config != nil && config.PreferredIP != "" {
		interfaces, err := m.GetAvailableInterfaces()
		if err == nil {
			for _, iface := range interfaces {
				for _, addr := range iface.Addresses {
					if addr == config.PreferredIP {
						return addr, nil
					}
				}
			}
		}
	}

	// 没有偏好或偏好已失效，返回第一个可用地址
	interfaces, err := m.GetAvailableInterfaces()
	if err != nil {
		return "", err
	}

	if len(interfaces) > 0 && len(interfaces[0].Addresses) > 0 {
		return interfaces[0].Addresses[0], nil
	}

	return "", team.ErrNoValidInterface
}

// GetAllIPs 获取所有可用 IP 地址
func (m *NetworkManager) GetAllIPs() ([]string, error) {
	interfaces, err := m.GetAvailableInterfaces()
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, iface := range interfaces {
		ips = append(ips, iface.Addresses...)
	}

	return ips, nil
}

// BuildMemberEndpoint 构建成员端点
func (m *NetworkManager) BuildMemberEndpoint(config *team.NetworkConfig, port int) (*team.MemberEndpoint, error) {
	primaryIP, err := m.GetPrimaryIP(config)
	if err != nil {
		return nil, err
	}

	allIPs, err := m.GetAllIPs()
	if err != nil {
		return nil, err
	}

	preferredIF := ""
	if config != nil {
		preferredIF = config.PreferredInterface
	}

	return &team.MemberEndpoint{
		PrimaryIP:   primaryIP,
		AllIPs:      allIPs,
		Port:        port,
		PreferredIF: preferredIF,
	}, nil
}

// ConnectToEndpoint 尝试连接端点，支持多地址回退
func (m *NetworkManager) ConnectToEndpoint(endpoint *team.MemberEndpoint, timeout time.Duration) (net.Conn, error) {
	addresses := endpoint.GetAllAddresses()
	if len(addresses) == 0 {
		addresses = []string{endpoint.GetAddress()}
	}

	var lastErr error
	for _, addr := range addresses {
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all addresses failed, last error: %w", lastErr)
}

// CheckPortAvailable 检查端口是否可用
func (m *NetworkManager) CheckPortAvailable(port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return team.ErrPortInUse
	}
	_ = listener.Close()
	return nil
}

// ParseEndpoint 解析端点字符串（IP:Port）
func ParseEndpoint(endpoint string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(endpoint)
	if err != nil {
		return "", 0, fmt.Errorf("invalid endpoint format: %w", err)
	}

	var port int
	_, err = fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %w", err)
	}

	return host, port, nil
}
