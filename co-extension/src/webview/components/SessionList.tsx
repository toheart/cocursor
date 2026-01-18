import React, { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { FixedSizeList as List } from "react-window";
import { apiService } from "../services/api";

interface Session {
  composerId: string;
  name: string;
  createdAt: number;
  lastUpdatedAt: number;
  totalLinesAdded: number;
  totalLinesRemoved: number;
  filesChangedCount: number;
}

const STORAGE_KEY = "cocursor_session_list_state";

interface SessionListState {
  offset: number;
  search: string;
}

export const SessionList: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [total, setTotal] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  
  // 从 sessionStorage 恢复状态
  const getSavedState = (): SessionListState => {
    try {
      const saved = sessionStorage.getItem(STORAGE_KEY);
      if (saved) {
        return JSON.parse(saved);
      }
    } catch (err) {
      console.error("读取 sessionStorage 失败:", err);
    }
    return { offset: 0, search: "" };
  };

  const savedState = getSavedState();
  const [search, setSearch] = useState(savedState.search);
  const [offset, setOffset] = useState(savedState.offset);
  const limit = 20;
  const isMountedRef = React.useRef(true);

  // 保存状态到 sessionStorage
  const saveState = (newOffset: number, newSearch: string) => {
    try {
      sessionStorage.setItem(STORAGE_KEY, JSON.stringify({
        offset: newOffset,
        search: newSearch
      }));
    } catch (err) {
      console.error("保存 sessionStorage 失败:", err);
    }
  };

  // 移除页面内的返回按钮，使用导航栏

  useEffect(() => {
    isMountedRef.current = true;
    loadSessions();
    
    return () => {
      isMountedRef.current = false;
    };
  }, [offset, search]);

  // 保存状态变化
  useEffect(() => {
    saveState(offset, search);
  }, [offset, search]);

  const loadSessions = async (): Promise<void> => {
    if (!isMountedRef.current) return;
    
    try {
      setLoading(true);
      setError(null);
      const result = await apiService.getSessionList("", limit, offset, search);
      
      // 检查组件是否已卸载
      if (!isMountedRef.current) return;
      
      if (result && typeof result === "object" && "data" in result) {
        const response = result as any;
        setSessions(response.data || []);
        if (response.page) {
          setTotal(response.page.total || 0);
          setHasMore((response.page.page || 1) * (response.page.pageSize || limit) < (response.page.total || 0));
        }
      }
    } catch (err) {
      // 组件已卸载，不更新状态
      if (!isMountedRef.current) return;
      setError(err instanceof Error ? err.message : "未知错误");
    } finally {
      if (isMountedRef.current) {
        setLoading(false);
      }
    }
  };

  const handleSessionClick = (sessionId: string): void => {
    navigate(`/sessions/${sessionId}`);
  };

  const formatDate = (timestamp: number): string => {
    const date = new Date(timestamp);
    return date.toLocaleString("zh-CN");
  };

  const Row = ({ index, style }: { index: number; style: React.CSSProperties }) => {
    const session = sessions[index];
    if (!session) return null;

    return (
      <div
        style={style}
        className="cocursor-session-item"
        onClick={() => handleSessionClick(session.composerId)}
      >
        <div className="cocursor-session-header">
          <h3>{session.name || "未命名会话"}</h3>
          <span className="cocursor-session-time">{formatDate(session.lastUpdatedAt)}</span>
        </div>
        <div className="cocursor-session-stats">
          <span>+{session.totalLinesAdded} / -{session.totalLinesRemoved} 行</span>
          <span>{session.filesChangedCount} 个文件</span>
        </div>
      </div>
    );
  };

  return (
    <div className="cocursor-session-list">
      <div className="cocursor-search">
        <input
          type="text"
          placeholder="搜索会话..."
          value={search}
          onChange={(e) => {
            const newSearch = e.target.value;
            setSearch(newSearch);
            setOffset(0);
            saveState(0, newSearch);
          }}
        />
      </div>

      <main className="cocursor-main" style={{ padding: "20px" }}>
        {loading && <div className="cocursor-loading">加载中...</div>}
        {error && <div className="cocursor-error">错误: {error}</div>}
        {!loading && !error && (
          <>
            {sessions.length === 0 ? (
              <div className="cocursor-empty">暂无会话</div>
            ) : (
              <>
                {total > 0 && (
                  <div style={{ 
                    marginBottom: "16px", 
                    fontSize: "13px", 
                    color: "var(--vscode-descriptionForeground)",
                    fontWeight: 500
                  }}>
                    共 {total} 个会话
                  </div>
                )}
                <List
                  height={600}
                  itemCount={sessions.length}
                  itemSize={120}
                  width="100%"
                >
                  {Row}
                </List>
                {hasMore && (
                  <button
                    onClick={() => {
                      const newOffset = offset + limit;
                      setOffset(newOffset);
                      saveState(newOffset, search);
                    }}
                    className="cocursor-load-more"
                  >
                    加载更多
                  </button>
                )}
              </>
            )}
          </>
        )}
      </main>
    </div>
  );
};
