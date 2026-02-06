package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig_DefaultPorts(t *testing.T) {
	t.Setenv(EnvHTTPPort, "")
	t.Setenv(EnvMCPPort, "")

	cfg := NewConfig()
	assert.Equal(t, ":19960", cfg.Server.HTTPPort)
	assert.Equal(t, ":19961", cfg.Server.MCPPort)
}

func TestNewConfig_EnvOverridePorts(t *testing.T) {
	t.Setenv(EnvHTTPPort, ":29960")
	t.Setenv(EnvMCPPort, ":29961")

	cfg := NewConfig()
	assert.Equal(t, ":29960", cfg.Server.HTTPPort)
	assert.Equal(t, ":29961", cfg.Server.MCPPort)
}

func TestNewConfig_PartialEnvOverride(t *testing.T) {
	t.Setenv(EnvHTTPPort, ":30000")
	t.Setenv(EnvMCPPort, "")

	cfg := NewConfig()
	assert.Equal(t, ":30000", cfg.Server.HTTPPort)
	assert.Equal(t, ":19961", cfg.Server.MCPPort, "未设置的端口应使用默认值")
}
