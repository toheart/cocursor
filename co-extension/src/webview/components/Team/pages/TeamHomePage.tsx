/**
 * 团队首页 - 列表 + 身份栏 + 操作入口
 * 紧凑布局，最大化内容空间
 */

import React, { useEffect, useCallback, useState, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { useIdentityStore, useTeamStore } from "../stores";
import { TeamCard } from "../cards";
import { EmptyState, LoadingState, ConfirmDialog } from "../shared";
import { ToastContainer } from "../../shared/ToastContainer";
import { useToast } from "../../../hooks";

// 自动刷新间隔（30秒）
const AUTO_REFRESH_INTERVAL = 30 * 1000;

export const TeamHomePage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { showToast, toasts } = useToast();

  // 全局状态
  const { identity, hasIdentity, fetchIdentity } = useIdentityStore();
  const { teams, loading, fetchTeams, leaveTeam, dissolveTeam } =
    useTeamStore();

  // 确认弹窗状态
  const [confirmAction, setConfirmAction] = useState<{
    type: "leave" | "dissolve";
    teamId: string;
    teamName: string;
  } | null>(null);
  const [confirmLoading, setConfirmLoading] = useState(false);

  // 初始加载
  useEffect(() => {
    fetchIdentity();
    fetchTeams();
  }, [fetchIdentity, fetchTeams]);

  // 自动刷新（带页面可见性检测）
  const refreshTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    const startRefresh = () => {
      refreshTimerRef.current = setInterval(() => {
        if (document.visibilityState === "visible") {
          fetchTeams();
        }
      }, AUTO_REFRESH_INTERVAL);
    };

    startRefresh();

    return () => {
      if (refreshTimerRef.current) {
        clearInterval(refreshTimerRef.current);
      }
    };
  }, [fetchTeams]);

  // 处理离开团队
  const handleLeave = useCallback(
    async (teamId: string) => {
      setConfirmLoading(true);
      try {
        await leaveTeam(teamId);
        showToast(t("team.leaveSuccess"), "success");
        setConfirmAction(null);
      } catch {
        showToast(t("team.leaveFailed"), "error");
      } finally {
        setConfirmLoading(false);
      }
    },
    [leaveTeam, showToast, t],
  );

  // 处理解散团队
  const handleDissolve = useCallback(
    async (teamId: string) => {
      setConfirmLoading(true);
      try {
        await dissolveTeam(teamId);
        showToast(t("team.dissolveSuccess"), "success");
        setConfirmAction(null);
      } catch {
        showToast(t("team.dissolveFailed"), "error");
      } finally {
        setConfirmLoading(false);
      }
    },
    [dissolveTeam, showToast, t],
  );

  return (
    <div className="ct-home">
      <ToastContainer toasts={toasts} />

      {/* 紧凑工具栏 */}
      <div className="ct-home-toolbar">
        <div className="ct-home-toolbar-left">
          {hasIdentity ? (
            <button
              className="ct-identity-chip"
              onClick={() => navigate("/identity")}
              title={t("common.edit")}
            >
              <span className="ct-identity-avatar">
                {identity?.name.charAt(0).toUpperCase()}
              </span>
              <span className="ct-identity-name">{identity?.name}</span>
              <span className="codicon codicon-edit" />
            </button>
          ) : (
            <button
              className="ct-btn primary small"
              onClick={() => navigate("/identity")}
            >
              <span className="codicon codicon-account" />
              {t("team.setupIdentity")}
            </button>
          )}
        </div>
        <div className="ct-home-toolbar-right">
          <button
            className="ct-btn primary small"
            onClick={() => navigate("/create")}
            disabled={!hasIdentity}
            title={!hasIdentity ? t("team.identityRequired") : t("team.createTeam")}
          >
            <span className="codicon codicon-add" />
            {t("team.createTeam")}
          </button>
          <button
            className="ct-btn secondary small"
            onClick={() => navigate("/join")}
            disabled={!hasIdentity}
            title={!hasIdentity ? t("team.identityRequired") : t("team.discoverTeams")}
          >
            <span className="codicon codicon-search" />
            {t("team.discoverTeams")}
          </button>
          <button
            className="ct-btn-icon"
            onClick={() => navigate("/network")}
            title={t("network.settings")}
          >
            <span className="codicon codicon-settings-gear" />
          </button>
          <button
            className="ct-btn-icon"
            onClick={fetchTeams}
            title={t("common.refresh")}
          >
            <span className="codicon codicon-refresh" />
          </button>
        </div>
      </div>

      {/* 团队列表 */}
      <div className="ct-home-list">
        {loading && teams.length === 0 ? (
          <LoadingState />
        ) : teams.length === 0 ? (
          <EmptyState
            icon="organization"
            title={t("team.noTeams")}
            description={t("team.noTeamsDesc")}
            action={
              hasIdentity ? (
                <button
                  className="ct-btn primary"
                  onClick={() => navigate("/create")}
                >
                  <span className="codicon codicon-add" />
                  {t("team.createTeam")}
                </button>
              ) : undefined
            }
          />
        ) : (
          teams.map((team) => (
            <TeamCard
              key={team.id}
              team={team}
              onLeave={() =>
                setConfirmAction({
                  type: "leave",
                  teamId: team.id,
                  teamName: team.name,
                })
              }
              onDissolve={() =>
                setConfirmAction({
                  type: "dissolve",
                  teamId: team.id,
                  teamName: team.name,
                })
              }
            />
          ))
        )}
      </div>

      {/* 确认弹窗 */}
      {confirmAction && (
        <ConfirmDialog
          title={
            confirmAction.type === "dissolve"
              ? t("team.dissolve")
              : t("team.leave")
          }
          message={
            confirmAction.type === "dissolve"
              ? t("team.dissolveConfirm")
              : t("team.leaveConfirmMessage", {
                  name: confirmAction.teamName,
                })
          }
          confirmText={
            confirmAction.type === "dissolve"
              ? t("team.dissolve")
              : t("team.leave")
          }
          danger={confirmAction.type === "dissolve"}
          loading={confirmLoading}
          onConfirm={() =>
            confirmAction.type === "dissolve"
              ? handleDissolve(confirmAction.teamId)
              : handleLeave(confirmAction.teamId)
          }
          onCancel={() => setConfirmAction(null)}
        />
      )}
    </div>
  );
};
