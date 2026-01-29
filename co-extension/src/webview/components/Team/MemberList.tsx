/**
 * 团队成员列表和技能管理组件
 */

import React, { useState, useCallback, useMemo, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { Team, TeamMember, TeamSkillEntry } from "../../types";
import { useApi, useToast, useTeamWebSocket } from "../../hooks";
import { TeamEvent, MemberStatusChangedEvent } from "../../services/teamWebSocket";
import { SkillPublish } from "./SkillPublish";
import { ToastContainer } from "../shared/ToastContainer";
import { WeeklyReport } from "./WeeklyReport";

interface MemberListProps {
  team: Team;
  onBack: () => void;
  onRefresh: () => void;
}

export const MemberList: React.FC<MemberListProps> = ({ team, onBack, onRefresh }) => {
  const { t } = useTranslation();
  const { showToast, toasts } = useToast();
  const [activeTab, setActiveTab] = useState<"members" | "skills" | "weekly">("members");
  const [showPublish, setShowPublish] = useState(false);
  const [memberStatuses, setMemberStatuses] = useState<Record<string, { project: string; file: string }>>({});

  // 获取成员列表
  const fetchMembers = useCallback(async () => {
    const resp = await apiService.getTeamMembers(team.id) as { members: TeamMember[] };
    return resp.members || [];
  }, [team.id]);

  const { data: members, loading: loadingMembers, refetch: refetchMembers } = useApi<TeamMember[]>(fetchMembers);

  // 获取技能列表
  const fetchSkills = useCallback(async () => {
    const resp = await apiService.getTeamSkillIndex(team.id) as { entries: TeamSkillEntry[] };
    return resp.entries || [];
  }, [team.id]);

  const { data: skills, loading: loadingSkills, refetch: refetchSkills } = useApi<TeamSkillEntry[]>(fetchSkills);

  // WebSocket 事件处理
  const handleTeamEvent = useCallback((event: TeamEvent) => {
    console.log("[MemberList] Team event received:", event.type);
    
    switch (event.type) {
      case "member_joined":
      case "member_left":
      case "member_online":
      case "member_offline":
        // 成员变化时刷新成员列表
        refetchMembers();
        break;
      case "skill_published":
      case "skill_deleted":
      case "skill_index_updated":
        // 技能变化时刷新技能列表
        refetchSkills();
        break;
      case "team_dissolved":
        // 团队解散时返回列表
        showToast(t("team.teamDissolved"), "error");
        onBack();
        onRefresh();
        break;
      case "member_status_changed": {
        // 成员状态变更事件
        const statusEvent = event as MemberStatusChangedEvent;
        if (statusEvent.payload.status_visible) {
          setMemberStatuses(prev => ({
            ...prev,
            [statusEvent.payload.member_id]: {
              project: statusEvent.payload.project_name,
              file: statusEvent.payload.current_file,
            },
          }));
        } else {
          setMemberStatuses(prev => {
            const { [statusEvent.payload.member_id]: _, ...rest } = prev;
            return rest;
          });
        }
        break;
      }
    }
  }, [refetchMembers, refetchSkills, showToast, t, onBack, onRefresh]);

  // 连接 WebSocket
  const { isConnected } = useTeamWebSocket({
    teamId: team.id,
    leaderEndpoint: team.leader_endpoint,
    onEvent: handleTeamEvent,
    enabled: true,
  });

  const handlePublished = useCallback(() => {
    setShowPublish(false);
    refetchSkills();
    showToast(t("team.publishSuccess"), "success");
  }, [refetchSkills, showToast, t]);

  const handleDownload = useCallback(async (skill: TeamSkillEntry) => {
    try {
      await apiService.downloadTeamSkill(
        team.id,
        skill.plugin_id,
        skill.author_endpoint,
        skill.checksum
      );
      showToast(t("team.downloadSuccess"), "success");
      refetchSkills();
    } catch (err: any) {
      showToast(err.message || t("team.downloadFailed"), "error");
    }
  }, [team.id, showToast, refetchSkills, t]);

  const handleInstall = useCallback(async (skill: TeamSkillEntry) => {
    try {
      await apiService.installTeamSkill(team.id, skill.plugin_id, skill.version);
      showToast(t("team.installSuccess"), "success");
    } catch (err: any) {
      showToast(err.message || t("team.installFailed"), "error");
    }
  }, [team.id, showToast, t]);

  const handleUninstall = useCallback(async (skill: TeamSkillEntry) => {
    try {
      await apiService.uninstallTeamSkill(team.id, skill.plugin_id);
      showToast(t("team.uninstallSuccess"), "success");
    } catch (err: any) {
      showToast(err.message || t("team.uninstallFailed"), "error");
    }
  }, [team.id, showToast, t]);

  const onlineCount = useMemo(() => members?.filter(m => m.is_online).length || 0, [members]);

  return (
    <div className="cocursor-team-detail">
      <ToastContainer toasts={toasts} />

      {/* 头部 */}
      <div className="cocursor-team-detail-header">
        <button className="cocursor-team-detail-back" onClick={onBack}>
          ← {t("common.back")}
        </button>
        <div className="cocursor-team-detail-title">
          <h2>{team.name}</h2>
          {team.is_leader && (
            <span className="cocursor-team-card-badge leader">{t("team.leader")}</span>
          )}
        </div>
      </div>

      {/* 团队信息 */}
      <div className="cocursor-team-detail-info">
        <div className="cocursor-team-detail-info-item">
          <span className="cocursor-team-detail-info-label">{t("team.leaderLabel")}</span>
          <span className="cocursor-team-detail-info-value">{team.leader_name}</span>
        </div>
        <div className="cocursor-team-detail-info-item">
          <span className="cocursor-team-detail-info-label">{t("team.onlineMembers")}</span>
          <span className="cocursor-team-detail-info-value">{onlineCount} / {members?.length || 0}</span>
        </div>
        <div className="cocursor-team-detail-info-item">
          <span className="cocursor-team-detail-info-label">{t("team.totalSkills")}</span>
          <span className="cocursor-team-detail-info-value">{skills?.length || 0}</span>
        </div>
      </div>

      {/* 选项卡 */}
      <div className="cocursor-team-detail-tabs">
        <button
          className={`cocursor-team-detail-tab ${activeTab === "members" ? "active" : ""}`}
          onClick={() => setActiveTab("members")}
        >
          <span className="cocursor-team-detail-tab-icon">👥</span>
          {t("team.members")} ({members?.length || 0})
        </button>
        <button
          className={`cocursor-team-detail-tab ${activeTab === "skills" ? "active" : ""}`}
          onClick={() => setActiveTab("skills")}
        >
          <span className="cocursor-team-detail-tab-icon">📦</span>
          {t("team.skills")} ({skills?.length || 0})
        </button>
        <button
          className={`cocursor-team-detail-tab ${activeTab === "weekly" ? "active" : ""}`}
          onClick={() => setActiveTab("weekly")}
        >
          <span className="cocursor-team-detail-tab-icon">📊</span>
          {t("weeklyReport.title")}
        </button>
      </div>

      {/* 成员列表 */}
      {activeTab === "members" && (
        <div className="cocursor-team-members">
          <div className="cocursor-team-section-header">
            <h3>{t("team.memberList")}</h3>
            <button className="cocursor-btn secondary" onClick={refetchMembers}>
              {t("common.refresh")}
            </button>
          </div>

          {loadingMembers ? (
            <div className="cocursor-team-loading">
              <div className="cocursor-team-loading-spinner"></div>
            </div>
          ) : members?.length === 0 ? (
            <div className="cocursor-team-empty-section">
              <span>{t("team.noMembers")}</span>
            </div>
          ) : (
            <div className="cocursor-team-member-list">
              {members?.map(member => (
                <MemberCard 
                  key={member.id} 
                  member={member} 
                  workStatus={memberStatuses[member.id]}
                />
              ))}
            </div>
          )}
        </div>
      )}

      {/* 技能列表 */}
      {activeTab === "skills" && (
        <div className="cocursor-team-skills">
          <div className="cocursor-team-section-header">
            <h3>{t("team.skillList")}</h3>
            <div className="cocursor-team-section-actions">
              <button className="cocursor-btn secondary" onClick={refetchSkills}>
                {t("common.refresh")}
              </button>
              <button className="cocursor-btn primary" onClick={() => setShowPublish(true)}>
                <span className="cocursor-btn-icon">📤</span>
                {t("team.publishSkill")}
              </button>
            </div>
          </div>

          {loadingSkills ? (
            <div className="cocursor-team-loading">
              <div className="cocursor-team-loading-spinner"></div>
            </div>
          ) : skills?.length === 0 ? (
            <div className="cocursor-team-empty-section">
              <span className="cocursor-team-empty-icon">📦</span>
              <span>{t("team.noSkills")}</span>
              <p>{t("team.noSkillsDesc")}</p>
            </div>
          ) : (
            <div className="cocursor-team-skill-list">
              {skills?.map(skill => (
                <SkillCard 
                  key={skill.plugin_id} 
                  skill={skill} 
                  onDownload={() => handleDownload(skill)}
                  onInstall={() => handleInstall(skill)}
                  onUninstall={() => handleUninstall(skill)}
                />
              ))}
            </div>
          )}
        </div>
      )}

      {/* 周报视图 */}
      {activeTab === "weekly" && (
        <WeeklyReport teamId={team.id} isLeader={team.is_leader} />
      )}

      {/* 发布技能弹窗 */}
      {showPublish && (
        <SkillPublish
          teamId={team.id}
          onClose={() => setShowPublish(false)}
          onSuccess={handlePublished}
        />
      )}
    </div>
  );
};

// 成员卡片
interface MemberCardProps {
  member: TeamMember;
  workStatus?: { project: string; file: string };
}

const MemberCard: React.FC<MemberCardProps> = ({ member, workStatus }) => {
  const { t } = useTranslation();

  return (
    <div className={`cocursor-team-member-card ${member.is_online ? "online" : "offline"}`}>
      <div className="cocursor-team-member-avatar">
        {member.name.charAt(0).toUpperCase()}
        <span className={`cocursor-team-member-status ${member.is_online ? "online" : "offline"}`}></span>
      </div>
      <div className="cocursor-team-member-info">
        <div className="cocursor-team-member-name">
          {member.name}
          {member.is_leader && (
            <span className="cocursor-team-card-badge leader small">{t("team.leader")}</span>
          )}
        </div>
        <div className="cocursor-team-member-meta">
          <span className={member.is_online ? "online" : "offline"}>
            {member.is_online ? t("team.online") : t("team.offline")}
          </span>
        </div>
        {/* 工作状态 */}
        {member.is_online && workStatus && (
          <div className="cocursor-team-member-work-status">
            <span className="cocursor-team-member-work-icon">💻</span>
            <span className="cocursor-team-member-work-project">{workStatus.project}</span>
            {workStatus.file && (
              <span className="cocursor-team-member-work-file">• {workStatus.file}</span>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

// 技能卡片
interface SkillCardProps {
  skill: TeamSkillEntry;
  onDownload: () => void;
  onInstall: () => void;
  onUninstall: () => void;
}

const SkillCard: React.FC<SkillCardProps> = ({ skill, onDownload, onInstall, onUninstall }) => {
  const { t } = useTranslation();
  const [downloading, setDownloading] = useState(false);
  const [installing, setInstalling] = useState(false);
  const [uninstalling, setUninstalling] = useState(false);

  const handleDownload = useCallback(async () => {
    setDownloading(true);
    try {
      await onDownload();
    } finally {
      setDownloading(false);
    }
  }, [onDownload]);

  const handleInstall = useCallback(async () => {
    setInstalling(true);
    try {
      await onInstall();
    } finally {
      setInstalling(false);
    }
  }, [onInstall]);

  const handleUninstall = useCallback(async () => {
    setUninstalling(true);
    try {
      await onUninstall();
    } finally {
      setUninstalling(false);
    }
  }, [onUninstall]);

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <div className="cocursor-team-skill-card">
      <div className="cocursor-team-skill-header">
        <div className="cocursor-team-skill-icon">📦</div>
        <div className="cocursor-team-skill-info">
          <h4 className="cocursor-team-skill-name">{skill.name}</h4>
          <div className="cocursor-team-skill-meta">
            <span className="cocursor-team-skill-version">v{skill.version}</span>
            <span className="cocursor-team-skill-author">{skill.author_name}</span>
            <span className="cocursor-team-skill-size">{formatSize(skill.total_size)}</span>
          </div>
        </div>
        <div className="cocursor-team-skill-actions">
          <button
            className="cocursor-btn secondary"
            onClick={handleDownload}
            disabled={downloading || installing || uninstalling}
            title={t("team.download")}
          >
            {downloading ? (
              <span className="cocursor-btn-spinner"></span>
            ) : (
              <span className="cocursor-btn-icon">⬇️</span>
            )}
          </button>
          <button
            className="cocursor-btn primary"
            onClick={handleInstall}
            disabled={downloading || installing || uninstalling}
            title={t("team.install")}
          >
            {installing ? (
              <>
                <span className="cocursor-btn-spinner"></span>
                {t("team.installing")}
              </>
            ) : (
              t("team.install")
            )}
          </button>
          <button
            className="cocursor-btn danger"
            onClick={handleUninstall}
            disabled={downloading || installing || uninstalling}
            title={t("team.uninstall")}
          >
            {uninstalling ? (
              <>
                <span className="cocursor-btn-spinner"></span>
                {t("team.uninstalling")}
              </>
            ) : (
              t("team.uninstall")
            )}
          </button>
        </div>
      </div>
      <p className="cocursor-team-skill-description">{skill.description}</p>
    </div>
  );
};
