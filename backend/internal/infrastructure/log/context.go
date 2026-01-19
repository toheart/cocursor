package log

import (
	"context"
	"log/slog"
)

// 上下文键定义
const (
	// RequestContextID HTTP 请求 ID
	RequestContextID = "request_id"

	// WorkspaceContextID 工作区 ID
	WorkspaceContextID = "workspace_id"

	// SessionContextID 会话 ID
	SessionContextID = "session_id"

	// ProjectContextID 项目 ID
	ProjectContextID = "project_id"

	// UserContextID 用户 ID
	UserContextID = "user_id"
)

// WithRequestID 在上下文中添加请求 ID
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestContextID, requestID)
}

// WithWorkspaceID 在上下文中添加工作区 ID
func WithWorkspaceID(ctx context.Context, workspaceID string) context.Context {
	return context.WithValue(ctx, WorkspaceContextID, workspaceID)
}

// WithSessionID 在上下文中添加会话 ID
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, SessionContextID, sessionID)
}

// WithProjectID 在上下文中添加项目 ID
func WithProjectID(ctx context.Context, projectID string) context.Context {
	return context.WithValue(ctx, ProjectContextID, projectID)
}

// WithUserID 在上下文中添加用户 ID
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserContextID, userID)
}

// LogCtxFromContext 从上下文中提取日志字段
func LogCtxFromContext(ctx context.Context) []slog.Attr {
	var attrs []slog.Attr

	if requestID := ctx.Value(RequestContextID); requestID != nil {
		attrs = append(attrs, slog.String("request_id", requestID.(string)))
	}
	if workspaceID := ctx.Value(WorkspaceContextID); workspaceID != nil {
		attrs = append(attrs, slog.String("workspace_id", workspaceID.(string)))
	}
	if sessionID := ctx.Value(SessionContextID); sessionID != nil {
		attrs = append(attrs, slog.String("session_id", sessionID.(string)))
	}
	if projectID := ctx.Value(ProjectContextID); projectID != nil {
		attrs = append(attrs, slog.String("project_id", projectID.(string)))
	}
	if userID := ctx.Value(UserContextID); userID != nil {
		attrs = append(attrs, slog.String("user_id", userID.(string)))
	}

	return attrs
}
