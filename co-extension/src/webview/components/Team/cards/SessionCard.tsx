/**
 * 共享会话卡片组件
 */

import React from "react";
import { useTranslation } from "react-i18next";

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

interface SessionCardProps {
  session: SharedSessionItem;
  onClick: () => void;
}

/** 格式化相对时间 */
function useFormatTime() {
  const { t } = useTranslation();

  return (dateStr: string): string => {
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
}

export const SessionCard: React.FC<SessionCardProps> = ({
  session,
  onClick,
}) => {
  const { t } = useTranslation();
  const formatTime = useFormatTime();

  return (
    <div className="ct-session-card" onClick={onClick}>
      <div className="ct-session-card-header">
        <div className="ct-session-card-avatar">
          {session.sharer_name.charAt(0).toUpperCase()}
        </div>
        <div className="ct-session-card-meta">
          <span className="ct-session-card-author">
            {session.sharer_name}
          </span>
          <span className="ct-session-card-time">
            {formatTime(session.shared_at)}
          </span>
        </div>
      </div>
      <h4 className="ct-session-card-title">{session.title}</h4>
      {session.description && (
        <p className="ct-session-card-desc">{session.description}</p>
      )}
      <div className="ct-session-card-stats">
        <span className="ct-session-card-stat">
          <span className="codicon codicon-comment-discussion" />
          {session.message_count} {t("session.messages")}
        </span>
        <span className="ct-session-card-stat">
          <span className="codicon codicon-note" />
          {session.comment_count} {t("session.comments")}
        </span>
      </div>
    </div>
  );
};

export type { SharedSessionItem };
