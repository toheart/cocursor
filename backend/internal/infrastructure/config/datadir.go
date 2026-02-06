package config

import (
	"os"
	"path/filepath"
	"sync"
)

const (
	// EnvDataDir 数据目录环境变量名
	EnvDataDir = "COCURSOR_DATA_DIR"
	// DefaultDataDirName 默认数据目录名
	DefaultDataDirName = ".cocursor"
)

var (
	dataDirOnce sync.Once
	dataDirPath string
)

// GetDataDir 获取 CoCursor 数据根目录
// 优先读取 COCURSOR_DATA_DIR 环境变量，默认 ~/.cocursor/
// 此函数是所有 cocursor 数据路径的唯一入口，禁止直接拼接 homeDir + ".cocursor"
func GetDataDir() string {
	dataDirOnce.Do(func() {
		if dir := os.Getenv(EnvDataDir); dir != "" {
			dataDirPath = dir
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				// 回退到当前目录
				dataDirPath = DefaultDataDirName
				return
			}
			dataDirPath = filepath.Join(homeDir, DefaultDataDirName)
		}
	})
	return dataDirPath
}

// ResetDataDir 重置数据目录缓存（仅用于测试）
func ResetDataDir() {
	dataDirOnce = sync.Once{}
	dataDirPath = ""
}
