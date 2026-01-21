import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { ProjectSummary } from "../../services/api";

interface ProjectDetailsProps {
  projects: ProjectSummary[];
  screenshotMode?: boolean; // 截图模式时自动展开
}

// 工作分类图标映射
const CATEGORY_ICONS: Record<string, string> = {
  coding: "💻",
  problem_solving: "🔍",
  refactoring: "♻️",
  code_review: "👀",
  documentation: "📝",
  testing: "🧪",
  requirements_discussion: "💬",
  other: "📌",
};

/**
 * 项目详情组件
 * 展示各项目的工作项、会话列表、活跃时段
 */
export const ProjectDetails: React.FC<ProjectDetailsProps> = ({ projects, screenshotMode = false }) => {
  const { t } = useTranslation();
  const [expandedProjects, setExpandedProjects] = useState<Set<string>>(new Set());

  // 截图模式时自动展开所有项目
  useEffect(() => {
    if (screenshotMode) {
      setExpandedProjects(new Set(projects.map((p) => p.project_name)));
    }
  }, [screenshotMode, projects]);

  if (!projects || projects.length === 0) return null;

  const toggleProject = (projectName: string) => {
    if (screenshotMode) return; // 截图模式不允许折叠
    setExpandedProjects((prev) => {
      const next = new Set(prev);
      if (next.has(projectName)) {
        next.delete(projectName);
      } else {
        next.add(projectName);
      }
      return next;
    });
  };

  // 格式化活跃时段
  const formatActiveHours = (hours: number[]): string => {
    if (!hours || hours.length === 0) return "-";
    // 合并连续时段
    const sorted = [...hours].sort((a, b) => a - b);
    const ranges: string[] = [];
    let start = sorted[0];
    let end = sorted[0];

    for (let i = 1; i <= sorted.length; i++) {
      if (i < sorted.length && sorted[i] === end + 1) {
        end = sorted[i];
      } else {
        ranges.push(start === end ? `${start}:00` : `${start}:00-${end + 1}:00`);
        if (i < sorted.length) {
          start = sorted[i];
          end = sorted[i];
        }
      }
    }
    return ranges.join(", ");
  };

  // 格式化时长（毫秒转分钟）
  const formatDuration = (ms: number): string => {
    const minutes = Math.round(ms / 60000);
    if (minutes < 60) return `${minutes}${t("dailyReport.minutesShort")}`;
    const hours = Math.floor(minutes / 60);
    const remainMins = minutes % 60;
    return remainMins > 0 ? `${hours}${t("dailyReport.hoursShort")}${remainMins}${t("dailyReport.minutesShort")}` : `${hours}${t("dailyReport.hoursShort")}`;
  };

  return (
    <div className="cocursor-daily-report-section">
      <h4 className="cocursor-daily-report-section-title">
        <span className="section-icon">📁</span>
        {t("dailyReport.projectDetails")}
      </h4>
      <div className="cocursor-project-details-list">
        {projects.map((project) => {
          const isExpanded = expandedProjects.has(project.project_name);
          return (
            <div key={project.project_name} className="cocursor-project-detail-card">
              <div
                className="cocursor-project-detail-header"
                onClick={() => toggleProject(project.project_name)}
              >
                <div className="project-info">
                  <span className="project-icon">📦</span>
                  <span className="project-name">{project.project_name}</span>
                  <span className="project-session-count">
                    ({project.session_count} {t("dailyReport.sessionsShort")})
                  </span>
                </div>
                {!screenshotMode && (
                  <span className={`expand-icon ${isExpanded ? "expanded" : ""}`}>
                    ▼
                  </span>
                )}
              </div>
              {isExpanded && (
                <div className="cocursor-project-detail-content">
                  {/* 工作项列表 */}
                  {project.work_items && project.work_items.length > 0 && (
                    <div className="project-work-items">
                      <div className="work-items-label">{t("dailyReport.workItems")}:</div>
                      <ul className="work-items-list">
                        {project.work_items.map((item, idx) => (
                          <li key={idx} className="work-item">
                            <span className="work-item-category">
                              {CATEGORY_ICONS[item.category] || "📌"}
                            </span>
                            <span className="work-item-description">{item.description}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}
                  {/* 活跃时段 */}
                  {project.active_hours && project.active_hours.length > 0 && (
                    <div className="project-active-hours">
                      <span className="active-hours-label">{t("dailyReport.activeHours")}:</span>
                      <span className="active-hours-value">
                        {formatActiveHours(project.active_hours)}
                      </span>
                    </div>
                  )}
                  {/* 代码变更 */}
                  {project.code_changes && (
                    <div className="project-code-changes">
                      <span className="code-changes-label">{t("dailyReport.codeChanges")}:</span>
                      <span className="code-changes-added">+{project.code_changes.lines_added}</span>
                      <span className="code-changes-removed">-{project.code_changes.lines_removed}</span>
                      <span className="code-changes-files">
                        {project.code_changes.files_changed} {t("dailyReport.files")}
                      </span>
                    </div>
                  )}
                  {/* 会话列表 */}
                  {project.sessions && project.sessions.length > 0 && (
                    <div className="project-sessions">
                      <div className="sessions-label">{t("dailyReport.relatedSessions")}:</div>
                      <ul className="sessions-list">
                        {project.sessions.slice(0, 5).map((session) => (
                          <li key={session.session_id} className="session-item">
                            <span className="session-name">{session.name || t("dailyReport.unnamedSession")}</span>
                            <span className="session-meta">
                              ({formatDuration(session.duration)}, {session.message_count} {t("dailyReport.messagesShort")})
                            </span>
                          </li>
                        ))}
                        {project.sessions.length > 5 && (
                          <li className="session-more">
                            +{project.sessions.length - 5} {t("dailyReport.moreSessions")}
                          </li>
                        )}
                      </ul>
                    </div>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};
