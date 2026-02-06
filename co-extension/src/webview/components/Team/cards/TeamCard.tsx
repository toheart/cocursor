/**
 * 团队卡片组件
 * 不再创建 WebSocket 连接，仅展示团队基本信息
 */

import React from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { Team } from "../../../types";
import { StatusDot } from "../shared";

interface TeamCardProps {
  team: Team;
  onLeave: () => void;
  onDissolve: () => void;
}

export const TeamCard: React.FC<TeamCardProps> = ({
  team,
  onLeave,
  onDissolve,
}) => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <div
      className={`ct-team-card ${team.is_leader ? "leader" : ""}`}
      onClick={() => navigate(`/team/${team.id}/members`)}
    >
      <div className="ct-team-card-header">
        <div className="ct-team-card-icon">
          <span className={`codicon codicon-${team.is_leader ? "star-full" : "organization"}`} />
        </div>
        <div className="ct-team-card-info">
          <h3 className="ct-team-card-name">
            {team.name}
            {team.is_leader && (
              <span className="ct-badge leader">{t("team.leader")}</span>
            )}
          </h3>
          <div className="ct-team-card-meta">
            <span>{t("team.leaderLabel")}: {team.leader_name}</span>
            <StatusDot
              online={team.is_leader || team.leader_online}
              label={
                team.is_leader || team.leader_online
                  ? t("team.connected")
                  : t("team.disconnected")
              }
            />
          </div>
        </div>
      </div>

      <div className="ct-team-card-stats">
        <div className="ct-team-card-stat">
          <span className="ct-team-card-stat-value">{team.member_count}</span>
          <span className="ct-team-card-stat-label">{t("team.members")}</span>
        </div>
        <div className="ct-team-card-stat">
          <span className="ct-team-card-stat-value">{team.skill_count}</span>
          <span className="ct-team-card-stat-label">{t("team.skills")}</span>
        </div>
      </div>

      <div className="ct-team-card-actions" onClick={(e) => e.stopPropagation()}>
        {team.is_leader ? (
          <button className="ct-btn danger small" onClick={onDissolve}>
            {t("team.dissolve")}
          </button>
        ) : (
          <button className="ct-btn secondary small" onClick={onLeave}>
            {t("team.leave")}
          </button>
        )}
      </div>
    </div>
  );
};
