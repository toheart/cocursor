/**
 * 成员列表 Tab 页面
 */

import React from "react";
import { useTranslation } from "react-i18next";
import { useMemberStore } from "../stores";
import { MemberCard } from "../cards";
import { EmptyState, LoadingState } from "../shared";

export const MembersPage: React.FC = () => {
  const { t } = useTranslation();
  const { members, loadingMembers, memberStatuses, fetchMembers, currentTeamId } =
    useMemberStore();

  return (
    <div className="ct-members-page">
      <div className="ct-section-header">
        <h3 className="ct-section-title">{t("team.memberList")}</h3>
        <button
          className="ct-btn-icon"
          onClick={() => currentTeamId && fetchMembers(currentTeamId)}
          title={t("common.refresh")}
        >
          <span className="codicon codicon-refresh" />
        </button>
      </div>

      {loadingMembers && members.length === 0 ? (
        <LoadingState />
      ) : members.length === 0 ? (
        <EmptyState icon="people" title={t("team.noMembers")} />
      ) : (
        <div className="ct-member-grid">
          {members.map((member) => (
            <MemberCard
              key={member.id}
              member={member}
              workStatus={memberStatuses[member.id]}
            />
          ))}
        </div>
      )}
    </div>
  );
};
