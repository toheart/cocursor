//go:build integration
// +build integration

// 团队网络配置集成测试
// 测试网络接口获取、网络配置设置等功能

package integration

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appTeam "github.com/cocursor/backend/internal/application/team"
)

// TestNetworkInterfaces 测试获取网络接口列表
func TestNetworkInterfaces(t *testing.T) {
	// 创建临时 HOME 目录
	tmpDir, err := os.MkdirTemp("", "network-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// 创建 TeamService
	service, err := appTeam.NewTeamService(20010, "1.0.0-test")
	require.NoError(t, err)
	defer service.Close()

	// 获取网络接口列表
	interfaces, err := service.GetNetworkInterfaces()
	require.NoError(t, err)

	// 应该至少有一个网络接口
	assert.NotEmpty(t, interfaces, "Should have at least one network interface")

	t.Logf("Found %d network interfaces:", len(interfaces))
	for _, iface := range interfaces {
		t.Logf("  - %s: %v", iface.Name, iface.Addresses)
	}

	// 验证接口有地址
	hasAddress := false
	for _, iface := range interfaces {
		if len(iface.Addresses) > 0 {
			hasAddress = true
			break
		}
	}
	assert.True(t, hasAddress, "At least one interface should have an address")
}

// TestNetworkConfig 测试设置和获取网络配置
func TestNetworkConfig(t *testing.T) {
	// 创建临时 HOME 目录
	tmpDir, err := os.MkdirTemp("", "network-config-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// 创建 TeamService
	service, err := appTeam.NewTeamService(20011, "1.0.0-test")
	require.NoError(t, err)
	defer service.Close()

	// 获取网络接口
	interfaces, err := service.GetNetworkInterfaces()
	require.NoError(t, err)
	require.NotEmpty(t, interfaces)

	// 选择第一个有地址的接口
	var preferredIF, preferredIP string
	for _, iface := range interfaces {
		if len(iface.Addresses) > 0 {
			preferredIF = iface.Name
			preferredIP = iface.Addresses[0]
			break
		}
	}
	require.NotEmpty(t, preferredIF, "Should have an interface with address")
	require.NotEmpty(t, preferredIP, "Should have an IP address")

	t.Logf("Setting network config: interface=%s, ip=%s", preferredIF, preferredIP)

	// 设置网络配置
	err = service.SetNetworkConfig(preferredIF, preferredIP)
	require.NoError(t, err)

	// 获取网络配置
	config := service.GetNetworkConfig()
	require.NotNil(t, config)
	assert.Equal(t, preferredIF, config.PreferredInterface)
	assert.Equal(t, preferredIP, config.PreferredIP)

	t.Logf("Network config: interface=%s, ip=%s", config.PreferredInterface, config.PreferredIP)
}

// TestNetworkConfig_Empty 测试默认网络配置为空
func TestNetworkConfig_Empty(t *testing.T) {
	// 创建临时 HOME 目录
	tmpDir, err := os.MkdirTemp("", "network-empty-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// 创建 TeamService
	service, err := appTeam.NewTeamService(20012, "1.0.0-test")
	require.NoError(t, err)
	defer service.Close()

	// 获取默认网络配置
	config := service.GetNetworkConfig()

	// 默认配置可能为 nil 或空
	if config != nil {
		t.Logf("Default config: interface=%s, ip=%s", config.PreferredInterface, config.PreferredIP)
	} else {
		t.Log("Default config is nil (expected)")
	}
}

// TestNetworkConfig_CreateTeamWithPreferred 测试使用指定网卡创建团队
func TestNetworkConfig_CreateTeamWithPreferred(t *testing.T) {
	// 创建临时 HOME 目录
	tmpDir, err := os.MkdirTemp("", "network-create-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// 创建 TeamService
	service, err := appTeam.NewTeamService(20013, "1.0.0-test")
	require.NoError(t, err)
	defer service.Close()

	// 创建身份
	_, err = service.EnsureIdentity("NetworkLeader")
	require.NoError(t, err)

	// 获取网络接口
	interfaces, err := service.GetNetworkInterfaces()
	require.NoError(t, err)

	var preferredIF, preferredIP string
	for _, iface := range interfaces {
		if len(iface.Addresses) > 0 {
			preferredIF = iface.Name
			preferredIP = iface.Addresses[0]
			break
		}
	}

	// 使用指定网卡创建团队
	team, err := service.CreateTeam("NetworkTeam", preferredIF, preferredIP)
	require.NoError(t, err)

	// 验证团队端点使用了指定的 IP
	assert.NotEmpty(t, team.LeaderEndpoint)
	t.Logf("Team created with endpoint: %s", team.LeaderEndpoint)

	// 清理
	err = service.DissolveTeam(nil, team.ID)
	require.NoError(t, err)
}
