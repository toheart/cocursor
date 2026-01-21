import React from "react";
import { useTranslation } from "react-i18next";
import { EfficiencyMetricsSummary } from "../../services/api";

interface DailyReportStatsProps {
  totalSessions: number;
  projectCount: number;
  efficiencyMetrics?: EfficiencyMetricsSummary;
}

/**
 * 日报统计卡片组件
 * 展示会话数、项目数、活跃时长、消息数
 */
export const DailyReportStats: React.FC<DailyReportStatsProps> = ({
  totalSessions,
  projectCount,
  efficiencyMetrics,
}) => {
  const { t } = useTranslation();

  // 格式化时长显示
  const formatDuration = (hours: number): string => {
    if (hours < 1) {
      return `${Math.round(hours * 60)}${t("dailyReport.minutes")}`;
    }
    return `${hours.toFixed(1)}${t("dailyReport.hours")}`;
  };

  return (
    <div className="cocursor-daily-report-stats-grid">
      <div className="cocursor-daily-report-stat-card">
        <span className="cocursor-daily-report-stat-icon">💬</span>
        <span className="cocursor-daily-report-stat-value">{totalSessions}</span>
        <span className="cocursor-daily-report-stat-label">{t("dailyReport.sessions")}</span>
      </div>
      <div className="cocursor-daily-report-stat-card">
        <span className="cocursor-daily-report-stat-icon">📁</span>
        <span className="cocursor-daily-report-stat-value">{projectCount}</span>
        <span className="cocursor-daily-report-stat-label">{t("dailyReport.projects")}</span>
      </div>
      {efficiencyMetrics && (
        <>
          <div className="cocursor-daily-report-stat-card">
            <span className="cocursor-daily-report-stat-icon">⏱️</span>
            <span className="cocursor-daily-report-stat-value">
              {formatDuration(efficiencyMetrics.total_active_time)}
            </span>
            <span className="cocursor-daily-report-stat-label">{t("dailyReport.activeTime")}</span>
          </div>
          <div className="cocursor-daily-report-stat-card">
            <span className="cocursor-daily-report-stat-icon">📝</span>
            <span className="cocursor-daily-report-stat-value">
              {Math.round(efficiencyMetrics.avg_messages_per_session * totalSessions)}
            </span>
            <span className="cocursor-daily-report-stat-label">{t("dailyReport.messages")}</span>
          </div>
        </>
      )}
    </div>
  );
};
