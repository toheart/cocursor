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
  summary?: {
    main_topic: string;
    problem: string;
    solution: string;
    tech_stack: string[];
    code_snippets: string[];
    key_points: string[];
    lessons: string[];
    tags: string[];
    summary: string;
    context: string;
  };
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
        // 检查是否是 RAG 未配置错误
        if (response.error.includes("not initialized") || 
            response.error.includes("not configured") ||
            response.error.includes("Please configure")) {
          throw new Error(t("rag.search.notConfigured"));
        }
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
  }, [debouncedQuery, selectedProjects, t]);

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

  // 解析 summary JSON 字符串
  const parseSummary = useCallback((summaryStr: unknown) => {
    if (!summaryStr || typeof summaryStr !== 'string') return null;
    try {
      return JSON.parse(summaryStr) as SearchResult['summary'];
    } catch (error) {
      console.error("Failed to parse summary:", error);
      return null;
    }
  }, []);

  // 格式化时间
  const formatTime = useCallback((timestamp: number) => {
    if (!timestamp) return "";
    const date = new Date(timestamp);
    return date.toLocaleString();
  }, []);

  return (
    <div className="cocursor-rag-search">
      <div className="cocursor-rag-search-header">
        <h2>{t("rag.search.title")}</h2>
        <button
          className="cocursor-rag-config-button secondary"
          onClick={() => {
            const vscode = getVscodeApi();
            vscode.postMessage({
              command: "openRAGSearch",
              payload: { route: "/config" },
            });
          }}
        >
          ⚙️ {t("rag.config.title")}
        </button>
      </div>

      {/* 搜索框 */}
      <div className="cocursor-rag-search-box">
        <input
          type="text"
          className="cocursor-rag-search-input"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyPress={(e) => e.key === "Enter" && handleSearch()}
          placeholder={t("rag.search.placeholder")}
        />
        <button
          className="cocursor-rag-search-button"
          onClick={handleSearch}
          disabled={loading || !query.trim()}
        >
          {loading ? t("common.loading") : t("common.search")}
        </button>
      </div>

      {/* 错误提示 */}
      {error && (
        <div className="cocursor-error">
          {t("rag.search.error")}: {(error as any)?.message || String(error)}
        </div>
      )}

      {/* 搜索结果 */}
      {results.length > 0 && (
        <div className="cocursor-rag-results">
          <div className="cocursor-rag-results-header">
            {t("rag.search.found")} {results.length} {t("rag.search.results")}
          </div>

          <div className="cocursor-rag-results-list">
            {results.map((result, index) => {
              const resultId = `${result.session_id}-${result.type}-${index}`;
              const isExpanded = expandedResults.has(resultId);
              const isTurn = result.type === "turn";

              return (
                <div
                  key={resultId}
                  className="cocursor-rag-result-item"
                  onClick={() => handleResultClick(result.session_id)}
                >
                  <div className="cocursor-rag-result-header">
                    <div>
                      <strong className="cocursor-rag-result-project">
                        {result.project_name || result.project_id}
                      </strong>
                      <span className="cocursor-rag-result-meta">
                        {formatTime(result.timestamp)}
                      </span>
                    </div>
                    <div className="cocursor-rag-result-score">
                      {t("rag.search.score")}: {(result.score * 100).toFixed(1)}%
                    </div>
                  </div>

                  {isTurn && result.summary ? (() => {
                    const summaryData = parseSummary(result.summary);
                    if (!summaryData) return null;

                    return (
                      <div className="cocursor-rag-result-turn">
                        {/* 总结信息 */}
                        <div className="cocursor-rag-result-summary">
                          <div className="cocursor-rag-result-summary-topic">
                            {summaryData.main_topic}
                          </div>
                          <div className="cocursor-rag-result-summary-text">
                            {summaryData.summary}
                          </div>
                          {summaryData.key_points && summaryData.key_points.length > 0 && (
                            <div className="cocursor-rag-result-key-points">
                              <strong>关键知识点:</strong>
                              <ul>
                                {summaryData.key_points.map((point: string, idx: number) => (
                                  <li key={idx}>{point}</li>
                                ))}
                              </ul>
                            </div>
                          )}
                          {summaryData.tags && summaryData.tags.length > 0 && (
                            <div className="cocursor-rag-result-tags">
                              {summaryData.tags.map((tag: string, idx: number) => (
                                <span
                                  key={idx}
                                  className="cocursor-rag-result-tag"
                                >
                                  {tag}
                                </span>
                              ))}
                            </div>
                          )}
                        </div>
                        {/* 查看原始对话按钮 */}
                        <div>
                          <button
                            className="cocursor-rag-result-expand-button"
                            onClick={(e) => {
                              e.stopPropagation();
                              toggleExpand(resultId);
                            }}
                          >
                            {isExpanded
                              ? t("rag.search.collapse")
                              : "查看原始对话"}
                          </button>
                        </div>
                      </div>
                    );
                  })() : isTurn ? (
                    <div className="cocursor-rag-result-turn">
                      <div className="cocursor-rag-result-message">
                        <strong>{t("rag.search.user")}:</strong> {result.user_text}
                      </div>
                      <div className="cocursor-rag-result-message">
                        <strong>{t("rag.search.ai")}:</strong> {result.ai_text}
                      </div>
                    </div>
                  ) : (
                    <div className="cocursor-rag-result-content">{result.content}</div>
                  )}

                  {isExpanded && result.message_ids && (
                    <div className="cocursor-rag-result-expanded">
                      <div className="cocursor-rag-result-expanded-title">
                        {t("rag.search.messagesInTurn")}:
                      </div>
                      {result.message_ids.map((msgId, idx) => (
                        <div key={idx} className="cocursor-rag-result-expanded-list">
                          • {msgId}
                        </div>
                      ))}
                    </div>
                  )}
                  {isExpanded && !result.message_ids && result.user_text && result.ai_text && (
                    <div className="cocursor-rag-result-expanded">
                      <div className="cocursor-rag-result-expanded-title">
                        {t("rag.search.user")}:
                      </div>
                      <div className="cocursor-rag-result-expanded-content">
                        {result.user_text}
                      </div>
                      <div className="cocursor-rag-result-expanded-title">
                        {t("rag.search.ai")}:
                      </div>
                      <div className="cocursor-rag-result-expanded-content">
                        {result.ai_text}
                      </div>
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
        <div className="cocursor-rag-empty">
          {t("rag.search.noResults")}
        </div>
      )}

      {/* Toast 提示 */}
      <ToastContainer toasts={toasts} />
    </div>
  );
};
