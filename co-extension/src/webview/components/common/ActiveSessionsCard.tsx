import React, { useState } from "react";
import { useTranslation } from "react-i18next";

/**
 * 活跃会话数据接口
 */
export interface ActiveSession {
  composer_id: string;
  name: string;
  entropy: number;
  context_usage_percent: number;
  status: "healthy" | "warning" | "critical";
  warning?: string;
  last_updated_at: number;
}

/**
 * 活跃会话概览接口
 */
export interface ActiveSessionsOverview {
  focused: ActiveSession | null;
  open_sessions: ActiveSession[];
  closed_count: number;
  archived_count: number;
}

interface ActiveSessionsCardProps {
  data: ActiveSessionsOverview | null;
  loading?: boolean;
}

/**
 * 获取健康状态颜色
 */
const getStatusColor = (status: string): string => {
  switch (status) {
    case "healthy":
      return "var(--vscode-testing-iconPassed)";
    case "warning":
      return "var(--vscode-testing-iconQueued)";
    case "critical":
      return "var(--vscode-testing-iconFailed)";
    default:
      return "var(--vscode-foreground)";
  }
};

/**
 * 获取健康状态图标
 */
const getStatusIcon = (status: string): string => {
  switch (status) {
    case "healthy":
      return "✓";
    case "warning":
      return "⚠";
    case "critical":
      return "❗";
    default:
      return "○";
  }
};

/**
 * 活跃会话卡片组件
 */
export const ActiveSessionsCard: React.FC<ActiveSessionsCardProps> = ({
  data,
  loading = false,
}) => {
  const { t } = useTranslation();
  const [collapsed, setCollapsed] = useState(false);

  if (loading) {
    return (
      <div className="cocursor-active-sessions-card loading">
        <div className="cocursor-active-sessions-header">
          <h2>{t("workAnalysis.activeSessions.title")}</h2>
        </div>
        <div className="cocursor-active-sessions-loading">
          <div className="cocursor-loading-spinner"></div>
        </div>
      </div>
    );
  }

  if (!data) {
    return null;
  }

  const { focused, open_sessions, closed_count, archived_count } = data;
  const hasActiveSessions = focused || open_sessions.length > 0;

  return (
    <div className={`cocursor-active-sessions-card ${collapsed ? "collapsed" : ""}`}>
      {/* 头部 */}
      <div
        className="cocursor-active-sessions-header"
        onClick={() => setCollapsed(!collapsed)}
      >
        <div className="cocursor-active-sessions-title-section">
          <span className="cocursor-active-sessions-collapse-icon">
            {collapsed ? "▶" : "▼"}
          </span>
          <h2>{t("workAnalysis.activeSessions.title")}</h2>
          {collapsed && focused && (
            <span className="cocursor-active-sessions-summary">
              <span
                className="cocursor-active-sessions-status-dot"
                style={{ backgroundColor: getStatusColor(focused.status) }}
              />
              {focused.name} ({focused.entropy.toFixed(1)})
            </span>
          )}
        </div>
        <div className="cocursor-active-sessions-stats">
          {(open_sessions.length + (focused ? 1 : 0)) > 0 && (
            <span className="cocursor-active-sessions-stat open">
              {open_sessions.length + (focused ? 1 : 0)} {t("workAnalysis.activeSessions.open")}
            </span>
          )}
          {closed_count > 0 && (
            <span className="cocursor-active-sessions-stat closed">
              {closed_count} {t("workAnalysis.activeSessions.closed")}
            </span>
          )}
          {archived_count > 0 && (
            <span className="cocursor-active-sessions-stat archived">
              {archived_count} {t("workAnalysis.activeSessions.archived")}
            </span>
          )}
        </div>
      </div>

      {/* 内容区域（可折叠） */}
      {!collapsed && (
        <div className="cocursor-active-sessions-content">
          {!hasActiveSessions ? (
            <div className="cocursor-active-sessions-empty">
              <span className="cocursor-active-sessions-empty-icon">💤</span>
              <span>{t("workAnalysis.activeSessions.noActive")}</span>
            </div>
          ) : (
            <>
              {/* 聚焦会话 */}
              {focused && (
                <div className="cocursor-active-session-focused">
                  <div className="cocursor-active-session-label">
                    <span className="cocursor-active-session-focus-indicator">●</span>
                    {t("workAnalysis.activeSessions.focused")}
                  </div>
                  <SessionItem session={focused} isFocused />
                </div>
              )}

              {/* 其他打开的会话 */}
              {open_sessions.length > 0 && (
                <div className="cocursor-active-sessions-open-list">
                  <div className="cocursor-active-session-label">
                    {t("workAnalysis.activeSessions.otherOpen")} ({open_sessions.length})
                  </div>
                  {open_sessions.map((session) => (
                    <SessionItem key={session.composer_id} session={session} />
                  ))}
                </div>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
};

/**
 * 单个会话项组件
 */
interface SessionItemProps {
  session: ActiveSession;
  isFocused?: boolean;
}

const SessionItem: React.FC<SessionItemProps> = ({ session, isFocused = false }) => {
  const { t } = useTranslation();

  return (
    <div className={`cocursor-active-session-item ${isFocused ? "focused" : ""}`}>
      <div className="cocursor-active-session-main">
        <div className="cocursor-active-session-name" title={session.name}>
          {session.name.length > 40 ? session.name.substring(0, 40) + "..." : session.name}
        </div>
        <div className="cocursor-active-session-metrics">
          <span
            className="cocursor-active-session-status-badge"
            style={{ color: getStatusColor(session.status) }}
          >
            {getStatusIcon(session.status)}
          </span>
          <span className="cocursor-active-session-entropy">
            {t("workAnalysis.activeSessions.entropy")}: {session.entropy.toFixed(1)}
          </span>
          <span className="cocursor-active-session-context">
            {t("workAnalysis.activeSessions.context")}: {session.context_usage_percent.toFixed(0)}%
          </span>
        </div>
      </div>
      {session.warning && (
        <div
          className="cocursor-active-session-warning"
          style={{ color: getStatusColor(session.status) }}
        >
          {session.warning}
        </div>
      )}
      {/* 进度条 */}
      <div className="cocursor-active-session-progress">
        <div
          className="cocursor-active-session-progress-bar"
          style={{
            width: `${Math.min(session.context_usage_percent, 100)}%`,
            backgroundColor: getStatusColor(session.status),
          }}
        />
      </div>
    </div>
  );
};

export default ActiveSessionsCard;
