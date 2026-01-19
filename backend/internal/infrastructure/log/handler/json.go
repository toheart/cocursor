package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"time"
)

// JSONHandler JSON 格式日志处理器
type JSONHandler struct {
	opts *slog.HandlerOptions
	mu   sync.Mutex
	out  io.Writer
	enc  *json.Encoder
}

// NewJSONHandler 创建 JSON 处理器
func NewJSONHandler(out io.Writer, cfg interface{}) *JSONHandler {
	return &JSONHandler{
		out:  out,
		opts: &slog.HandlerOptions{},
		enc:  json.NewEncoder(out),
	}
}

// Enabled 检查日志级别是否启用
func (h *JSONHandler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := h.opts.Level
	if minLevel == nil {
		return true
	}
	return level >= minLevel.Level()
}

// Handle 处理日志记录
func (h *JSONHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 创建 JSON 对象
	obj := make(map[string]any)

	// 基础字段
	obj["time"] = r.Time.Format(time.RFC3339Nano)
	obj["level"] = r.Level.String()
	obj["msg"] = r.Message

	// 复制所有属性
	r.Attrs(func(a slog.Attr) bool {
		obj[a.Key] = a.Value.Any()
		return true
	})

	// 编码为 JSON
	return h.enc.Encode(obj)
}

// WithAttrs 返回带有额外属性的处理器
func (h *JSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup 返回带有分组的处理器
func (h *JSONHandler) WithGroup(name string) slog.Handler {
	return h
}
