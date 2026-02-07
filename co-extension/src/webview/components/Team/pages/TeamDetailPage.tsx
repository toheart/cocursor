/**
 * 团队详情页（Tab 容器 + 嵌套路由）
 * 进入时建立 WebSocket 连接，离开时延迟断开
 */

import React, { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useParams, Outlet, useNavigate } from "react-router-dom";
import { useTeamStore, useMemberStore, useWsStore, createTeamEventHandler } from "../stores";
import { PageHeader, TabNav, StatusDot, LoadingState } from "../shared";
import { teamWebSocket } from "../../../services/teamWebSocket";
import type { TabItem } from "../shared";

export const TeamDetailPage: React.FC = () => {
  const { t } = useTranslation();
  const { teamId } = useParams<{ teamId: string }>();
  const navigate = useNavigate();

  const { getTeamById, fetchTeams } = useTeamStore();
  const { members, skills, fetchMembers, fetchSkills, setCurrentTeam, reset } =
    useMemberStore();
  const { connectTeam, disconnectTeam, isConnected } = useWsStore();

  const team = teamId ? getTeamById(teamId) : undefined;
  const connected = teamId ? isConnected(teamId) : false;

  // 初始化：加载数据 + 连接 WebSocket
  useEffect(() => {
    if (!teamId) return;

    setCurrentTeam(teamId);
    fetchMembers(teamId);
    fetchSkills(teamId);

    // 如果有团队信息，建立 WebSocket
    if (team?.leader_endpoint) {
      connectTeam(teamId, team.leader_endpoint);

      // 注册事件处理
      const handler = createTeamEventHandler(teamId);
      const unsubscribe = teamWebSocket.on(teamId, handler);

      // 注册团队解散处理
      const dissolveHandler = teamWebSocket.on(teamId, (event) => {
        if (event.type === "team_dissolved") {
          navigate("/", { replace: true });
          fetchTeams();
        }
      });

      return () => {
        unsubscribe();
        dissolveHandler();
        disconnectTeam(teamId);
        reset();
      };
    }

    return () => {
      if (teamId) {
        disconnectTeam(teamId);
      }
      reset();
    };
  }, [teamId, team?.leader_endpoint]);  // eslint-disable-line react-hooks/exhaustive-deps

  // Tab 配置
  const tabs: TabItem[] = useMemo(
    () => [
      {
        id: "members",
        label: t("team.members"),
        icon: "people",
        count: members?.length || 0,
        path: "members",
      },
      {
        id: "weekly",
        label: t("weeklyReport.title"),
        icon: "graph",
        path: "weekly",
      },
      {
        id: "sessions",
        label: t("session.sharedSessions"),
        icon: "comment-discussion",
        path: "sessions",
      },
      // 团队技能功能暂时隐藏，技能通过内建技能市场分发
      // {
      //   id: "skills",
      //   label: t("team.skills"),
      //   icon: "package",
      //   count: skills?.length || 0,
      //   path: "skills",
      // },
    ],
    [t, members?.length],
  );

  if (!team) {
    return <LoadingState />;
  }

  const onlineCount = members?.filter((m) => m.is_online).length || 0;

  return (
    <div className="ct-detail">
      <PageHeader
        title={team.name}
        backTo="/"
        badge={
          team.is_leader ? (
            <span className="ct-badge leader">{t("team.leader")}</span>
          ) : undefined
        }
        actions={
          <div className="ct-detail-header-actions">
            <StatusDot
              online={team.is_leader || connected}
              label={
                team.is_leader || connected
                  ? t("team.connected")
                  : t("team.disconnected")
              }
              size="medium"
            />
            <span className="ct-detail-stat">
              {onlineCount}/{members?.length || 0} {t("team.online")}
            </span>
          </div>
        }
      />

      <TabNav tabs={tabs} />

      {/* 嵌套路由内容 */}
      <div className="ct-detail-content">
        <Outlet />
      </div>
    </div>
  );
};
