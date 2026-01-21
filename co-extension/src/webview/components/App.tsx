/**
 * 主应用组件（重构版）
 */

import React, { useState, useCallback, useMemo, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Session, SessionHealth } from "../types";
import { apiService, getVscodeApi } from "../services/api";
import { useApi, useMounted, useVisibilityInterval, useDataRefresh } from "../hooks";
import { shouldWarnAboutEntropy, formatShortDate } from "../utils";
import { Button, Loading, EmptyState, ErrorState, SessionHealthCard } from "./shared";

// ========== 常量定义 ==========

const SESSION_HEALTH_POLL_INTERVAL = 30000; // 30秒
const SESSION_LOAD_LIMIT = 10;
const ENTROPY_WARNING_THRESHOLD = 70;
const ENTROPY_HEALTHY_THRESHOLD = 40;

// ========== 主组件 ==========

export const App: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const isMounted = useMounted();

  // ========== 状态管理 ==========
  const [previousEntropy, setPreviousEntropy] = useState<number | null>(null);

  // ========== API 请求 ==========
  const fetchChats = useCallback(async () => {
    const response = await apiService.getSessionList("", SESSION_LOAD_LIMIT, 0, "");
    return response as { data: Session[] };
  }, []);

  const fetchSessionHealth = useCallback(async () => {
    const workspacePath = (window as any).__WORKSPACE_PATH__;
    return await apiService.getCurrentSessionHealth(workspacePath) as SessionHealth;
  }, []);

  const {
    data: chatsResponse,
    loading,
    error,
    refetch: loadChats,
  } = useApi<{ data: Session[] }>(fetchChats);

  const {
    data: sessionHealth,
    refetch: loadSessionHealth,
  } = useApi<SessionHealth>(fetchSessionHealth);

  const chats = chatsResponse?.data || [];

  // ========== 事件处理 ==========
  const handleChatClick = useCallback(
    (chat: Session) => {
      if (chat.composerId) {
        navigate(`/sessions/${chat.composerId}`);
      }
    },
    [navigate]
  );

  const sendEntropyWarning = useCallback(
    (entropy: number, message: string) => {
      if (!isMounted.current) return;

      try {
        const vscode = getVscodeApi();
        vscode.postMessage({
          command: "showEntropyWarning",
          payload: {
            entropy,
            message: t("app.entropyWarning"),
          },
        });
      } catch (err) {
        console.error("发送熵值警告失败:", err);
      }
    },
    [isMounted]
  );

  // ========== 副作用 ==========
  // 检查是否需要发送熵值警告
  useEffect(() => {
    if (
      sessionHealth &&
      shouldWarnAboutEntropy(previousEntropy, sessionHealth.entropy)
    ) {
      sendEntropyWarning(sessionHealth.entropy, t("app.entropyWarning"));
    }
    setPreviousEntropy(sessionHealth?.entropy || null);
  }, [sessionHealth, previousEntropy, sendEntropyWarning]);

  // 定时刷新会话健康状态（页面可见时才轮询）
  useVisibilityInterval(() => {
    if (isMounted.current) {
      loadSessionHealth();
    }
  }, SESSION_HEALTH_POLL_INTERVAL);

  // 监听来自 Extension 的刷新通知
  useDataRefresh(
    useCallback(() => {
      console.log("[App] received refresh notification, reloading data");
      loadChats();
      loadSessionHealth();
    }, [loadChats, loadSessionHealth])
  );

  // ========== 渲染 ==========
  return (
    <div className="cocursor-app">
      <AppHeader onRefresh={loadChats} loading={loading} />

      <main className="cocursor-main" style={{ padding: "16px" }}>

        {sessionHealth && (
          <SessionHealthCard
            health={sessionHealth}
            className="cocursor-session-health-main"
          />
        )}

        {error && <ErrorState error={error} onRetry={loadChats} />}

        {loading ? (
          <Loading message={t("app.loadingChats")} />
        ) : (
          <ChatList
            chats={chats}
            onChatClick={handleChatClick}
          />
        )}
      </main>
    </div>
  );
};

// ========== 子组件 ==========

interface AppHeaderProps {
  onRefresh: () => void;
  loading: boolean;
}

const AppHeader: React.FC<AppHeaderProps> = ({ onRefresh, loading }) => {
  const { t } = useTranslation();
  return (
    <div
      style={{
        padding: "12px 16px",
        borderBottom: "1px solid var(--vscode-panel-border)",
        display: "flex",
        justifyContent: "space-between",
        alignItems: "center",
      }}
    >
      <h2
        style={{
          margin: 0,
          fontSize: "14px",
          fontWeight: 600,
        }}
      >
        {t("app.title")}
      </h2>
      <Button
        onClick={onRefresh}
        disabled={loading}
        loading={loading}
        size="small"
        variant="secondary"
      >
        {loading ? t("common.loading") : t("common.refresh")}
      </Button>
    </div>
  );
};

interface ChatListProps {
  chats: Session[];
  onChatClick: (chat: Session) => void;
}

const ChatList: React.FC<ChatListProps> = ({ chats, onChatClick }) => {
  const { t } = useTranslation();
  return (
    <div className="cocursor-chats">
      {chats.length === 0 ? (
        <EmptyState
          icon="💬"
          title={t("app.noChats")}
          description={t("app.noChatsDesc")}
        />
      ) : (
        <ul
          style={{
            listStyle: "none",
            padding: 0,
            margin: 0,
          }}
        >
          {chats.map((chat, index) => (
            <ChatItem key={chat.composerId || index} chat={chat} onClick={onChatClick} />
          ))}
        </ul>
      )}
    </div>
  );
};

interface ChatItemProps {
  chat: Session;
  onClick: (chat: Session) => void;
}

const ChatItem: React.FC<ChatItemProps> = ({ chat, onClick }) => {
  const { t } = useTranslation();
  return (
    <li
      style={{
        padding: "12px 16px",
        marginBottom: "8px",
        backgroundColor: "var(--vscode-list-activeSelectionBackground)",
        borderRadius: "4px",
        border: "1px solid var(--vscode-list-activeSelectionBorder)",
        cursor: "pointer",
        transition: "background-color 0.2s, transform 0.2s",
      }}
      onClick={() => onClick(chat)}
      onMouseEnter={(e) => {
        e.currentTarget.style.backgroundColor = "var(--vscode-list-hoverBackground)";
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.backgroundColor = "var(--vscode-list-activeSelectionBackground)";
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: "4px",
        }}
      >
        <h3
          style={{
            margin: 0,
            fontSize: "14px",
            fontWeight: 600,
          }}
        >
          {chat.name || t("app.unnamedSession")}
        </h3>
        <span
          style={{
            fontSize: "12px",
            color: "var(--vscode-descriptionForeground)",
          }}
        >
          {formatShortDate(chat.lastUpdatedAt)}
        </span>
      </div>
      {(chat.totalLinesAdded !== undefined ||
        chat.totalLinesRemoved !== undefined ||
        chat.filesChangedCount !== undefined) && (
        <div
          style={{
            fontSize: "12px",
            color: "var(--vscode-descriptionForeground)",
            display: "flex",
            gap: "12px",
          }}
        >
          {chat.totalLinesAdded !== undefined &&
            chat.totalLinesRemoved !== undefined && (
              <span>
                +{chat.totalLinesAdded} / -{chat.totalLinesRemoved} {t("app.lines")}
              </span>
            )}
          {chat.filesChangedCount !== undefined && (
            <span>{chat.filesChangedCount} {t("app.files")}</span>
          )}
        </div>
      )}
    </li>
  );
};
