import React, { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { apiService, SessionHealth, getVscodeApi } from "../services/api";

interface ChatItem {
  composerId: string;
  name: string;
  lastUpdatedAt: number;
  totalLinesAdded?: number;
  totalLinesRemoved?: number;
  filesChangedCount?: number;
}

interface AppState {
  loading: boolean;
  error: string | null;
  chats: ChatItem[];
  sessionHealth: SessionHealth | null;
  previousEntropy: number | null;
}

export const App: React.FC = () => {
  const navigate = useNavigate();
  const [state, setState] = useState<AppState>({
    loading: true,
    error: null,
    chats: [],
    sessionHealth: null,
    previousEntropy: null
  });

  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const isMountedRef = useRef(true);

  useEffect(() => {
    console.log("App: 组件已挂载，开始加载对话");
    isMountedRef.current = true;
    loadChats();
    loadSessionHealth();

    // 设置定时器，每 30 秒轮询一次会话健康状态
    intervalRef.current = setInterval(() => {
      if (isMountedRef.current) {
        loadSessionHealth();
      }
    }, 30000);

    // 清理定时器
    return () => {
      isMountedRef.current = false;
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, []);

  const loadChats = async (): Promise<void> => {
    if (!isMountedRef.current) return;
    
    try {
      setState((prev) => ({ ...prev, loading: true, error: null }));
      // 使用会话列表 API 替代 chats API
      const result = await apiService.getSessionList("", 10, 0, "");
      
      // 检查组件是否已卸载
      if (!isMountedRef.current) return;
      
      if (result && typeof result === "object" && "data" in result) {
        const response = result as any;
        const sessions = (response.data || []) as ChatItem[];
        setState((prev) => ({
          ...prev,
          loading: false,
          error: null,
          chats: Array.isArray(sessions) ? sessions : []
        }));
      } else {
        setState((prev) => ({
          ...prev,
          loading: false,
          error: null,
          chats: []
        }));
      }
    } catch (error) {
      // 检查组件是否已卸载
      if (!isMountedRef.current) return;
      
      setState((prev) => ({
        ...prev,
        loading: false,
        error: error instanceof Error ? error.message : "未知错误",
        chats: []
      }));
    }
  };

  const handleChatClick = (chat: ChatItem): void => {
    if (chat.composerId) {
      navigate(`/sessions/${chat.composerId}`);
    }
  };

  const formatDate = (timestamp: number): string => {
    const date = new Date(timestamp);
    return date.toLocaleString("zh-CN", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit"
    });
  };

  const loadSessionHealth = async (): Promise<void> => {
    if (!isMountedRef.current) return;
    
    try {
      // 获取当前工作区路径
      const workspacePath = (window as any).__WORKSPACE_PATH__;
      const health = await apiService.getCurrentSessionHealth(workspacePath);
      
      // 检查组件是否已卸载
      if (!isMountedRef.current) return;
      
      setState((prev) => {
        // 检查熵值是否从健康变为危险
        const wasHealthy = prev.previousEntropy === null || prev.previousEntropy < 70;
        const isDangerous = health.entropy >= 70;
        const shouldNotify = wasHealthy && isDangerous;

        if (shouldNotify && prev.sessionHealth) {
          // 发送消息到 Extension，触发 VS Code 通知
          try {
            const vscode = getVscodeApi();
            vscode.postMessage({
              command: "showEntropyWarning",
              payload: {
                entropy: health.entropy,
                message: "会话熵值过高，建议重启会话"
              }
            });
          } catch (err) {
            console.error("发送熵值警告失败:", err);
          }
        }

        return {
          ...prev,
          sessionHealth: health,
          previousEntropy: health.entropy
        };
      });
    } catch (error) {
      // 组件已卸载，不更新状态
      if (!isMountedRef.current) return;
      console.error("加载会话健康状态失败:", error);
      // 不显示错误，静默失败
    }
  };

  const getEntropyColor = (entropy: number): string => {
    if (entropy < 40) {
      return "var(--vscode-testing-iconPassed)"; // 绿色
    } else if (entropy < 70) {
      return "var(--vscode-testing-iconQueued)"; // 黄色
    } else {
      return "var(--vscode-testing-iconFailed)"; // 红色
    }
  };

  const getEntropyStatusText = (status: string): string => {
    switch (status) {
      case "healthy":
        return "健康";
      case "sub_healthy":
        return "亚健康";
      case "dangerous":
        return "危险";
      default:
        return "未知";
    }
  };

  return (
    <div className="cocursor-app">
      <div style={{ padding: "12px 16px", borderBottom: "1px solid var(--vscode-panel-border)", display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <h2 style={{ margin: 0, fontSize: "14px", fontWeight: 600 }}>CoCursor 仪表板</h2>
        <button onClick={loadChats} disabled={state.loading} style={{ padding: "4px 8px", fontSize: "12px" }}>
          {state.loading ? "加载中..." : "刷新"}
        </button>
      </div>

      <main className="cocursor-main" style={{ padding: "16px" }}>
        {/* 会话熵值展示 */}
        {state.sessionHealth && (
          <div className="cocursor-session-health">
            <div className="cocursor-session-health-header">
              <h2>会话健康状态</h2>
            </div>
            <div className="cocursor-entropy-display">
              <div className="cocursor-entropy-value">
                <span className="cocursor-entropy-label">熵值:</span>
                <span
                  className="cocursor-entropy-number"
                  style={{
                    color: getEntropyColor(state.sessionHealth.entropy),
                    fontWeight: "bold"
                  }}
                >
                  {state.sessionHealth.entropy.toFixed(2)}
                </span>
              </div>
              <div className="cocursor-entropy-status">
                <span className="cocursor-entropy-label">状态:</span>
                <span
                  className="cocursor-entropy-status-text"
                  style={{
                    color: getEntropyColor(state.sessionHealth.entropy)
                  }}
                >
                  {getEntropyStatusText(state.sessionHealth.status)}
                </span>
              </div>
              {/* 进度条 */}
              <div className="cocursor-entropy-progress">
                <div
                  className="cocursor-entropy-progress-bar"
                  style={{
                    width: `${Math.min((state.sessionHealth.entropy / 100) * 100, 100)}%`,
                    backgroundColor: getEntropyColor(state.sessionHealth.entropy),
                    animation:
                      state.sessionHealth.entropy >= 70
                        ? "pulse 1s ease-in-out infinite"
                        : "none"
                  }}
                />
              </div>
              {state.sessionHealth.warning && (
                <div
                  className="cocursor-entropy-warning"
                  style={{
                    color: "var(--vscode-errorForeground)",
                    marginTop: "8px",
                    padding: "8px",
                    backgroundColor: "var(--vscode-inputValidation-errorBackground)",
                    borderRadius: "4px"
                  }}
                >
                  ⚠️ {state.sessionHealth.warning}
                </div>
              )}
            </div>
          </div>
        )}

        {state.error && (
          <div className="cocursor-error">错误: {state.error}</div>
        )}

        {state.loading ? (
          <div className="cocursor-loading">加载中...</div>
        ) : (
          <div className="cocursor-chats">
            {state.chats.length === 0 ? (
              <div className="cocursor-empty" style={{ padding: "16px", textAlign: "center", color: "var(--vscode-descriptionForeground)" }}>
                暂无对话数据
              </div>
            ) : (
              <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                {state.chats.map((chat, index) => (
                  <li
                    key={chat.composerId || index}
                    onClick={() => handleChatClick(chat)}
                    style={{
                      padding: "12px 16px",
                      borderBottom: "1px solid var(--vscode-panel-border)",
                      cursor: "pointer",
                      transition: "background-color 0.2s"
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.backgroundColor = "var(--vscode-list-hoverBackground)";
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.backgroundColor = "transparent";
                    }}
                  >
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "4px" }}>
                      <h3 style={{ margin: 0, fontSize: "14px", fontWeight: 600 }}>
                        {chat.name || "未命名会话"}
                      </h3>
                      <span style={{ fontSize: "12px", color: "var(--vscode-descriptionForeground)" }}>
                        {formatDate(chat.lastUpdatedAt)}
                      </span>
                    </div>
                    {(chat.totalLinesAdded !== undefined || chat.totalLinesRemoved !== undefined || chat.filesChangedCount !== undefined) && (
                      <div style={{ fontSize: "12px", color: "var(--vscode-descriptionForeground)", display: "flex", gap: "12px" }}>
                        {chat.totalLinesAdded !== undefined && chat.totalLinesRemoved !== undefined && (
                          <span>+{chat.totalLinesAdded} / -{chat.totalLinesRemoved} 行</span>
                        )}
                        {chat.filesChangedCount !== undefined && (
                          <span>{chat.filesChangedCount} 个文件</span>
                        )}
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </main>
    </div>
  );
};
