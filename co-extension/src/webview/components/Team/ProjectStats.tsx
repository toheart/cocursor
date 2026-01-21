/**
 * 项目周统计汇总组件
 */

import React from "react";
import { useTranslation } from "react-i18next";
import { ProjectWeekStats, ContributorStats } from "../../types";

interface ProjectStatsProps {
  projects: ProjectWeekStats[];
}

export const ProjectStats: React.FC<ProjectStatsProps> = ({ projects }) => {
  const { t } = useTranslation();

  if (!projects || projects.length === 0) {
    return null;
  }

  return (
    <div className="cocursor-project-stats">
      <h3 className="cocursor-project-stats-title">
        <span className="cocursor-section-icon">📊</span>
        {t("weeklyReport.projectSummary")}
      </h3>

      <div className="cocursor-project-stats-grid">
        {projects.map((project, idx) => (
          <ProjectCard key={idx} project={project} />
        ))}
      </div>
    </div>
  );
};

// 项目卡片组件
interface ProjectCardProps {
  project: ProjectWeekStats;
}

const ProjectCard: React.FC<ProjectCardProps> = ({ project }) => {
  const { t } = useTranslation();

  // 排序贡献者（按提交数降序）
  const sortedContributors = React.useMemo(() => {
    if (!project.contributors) return [];
    return [...project.contributors].sort((a, b) => b.commits - a.commits);
  }, [project.contributors]);

  return (
    <div className="cocursor-project-card">
      <div className="cocursor-project-card-header">
        <h4 className="cocursor-project-card-name">{project.project_name}</h4>
        <span className="cocursor-project-card-url" title={project.repo_url}>
          {project.repo_url}
        </span>
      </div>

      {/* 项目统计 */}
      <div className="cocursor-project-card-stats">
        <div className="cocursor-project-stat">
          <span className="cocursor-project-stat-value">{project.total_commits}</span>
          <span className="cocursor-project-stat-label">{t("weeklyReport.commits")}</span>
        </div>
        <div className="cocursor-project-stat positive">
          <span className="cocursor-project-stat-value">+{project.total_added}</span>
          <span className="cocursor-project-stat-label">{t("weeklyReport.added")}</span>
        </div>
        <div className="cocursor-project-stat negative">
          <span className="cocursor-project-stat-value">-{project.total_removed}</span>
          <span className="cocursor-project-stat-label">{t("weeklyReport.removed")}</span>
        </div>
      </div>

      {/* 贡献者排名 */}
      {sortedContributors.length > 0 && (
        <div className="cocursor-project-card-contributors">
          <h5 className="cocursor-contributors-title">{t("weeklyReport.contributors")}</h5>
          <div className="cocursor-contributors-list">
            {sortedContributors.slice(0, 5).map((contributor, idx) => (
              <ContributorItem
                key={contributor.member_id}
                contributor={contributor}
                rank={idx + 1}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

// 贡献者列表项
interface ContributorItemProps {
  contributor: ContributorStats;
  rank: number;
}

const ContributorItem: React.FC<ContributorItemProps> = ({ contributor, rank }) => {
  // 排名颜色
  const rankColors = ["#ffd700", "#c0c0c0", "#cd7f32", "#9ca3af", "#9ca3af"];
  const rankColor = rankColors[rank - 1] || "#9ca3af";

  return (
    <div className="cocursor-contributor-item">
      <span className="cocursor-contributor-rank" style={{ color: rankColor }}>
        #{rank}
      </span>
      <div className="cocursor-contributor-avatar">
        {contributor.member_name.charAt(0).toUpperCase()}
      </div>
      <span className="cocursor-contributor-name">{contributor.member_name}</span>
      <div className="cocursor-contributor-stats">
        <span className="cocursor-contributor-commits">{contributor.commits}</span>
        <span className="cocursor-contributor-lines positive">+{contributor.lines_added}</span>
        <span className="cocursor-contributor-lines negative">-{contributor.lines_removed}</span>
      </div>
    </div>
  );
};

export default ProjectStats;
