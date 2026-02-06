/**
 * 共享会话 Tab 页面
 */

import React, { useState, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useParams } from "react-router-dom";
import { apiService } from "../../../services/api";
import { useApi } from "../../../hooks";
import { SessionCard, SharedSessionItem } from "../cards";
import { EmptyState, LoadingState } from "../shared";
import { SharedSessionDetail } from "../SharedSessionDetail";

export const SessionsPage: React.FC = () => {
  const { t } = useTranslation();
  const { teamId } = useParams<{ teamId: string }>();

  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(
    null,
  );
  const [page, setPage] = useState(1);
  const pageSize = 20;

  // 获取共享会话列表
  const fetchSessions = useCallback(async () => {
    if (!teamId) return { sessions: [], total: 0 };
    const resp = (await apiService.getSharedSessions(
      teamId,
      page,
      pageSize,
    )) as {
      sessions: SharedSessionItem[];
      total: number;
    };
    return resp;
  }, [teamId, page, pageSize]);

  const { data, loading, refetch } = useApi<{
    sessions: SharedSessionItem[];
    total: number;
  }>(fetchSessions);

  const sessions = useMemo(() => data?.sessions || [], [data]);
  const total = data?.total || 0;
  const totalPages = Math.ceil(total / pageSize);

  // 如果选中了某个会话，显示详情
  if (selectedSessionId && teamId) {
    return (
      <SharedSessionDetail
        teamId={teamId}
        sessionId={selectedSessionId}
        onBack={() => {
          setSelectedSessionId(null);
          refetch();
        }}
      />
    );
  }

  return (
    <div className="ct-sessions-page">
      <div className="ct-section-header">
        <h3 className="ct-section-title">{t("session.sharedSessions")}</h3>
        <button
          className="ct-btn-icon"
          onClick={refetch}
          title={t("common.refresh")}
        >
          <span className="codicon codicon-refresh" />
        </button>
      </div>

      {loading && sessions.length === 0 ? (
        <LoadingState />
      ) : sessions.length === 0 ? (
        <EmptyState
          icon="comment-discussion"
          title={t("session.noSharedSessions")}
          description={t("session.noSharedSessionsDesc")}
        />
      ) : (
        <>
          <div className="ct-session-list">
            {sessions.map((session) => (
              <SessionCard
                key={session.id}
                session={session}
                onClick={() => setSelectedSessionId(session.id)}
              />
            ))}
          </div>

          {/* 分页 */}
          {totalPages > 1 && (
            <div className="ct-pagination">
              <button
                className="ct-btn-icon"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                <span className="codicon codicon-chevron-left" />
              </button>
              <span className="ct-pagination-info">
                {page} / {totalPages}
              </span>
              <button
                className="ct-btn-icon"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
                <span className="codicon codicon-chevron-right" />
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
};
