/**
 * 技能卡片组件（带状态感知）
 * 根据下载/安装状态显示不同操作按钮
 */

import React, { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { TeamSkillEntry } from "../../../types";

interface SkillCardProps {
  skill: TeamSkillEntry;
  onDownload: () => Promise<void>;
  onInstall: () => Promise<void>;
  onUninstall: () => Promise<void>;
}

/** 格式化文件大小 */
function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export const SkillCard: React.FC<SkillCardProps> = ({
  skill,
  onDownload,
  onInstall,
  onUninstall,
}) => {
  const { t } = useTranslation();
  const [actionLoading, setActionLoading] = useState(false);

  const handleAction = useCallback(
    async (action: () => Promise<void>) => {
      setActionLoading(true);
      try {
        await action();
      } finally {
        setActionLoading(false);
      }
    },
    [],
  );

  // 判断技能状态（基于已有字段推断）
  // TODO: 后端应增加 is_downloaded / is_installed 字段
  const isDownloaded = (skill as any).is_downloaded ?? false;
  const isInstalled = (skill as any).is_installed ?? false;

  return (
    <div className="ct-skill-card">
      <div className="ct-skill-card-header">
        <div className="ct-skill-card-icon">
          <span className="codicon codicon-package" />
        </div>
        <div className="ct-skill-card-info">
          <h4 className="ct-skill-card-name">
            {skill.name}
            {isInstalled && (
              <span className="ct-badge success">{t("team.installed")}</span>
            )}
            {!isInstalled && isDownloaded && (
              <span className="ct-badge info">{t("team.downloaded")}</span>
            )}
          </h4>
          <div className="ct-skill-card-meta">
            <span className="ct-skill-version">v{skill.version}</span>
            <span>{skill.author_name}</span>
            <span>{formatSize(skill.total_size)}</span>
          </div>
        </div>
        <div className="ct-skill-card-actions">
          {/* 状态感知的按钮：根据状态显示不同操作 */}
          {isInstalled ? (
            <button
              className="ct-btn danger small"
              onClick={() => handleAction(onUninstall)}
              disabled={actionLoading}
              title={t("team.uninstall")}
            >
              {actionLoading ? (
                <span className="ct-btn-spinner" />
              ) : (
                <span className="codicon codicon-trash" />
              )}
              {t("team.uninstall")}
            </button>
          ) : isDownloaded ? (
            <button
              className="ct-btn primary small"
              onClick={() => handleAction(onInstall)}
              disabled={actionLoading}
              title={t("team.install")}
            >
              {actionLoading ? (
                <span className="ct-btn-spinner" />
              ) : (
                <span className="codicon codicon-add" />
              )}
              {t("team.install")}
            </button>
          ) : (
            <button
              className="ct-btn secondary small"
              onClick={() => handleAction(onDownload)}
              disabled={actionLoading}
              title={t("team.download")}
            >
              {actionLoading ? (
                <span className="ct-btn-spinner" />
              ) : (
                <span className="codicon codicon-cloud-download" />
              )}
              {t("team.download")}
            </button>
          )}
        </div>
      </div>
      {skill.description && (
        <p className="ct-skill-card-desc">{skill.description}</p>
      )}
    </div>
  );
};
