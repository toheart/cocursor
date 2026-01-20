package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cocursor/backend/internal/domain/team"
)

func TestNetworkManager_GetAvailableInterfaces(t *testing.T) {
	mgr := NewNetworkManager()

	interfaces, err := mgr.GetAvailableInterfaces()
	require.NoError(t, err)

	// 应该至少有一个接口（除非在特殊环境）
	t.Logf("Found %d interfaces", len(interfaces))

	for _, iface := range interfaces {
		t.Logf("Interface: %s, Addresses: %v", iface.Name, iface.Addresses)
		
		// 验证基本属性
		assert.NotEmpty(t, iface.Name)
		assert.True(t, iface.IsUp)
		assert.False(t, iface.IsLoopback)
		
		// 验证地址是私有地址
		for _, addr := range iface.Addresses {
			assert.NotEmpty(t, addr)
			// 不应包含回环地址
			assert.NotEqual(t, "127.0.0.1", addr)
		}
	}
}

func TestNetworkManager_GetAllIPs(t *testing.T) {
	mgr := NewNetworkManager()

	ips, err := mgr.GetAllIPs()
	require.NoError(t, err)

	t.Logf("Found %d IPs: %v", len(ips), ips)

	for _, ip := range ips {
		assert.NotEmpty(t, ip)
		assert.NotEqual(t, "127.0.0.1", ip)
	}
}

func TestNetworkManager_BuildMemberEndpoint(t *testing.T) {
	mgr := NewNetworkManager()

	endpoint, err := mgr.BuildMemberEndpoint(nil, 19960)
	// 可能在某些环境（如 CI）没有有效接口
	if err == team.ErrNoValidInterface {
		t.Skip("No valid network interface available")
	}
	require.NoError(t, err)
	require.NotNil(t, endpoint)

	assert.NotEmpty(t, endpoint.PrimaryIP)
	assert.Equal(t, 19960, endpoint.Port)
	assert.NotEmpty(t, endpoint.AllIPs)
	
	// 验证 GetAddress
	addr := endpoint.GetAddress()
	assert.Contains(t, addr, ":19960")

	// 验证 GetAllAddresses
	allAddrs := endpoint.GetAllAddresses()
	assert.GreaterOrEqual(t, len(allAddrs), 1)
}

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantHost string
		wantPort int
		wantErr  bool
	}{
		{
			name:     "valid endpoint",
			endpoint: "192.168.1.100:19960",
			wantHost: "192.168.1.100",
			wantPort: 19960,
			wantErr:  false,
		},
		{
			name:     "localhost",
			endpoint: "127.0.0.1:8080",
			wantHost: "127.0.0.1",
			wantPort: 8080,
			wantErr:  false,
		},
		{
			name:     "invalid no port",
			endpoint: "192.168.1.100",
			wantErr:  true,
		},
		{
			name:     "invalid empty",
			endpoint: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := ParseEndpoint(tt.endpoint)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantHost, host)
				assert.Equal(t, tt.wantPort, port)
			}
		})
	}
}

func TestMemberEndpoint_Methods(t *testing.T) {
	endpoint := &team.MemberEndpoint{
		PrimaryIP: "192.168.1.100",
		AllIPs:    []string{"192.168.1.100", "10.0.0.50"},
		Port:      19960,
	}

	// 测试 GetAddress
	assert.Equal(t, "192.168.1.100:19960", endpoint.GetAddress())

	// 测试 GetAllAddresses
	allAddrs := endpoint.GetAllAddresses()
	assert.Len(t, allAddrs, 2)
	assert.Contains(t, allAddrs, "192.168.1.100:19960")
	assert.Contains(t, allAddrs, "10.0.0.50:19960")
}
