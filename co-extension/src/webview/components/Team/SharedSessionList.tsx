/**
 * 团队共享会话列表组件
 */

import React, { useState, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { useApi } from "../../hooks";
import { SharedSessionDetail } from "./SharedSessionDetail";

// 共享会话列表项
interface SharedSessionItem {
  id: string;
  sharer_id: string;
  sharer_name: string;
  title: string;
  message_count: number;
  description?: string;
  shared_at: string;
  comment_count: number;
}

interface SharedSessionListProps {
  teamId: string;
}

export const SharedSessionList: React.FC<SharedSessionListProps> = ({ teamId }) => {
  const { t } = useTranslation();
  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const pageSize = 20;

  // 获取共享会话列表
  const fetchSessions = useCallback(async () => {
    const resp = await apiService.getSharedSessions(teamId, page, pageSize) as {
      sessions: SharedSessionItem[];
      total: number;
    };
    return resp;
  }, [teamId, page, pageSize]);

  const { data, loading, refetch } = useApi<{ sessions: SharedSessionItem[]; total: number }>(fetchSessions);

  const sessions = useMemo(() => data?.sessions || [], [data]);
  const total = data?.total || 0;
  const totalPages = Math.ceil(total / pageSize);

  // 格式化时间
  const formatTime = (dateStr: string): string => {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return t("session.justNow");
    if (minutes < 60) return t("session.minutesAgo", { count: minutes });
    if (hours < 24) return t("session.hoursAgo", { count: hours });
    if (days < 7) return t("session.daysAgo", { count: days });
    
    return date.toLocaleDateString();
  };

  // 如果选中了某个会话，显示详情
  if (selectedSessionId) {
    return (
      <SharedSessionDetail
        teamId={teamId}
        sessionId={selectedSessionId}
        onBack={() => {
          setSelectedSessionId(null);
          refetch(); // 返回时刷新列表以更新评论数
        }}
      />
    );
  }

  return (
    <div className="cocursor-shared-sessions">
      <div className="cocursor-team-section-header">
        <h3>{t("session.sharedSessions")}</h3>
        <button className="cocursor-btn secondary" onClick={refetch}>
          {t("common.refresh")}
        </button>
      </div>

      {loading ? (
        <div className="cocursor-team-loading">
          <div className="cocursor-team-loading-spinner"></div>
        </div>
      ) : sessions.length === 0 ? (
        <div className="cocursor-team-empty-section">
          <span className="cocursor-team-empty-icon">💬</span>
          <span>{t("session.noSharedSessions")}</span>
          <p>{t("session.noSharedSessionsDesc")}</p>
        </div>
      ) : (
        <>
          <div className="cocursor-shared-session-list">
            {sessions.map((session) => (
              <div
                key={session.id}
                className="cocursor-shared-session-card"
                onClick={() => setSelectedSessionId(session.id)}
              >
                <div className="cocursor-shared-session-header">
                  <div className="cocursor-shared-session-avatar">
                    {session.sharer_name.charAt(0).toUpperCase()}
                  </div>
                  <div className="cocursor-shared-session-meta">
                    <span className="cocursor-shared-session-author">
                      {session.sharer_name}
                    </span>
                    <span className="cocursor-shared-session-time">
                      {formatTime(session.shared_at)}
                    </span>
                  </div>
                </div>
                <h4 className="cocursor-shared-session-title">{session.title}</h4>
                {session.description && (
                  <p className="cocursor-shared-session-description">
                    {session.description}
                  </p>
                )}
                <div className="cocursor-shared-session-stats">
                  <span className="cocursor-shared-session-stat">
                    💬 {session.message_count} {t("session.messages")}
                  </span>
                  <span className="cocursor-shared-session-stat">
                    📝 {session.comment_count} {t("session.comments")}
                  </span>
                </div>
              </div>
            ))}
          </div>

          {/* 分页 */}
          {totalPages > 1 && (
            <div className="cocursor-pagination">
              <button
                className="cocursor-pagination-btn"
                disabled={page <= 1}
                onClick={() => setPage(p => p - 1)}
              >
                ←
              </button>
              <span className="cocursor-pagination-info">
                {page} / {totalPages}
              </span>
              <button
                className="cocursor-pagination-btn"
                disabled={page >= totalPages}
                onClick={() => setPage(p => p + 1)}
              >
                →
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
};
