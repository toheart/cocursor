package p2p

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"sync"

	"github.com/grandcat/zeroconf"

	"github.com/cocursor/backend/internal/domain/p2p"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// MDNSAdvertiser mDNS 服务广播器
type MDNSAdvertiser struct {
	mu       sync.RWMutex
	server   *zeroconf.Server
	info     *p2p.ServiceInfo
	running  bool
	logger   *slog.Logger
	netMgr   *NetworkManager
}

// NewMDNSAdvertiser 创建 mDNS 广播器
func NewMDNSAdvertiser(netMgr *NetworkManager) *MDNSAdvertiser {
	return &MDNSAdvertiser{
		logger: log.NewModuleLogger("p2p", "mdns_advertiser"),
		netMgr: netMgr,
	}
}

// Start 开始广播服务
func (a *MDNSAdvertiser) Start(info p2p.ServiceInfo) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return fmt.Errorf("advertiser is already running")
	}

	// 构建 TXT 记录
	var txtRecords []string
	for k, v := range info.TxtRecords {
		txtRecords = append(txtRecords, fmt.Sprintf("%s=%s", k, v))
	}

	// 获取所有可用接口的 IP
	interfaces, err := a.netMgr.GetAvailableInterfaces()
	if err != nil {
		return fmt.Errorf("failed to get interfaces: %w", err)
	}

	var ips []string
	var ifaces []net.Interface
	for _, iface := range interfaces {
		ips = append(ips, iface.Addresses...)
		
		// 获取系统接口用于广播
		sysIface, err := net.InterfaceByName(iface.Name)
		if err == nil {
			ifaces = append(ifaces, *sysIface)
		}
	}

	if len(ips) == 0 {
		return fmt.Errorf("no available IP addresses")
	}

	a.logger.Info("starting mDNS advertiser",
		"instance", info.InstanceName,
		"port", info.Port,
		"ips", ips,
		"txt_records", txtRecords,
	)

	// 创建服务器
	// 服务类型：_cocursor._tcp
	server, err := zeroconf.RegisterProxy(
		info.InstanceName,          // 实例名称（团队名称）
		"_cocursor._tcp",           // 服务类型
		"local.",                   // 域
		info.Port,                  // 端口
		info.InstanceName,          // 主机名（使用实例名称）
		ips,                        // IP 地址
		txtRecords,                 // TXT 记录
		ifaces,                     // 网络接口
	)
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	a.server = server
	a.info = &info
	a.running = true

	a.logger.Info("mDNS advertiser started",
		"team_id", info.TxtRecords["team_id"],
		"name", info.InstanceName,
	)

	return nil
}

// Stop 停止广播
func (a *MDNSAdvertiser) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	if a.server != nil {
		a.server.Shutdown()
		a.server = nil
	}

	a.running = false
	a.info = nil

	a.logger.Info("mDNS advertiser stopped")

	return nil
}

// UpdateTxtRecords 更新 TXT 记录
// 注意：zeroconf 库不支持动态更新，需要重启服务
func (a *MDNSAdvertiser) UpdateTxtRecords(records map[string]string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running || a.info == nil {
		return fmt.Errorf("advertiser is not running")
	}

	// 更新 info 中的记录
	for k, v := range records {
		a.info.TxtRecords[k] = v
	}

	// 重启服务以应用更新
	info := *a.info

	// 先停止
	if a.server != nil {
		a.server.Shutdown()
		a.server = nil
	}
	a.running = false

	// 重新启动（不持有锁）
	a.mu.Unlock()
	err := a.Start(info)
	a.mu.Lock()

	return err
}

// IsRunning 是否正在广播
func (a *MDNSAdvertiser) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

// GetInfo 获取当前广播的服务信息
func (a *MDNSAdvertiser) GetInfo() *p2p.ServiceInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.info == nil {
		return nil
	}
	infoCopy := *a.info
	return &infoCopy
}

// BuildServiceInfo 构建服务信息
func BuildServiceInfo(teamID, teamName, leaderName string, port, memberCount int, version string) p2p.ServiceInfo {
	return p2p.ServiceInfo{
		InstanceName: teamName,
		ServiceType:  "_cocursor._tcp",
		Domain:       "local.",
		Port:         port,
		TxtRecords: map[string]string{
			"team_id":      teamID,
			"leader_name":  leaderName,
			"member_count": strconv.Itoa(memberCount),
			"version":      version,
		},
	}
}
