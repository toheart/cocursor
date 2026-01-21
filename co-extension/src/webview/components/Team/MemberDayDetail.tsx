/**
 * 成员日详情弹窗组件
 */

import React from "react";
import { useTranslation } from "react-i18next";
import { MemberDailyDetail, ProjectGitStats, WorkItemSummary } from "../../types";

interface MemberDayDetailModalProps {
  detail: MemberDailyDetail | null;
  loading: boolean;
  date: string;
  teamId: string;
  onClose: () => void;
}

// 格式化日期
function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString(undefined, {
    year: "numeric",
    month: "long",
    day: "numeric",
    weekday: "long",
  });
}

// 工作类型标签颜色
const CATEGORY_COLORS: Record<string, string> = {
  requirements_discussion: "#6366f1",
  coding: "#22c55e",
  problem_solving: "#f59e0b",
  refactoring: "#8b5cf6",
  code_review: "#ec4899",
  documentation: "#3b82f6",
  testing: "#14b8a6",
  other: "#6b7280",
};

export const MemberDayDetailModal: React.FC<MemberDayDetailModalProps> = ({
  detail,
  loading,
  date,
  onClose,
}) => {
  const { t } = useTranslation();

  return (
    <div className="cocursor-modal-overlay" onClick={onClose}>
      <div
        className="cocursor-modal cocursor-member-day-detail-modal"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="cocursor-modal-header">
          <div className="cocursor-member-day-detail-title">
            {loading ? (
              <h2>{t("common.loading")}</h2>
            ) : detail ? (
              <>
                <div className="cocursor-member-day-detail-avatar">
                  {detail.member_name.charAt(0).toUpperCase()}
                </div>
                <div className="cocursor-member-day-detail-info">
                  <h2>{detail.member_name}</h2>
                  <span className="cocursor-member-day-detail-date">
                    {formatDate(detail.date)}
                  </span>
                  <div className="cocursor-member-day-detail-badges">
                    {detail.is_online ? (
                      <span className="cocursor-badge online">{t("team.online")}</span>
                    ) : (
                      <span className="cocursor-badge offline">{t("team.offline")}</span>
                    )}
                    {detail.is_cached && (
                      <span className="cocursor-badge cached">{t("weeklyReport.cached")}</span>
                    )}
                    {detail.has_report && (
                      <span className="cocursor-badge report">{t("weeklyReport.hasReport")}</span>
                    )}
                  </div>
                </div>
              </>
            ) : (
              <h2>{formatDate(date)}</h2>
            )}
          </div>
          <button className="cocursor-modal-close" onClick={onClose}>
            ×
          </button>
        </div>

        <div className="cocursor-modal-body">
          {loading ? (
            <div className="cocursor-team-loading">
              <div className="cocursor-team-loading-spinner"></div>
            </div>
          ) : !detail ? (
            <div className="cocursor-team-empty-section">
              <span className="cocursor-team-empty-icon">📭</span>
              <span>{t("weeklyReport.noDetailData")}</span>
            </div>
          ) : (
            <div className="cocursor-member-day-detail-content">
              {/* Git 活动区 */}
              {detail.git_stats && (
                <section className="cocursor-member-day-detail-section">
                  <h3 className="cocursor-member-day-detail-section-title">
                    <span className="cocursor-section-icon">💻</span>
                    {t("weeklyReport.gitActivity")}
                  </h3>
                  
                  {/* Git 总览 */}
                  <div className="cocursor-member-day-detail-stats">
                    <div className="cocursor-stat-item">
                      <span className="cocursor-stat-value">{detail.git_stats.total_commits}</span>
                      <span className="cocursor-stat-label">{t("weeklyReport.commits")}</span>
                    </div>
                    <div className="cocursor-stat-item positive">
                      <span className="cocursor-stat-value">+{detail.git_stats.total_added}</span>
                      <span className="cocursor-stat-label">{t("weeklyReport.linesAdded")}</span>
                    </div>
                    <div className="cocursor-stat-item negative">
                      <span className="cocursor-stat-value">-{detail.git_stats.total_removed}</span>
                      <span className="cocursor-stat-label">{t("weeklyReport.linesRemoved")}</span>
                    </div>
                  </div>

                  {/* 项目细分 */}
                  {detail.git_stats.projects && detail.git_stats.projects.length > 0 && (
                    <div className="cocursor-member-day-detail-projects">
                      {detail.git_stats.projects.map((project, idx) => (
                        <ProjectGitDetail key={idx} project={project} />
                      ))}
                    </div>
                  )}
                </section>
              )}

              {/* Cursor 统计区 */}
              {detail.cursor_stats && (
                <section className="cocursor-member-day-detail-section">
                  <h3 className="cocursor-member-day-detail-section-title">
                    <span className="cocursor-section-icon">🤖</span>
                    {t("weeklyReport.cursorStats")}
                  </h3>
                  <div className="cocursor-member-day-detail-stats">
                    <div className="cocursor-stat-item">
                      <span className="cocursor-stat-value">{detail.cursor_stats.session_count}</span>
                      <span className="cocursor-stat-label">{t("weeklyReport.sessions")}</span>
                    </div>
                    <div className="cocursor-stat-item">
                      <span className="cocursor-stat-value">{detail.cursor_stats.tokens_used.toLocaleString()}</span>
                      <span className="cocursor-stat-label">{t("weeklyReport.tokens")}</span>
                    </div>
                    <div className="cocursor-stat-item positive">
                      <span className="cocursor-stat-value">+{detail.cursor_stats.lines_added}</span>
                      <span className="cocursor-stat-label">{t("weeklyReport.aiAdded")}</span>
                    </div>
                    <div className="cocursor-stat-item negative">
                      <span className="cocursor-stat-value">-{detail.cursor_stats.lines_removed}</span>
                      <span className="cocursor-stat-label">{t("weeklyReport.aiRemoved")}</span>
                    </div>
                  </div>
                </section>
              )}

              {/* 工作内容区 */}
              {detail.work_items && detail.work_items.length > 0 && (
                <section className="cocursor-member-day-detail-section">
                  <h3 className="cocursor-member-day-detail-section-title">
                    <span className="cocursor-section-icon">📋</span>
                    {t("weeklyReport.workItems")}
                  </h3>
                  <div className="cocursor-member-day-detail-work-items">
                    {detail.work_items.map((item, idx) => (
                      <WorkItemCard key={idx} item={item} />
                    ))}
                  </div>
                </section>
              )}

              {/* 无数据提示 */}
              {!detail.git_stats && !detail.cursor_stats && (!detail.work_items || detail.work_items.length === 0) && (
                <div className="cocursor-team-empty-section">
                  <span className="cocursor-team-empty-icon">📭</span>
                  <span>{t("weeklyReport.noActivityData")}</span>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

// 项目 Git 详情组件
interface ProjectGitDetailProps {
  project: ProjectGitStats;
}

const ProjectGitDetail: React.FC<ProjectGitDetailProps> = ({ project }) => {
  const { t } = useTranslation();
  const [expanded, setExpanded] = React.useState(false);

  return (
    <div className="cocursor-project-git-detail">
      <div
        className="cocursor-project-git-header"
        onClick={() => setExpanded(!expanded)}
      >
        <span className="cocursor-project-git-name">{project.project_name}</span>
        <div className="cocursor-project-git-summary">
          <span className="cocursor-project-git-commits">
            {project.commits} {t("weeklyReport.commits")}
          </span>
          <span className="cocursor-project-git-lines">
            <span className="positive">+{project.lines_added}</span>
            <span className="negative">-{project.lines_removed}</span>
          </span>
          <span className={`cocursor-expand-icon ${expanded ? "expanded" : ""}`}>▶</span>
        </div>
      </div>

      {expanded && project.commit_messages && project.commit_messages.length > 0 && (
        <div className="cocursor-project-git-commits-list">
          {project.commit_messages.map((commit, idx) => (
            <div key={idx} className="cocursor-commit-item">
              <span className="cocursor-commit-hash">{commit.hash}</span>
              <span className="cocursor-commit-message">{commit.message}</span>
              <span className="cocursor-commit-files">
                {commit.files_count} {t("weeklyReport.files")}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

// 工作条目卡片
interface WorkItemCardProps {
  item: WorkItemSummary;
}

const WorkItemCard: React.FC<WorkItemCardProps> = ({ item }) => {
  const { t } = useTranslation();
  const categoryColor = CATEGORY_COLORS[item.category] || CATEGORY_COLORS.other;

  return (
    <div className="cocursor-work-item-card">
      <div className="cocursor-work-item-header">
        <span
          className="cocursor-work-item-category"
          style={{ backgroundColor: categoryColor }}
        >
          {t(`weeklyReport.category.${item.category}`)}
        </span>
        <span className="cocursor-work-item-project">{item.project}</span>
      </div>
      <p className="cocursor-work-item-description">{item.description}</p>
    </div>
  );
};

export default MemberDayDetailModal;
