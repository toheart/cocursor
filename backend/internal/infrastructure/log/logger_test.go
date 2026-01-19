package log

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"invalid", slog.LevelInfo}, // 默认值
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewConfigFromEnv(t *testing.T) {
	// 保存原始环境变量
	oldLogLevel := os.Getenv("LOG_LEVEL")
	oldLogFormat := os.Getenv("LOG_FORMAT")
	oldEnv := os.Getenv("ENV")

	defer func() {
		// 恢复环境变量
		if oldLogLevel != "" {
			os.Setenv("LOG_LEVEL", oldLogLevel)
		} else {
			os.Unsetenv("LOG_LEVEL")
		}
		if oldLogFormat != "" {
			os.Setenv("LOG_FORMAT", oldLogFormat)
		} else {
			os.Unsetenv("LOG_FORMAT")
		}
		if oldEnv != "" {
			os.Setenv("ENV", oldEnv)
		} else {
			os.Unsetenv("ENV")
		}
	}()

	t.Run("default config", func(t *testing.T) {
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("LOG_FORMAT")
		os.Unsetenv("ENV")

		cfg := NewConfigFromEnv()

		if cfg.Level != "info" {
			t.Errorf("expected default level info, got %s", cfg.Level)
		}
		if cfg.Format != "console" {
			t.Errorf("expected default format console, got %s", cfg.Format)
		}
	})

	t.Run("custom config", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("LOG_FORMAT", "json")

		cfg := NewConfigFromEnv()

		if cfg.Level != "debug" {
			t.Errorf("expected level debug, got %s", cfg.Level)
		}
		if cfg.Format != "json" {
			t.Errorf("expected format json, got %s", cfg.Format)
		}
	})

	t.Run("development mode", func(t *testing.T) {
		os.Setenv("ENV", "development")
		os.Setenv("LOG_LEVEL", "error") // 应该被覆盖

		cfg := NewConfigFromEnv()

		// 开发环境应该覆盖为 debug
		if cfg.Level != "debug" {
			t.Errorf("expected debug in development, got %s", cfg.Level)
		}
		if cfg.Format != "console" {
			t.Errorf("expected console in development, got %s", cfg.Format)
		}
		if !cfg.AddSource {
			t.Error("expected AddSource true in development")
		}
	})
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		defaultValue  bool
		envValue      string
		expected      bool
	}{
		{"true value", "TEST_BOOL", false, "true", true},
		{"false value", "TEST_BOOL", true, "false", false},
		{"invalid value", "TEST_BOOL", true, "invalid", true}, // 默认值
		{"missing env", "MISSING_BOOL", false, "", false},  // 默认值
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvBool(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvBool(%q, %v) = %v, want %v", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestInit(t *testing.T) {
	// 临时设置环境变量
	oldLevel := os.Getenv("LOG_LEVEL")
	oldFormat := os.Getenv("LOG_FORMAT")
	defer func() {
		if oldLevel != "" {
			os.Setenv("LOG_LEVEL", oldLevel)
		} else {
			os.Unsetenv("LOG_LEVEL")
		}
		if oldFormat != "" {
			os.Setenv("LOG_FORMAT", oldFormat)
		} else {
			os.Unsetenv("LOG_FORMAT")
		}
	}()

	t.Run("init with defaults", func(t *testing.T) {
		Init(nil)

		logger := GetLogger()
		if logger == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("init with custom config", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "debug")
		cfg := NewConfigFromEnv()

		Init(cfg)

		if !IsDebugMode() {
			t.Error("expected debug mode")
		}
	})
}

func TestNewModuleLogger(t *testing.T) {
	Init(nil)

	logger := NewModuleLogger("test", "component")
	if logger == nil {
		t.Error("expected non-nil logger")
	}

	// 测试日志输出（只验证不 panic）
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	testLogger := slog.New(handler).With("module", "test", "component", "component")

	testLogger.Info("test message")

	if !strings.Contains(buf.String(), "test message") {
		t.Error("expected log message in output")
	}
}
