package log

import (
	"log/slog"
	"os"
	"strings"
)

// 全局 logger 实例
var (
	defaultLogger *slog.Logger
	debugMode     bool
)

// Init 初始化日志系统
func Init(cfg *Config) {
	if cfg == nil {
		cfg = NewConfigFromEnv()
	}

	// 创建 handler options
	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.Level),
	}

	// 在开发环境添加源文件信息
	if cfg.AddSource {
		opts.AddSource = true
	}

	// 根据格式选择处理器
	var logHandler slog.Handler
	if strings.ToLower(cfg.Format) == "json" {
		logHandler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, opts)
	}

	// 添加服务标识
	defaultLogger = slog.New(logHandler.WithAttrs([]slog.Attr{
		slog.String("service", "cocursor-backend"),
	}))

	debugMode = strings.ToLower(cfg.Level) == "debug"

	slog.SetDefault(defaultLogger)
}

// GetLogger 获取默认 logger
func GetLogger() *slog.Logger {
	if defaultLogger == nil {
		// 未初始化，使用默认配置
		Init(nil)
	}
	return defaultLogger
}

// With 创建带有额外字段的 logger
func With(args ...any) *slog.Logger {
	return GetLogger().With(args...)
}

// NewModuleLogger 为特定模块创建 logger
func NewModuleLogger(module, component string) *slog.Logger {
	return GetLogger().With(
		slog.String("module", module),
		slog.String("component", component),
	)
}

// IsDebugMode 检查是否为调试模式
func IsDebugMode() bool {
	return debugMode
}

// parseLevel 解析日志级别
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
