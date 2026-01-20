/**
 * RAG 搜索组件
 * 支持新的 KnowledgeChunk 模型
 */

import React, { useState, useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { apiService } from "../services/api";
import { getVscodeApi } from "../services/api";
import { useApi, useToast } from "../hooks";
import { ToastContainer } from "./shared";
import type { ChunkSearchResult } from "../types";

// 旧的搜索结果格式（兼容）
interface LegacySearchResult {
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

// 统一的搜索结果类型
type SearchResult = ChunkSearchResult | LegacySearchResult;

// 类型守卫
function isChunkResult(result: SearchResult): result is ChunkSearchResult {
  return 'chunk_id' in result;
}

// 项目信息接口
interface IndexedProject {
  project_id: string;
  project_name: string;
  chunk_count: number;
}

export const RAGSearch: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { showToast, toasts } = useToast();

  const [query, setQuery] = useState("");
  const [selectedProjects, setSelectedProjects] = useState<string[]>([]);
  const [expandedResults, setExpandedResults] = useState<Set<string>>(new Set());
  const [availableProjects, setAvailableProjects] = useState<IndexedProject[]>([]);
  const [loadingProjects, setLoadingProjects] = useState(true);

  // 加载已索引的项目列表
  useEffect(() => {
    const loadProjects = async () => {
      try {
        setLoadingProjects(true);
        const response = await apiService.getIndexedProjects();
        if (response.projects) {
          setAvailableProjects(response.projects);
        }
      } catch (error) {
        console.error("Failed to load indexed projects:", error);
      } finally {
        setLoadingProjects(false);
      }
    };
    loadProjects();
  }, []);

  // 处理项目选择变化
  const handleProjectToggle = useCallback((projectId: string) => {
    setSelectedProjects(prev => {
      if (prev.includes(projectId)) {
        return prev.filter(id => id !== projectId);
      }
      return [...prev, projectId];
    });
  }, []);

  // 清除所有项目过滤
  const handleClearProjectFilter = useCallback(() => {
    setSelectedProjects([]);
  }, []);

  // 搜索（优先使用新的 chunks 接口）
  const performSearch = useCallback(async () => {
    if (!query.trim()) {
      return { results: [], count: 0 };
    }

    try {
      // 优先尝试新的 chunks 接口
      const response = await apiService.searchChunks(
        query,
        selectedProjects.length > 0 ? selectedProjects : undefined,
        20
      ) as { results?: ChunkSearchResult[]; count?: number; error?: string };

      if (response.error) {
        // 如果新接口失败，回退到旧接口
        console.warn("Chunks search failed, falling back to legacy search:", response.error);
        return performLegacySearch();
      }

      return {
        results: response.results || [],
        count: response.count || 0,
      };
    } catch (error) {
      console.warn("Chunks search error, falling back to legacy:", error);
      return performLegacySearch();
    }
  }, [query, selectedProjects]);

  // 旧的搜索接口（兼容）
  const performLegacySearch = useCallback(async () => {
    try {
      const response = await apiService.searchRAG(
        query,
        selectedProjects.length > 0 ? selectedProjects : undefined,
        20
      ) as { results?: LegacySearchResult[]; count?: number; error?: string };

      if (response.error) {
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
  }, [query, selectedProjects, t]);

  const {
    data: searchResponse,
    loading,
    error,
    refetch: search,
  } = useApi<{ results: SearchResult[]; count: number }>(performSearch, { initialData: { results: [], count: 0 } });

  const results = searchResponse?.results || [];

  // 手动触发搜索（点击按钮或按回车）
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
        <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
          <h2>{t("rag.search.title")}</h2>
          <span
            className="cocursor-beta-badge"
            title={t("rag.betaTooltip")}
          >
            {t("rag.beta")}
          </span>
        </div>
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

      {/* 项目过滤器 */}
      <div className="cocursor-rag-project-filter">
        <div className="cocursor-rag-project-filter-header">
          <span className="cocursor-rag-project-filter-label">
            {t("rag.search.filterByProject")}:
          </span>
          {selectedProjects.length > 0 && (
            <button
              className="cocursor-rag-project-filter-clear"
              onClick={handleClearProjectFilter}
            >
              {t("rag.search.clearFilter")}
            </button>
          )}
        </div>
        <div className="cocursor-rag-project-filter-list">
          {loadingProjects ? (
            <span className="cocursor-rag-project-filter-loading">
              {t("common.loading")}...
            </span>
          ) : availableProjects.length === 0 ? (
            <span className="cocursor-rag-project-filter-empty">
              {t("rag.search.noProjectsIndexed")}
            </span>
          ) : (
            availableProjects.map((project) => (
              <button
                key={project.project_id}
                className={`cocursor-rag-project-filter-item ${
                  selectedProjects.includes(project.project_id) ? "selected" : ""
                }`}
                onClick={() => handleProjectToggle(project.project_id)}
                title={`${project.project_name} (${project.chunk_count} ${t("rag.search.chunks")})`}
              >
                <span className="cocursor-rag-project-filter-name">
                  {project.project_name.replace(/^Users-[^-]+-code-/, "")}
                </span>
                <span className="cocursor-rag-project-filter-count">
                  {project.chunk_count}
                </span>
              </button>
            ))
          )}
        </div>
        {selectedProjects.length > 0 && (
          <div className="cocursor-rag-project-filter-selected">
            {t("rag.search.selectedProjects")}: {selectedProjects.length}
          </div>
        )}
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
              // 处理新的 ChunkSearchResult 格式
              if (isChunkResult(result)) {
                const resultId = `chunk-${result.chunk_id}`;
                const isExpanded = expandedResults.has(resultId);

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
                        {result.is_enriched && (
                          <span className="cocursor-rag-enriched-badge" title={t("rag.search.enriched")}>
                            ✨
                          </span>
                        )}
                        {result.has_code && (
                          <span className="cocursor-rag-code-badge" title={t("rag.search.hasCode")}>
                            {"</>"}
                          </span>
                        )}
                      </div>
                      <div className="cocursor-rag-result-score">
                        {t("rag.search.score")}: {(result.score * 100).toFixed(1)}%
                      </div>
                    </div>

                    {/* 主题和摘要 */}
                    <div className="cocursor-rag-result-turn">
                      {result.main_topic && (
                        <div className="cocursor-rag-result-summary-topic">
                          {result.main_topic}
                        </div>
                      )}
                      <div className="cocursor-rag-result-summary-text">
                        {result.summary || result.user_query_preview}
                      </div>

                      {/* 工具标签 */}
                      {result.tools_used && result.tools_used.length > 0 && (
                        <div className="cocursor-rag-result-tools">
                          <span className="cocursor-rag-tools-label">{t("rag.search.toolsUsed")}:</span>
                          {result.tools_used.slice(0, 5).map((tool, idx) => (
                            <span key={idx} className="cocursor-rag-tool-tag">
                              {tool}
                            </span>
                          ))}
                        </div>
                      )}

                      {/* 修改的文件 */}
                      {result.files_modified && result.files_modified.length > 0 && (
                        <div className="cocursor-rag-result-files">
                          <span className="cocursor-rag-files-label">{t("rag.search.filesModified")}:</span>
                          {result.files_modified.slice(0, 3).map((file, idx) => (
                            <span key={idx} className="cocursor-rag-file-tag">
                              {file}
                            </span>
                          ))}
                        </div>
                      )}

                      {/* 标签 */}
                      {result.tags && result.tags.length > 0 && (
                        <div className="cocursor-rag-result-tags">
                          {result.tags.map((tag, idx) => (
                            <span key={idx} className="cocursor-rag-result-tag">
                              {tag}
                            </span>
                          ))}
                        </div>
                      )}

                      {/* 展开按钮 */}
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
                            : t("rag.search.viewOriginalConversation")}
                        </button>
                      </div>
                    </div>

                    {/* 展开的详情 */}
                    {isExpanded && (
                      <div className="cocursor-rag-result-expanded">
                        <div className="cocursor-rag-result-expanded-title">
                          {t("rag.search.user")}:
                        </div>
                        <div className="cocursor-rag-result-expanded-content">
                          {result.user_query_preview}
                        </div>
                      </div>
                    )}
                  </div>
                );
              }

              // 处理旧的 LegacySearchResult 格式
              const legacyResult = result as LegacySearchResult;
              const resultId = `${legacyResult.session_id}-${legacyResult.type}-${index}`;
              const isExpanded = expandedResults.has(resultId);
              const isTurn = legacyResult.type === "turn";

              return (
                <div
                  key={resultId}
                  className="cocursor-rag-result-item"
                  onClick={() => handleResultClick(legacyResult.session_id)}
                >
                  <div className="cocursor-rag-result-header">
                    <div>
                      <strong className="cocursor-rag-result-project">
                        {legacyResult.project_name || legacyResult.project_id}
                      </strong>
                      <span className="cocursor-rag-result-meta">
                        {formatTime(legacyResult.timestamp)}
                      </span>
                    </div>
                    <div className="cocursor-rag-result-score">
                      {t("rag.search.score")}: {(legacyResult.score * 100).toFixed(1)}%
                    </div>
                  </div>

                  {isTurn && legacyResult.summary ? (() => {
                    const summaryData = parseSummary(legacyResult.summary);
                    if (!summaryData) return null;

                    return (
                      <div className="cocursor-rag-result-turn">
                        <div className="cocursor-rag-result-summary">
                          <div className="cocursor-rag-result-summary-topic">
                            {summaryData.main_topic}
                          </div>
                          <div className="cocursor-rag-result-summary-text">
                            {summaryData.summary}
                          </div>
                          {summaryData.key_points && summaryData.key_points.length > 0 && (
                            <div className="cocursor-rag-result-key-points">
                              <strong>{t("rag.search.keyPoints")}:</strong>
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
                                <span key={idx} className="cocursor-rag-result-tag">
                                  {tag}
                                </span>
                              ))}
                            </div>
                          )}
                        </div>
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
                              : t("rag.search.viewOriginalConversation")}
                          </button>
                        </div>
                      </div>
                    );
                  })() : isTurn ? (
                    <div className="cocursor-rag-result-turn">
                      <div className="cocursor-rag-result-message">
                        <strong>{t("rag.search.user")}:</strong> {legacyResult.user_text}
                      </div>
                      <div className="cocursor-rag-result-message">
                        <strong>{t("rag.search.ai")}:</strong> {legacyResult.ai_text}
                      </div>
                    </div>
                  ) : (
                    <div className="cocursor-rag-result-content">{legacyResult.content}</div>
                  )}

                  {isExpanded && legacyResult.message_ids && (
                    <div className="cocursor-rag-result-expanded">
                      <div className="cocursor-rag-result-expanded-title">
                        {t("rag.search.messagesInTurn")}:
                      </div>
                      {legacyResult.message_ids.map((msgId, idx) => (
                        <div key={idx} className="cocursor-rag-result-expanded-list">
                          • {msgId}
                        </div>
                      ))}
                    </div>
                  )}
                  {isExpanded && !legacyResult.message_ids && legacyResult.user_text && legacyResult.ai_text && (
                    <div className="cocursor-rag-result-expanded">
                      <div className="cocursor-rag-result-expanded-title">
                        {t("rag.search.user")}:
                      </div>
                      <div className="cocursor-rag-result-expanded-content">
                        {legacyResult.user_text}
                      </div>
                      <div className="cocursor-rag-result-expanded-title">
                        {t("rag.search.ai")}:
                      </div>
                      <div className="cocursor-rag-result-expanded-content">
                        {legacyResult.ai_text}
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
