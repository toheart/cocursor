/**
 * 团队列表组件
 */

import React, { useState, useCallback, useMemo, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { Team, Identity } from "../../types";
import { useApi, useToast, useTeamWebSocket } from "../../hooks";
import { TeamCreate } from "./TeamCreate";
import { TeamJoin } from "./TeamJoin";
import { MemberList } from "./MemberList";
import { IdentitySetup } from "./IdentitySetup";
import { NetworkSettings } from "./NetworkSettings";
import { ToastContainer } from "../shared/ToastContainer";

// 自动刷新间隔（30秒）
const AUTO_REFRESH_INTERVAL = 30 * 1000;

export const TeamList: React.FC = () => {
  const { t } = useTranslation();
  const { showToast, toasts } = useToast();
  
  const [showCreate, setShowCreate] = useState(false);
  const [showJoin, setShowJoin] = useState(false);
  const [showIdentity, setShowIdentity] = useState(false);
  const [showNetworkSettings, setShowNetworkSettings] = useState(false);
  const [selectedTeam, setSelectedTeam] = useState<string | null>(null);

  // 获取身份
  const fetchIdentity = useCallback(async () => {
    const resp = await apiService.getTeamIdentity() as { exists: boolean; identity?: Identity };
    return resp;
  }, []);

  const { data: identityData, refetch: refetchIdentity } = useApi<{ exists: boolean; identity?: Identity }>(fetchIdentity);

  // 获取团队列表
  const fetchTeams = useCallback(async () => {
    const resp = await apiService.getTeamList() as { teams: Team[]; total: number };
    return resp;
  }, []);

  const { data: teamsData, loading, refetch: loadTeams } = useApi<{ teams: Team[]; total: number }>(fetchTeams);

  const teams = useMemo(() => teamsData?.teams || [], [teamsData]);

  // 记录每个团队的 WebSocket 连接状态
  const [wsConnectionStates, setWsConnectionStates] = useState<Record<string, boolean>>({});

  // 自动刷新定时器
  const refreshTimerRef = useRef<NodeJS.Timeout | null>(null);

  // 自动刷新团队列表
  useEffect(() => {
    // 设置定时刷新
    refreshTimerRef.current = setInterval(() => {
      loadTeams();
    }, AUTO_REFRESH_INTERVAL);

    return () => {
      if (refreshTimerRef.current) {
        clearInterval(refreshTimerRef.current);
      }
    };
  }, [loadTeams]);

  // 更新 WebSocket 连接状态的回调
  const updateWsConnectionState = useCallback((teamId: string, connected: boolean) => {
    setWsConnectionStates(prev => ({
      ...prev,
      [teamId]: connected
    }));
  }, []);
  const hasIdentity = identityData?.exists ?? false;
  const identity = identityData?.identity;

  const handleLeaveTeam = useCallback(async (teamId: string) => {
    try {
      await apiService.leaveTeam(teamId);
      showToast(t("team.leaveSuccess"), "success");
      loadTeams();
    } catch (error) {
      showToast(t("team.leaveFailed"), "error");
    }
  }, [showToast, loadTeams, t]);

  const handleDissolveTeam = useCallback(async (teamId: string) => {
    if (!confirm(t("team.dissolveConfirm"))) return;
    try {
      await apiService.dissolveTeam(teamId);
      showToast(t("team.dissolveSuccess"), "success");
      loadTeams();
    } catch (error) {
      showToast(t("team.dissolveFailed"), "error");
    }
  }, [showToast, loadTeams, t]);

  const handleTeamCreated = useCallback(() => {
    setShowCreate(false);
    loadTeams();
    showToast(t("team.createSuccess"), "success");
  }, [loadTeams, showToast, t]);

  const handleTeamJoined = useCallback(() => {
    setShowJoin(false);
    loadTeams();
    showToast(t("team.joinSuccess"), "success");
  }, [loadTeams, showToast, t]);

  const handleIdentitySet = useCallback(() => {
    setShowIdentity(false);
    refetchIdentity();
  }, [refetchIdentity]);

  // 如果选中了某个团队，显示成员列表
  if (selectedTeam) {
    const team = teams.find(t => t.id === selectedTeam);
    if (team) {
      return (
        <MemberList 
          team={team} 
          onBack={() => setSelectedTeam(null)}
          onRefresh={loadTeams}
        />
      );
    }
  }

  return (
    <div className="cocursor-team">
      <ToastContainer toasts={toasts} />

      {/* Hero 区域 */}
      <div className="cocursor-team-hero">
        <div className="cocursor-team-title-row">
          <div className="cocursor-team-title-left">
            <h1 className="cocursor-team-title">{t("team.title")}</h1>
            <span
              className="cocursor-beta-badge"
              title={t("team.betaTooltip")}
            >
              {t("team.beta")}
            </span>
          </div>
          <button
            className="cocursor-team-settings-btn"
            onClick={() => setShowNetworkSettings(true)}
            title={t("network.settings")}
          >
            ⚙️
          </button>
        </div>
        <p className="cocursor-team-subtitle">{t("team.subtitle")}</p>
      </div>

      {/* 身份信息 */}
      <div className="cocursor-team-identity-bar">
        {hasIdentity ? (
          <div className="cocursor-team-identity-info">
            <span className="cocursor-team-identity-label">{t("team.identity")}：</span>
            <span className="cocursor-team-identity-name">{identity?.name}</span>
            <button 
              className="cocursor-team-identity-edit"
              onClick={() => setShowIdentity(true)}
            >
              {t("common.edit")}
            </button>
          </div>
        ) : (
          <button 
            className="cocursor-team-setup-identity"
            onClick={() => setShowIdentity(true)}
          >
            {t("team.setupIdentity")}
          </button>
        )}
      </div>

      {/* 操作按钮 */}
      <div className="cocursor-team-actions">
        <button 
          className="cocursor-team-action-btn primary"
          onClick={() => setShowCreate(true)}
          disabled={!hasIdentity}
          title={!hasIdentity ? t("team.identityRequired") : ""}
        >
          <span className="cocursor-team-action-icon">👑</span>
          {t("team.createTeam")}
        </button>
        <button 
          className="cocursor-team-action-btn secondary"
          onClick={() => setShowJoin(true)}
          disabled={!hasIdentity}
          title={!hasIdentity ? t("team.identityRequired") : ""}
        >
          <span className="cocursor-team-action-icon">🔍</span>
          {t("team.discoverTeams")}
        </button>
        <button 
          className="cocursor-team-action-btn secondary"
          onClick={loadTeams}
        >
          <span className="cocursor-team-action-icon">🔄</span>
          {t("common.refresh")}
        </button>
      </div>

      {/* 团队列表 */}
      <div className="cocursor-team-list">
        {loading ? (
          <div className="cocursor-team-loading">
            <div className="cocursor-team-loading-spinner"></div>
            <span>{t("common.loading")}</span>
          </div>
        ) : teams.length === 0 ? (
          <div className="cocursor-team-empty">
            <div className="cocursor-team-empty-icon">👥</div>
            <p>{t("team.noTeams")}</p>
            <span>{t("team.noTeamsDesc")}</span>
          </div>
        ) : (
          teams.map(team => (
            <TeamCardWithConnection
              key={team.id}
              team={team}
              onClick={() => setSelectedTeam(team.id)}
              onLeave={() => handleLeaveTeam(team.id)}
              onDissolve={() => handleDissolveTeam(team.id)}
              onConnectionChange={(connected) => updateWsConnectionState(team.id, connected)}
              wsConnected={wsConnectionStates[team.id]}
            />
          ))
        )}
      </div>

      {/* 弹窗 */}
      {showIdentity && (
        <IdentitySetup 
          identity={identity}
          onClose={() => setShowIdentity(false)}
          onSuccess={handleIdentitySet}
        />
      )}

      {showCreate && (
        <TeamCreate 
          onClose={() => setShowCreate(false)}
          onSuccess={handleTeamCreated}
        />
      )}

      {showJoin && (
        <TeamJoin 
          onClose={() => setShowJoin(false)}
          onSuccess={handleTeamJoined}
        />
      )}

      {showNetworkSettings && (
        <NetworkSettings onClose={() => setShowNetworkSettings(false)} />
      )}
    </div>
  );
};

// 团队卡片组件（带 WebSocket 连接监控）
interface TeamCardWithConnectionProps {
  team: Team;
  onClick: () => void;
  onLeave: () => void;
  onDissolve: () => void;
  onConnectionChange: (connected: boolean) => void;
  wsConnected?: boolean;
}

const TeamCardWithConnection: React.FC<TeamCardWithConnectionProps> = ({ 
  team, 
  onClick, 
  onLeave, 
  onDissolve,
  onConnectionChange,
  wsConnected 
}) => {
  const { t } = useTranslation();

  // 对于非 Leader 成员，连接 Leader 的 WebSocket 来监测连接状态
  // 这里使用轻量级的连接（仅用于状态检测）
  const { isConnected } = useTeamWebSocket({
    teamId: team.id,
    leaderEndpoint: team.leader_endpoint,
    enabled: !team.is_leader, // Leader 自己不需要连接自己
    onEvent: () => {}, // 不需要处理事件
  });

  // 当连接状态变化时通知父组件
  useEffect(() => {
    // 如果是 Leader，始终认为已连接
    const connected = team.is_leader ? true : isConnected;
    onConnectionChange(connected);
  }, [isConnected, team.is_leader, onConnectionChange]);

  // 确定实际的连接状态
  // Leader 始终显示为已连接，成员显示实际 WebSocket 连接状态
  const actualConnected = team.is_leader ? true : (wsConnected ?? isConnected);

  return (
    <div className={`cocursor-team-card ${team.is_leader ? "leader" : ""}`} onClick={onClick}>
      <div className="cocursor-team-card-header">
        <div className="cocursor-team-card-icon">
          {team.is_leader ? "👑" : "👥"}
        </div>
        <div className="cocursor-team-card-info">
          <h3 className="cocursor-team-card-name">
            {team.name}
            {team.is_leader && (
              <span className="cocursor-team-card-badge leader">{t("team.leader")}</span>
            )}
          </h3>
          <div className="cocursor-team-card-meta">
            <span>{t("team.leaderLabel")}: {team.leader_name}</span>
            {/* WebSocket 连接状态指示器 */}
            <span 
              className={`cocursor-team-card-connection ${actualConnected ? "connected" : "disconnected"}`}
              title={actualConnected ? t("team.wsConnected") : t("team.wsDisconnected")}
            >
              <span className="cocursor-team-card-connection-dot"></span>
              {actualConnected ? t("team.connected") : t("team.disconnected")}
            </span>
          </div>
        </div>
      </div>

      <div className="cocursor-team-card-stats">
        <div className="cocursor-team-card-stat">
          <span className="cocursor-team-card-stat-value">{team.member_count}</span>
          <span className="cocursor-team-card-stat-label">{t("team.members")}</span>
        </div>
        <div className="cocursor-team-card-stat">
          <span className="cocursor-team-card-stat-value">{team.skill_count}</span>
          <span className="cocursor-team-card-stat-label">{t("team.skills")}</span>
        </div>
      </div>

      <div className="cocursor-team-card-actions" onClick={e => e.stopPropagation()}>
        {team.is_leader ? (
          <button 
            className="cocursor-team-card-btn danger"
            onClick={onDissolve}
          >
            {t("team.dissolve")}
          </button>
        ) : (
          <button 
            className="cocursor-team-card-btn secondary"
            onClick={onLeave}
          >
            {t("team.leave")}
          </button>
        )}
      </div>
    </div>
  );
};
