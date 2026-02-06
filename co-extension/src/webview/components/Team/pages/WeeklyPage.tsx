/**
 * 周报 Tab 页面
 * 复用已有的 WeeklyReport 组件逻辑
 */

import React, { useState, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useParams } from "react-router-dom";
import { apiService } from "../../../services/api";
import { TeamWeeklyView, TeamProjectConfig, MemberDailyDetail } from "../../../types";
import { useApi, useToast } from "../../../hooks";
import { useTeamStore } from "../stores";
import { LoadingState, EmptyState } from "../shared";
import { ToastContainer } from "../../shared/ToastContainer";
import { WeeklyCalendar } from "../WeeklyCalendar";
import { MemberDayDetailModal } from "../MemberDayDetail";
import { ProjectConfig } from "../ProjectConfig";
import { ProjectStats } from "../ProjectStats";
import "../../../styles/weekly-report.css";

// 获取周一日期
function getWeekStart(date: Date): string {
  const d = new Date(date);
  const day = d.getDay();
  const diff = d.getDate() - day + (day === 0 ? -6 : 1);
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
  const startMonth = startDate.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
  const endMonth = endDate.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
  return `${startMonth} - ${endMonth}`;
}

export const WeeklyPage: React.FC = () => {
  const { t } = useTranslation();
  const { teamId } = useParams<{ teamId: string }>();
  const { showToast, toasts } = useToast();
  const { getTeamById } = useTeamStore();

  const team = teamId ? getTeamById(teamId) : undefined;
  const isLeader = team?.is_leader ?? false;

  const [weekStart, setWeekStart] = useState(() => getWeekStart(new Date()));
  const [showProjectConfig, setShowProjectConfig] = useState(false);
  const [selectedCell, setSelectedCell] = useState<{
    memberId: string;
    date: string;
  } | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [memberDetail, setMemberDetail] = useState<MemberDailyDetail | null>(
    null,
  );
  const [refreshing, setRefreshing] = useState(false);

  // 获取周报数据
  const fetchWeeklyReport = useCallback(async (): Promise<TeamWeeklyView> => {
    if (!teamId) return {} as TeamWeeklyView;
    const report = await apiService.getTeamWeeklyReport(teamId, weekStart);
    return report;
  }, [teamId, weekStart]);

  const {
    data: weeklyView,
    loading,
    refetch,
    error,
  } = useApi<TeamWeeklyView>(fetchWeeklyReport);

  // 获取项目配置
  const fetchProjectConfig = useCallback(async (): Promise<TeamProjectConfig> => {
    if (!teamId) return {} as TeamProjectConfig;
    const config = await apiService.getTeamProjectConfig(teamId);
    return config;
  }, [teamId]);

  const { data: projectConfig, refetch: refetchConfig } =
    useApi<TeamProjectConfig>(fetchProjectConfig);

  const weekEnd = useMemo(() => getWeekEnd(weekStart), [weekStart]);

  const handlePrevWeek = useCallback(() => {
    const d = new Date(weekStart);
    d.setDate(d.getDate() - 7);
    setWeekStart(d.toISOString().split("T")[0]);
  }, [weekStart]);

  const handleNextWeek = useCallback(() => {
    const d = new Date(weekStart);
    d.setDate(d.getDate() + 7);
    const thisWeekStart = getWeekStart(new Date());
    if (d.toISOString().split("T")[0] <= thisWeekStart) {
      setWeekStart(d.toISOString().split("T")[0]);
    }
  }, [weekStart]);

  const handleThisWeek = useCallback(() => {
    setWeekStart(getWeekStart(new Date()));
  }, []);

  const isCurrentWeek = useMemo(() => {
    return weekStart === getWeekStart(new Date());
  }, [weekStart]);

  const handleRefresh = useCallback(async () => {
    if (!teamId) return;
    setRefreshing(true);
    try {
      await apiService.refreshTeamWeeklyStats(teamId, weekStart);
      await refetch();
      showToast(t("weeklyReport.refreshSuccess"), "success");
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : t("weeklyReport.refreshFailed");
      showToast(message, "error");
    } finally {
      setRefreshing(false);
    }
  }, [teamId, weekStart, refetch, showToast, t]);

  const handleCellClick = useCallback(
    async (memberId: string, date: string) => {
      if (!teamId) return;
      setSelectedCell({ memberId, date });
      setDetailLoading(true);
      try {
        const detail = await apiService.getMemberDailyDetail(
          teamId,
          memberId,
          date,
        );
        setMemberDetail(detail);
      } catch (err: unknown) {
        const message =
          err instanceof Error
            ? err.message
            : t("weeklyReport.fetchDetailFailed");
        showToast(message, "error");
        setSelectedCell(null);
      } finally {
        setDetailLoading(false);
      }
    },
    [teamId, showToast, t],
  );

  const handleCloseDetail = useCallback(() => {
    setSelectedCell(null);
    setMemberDetail(null);
  }, []);

  const handleConfigUpdated = useCallback(() => {
    refetchConfig();
    refetch();
  }, [refetchConfig, refetch]);

  const hasProjects =
    projectConfig &&
    projectConfig.projects &&
    projectConfig.projects.length > 0;

  return (
    <div className="ct-weekly-page">
      <ToastContainer toasts={toasts} />

      {/* 周次导航 */}
      <div className="ct-weekly-nav">
        <div className="ct-weekly-nav-left">
          <button
            className="ct-btn-icon"
            onClick={handlePrevWeek}
            title={t("weeklyReport.prevWeek")}
          >
            <span className="codicon codicon-chevron-left" />
          </button>
          <div className="ct-weekly-nav-display">
            <span className="ct-weekly-nav-range">
              {formatWeekRange(weekStart, weekEnd)}
            </span>
            {!isCurrentWeek && (
              <button className="ct-btn-link small" onClick={handleThisWeek}>
                {t("weeklyReport.thisWeek")}
              </button>
            )}
          </div>
          <button
            className="ct-btn-icon"
            onClick={handleNextWeek}
            disabled={isCurrentWeek}
            title={t("weeklyReport.nextWeek")}
          >
            <span className="codicon codicon-chevron-right" />
          </button>
        </div>

        <div className="ct-weekly-nav-right">
          {isLeader && (
            <button
              className="ct-btn secondary small"
              onClick={() => setShowProjectConfig(true)}
            >
              <span className="codicon codicon-settings-gear" />
              {t("weeklyReport.projectConfig")}
            </button>
          )}
          <button
            className="ct-btn primary small"
            onClick={handleRefresh}
            disabled={refreshing || loading}
          >
            {refreshing ? (
              <span className="ct-btn-spinner" />
            ) : (
              <span className="codicon codicon-refresh" />
            )}
            {t("common.refresh")}
          </button>
        </div>
      </div>

      {/* 内容 */}
      <div className="ct-weekly-content">
        {loading ? (
          <LoadingState />
        ) : error ? (
          <EmptyState
            icon="error"
            title={t("weeklyReport.loadFailed")}
            action={
              <button className="ct-btn secondary" onClick={refetch}>
                {t("common.retry")}
              </button>
            }
          />
        ) : !hasProjects ? (
          <EmptyState
            icon="folder"
            title={t("weeklyReport.noProjectsConfigured")}
            description={t("weeklyReport.noProjectsConfiguredDesc")}
            action={
              isLeader ? (
                <button
                  className="ct-btn primary"
                  onClick={() => setShowProjectConfig(true)}
                >
                  {t("weeklyReport.configureProjects")}
                </button>
              ) : undefined
            }
          />
        ) : (
          <>
            <WeeklyCalendar
              calendar={weeklyView?.calendar || []}
              weekStart={weekStart}
              onCellClick={handleCellClick}
            />

            {weeklyView?.project_summary &&
              weeklyView.project_summary.length > 0 && (
                <ProjectStats projects={weeklyView.project_summary} />
              )}
          </>
        )}
      </div>

      {/* 项目配置弹窗（保留弹窗形式） */}
      {showProjectConfig && teamId && (
        <ProjectConfig
          teamId={teamId}
          config={projectConfig}
          onClose={() => setShowProjectConfig(false)}
          onUpdated={handleConfigUpdated}
        />
      )}

      {/* 成员日详情弹窗（保留弹窗形式） */}
      {selectedCell && teamId && (
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
