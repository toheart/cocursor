package p2p

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/grandcat/zeroconf"

	"github.com/cocursor/backend/internal/domain/p2p"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// MDNSDiscovery mDNS 服务发现器
type MDNSDiscovery struct {
	logger *slog.Logger
}

// NewMDNSDiscovery 创建 mDNS 发现器
func NewMDNSDiscovery() *MDNSDiscovery {
	return &MDNSDiscovery{
		logger: log.NewModuleLogger("p2p", "mdns_discovery"),
	}
}

// Discover 发现服务
func (d *MDNSDiscovery) Discover(ctx context.Context, timeout time.Duration) ([]p2p.ServiceInfo, error) {
	// 创建浏览器
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver: %w", err)
	}

	entries := make(chan *zeroconf.ServiceEntry, 10)
	var services []p2p.ServiceInfo

	// 在后台收集结果
	go func() {
		for entry := range entries {
			service := d.parseServiceEntry(entry)
			if service != nil {
				services = append(services, *service)
			}
		}
	}()

	// 创建超时上下文
	browseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 开始浏览
	// 服务类型：_cocursor._tcp
	err = resolver.Browse(browseCtx, "_cocursor._tcp", "local.", entries)
	if err != nil {
		return nil, fmt.Errorf("failed to browse services: %w", err)
	}

	// 等待超时
	<-browseCtx.Done()

	d.logger.Debug("mDNS discovery completed",
		"count", len(services),
	)

	return services, nil
}

// parseServiceEntry 解析服务条目
func (d *MDNSDiscovery) parseServiceEntry(entry *zeroconf.ServiceEntry) *p2p.ServiceInfo {
	if entry == nil {
		return nil
	}

	// 提取 IP 地址
	var ips []string
	for _, ip := range entry.AddrIPv4 {
		ips = append(ips, ip.String())
	}

	// 如果没有 IPv4 地址，跳过
	if len(ips) == 0 {
		d.logger.Debug("skipping service without IPv4 address",
			"instance", entry.Instance,
		)
		return nil
	}

	// 解析 TXT 记录
	txtRecords := make(map[string]string)
	for _, txt := range entry.Text {
		key, value := parseTxtRecord(txt)
		if key != "" {
			txtRecords[key] = value
		}
	}

	return &p2p.ServiceInfo{
		InstanceName: entry.Instance,
		ServiceType:  entry.Service,
		Domain:       entry.Domain,
		HostName:     entry.HostName,
		Port:         entry.Port,
		IPs:          ips,
		TxtRecords:   txtRecords,
	}
}

// parseTxtRecord 解析 TXT 记录（格式：key=value）
func parseTxtRecord(txt string) (string, string) {
	for i := 0; i < len(txt); i++ {
		if txt[i] == '=' {
			return txt[:i], txt[i+1:]
		}
	}
	return txt, ""
}

// Close 关闭发现器
func (d *MDNSDiscovery) Close() error {
	return nil
}

// DiscoverTeams 发现团队（便捷方法）
func (d *MDNSDiscovery) DiscoverTeams(ctx context.Context, timeout time.Duration) ([]DiscoveredTeamInfo, error) {
	services, err := d.Discover(ctx, timeout)
	if err != nil {
		return nil, err
	}

	var teams []DiscoveredTeamInfo
	for _, svc := range services {
		team := DiscoveredTeamInfo{
			TeamID:      svc.GetTeamID(),
			Name:        svc.InstanceName,
			LeaderName:  svc.GetLeaderName(),
			Version:     svc.GetVersion(),
		}

		// 构建端点
		if len(svc.IPs) > 0 {
			team.Endpoint = fmt.Sprintf("%s:%d", svc.IPs[0], svc.Port)
		}

		// 解析成员数量
		if countStr := svc.GetMemberCount(); countStr != "" {
			count, err := strconv.Atoi(countStr)
			if err == nil {
				team.MemberCount = count
			}
		}

		if team.TeamID != "" {
			teams = append(teams, team)
		}
	}

	return teams, nil
}

// DiscoveredTeamInfo 发现的团队信息
type DiscoveredTeamInfo struct {
	TeamID      string `json:"team_id"`
	Name        string `json:"name"`
	LeaderName  string `json:"leader_name"`
	Endpoint    string `json:"endpoint"`
	MemberCount int    `json:"member_count"`
	Version     string `json:"version"`
}
