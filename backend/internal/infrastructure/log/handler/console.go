package handler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

// ANSI 颜色代码
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
)

// ConsoleHandler 控制台日志处理器（彩色输出）
type ConsoleHandler struct {
	opts *slog.HandlerOptions
	mu   sync.Mutex
	out  io.Writer
	attrs []slog.Attr
}

// NewConsoleHandler 创建控制台处理器
func NewConsoleHandler(out io.Writer, cfg interface{}) *ConsoleHandler {
	return &ConsoleHandler{
		out:   out,
		opts:  &slog.HandlerOptions{},
		attrs: []slog.Attr{},
	}
}

// Enabled 检查日志级别是否启用
func (h *ConsoleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := h.opts.Level
	if minLevel == nil {
		return true
	}
	return level >= minLevel.Level()
}

// Handle 处理日志记录
func (h *ConsoleHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 格式化日志级别
	level := r.Level.String()
	levelColor := levelColor(r.Level)

	// 格式化时间
	timestamp := r.Time.Format("2006-01-02T15:04:05.000Z")

	// 提取模块和组件信息
	var module, component string
	r.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "module":
			module = a.Value.String()
		case "component":
			component = a.Value.String()
		}
		return true
	})

	// 构建模块前缀
	modulePrefix := ""
	if module != "" && component != "" {
		modulePrefix = fmt.Sprintf(" [%s/%s]", module, component)
	} else if module != "" {
		modulePrefix = fmt.Sprintf(" [%s]", module)
	}

	// 格式化日志
	fmt.Fprintf(h.out, "%s%s%s %s%s %s\n",
		levelColor, level, colorReset,
		timestamp,
		modulePrefix,
		r.Message,
	)

	// 输出属性
	r.Attrs(func(a slog.Attr) bool {
		// 跳过模块信息（已在前缀中显示）
		if a.Key == "module" || a.Key == "component" {
			return true
		}
		fmt.Fprintf(h.out, "  %s=%v\n", a.Key, a.Value)
		return true
	})

	return nil
}

// WithAttrs 返回带有额外属性的处理器
func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup 返回带有分组的处理器
func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	return h
}

// levelColor 返回日志级别对应的颜色
func levelColor(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return colorRed
	case level >= slog.LevelWarn:
		return colorYellow
	case level >= slog.LevelInfo:
		return colorGreen
	default:
		return colorBlue
	}
}
