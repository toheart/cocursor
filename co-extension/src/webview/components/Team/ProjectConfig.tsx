/**
 * 团队项目配置组件
 */

import React, { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { TeamProjectConfig as TeamProjectConfigType, ProjectMatcher } from "../../types";
import { useToast } from "../../hooks";
import { ToastContainer } from "../shared/ToastContainer";

interface ProjectConfigProps {
  teamId: string;
  config?: TeamProjectConfigType | null;
  onClose: () => void;
  onUpdated: () => void;
}

export const ProjectConfig: React.FC<ProjectConfigProps> = ({
  teamId,
  config,
  onClose,
  onUpdated,
}) => {
  const { t } = useTranslation();
  const { showToast, toasts } = useToast();

  // 表单状态
  const [projectName, setProjectName] = useState("");
  const [repoUrl, setRepoUrl] = useState("");
  const [adding, setAdding] = useState(false);
  const [selectingFolder, setSelectingFolder] = useState(false);
  const [removingId, setRemovingId] = useState<string | null>(null);

  // 项目列表
  const projects = config?.projects || [];

  // 添加项目
  const handleAdd = useCallback(async () => {
    if (!projectName.trim()) {
      showToast(t("weeklyReport.projectNameRequired"), "error");
      return;
    }
    if (!repoUrl.trim()) {
      showToast(t("weeklyReport.repoUrlRequired"), "error");
      return;
    }

    // 标准化 repo URL（移除协议和 .git 后缀）
    let normalizedUrl = repoUrl.trim();
    normalizedUrl = normalizedUrl.replace(/^(https?:\/\/|git@)/, "");
    normalizedUrl = normalizedUrl.replace(/\.git$/, "");
    normalizedUrl = normalizedUrl.replace(/:/, "/"); // git@ 格式转换

    setAdding(true);
    try {
      await apiService.addTeamProject(teamId, {
        name: projectName.trim(),
        repo_url: normalizedUrl,
      });
      showToast(t("weeklyReport.addProjectSuccess"), "success");
      setProjectName("");
      setRepoUrl("");
      onUpdated();
    } catch (err: any) {
      showToast(err.message || t("weeklyReport.addProjectFailed"), "error");
    } finally {
      setAdding(false);
    }
  }, [teamId, projectName, repoUrl, showToast, onUpdated, t]);

  // 移除项目
  const handleRemove = useCallback(async (projectId: string) => {
    setRemovingId(projectId);
    try {
      await apiService.removeTeamProject(teamId, projectId);
      showToast(t("weeklyReport.removeProjectSuccess"), "success");
      onUpdated();
    } catch (err: any) {
      showToast(err.message || t("weeklyReport.removeProjectFailed"), "error");
    } finally {
      setRemovingId(null);
    }
  }, [teamId, showToast, onUpdated, t]);

  // 选择文件夹添加项目
  const handleSelectFolder = useCallback(async () => {
    setSelectingFolder(true);
    try {
      const result = await apiService.selectFolder();
      if (result && result.path) {
        // 调用后端 API 通过路径添加项目
        await apiService.addTeamProjectByPath(teamId, result.path);
        showToast(t("weeklyReport.addProjectSuccess"), "success");
        onUpdated();
      }
    } catch (err: any) {
      showToast(err.message || t("weeklyReport.addProjectFailed"), "error");
    } finally {
      setSelectingFolder(false);
    }
  }, [teamId, showToast, onUpdated, t]);

  return (
    <div className="cocursor-modal-overlay" onClick={onClose}>
      <div
        className="cocursor-modal cocursor-project-config-modal"
        onClick={(e) => e.stopPropagation()}
      >
        <ToastContainer toasts={toasts} />

        <div className="cocursor-modal-header">
          <h2>{t("weeklyReport.projectConfig")}</h2>
          <button className="cocursor-modal-close" onClick={onClose}>
            ×
          </button>
        </div>

        <div className="cocursor-modal-body">
          {/* 说明 */}
          <div className="cocursor-project-config-desc">
            <p>{t("weeklyReport.projectConfigDesc")}</p>
          </div>

          {/* 选择文件夹快捷方式 */}
          <div className="cocursor-project-config-quick">
            <button
              className="cocursor-btn primary cocursor-btn-full-width"
              onClick={handleSelectFolder}
              disabled={selectingFolder}
            >
              {selectingFolder ? t("common.loading") : `📁 ${t("weeklyReport.selectFolder")}`}
            </button>
            <p className="cocursor-form-hint">{t("weeklyReport.selectFolderHint")}</p>
          </div>

          {/* 手动添加项目表单（可折叠） */}
          <details className="cocursor-project-config-form">
            <summary>{t("weeklyReport.manualAdd")}</summary>
            <div className="cocursor-form-row">
              <div className="cocursor-form-group">
                <label>{t("weeklyReport.projectName")}</label>
                <input
                  type="text"
                  value={projectName}
                  onChange={(e) => setProjectName(e.target.value)}
                  placeholder={t("weeklyReport.projectNamePlaceholder")}
                />
              </div>
              <div className="cocursor-form-group flex-grow">
                <label>{t("weeklyReport.repoUrl")}</label>
                <input
                  type="text"
                  value={repoUrl}
                  onChange={(e) => setRepoUrl(e.target.value)}
                  placeholder={t("weeklyReport.repoUrlPlaceholder")}
                />
              </div>
              <button
                className="cocursor-btn primary"
                onClick={handleAdd}
                disabled={adding}
              >
                {adding ? t("common.loading") : t("weeklyReport.add")}
              </button>
            </div>
            <p className="cocursor-form-hint">{t("weeklyReport.repoUrlHint")}</p>
          </details>

          {/* 项目列表 */}
          <div className="cocursor-project-config-list">
            <h3>{t("weeklyReport.configuredProjects")}</h3>
            {projects.length === 0 ? (
              <div className="cocursor-team-empty-section small">
                <span>{t("weeklyReport.noProjectsConfigured")}</span>
              </div>
            ) : (
              <div className="cocursor-project-list">
                {projects.map((project) => (
                  <ProjectItem
                    key={project.id}
                    project={project}
                    onRemove={() => handleRemove(project.id)}
                    removing={removingId === project.id}
                  />
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="cocursor-modal-footer">
          <button className="cocursor-btn secondary" onClick={onClose}>
            {t("common.close")}
          </button>
        </div>
      </div>
    </div>
  );
};

// 项目列表项组件
interface ProjectItemProps {
  project: ProjectMatcher;
  onRemove: () => void;
  removing: boolean;
}

const ProjectItem: React.FC<ProjectItemProps> = ({ project, onRemove, removing }) => {
  const { t } = useTranslation();

  return (
    <div className="cocursor-project-item">
      <div className="cocursor-project-item-info">
        <span className="cocursor-project-item-name">{project.name}</span>
        <span className="cocursor-project-item-url">{project.repo_url}</span>
      </div>
      <button
        className="cocursor-btn danger small"
        onClick={onRemove}
        disabled={removing}
      >
        {removing ? t("common.loading") : t("common.delete")}
      </button>
    </div>
  );
};

export default ProjectConfig;
