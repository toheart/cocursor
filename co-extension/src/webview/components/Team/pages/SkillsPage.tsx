/**
 * 技能管理 Tab 页面
 */

import React, { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useParams } from "react-router-dom";
import { apiService } from "../../../services/api";
import { TeamSkillEntry } from "../../../types";
import { useToast } from "../../../hooks";
import { useMemberStore } from "../stores";
import { SkillCard } from "../cards";
import { EmptyState, LoadingState } from "../shared";
import { SkillPublish } from "../SkillPublish";
import { ToastContainer } from "../../shared/ToastContainer";

export const SkillsPage: React.FC = () => {
  const { t } = useTranslation();
  const { teamId } = useParams<{ teamId: string }>();
  const { showToast, toasts } = useToast();
  const { skills, loadingSkills, fetchSkills } = useMemberStore();

  const [showPublish, setShowPublish] = useState(false);

  const handleDownload = useCallback(
    async (skill: TeamSkillEntry) => {
      if (!teamId) return;
      try {
        await apiService.downloadTeamSkill(
          teamId,
          skill.plugin_id,
          skill.author_endpoint,
          skill.checksum,
        );
        showToast(t("team.downloadSuccess"), "success");
        fetchSkills(teamId);
      } catch (err: unknown) {
        const message =
          err instanceof Error ? err.message : t("team.downloadFailed");
        showToast(message, "error");
      }
    },
    [teamId, showToast, fetchSkills, t],
  );

  const handleInstall = useCallback(
    async (skill: TeamSkillEntry) => {
      if (!teamId) return;
      try {
        await apiService.installTeamSkill(
          teamId,
          skill.plugin_id,
          skill.version,
        );
        showToast(t("team.installSuccess"), "success");
      } catch (err: unknown) {
        const message =
          err instanceof Error ? err.message : t("team.installFailed");
        showToast(message, "error");
      }
    },
    [teamId, showToast, t],
  );

  const handleUninstall = useCallback(
    async (skill: TeamSkillEntry) => {
      if (!teamId) return;
      try {
        await apiService.uninstallTeamSkill(teamId, skill.plugin_id);
        showToast(t("team.uninstallSuccess"), "success");
      } catch (err: unknown) {
        const message =
          err instanceof Error ? err.message : t("team.uninstallFailed");
        showToast(message, "error");
      }
    },
    [teamId, showToast, t],
  );

  const handlePublished = useCallback(() => {
    setShowPublish(false);
    if (teamId) fetchSkills(teamId);
    showToast(t("team.publishSuccess"), "success");
  }, [teamId, fetchSkills, showToast, t]);

  return (
    <div className="ct-skills-page">
      <ToastContainer toasts={toasts} />

      <div className="ct-section-header">
        <h3 className="ct-section-title">{t("team.skillList")}</h3>
        <div className="ct-section-actions">
          <button
            className="ct-btn-icon"
            onClick={() => teamId && fetchSkills(teamId)}
            title={t("common.refresh")}
          >
            <span className="codicon codicon-refresh" />
          </button>
          <button
            className="ct-btn primary small"
            onClick={() => setShowPublish(true)}
          >
            <span className="codicon codicon-cloud-upload" />
            {t("team.publishSkill")}
          </button>
        </div>
      </div>

      {loadingSkills && skills.length === 0 ? (
        <LoadingState />
      ) : skills.length === 0 ? (
        <EmptyState
          icon="package"
          title={t("team.noSkills")}
          description={t("team.noSkillsDesc")}
        />
      ) : (
        <div className="ct-skill-list">
          {skills.map((skill) => (
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

      {/* 发布技能弹窗（保留弹窗形式） */}
      {showPublish && teamId && (
        <SkillPublish
          teamId={teamId}
          onClose={() => setShowPublish(false)}
          onSuccess={handlePublished}
        />
      )}
    </div>
  );
};
