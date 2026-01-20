/**
 * 团队日报 Tab 组件
 */

import React, { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { apiService } from "../../services/api";
import { TeamDailySummary } from "../../types";
import { useApi, useToast } from "../../hooks";
import { ToastContainer } from "../shared/ToastContainer";

interface DailyReportTabProps {
  teamId: string;
  onRefresh?: () => void;
}

export const DailyReportTab: React.FC<DailyReportTabProps> = ({ teamId, onRefresh }) => {
  const { t } = useTranslation();
  const { showToast, toasts } = useToast();
  const [selectedDate, setSelectedDate] = useState(() => new Date().toISOString().split("T")[0]);
  const [selectedSummary, setSelectedSummary] = useState<TeamDailySummary | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  // 获取日报列表
  const fetchSummaries = useCallback(async () => {
    const resp = await apiService.getTeamDailySummaries(teamId, selectedDate) as { summaries: TeamDailySummary[] };
    return resp.summaries || [];
  }, [teamId, selectedDate]);

  const { data: summaries, loading, refetch } = useApi<TeamDailySummary[]>(fetchSummaries);

  // 分享我的日报
  const handleShare = useCallback(async () => {
    try {
      await apiService.shareTeamDailySummary(teamId, selectedDate);
      showToast(t("team.shareDailySummarySuccess"), "success");
      refetch();
      onRefresh?.();
    } catch (err: any) {
      showToast(err.message || t("team.shareDailySummaryFailed"), "error");
    }
  }, [teamId, selectedDate, showToast, refetch, onRefresh, t]);

  // 查看日报详情
  const handleViewDetail = useCallback(async (summary: TeamDailySummary) => {
    setDetailLoading(true);
    try {
      const detail = await apiService.getTeamDailySummaryDetail(
        teamId,
        summary.member_id,
        summary.date
      ) as TeamDailySummary;
      setSelectedSummary(detail);
    } catch (err: any) {
      showToast(err.message || t("team.fetchDailySummaryFailed"), "error");
    } finally {
      setDetailLoading(false);
    }
  }, [teamId, showToast, t]);

  // 日期变更
  const handleDateChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSelectedDate(e.target.value);
    setSelectedSummary(null);
  }, []);

  // 格式化时间
  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  };

  return (
    <div className="cocursor-team-daily-report">
      <ToastContainer toasts={toasts} />

      {/* 头部操作栏 */}
      <div className="cocursor-team-daily-report-header">
        <div className="cocursor-team-daily-report-date-picker">
          <label>{t("team.date")}:</label>
          <input
            type="date"
            value={selectedDate}
            onChange={handleDateChange}
            max={new Date().toISOString().split("T")[0]}
          />
        </div>
        <div className="cocursor-team-daily-report-actions">
          <button className="cocursor-btn secondary" onClick={refetch}>
            {t("common.refresh")}
          </button>
          <button className="cocursor-btn primary" onClick={handleShare}>
            <span className="cocursor-btn-icon">📤</span>
            {t("team.shareMyDailySummary")}
          </button>
        </div>
      </div>

      {/* 日报列表 */}
      <div className="cocursor-team-daily-report-list">
        {loading ? (
          <div className="cocursor-team-loading">
            <div className="cocursor-team-loading-spinner"></div>
          </div>
        ) : summaries?.length === 0 ? (
          <div className="cocursor-team-empty-section">
            <span className="cocursor-team-empty-icon">📝</span>
            <span>{t("team.noDailySummaries")}</span>
            <p>{t("team.noDailySummariesDesc")}</p>
          </div>
        ) : (
          summaries?.map((summary) => (
            <DailyReportCard
              key={`${summary.member_id}-${summary.date}`}
              summary={summary}
              onViewDetail={() => handleViewDetail(summary)}
              loading={detailLoading && selectedSummary?.member_id === summary.member_id}
            />
          ))
        )}
      </div>

      {/* 日报详情弹窗 */}
      {selectedSummary && selectedSummary.summary && (
        <DailyReportDetailModal
          summary={selectedSummary}
          onClose={() => setSelectedSummary(null)}
        />
      )}
    </div>
  );
};

// 日报卡片组件
interface DailyReportCardProps {
  summary: TeamDailySummary;
  onViewDetail: () => void;
  loading?: boolean;
}

const DailyReportCard: React.FC<DailyReportCardProps> = ({ summary, onViewDetail, loading }) => {
  const { t } = useTranslation();

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  };

  // 从 Markdown 中提取纯文本预览
  const getPreviewText = (markdown: string | undefined, maxLength: number = 120) => {
    if (!markdown) return "";
    // 移除 Markdown 语法，获取纯文本
    const plainText = markdown
      .replace(/#{1,6}\s+/g, "") // 移除标题标记
      .replace(/\*\*([^*]+)\*\*/g, "$1") // 移除粗体
      .replace(/\*([^*]+)\*/g, "$1") // 移除斜体
      .replace(/`([^`]+)`/g, "$1") // 移除行内代码
      .replace(/```[\s\S]*?```/g, "") // 移除代码块
      .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1") // 移除链接，保留文本
      .replace(/[-*+]\s+/g, "") // 移除列表标记
      .replace(/\n+/g, " ") // 换行替换为空格
      .trim();
    return plainText.length > maxLength ? plainText.slice(0, maxLength) + "..." : plainText;
  };

  // 从 Markdown 中提取要点（获取前几个列表项或标题）
  const extractHighlights = (markdown: string | undefined, maxItems: number = 3) => {
    if (!markdown) return [];
    const highlights: string[] = [];
    
    // 尝试匹配列表项
    const listItemRegex = /[-*+]\s+(.+)/g;
    let match;
    while ((match = listItemRegex.exec(markdown)) !== null && highlights.length < maxItems) {
      const item = match[1]
        .replace(/\*\*([^*]+)\*\*/g, "$1")
        .replace(/\*([^*]+)\*/g, "$1")
        .replace(/`([^`]+)`/g, "$1")
        .trim();
      if (item.length > 0 && item.length < 80) {
        highlights.push(item);
      }
    }
    
    return highlights;
  };

  const highlights = extractHighlights(summary.summary);
  const previewText = highlights.length === 0 ? getPreviewText(summary.summary) : "";

  return (
    <div className="cocursor-team-daily-report-card" onClick={onViewDetail}>
      <div className="cocursor-team-daily-report-card-header">
        <div className="cocursor-team-daily-report-card-avatar">
          {summary.member_name.charAt(0).toUpperCase()}
        </div>
        <div className="cocursor-team-daily-report-card-info">
          <h4 className="cocursor-team-daily-report-card-name">{summary.member_name}</h4>
          <div className="cocursor-team-daily-report-card-meta">
            <span className="cocursor-team-daily-report-card-stat">
              <span className="cocursor-stat-icon">💬</span>
              {summary.total_sessions} {t("team.sessions")}
            </span>
            <span className="cocursor-team-daily-report-card-stat">
              <span className="cocursor-stat-icon">📁</span>
              {summary.project_count} {t("team.projects")}
            </span>
          </div>
        </div>
        <div className="cocursor-team-daily-report-card-time">
          <span className="cocursor-time-icon">🕐</span>
          {formatTime(summary.shared_at)}
        </div>
      </div>
      
      {/* 内容预览区域 */}
      {(highlights.length > 0 || previewText) && (
        <div className="cocursor-team-daily-report-card-preview">
          {highlights.length > 0 ? (
            <ul className="cocursor-team-daily-report-card-highlights">
              {highlights.map((item, idx) => (
                <li key={idx}>{item}</li>
              ))}
            </ul>
          ) : (
            <p className="cocursor-team-daily-report-card-excerpt">{previewText}</p>
          )}
        </div>
      )}

      <div className="cocursor-team-daily-report-card-footer">
        <button
          className="cocursor-btn-text"
          onClick={(e) => {
            e.stopPropagation();
            onViewDetail();
          }}
          disabled={loading}
        >
          {loading ? (
            <>
              <span className="cocursor-btn-spinner"></span>
              {t("common.loading")}
            </>
          ) : (
            <>
              {t("team.viewDetail")}
              <span className="cocursor-btn-arrow">→</span>
            </>
          )}
        </button>
      </div>
    </div>
  );
};

// 日报详情弹窗
interface DailyReportDetailModalProps {
  summary: TeamDailySummary;
  onClose: () => void;
}

const DailyReportDetailModal: React.FC<DailyReportDetailModalProps> = ({ summary, onClose }) => {
  const { t } = useTranslation();

  // 格式化日期显示
  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString(undefined, {
      year: "numeric",
      month: "long",
      day: "numeric",
      weekday: "long"
    });
  };

  return (
    <div className="cocursor-modal-overlay" onClick={onClose}>
      <div className="cocursor-modal cocursor-daily-report-modal" onClick={(e) => e.stopPropagation()}>
        <div className="cocursor-modal-header">
          <div className="cocursor-daily-report-modal-title">
            <div className="cocursor-daily-report-modal-avatar">
              {summary.member_name.charAt(0).toUpperCase()}
            </div>
            <div className="cocursor-daily-report-modal-info">
              <h2>{summary.member_name}</h2>
              <span className="cocursor-daily-report-modal-date">{formatDate(summary.date)}</span>
            </div>
          </div>
          <button className="cocursor-modal-close" onClick={onClose}>×</button>
        </div>
        <div className="cocursor-modal-body">
          <div className="cocursor-daily-report-content">
            {/* 统计信息卡片 */}
            <div className="cocursor-daily-report-stats">
              <div className="cocursor-daily-report-stat">
                <span className="cocursor-daily-report-stat-icon">💬</span>
                <span className="cocursor-daily-report-stat-value">{summary.total_sessions}</span>
                <span className="cocursor-daily-report-stat-label">{t("team.sessions")}</span>
              </div>
              <div className="cocursor-daily-report-stat">
                <span className="cocursor-daily-report-stat-icon">📁</span>
                <span className="cocursor-daily-report-stat-value">{summary.project_count}</span>
                <span className="cocursor-daily-report-stat-label">{t("team.projects")}</span>
              </div>
            </div>
            {/* Markdown 渲染区域 */}
            <div className="cocursor-daily-report-markdown-container">
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                components={{
                  // 标题样式
                  h1: ({ children }) => <h1 className="cocursor-md-h1">{children}</h1>,
                  h2: ({ children }) => <h2 className="cocursor-md-h2">{children}</h2>,
                  h3: ({ children }) => <h3 className="cocursor-md-h3">{children}</h3>,
                  h4: ({ children }) => <h4 className="cocursor-md-h4">{children}</h4>,
                  // 段落
                  p: ({ children }) => <p className="cocursor-md-p">{children}</p>,
                  // 列表
                  ul: ({ children }) => <ul className="cocursor-md-ul">{children}</ul>,
                  ol: ({ children }) => <ol className="cocursor-md-ol">{children}</ol>,
                  li: ({ children }) => <li className="cocursor-md-li">{children}</li>,
                  // 代码
                  code: ({ className, children, ...props }) => {
                    const isInline = !className;
                    return isInline ? (
                      <code className="cocursor-md-code-inline" {...props}>{children}</code>
                    ) : (
                      <code className={`cocursor-md-code-block ${className || ""}`} {...props}>{children}</code>
                    );
                  },
                  pre: ({ children }) => <pre className="cocursor-md-pre">{children}</pre>,
                  // 引用
                  blockquote: ({ children }) => <blockquote className="cocursor-md-blockquote">{children}</blockquote>,
                  // 链接
                  a: ({ href, children }) => (
                    <a href={href} className="cocursor-md-link" target="_blank" rel="noopener noreferrer">{children}</a>
                  ),
                  // 强调
                  strong: ({ children }) => <strong className="cocursor-md-strong">{children}</strong>,
                  em: ({ children }) => <em className="cocursor-md-em">{children}</em>,
                  // 分割线
                  hr: () => <hr className="cocursor-md-hr" />,
                  // 表格
                  table: ({ children }) => <table className="cocursor-md-table">{children}</table>,
                  thead: ({ children }) => <thead className="cocursor-md-thead">{children}</thead>,
                  tbody: ({ children }) => <tbody className="cocursor-md-tbody">{children}</tbody>,
                  tr: ({ children }) => <tr className="cocursor-md-tr">{children}</tr>,
                  th: ({ children }) => <th className="cocursor-md-th">{children}</th>,
                  td: ({ children }) => <td className="cocursor-md-td">{children}</td>,
                }}
              >
                {summary.summary || ""}
              </ReactMarkdown>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default DailyReportTab;
