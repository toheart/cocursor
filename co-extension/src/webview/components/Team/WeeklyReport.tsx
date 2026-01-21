/**
 * 团队周报主组件
 */

import React, { useState, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { TeamWeeklyView, TeamProjectConfig, MemberDailyDetail } from "../../types";
import { useApi, useToast } from "../../hooks";
import { ToastContainer } from "../shared/ToastContainer";
import { WeeklyCalendar } from "./WeeklyCalendar";
import { MemberDayDetailModal } from "./MemberDayDetail";
import { ProjectConfig } from "./ProjectConfig";
import { ProjectStats } from "./ProjectStats";
import "../../styles/weekly-report.css";

interface WeeklyReportProps {
  teamId: string;
  isLeader: boolean;
  onRefresh?: () => void;
}

// 获取周一日期
function getWeekStart(date: Date): string {
  const d = new Date(date);
  const day = d.getDay();
  const diff = d.getDate() - day + (day === 0 ? -6 : 1); // 调整为周一
  d.setDate(diff);
  return d.toISOString().split("T")[0];
}

// 获取周日日期
function getWeekEnd(weekStart: string): string {
  const d = new Date(weekStart);
  d.setDate(d.getDate() + 6);
  return d.toISOString().split("T")[0];
}

// 格式化日期范围显示
function formatWeekRange(start: string, end: string): string {
  const startDate = new Date(start);
  const endDate = new Date(end);
  const startMonth = startDate.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  const endMonth = endDate.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  return `${startMonth} - ${endMonth}`;
}

export const WeeklyReport: React.FC<WeeklyReportProps> = ({ teamId, isLeader, onRefresh }) => {
  const { t } = useTranslation();
  const { showToast, toasts } = useToast();

  // 周次选择状态
  const [weekStart, setWeekStart] = useState(() => getWeekStart(new Date()));
  const [showProjectConfig, setShowProjectConfig] = useState(false);
  const [selectedCell, setSelectedCell] = useState<{ memberId: string; date: string } | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [memberDetail, setMemberDetail] = useState<MemberDailyDetail | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  // 获取周报数据
  const fetchWeeklyReport = useCallback(async () => {
    const report = await apiService.getTeamWeeklyReport(teamId, weekStart);
    return report;
  }, [teamId, weekStart]);

  const { data: weeklyView, loading, refetch, error } = useApi<TeamWeeklyView>(fetchWeeklyReport);

  // 获取项目配置
  const fetchProjectConfig = useCallback(async () => {
    const config = await apiService.getTeamProjectConfig(teamId);
    return config;
  }, [teamId]);

  const { data: projectConfig, refetch: refetchConfig } = useApi<TeamProjectConfig>(fetchProjectConfig);

  // 周次导航
  const weekEnd = useMemo(() => getWeekEnd(weekStart), [weekStart]);

  const handlePrevWeek = useCallback(() => {
    const d = new Date(weekStart);
    d.setDate(d.getDate() - 7);
    setWeekStart(d.toISOString().split("T")[0]);
  }, [weekStart]);

  const handleNextWeek = useCallback(() => {
    const d = new Date(weekStart);
    d.setDate(d.getDate() + 7);
    const today = new Date();
    const thisWeekStart = getWeekStart(today);
    // 不允许选择未来的周
    if (d.toISOString().split("T")[0] <= thisWeekStart) {
      setWeekStart(d.toISOString().split("T")[0]);
    }
  }, [weekStart]);

  const handleThisWeek = useCallback(() => {
    setWeekStart(getWeekStart(new Date()));
  }, []);

  // 判断是否为当前周
  const isCurrentWeek = useMemo(() => {
    return weekStart === getWeekStart(new Date());
  }, [weekStart]);

  // 刷新周报数据
  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await apiService.refreshTeamWeeklyStats(teamId, weekStart);
      await refetch();
      showToast(t("weeklyReport.refreshSuccess"), "success");
      onRefresh?.();
    } catch (err: any) {
      showToast(err.message || t("weeklyReport.refreshFailed"), "error");
    } finally {
      setRefreshing(false);
    }
  }, [teamId, weekStart, refetch, showToast, onRefresh, t]);

  // 查看成员日详情
  const handleCellClick = useCallback(async (memberId: string, date: string) => {
    setSelectedCell({ memberId, date });
    setDetailLoading(true);
    try {
      const detail = await apiService.getMemberDailyDetail(teamId, memberId, date);
      setMemberDetail(detail);
    } catch (err: any) {
      showToast(err.message || t("weeklyReport.fetchDetailFailed"), "error");
      setSelectedCell(null);
    } finally {
      setDetailLoading(false);
    }
  }, [teamId, showToast, t]);

  // 关闭详情弹窗
  const handleCloseDetail = useCallback(() => {
    setSelectedCell(null);
    setMemberDetail(null);
  }, []);

  // 项目配置更新后刷新
  const handleConfigUpdated = useCallback(() => {
    refetchConfig();
    refetch();
  }, [refetchConfig, refetch]);

  // 判断是否有配置项目
  const hasProjects = projectConfig && projectConfig.projects && projectConfig.projects.length > 0;

  return (
    <div className="cocursor-weekly-report">
      <ToastContainer toasts={toasts} />

      {/* 头部操作栏 */}
      <div className="cocursor-weekly-report-header">
        <div className="cocursor-weekly-report-nav">
          <button
            className="cocursor-btn icon"
            onClick={handlePrevWeek}
            title={t("weeklyReport.prevWeek")}
          >
            ←
          </button>
          <div className="cocursor-weekly-report-week-display">
            <span className="cocursor-weekly-report-week-range">
              {formatWeekRange(weekStart, weekEnd)}
            </span>
            {!isCurrentWeek && (
              <button
                className="cocursor-btn-text small"
                onClick={handleThisWeek}
              >
                {t("weeklyReport.thisWeek")}
              </button>
            )}
          </div>
          <button
            className="cocursor-btn icon"
            onClick={handleNextWeek}
            disabled={isCurrentWeek}
            title={t("weeklyReport.nextWeek")}
          >
            →
          </button>
        </div>

        <div className="cocursor-weekly-report-actions">
          {isLeader && (
            <button
              className="cocursor-btn secondary"
              onClick={() => setShowProjectConfig(true)}
            >
              <span className="cocursor-btn-icon">⚙️</span>
              {t("weeklyReport.projectConfig")}
            </button>
          )}
          <button
            className="cocursor-btn primary"
            onClick={handleRefresh}
            disabled={refreshing || loading}
          >
            {refreshing ? (
              <>
                <span className="cocursor-btn-spinner"></span>
                {t("common.loading")}
              </>
            ) : (
              <>
                <span className="cocursor-btn-icon">🔄</span>
                {t("common.refresh")}
              </>
            )}
          </button>
        </div>
      </div>

      {/* 主体内容 */}
      <div className="cocursor-weekly-report-content">
        {loading ? (
          <div className="cocursor-team-loading">
            <div className="cocursor-team-loading-spinner"></div>
          </div>
        ) : error ? (
          <div className="cocursor-team-error">
            <span className="cocursor-error-icon">❌</span>
            <span>{error || t("weeklyReport.loadFailed")}</span>
            <button className="cocursor-btn secondary" onClick={refetch}>
              {t("common.retry")}
            </button>
          </div>
        ) : !hasProjects ? (
          <div className="cocursor-team-empty-section">
            <span className="cocursor-team-empty-icon">📁</span>
            <span>{t("weeklyReport.noProjectsConfigured")}</span>
            <p>{t("weeklyReport.noProjectsConfiguredDesc")}</p>
            {isLeader && (
              <button
                className="cocursor-btn primary"
                onClick={() => setShowProjectConfig(true)}
              >
                {t("weeklyReport.configureProjects")}
              </button>
            )}
          </div>
        ) : (
          <>
            {/* 日历视图 */}
            <WeeklyCalendar
              calendar={weeklyView?.calendar || []}
              weekStart={weekStart}
              onCellClick={handleCellClick}
            />

            {/* 项目汇总 */}
            {weeklyView?.project_summary && weeklyView.project_summary.length > 0 && (
              <ProjectStats projects={weeklyView.project_summary} />
            )}
          </>
        )}
      </div>

      {/* 项目配置弹窗 */}
      {showProjectConfig && (
        <ProjectConfig
          teamId={teamId}
          config={projectConfig}
          onClose={() => setShowProjectConfig(false)}
          onUpdated={handleConfigUpdated}
        />
      )}

      {/* 成员日详情弹窗 */}
      {selectedCell && (
        <MemberDayDetailModal
          detail={memberDetail}
          loading={detailLoading}
          date={selectedCell.date}
          teamId={teamId}
          onClose={handleCloseDetail}
        />
      )}
    </div>
  );
};

export default WeeklyReport;
