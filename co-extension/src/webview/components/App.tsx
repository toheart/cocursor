import React, { useState, useEffect } from "react";
import { apiService } from "../services/api";

interface AppState {
  loading: boolean;
  error: string | null;
  chats: unknown[];
}

export const App: React.FC = () => {
  const [state, setState] = useState<AppState>({
    loading: true,
    error: null,
    chats: []
  });

  useEffect(() => {
    console.log("App: 组件已挂载，开始加载对话");
    loadChats();
  }, []);

  const loadChats = async (): Promise<void> => {
    try {
      setState((prev) => ({ ...prev, loading: true, error: null }));
      const chats = await apiService.getChats();
      setState({
        loading: false,
        error: null,
        chats: Array.isArray(chats) ? chats : []
      });
    } catch (error) {
      setState({
        loading: false,
        error: error instanceof Error ? error.message : "未知错误",
        chats: []
      });
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
