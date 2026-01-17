import React, { useState, useEffect, useRef } from "react";
import { apiService, SessionHealth } from "../services/api";

interface AppState {
  loading: boolean;
  error: string | null;
  chats: unknown[];
  sessionHealth: SessionHealth | null;
  previousEntropy: number | null;
}

export const App: React.FC = () => {
  const [state, setState] = useState<AppState>({
    loading: true,
    error: null,
    chats: [],
    sessionHealth: null,
    previousEntropy: null
  });

  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    console.log("App: 组件已挂载，开始加载对话");
    loadChats();
    loadSessionHealth();

    // 设置定时器，每 30 秒轮询一次会话健康状态
    intervalRef.current = setInterval(() => {
      loadSessionHealth();
    }, 30000);

    // 清理定时器
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  const loadChats = async (): Promise<void> => {
    try {
      setState((prev) => ({ ...prev, loading: true, error: null }));
      const chats = await apiService.getChats();
      setState((prev) => ({
        ...prev,
        loading: false,
        error: null,
        chats: Array.isArray(chats) ? chats : []
      }));
    } catch (error) {
      setState((prev) => ({
        ...prev,
        loading: false,
        error: error instanceof Error ? error.message : "未知错误",
        chats: []
      }));
    }
  };

  const loadSessionHealth = async (): Promise<void> => {
    try {
      const health = await apiService.getCurrentSessionHealth();
      setState((prev) => {
        // 检查熵值是否从健康变为危险
        const wasHealthy = prev.previousEntropy === null || prev.previousEntropy < 70;
        const isDangerous = health.entropy >= 70;
        const shouldNotify = wasHealthy && isDangerous;

        if (shouldNotify && prev.sessionHealth) {
          // 发送消息到 Extension，触发 VS Code 通知
          const vscode = acquireVsCodeApi();
          vscode.postMessage({
            command: "showEntropyWarning",
            payload: {
              entropy: health.entropy,
              message: "会话熵值过高，建议重启会话"
            }
          });
        }

        return {
          ...prev,
          sessionHealth: health,
          previousEntropy: health.entropy
        };
      });
    } catch (error) {
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
      <header className="cocursor-header">
        <h1>CoCursor</h1>
        <button onClick={loadChats} disabled={state.loading}>
          {state.loading ? "加载中..." : "刷新"}
        </button>
      </header>

      <main className="cocursor-main">
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
              <div className="cocursor-empty">暂无对话</div>
            ) : (
              <ul>
                {state.chats.map((chat, index) => (
                  <li key={index}>{JSON.stringify(chat)}</li>
                ))}
              </ul>
            )}
          </div>
        )}
      </main>
    </div>
  );
};
