package log

import (
	"os"
	"strconv"
	"strings"
)

// Config 日志配置
type Config struct {
	// Level 日志级别：debug, info, warn, error
	Level string `json:"level" env:"LOG_LEVEL"`

	// Format 日志格式：console, json
	Format string `json:"format" env:"LOG_FORMAT"`

	// Output 输出目标：stdout, file:/path/to/log
	Output string `json:"output" env:"LOG_OUTPUT"`

	// AddSource 是否添加源文件信息（开发环境）
	AddSource bool `json:"add_source" env:"LOG_ADD_SOURCE"`

	// AddCaller 是否添加调用者信息
	AddCaller bool `json:"add_caller" env:"LOG_ADD_CALLER"`
}

// NewConfigFromEnv 从环境变量创建配置
func NewConfigFromEnv() *Config {
	cfg := &Config{
		Level:     getEnvWithDefault("LOG_LEVEL", "info"),
		Format:    getEnvWithDefault("LOG_FORMAT", "console"),
		Output:    getEnvWithDefault("LOG_OUTPUT", "stdout"),
		AddSource: getEnvBool("LOG_ADD_SOURCE", false),
		AddCaller: getEnvBool("LOG_ADD_CALLER", false),
	}

	// 在开发环境自动设置
	if cfg.isDevelopment() {
		cfg.Level = "debug"
		cfg.Format = "console"
		cfg.AddSource = true
	}

	return cfg
}

// isDevelopment 检查是否为开发环境
func (c *Config) isDevelopment() bool {
	env := getEnvWithDefault("ENV", "production")
	return strings.ToLower(env) == "development"
}

// getEnvWithDefault 获取环境变量，带默认值
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvBool 获取布尔型环境变量
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
}
