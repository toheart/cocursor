/**
 * 成员卡片组件
 */

import React from "react";
import { useTranslation } from "react-i18next";
import { TeamMember } from "../../../types";
import { StatusDot } from "../shared";

interface MemberCardProps {
  member: TeamMember;
  workStatus?: { project: string; file: string };
}

export const MemberCard: React.FC<MemberCardProps> = ({
  member,
  workStatus,
}) => {
  const { t } = useTranslation();

  return (
    <div
      className={`ct-member-card ${member.is_online ? "online" : "offline"}`}
    >
      <div className="ct-member-avatar">
        {member.name.charAt(0).toUpperCase()}
        <StatusDot online={member.is_online} size="small" />
      </div>
      <div className="ct-member-info">
        <div className="ct-member-name">
          {member.name}
          {member.is_leader && (
            <span className="ct-badge leader small">{t("team.leader")}</span>
          )}
        </div>
        <div className="ct-member-meta">
          <span className={member.is_online ? "online" : "offline"}>
            {member.is_online ? t("team.online") : t("team.offline")}
          </span>
        </div>
        {/* 实时工作状态 */}
        {member.is_online && workStatus && (
          <div className="ct-member-work-status">
            <span className="codicon codicon-file-code" />
            <span className="ct-member-work-project">
              {workStatus.project}
            </span>
            {workStatus.file && (
              <span className="ct-member-work-file">
                &bull; {workStatus.file}
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
