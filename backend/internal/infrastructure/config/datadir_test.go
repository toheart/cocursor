package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDataDir_Default(t *testing.T) {
	// 确保环境变量未设置
	ResetDataDir()
	os.Unsetenv(EnvDataDir)

	dir := GetDataDir()

	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(homeDir, ".cocursor"), dir)
}

func TestGetDataDir_EnvOverride(t *testing.T) {
	ResetDataDir()
	t.Setenv(EnvDataDir, "/custom/data/path")

	dir := GetDataDir()
	assert.Equal(t, "/custom/data/path", dir)
}

func TestGetDataDir_Cached(t *testing.T) {
	// 第一次调用设置环境变量
	ResetDataDir()
	t.Setenv(EnvDataDir, "/first/path")
	dir1 := GetDataDir()
	assert.Equal(t, "/first/path", dir1)

	// 修改环境变量后再调用，应该返回缓存值
	os.Setenv(EnvDataDir, "/second/path")
	dir2 := GetDataDir()
	assert.Equal(t, "/first/path", dir2, "应该返回缓存值，不受环境变量修改影响")
}

func TestResetDataDir(t *testing.T) {
	ResetDataDir()
	t.Setenv(EnvDataDir, "/path/a")
	assert.Equal(t, "/path/a", GetDataDir())

	// 重置后重新读取
	ResetDataDir()
	t.Setenv(EnvDataDir, "/path/b")
	assert.Equal(t, "/path/b", GetDataDir())
}
