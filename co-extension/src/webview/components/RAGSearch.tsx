/**
 * RAG 搜索组件
 */

import React, { useState, useCallback, useMemo, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { apiService } from "../services/api";
import { getVscodeApi } from "../services/api";
import { useApi, useDebounce, useToast } from "../hooks";
import { ToastContainer } from "./shared";

interface SearchResult {
  type: "message" | "turn";
  session_id: string;
  score: number;
  content: string;
  user_text?: string;
  ai_text?: string;
  message_id?: string;
  turn_index?: number;
  project_id: string;
  project_name: string;
  timestamp: number;
  message_ids?: string[];
}

const DEBOUNCE_DELAY = 500;

export const RAGSearch: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { showToast, toasts } = useToast();

  const [query, setQuery] = useState("");
  const [selectedProjects, setSelectedProjects] = useState<string[]>([]);
  const [expandedResults, setExpandedResults] = useState<Set<string>>(new Set());
  const debouncedQuery = useDebounce(query, DEBOUNCE_DELAY);

  // 搜索
  const performSearch = useCallback(async () => {
    if (!debouncedQuery.trim()) {
      return { results: [], count: 0 };
    }

    try {
      const response = await apiService.searchRAG(
        debouncedQuery,
        selectedProjects.length > 0 ? selectedProjects : undefined,
        20
      ) as { results?: SearchResult[]; count?: number; error?: string };

      if (response.error) {
        throw new Error(response.error);
      }

      return {
        results: response.results || [],
        count: response.count || 0,
      };
    } catch (error) {
      console.error("RAG search failed:", error);
      throw error;
    }
  }, [debouncedQuery, selectedProjects]);

  const {
    data: searchResponse,
    loading,
    error,
    refetch: search,
  } = useApi<{ results: SearchResult[]; count: number }>(performSearch, { initialData: { results: [], count: 0 } });

  const results = searchResponse?.results || [];

  // 触发搜索（当 debouncedQuery 变化时自动搜索）
  useEffect(() => {
    if (debouncedQuery.trim()) {
      search();
    }
  }, [debouncedQuery]);

  // 手动触发搜索
  const handleSearch = useCallback(() => {
    if (!query.trim()) {
      showToast(t("rag.search.queryRequired"), "error");
      return;
    }
    search();
  }, [query, search, showToast, t]);

  // 展开/收起结果
  const toggleExpand = useCallback((resultId: string) => {
    setExpandedResults((prev) => {
      const next = new Set(prev);
      if (next.has(resultId)) {
        next.delete(resultId);
      } else {
        next.add(resultId);
      }
      return next;
    });
  }, []);

  // 跳转到会话详情
  const handleResultClick = useCallback((sessionId: string) => {
    navigate(`/sessions/${sessionId}`);
  }, [navigate]);

  // 格式化时间
  const formatTime = useCallback((timestamp: number) => {
    if (!timestamp) return "";
    const date = new Date(timestamp);
    return date.toLocaleString();
  }, []);

  return (
    <div style={{ padding: "20px", maxWidth: "1200px", margin: "0 auto" }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "24px" }}>
        <h2 style={{ margin: 0 }}>{t("rag.search.title")}</h2>
        <button
          onClick={() => {
            const vscode = getVscodeApi();
            vscode.postMessage({
              command: "openRAGSearch",
              payload: { route: "/config" },
            });
          }}
          style={{
            padding: "8px 16px",
            backgroundColor: "var(--vscode-button-secondaryBackground)",
            color: "var(--vscode-button-secondaryForeground)",
            border: "1px solid var(--vscode-button-border)",
            cursor: "pointer",
          }}
        >
          ⚙️ {t("rag.config.title")}
        </button>
      </div>

      {/* 搜索框 */}
      <div style={{ marginBottom: "24px", display: "flex", gap: "8px" }}>
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyPress={(e) => e.key === "Enter" && handleSearch()}
          placeholder={t("rag.search.placeholder")}
          style={{
            flex: 1,
            padding: "10px",
            fontSize: "16px",
            border: "1px solid var(--vscode-input-border)",
            backgroundColor: "var(--vscode-input-background)",
            color: "var(--vscode-input-foreground)",
          }}
        />
        <button
          onClick={handleSearch}
          disabled={loading || !query.trim()}
          style={{
            padding: "10px 20px",
            backgroundColor: "var(--vscode-button-background)",
            color: "var(--vscode-button-foreground)",
            border: "none",
            cursor: loading || !query.trim() ? "not-allowed" : "pointer",
          }}
        >
          {loading ? t("common.loading") : t("common.search")}
        </button>
      </div>

      {/* 错误提示 */}
      {error && (
        <div style={{
          padding: "12px",
          marginBottom: "16px",
          backgroundColor: "var(--vscode-inputValidation-errorBackground)",
          color: "var(--vscode-errorForeground)",
          borderRadius: "4px",
        }}>
          {t("rag.search.error")}: {error instanceof Error ? error.message : String(error)}
        </div>
      )}

      {/* 搜索结果 */}
      {results.length > 0 && (
        <div style={{ marginTop: "24px" }}>
          <div style={{ marginBottom: "16px", color: "var(--vscode-descriptionForeground)" }}>
            {t("rag.search.found")} {results.length} {t("rag.search.results")}
          </div>

          <div style={{ display: "flex", flexDirection: "column", gap: "12px" }}>
            {results.map((result, index) => {
              const resultId = `${result.session_id}-${result.type}-${index}`;
              const isExpanded = expandedResults.has(resultId);
              const isTurn = result.type === "turn";

              return (
                <div
                  key={resultId}
                  style={{
                    padding: "16px",
                    border: "1px solid var(--vscode-panel-border)",
                    borderRadius: "4px",
                    backgroundColor: "var(--vscode-editor-background)",
                    cursor: "pointer",
                  }}
                  onClick={() => handleResultClick(result.session_id)}
                >
                  <div style={{ display: "flex", justifyContent: "space-between", marginBottom: "8px" }}>
                    <div>
                      <strong style={{ color: "var(--vscode-textLink-foreground)" }}>
                        {result.project_name || result.project_id}
                      </strong>
                      <span style={{ marginLeft: "8px", color: "var(--vscode-descriptionForeground)", fontSize: "12px" }}>
                        {formatTime(result.timestamp)}
                      </span>
                    </div>
                    <div style={{ color: "var(--vscode-descriptionForeground)", fontSize: "12px" }}>
                      {t("rag.search.score")}: {(result.score * 100).toFixed(1)}%
                    </div>
                  </div>

                  {isTurn ? (
                    <div>
                      <div style={{ marginBottom: "8px" }}>
                        <strong>{t("rag.search.user")}:</strong> {result.user_text}
                      </div>
                      <div>
                        <strong>{t("rag.search.ai")}:</strong> {result.ai_text}
                      </div>
                      {result.message_ids && result.message_ids.length > 0 && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            toggleExpand(resultId);
                          }}
                          style={{
                            marginTop: "8px",
                            padding: "4px 8px",
                            fontSize: "12px",
                            backgroundColor: "transparent",
                            border: "1px solid var(--vscode-button-border)",
                            color: "var(--vscode-foreground)",
                            cursor: "pointer",
                          }}
                        >
                          {isExpanded 
                            ? t("rag.search.collapse") 
                            : t("rag.search.expand")} ({result.message_ids.length} {t("rag.search.messages")})
                        </button>
                      )}
                    </div>
                  ) : (
                    <div>{result.content}</div>
                  )}

                  {isExpanded && result.message_ids && (
                    <div style={{ marginTop: "12px", paddingLeft: "16px", borderLeft: "2px solid var(--vscode-panel-border)" }}>
                      <div style={{ fontSize: "12px", color: "var(--vscode-descriptionForeground)", marginBottom: "8px" }}>
                        {t("rag.search.messagesInTurn")}:
                      </div>
                      {result.message_ids.map((msgId, idx) => (
                        <div key={idx} style={{ fontSize: "12px", marginBottom: "4px" }}>
                          • {msgId}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* 空状态 */}
      {!loading && !error && query && results.length === 0 && (
        <div style={{ textAlign: "center", padding: "40px", color: "var(--vscode-descriptionForeground)" }}>
          {t("rag.search.noResults")}
        </div>
      )}

      {/* Toast 提示 */}
      <ToastContainer toasts={toasts} />
    </div>
  );
};
