/**
 * 发布技能组件
 */

import React, { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { SkillValidationResult } from "../../types";

interface SkillPublishProps {
  teamId: string;
  onClose: () => void;
  onSuccess: () => void;
}

export const SkillPublish: React.FC<SkillPublishProps> = ({
  teamId,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const [step, setStep] = useState<"select" | "validate" | "publish">("select");
  const [selectedPath, setSelectedPath] = useState("");
  const [pluginId, setPluginId] = useState("");
  const [validation, setValidation] = useState<SkillValidationResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSelectDirectory = useCallback(async () => {
    try {
      const result = await apiService.selectDirectory() as { path?: string; cancelled?: boolean };
      if (result.path) {
        setSelectedPath(result.path);
        setError(null);
      }
    } catch (err: any) {
      setError(err.message || t("team.selectDirectoryFailed"));
    }
  }, [t]);

  const handleValidate = useCallback(async () => {
    if (!selectedPath) {
      setError(t("team.noDirectorySelected"));
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await apiService.validateSkillDirectory(selectedPath) as SkillValidationResult;
      
      if (!result.valid) {
        setError(result.error || t("team.validationFailed"));
        return;
      }

      setValidation(result);
      // 从目录名或技能名生成默认插件 ID
      const defaultId = selectedPath.split("/").pop()?.replace(/[^a-zA-Z0-9-_]/g, "-") || "my-skill";
      setPluginId(defaultId);
      setStep("validate");
    } catch (err: any) {
      setError(err.message || t("team.validationFailed"));
    } finally {
      setLoading(false);
    }
  }, [selectedPath, t]);

  const handlePublish = useCallback(async () => {
    if (!validation || !pluginId.trim()) {
      setError(t("team.pluginIdRequired"));
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await apiService.publishTeamSkill(teamId, pluginId.trim(), selectedPath);
      onSuccess();
    } catch (err: any) {
      setError(err.message || t("team.publishFailed"));
    } finally {
      setLoading(false);
    }
  }, [validation, pluginId, teamId, selectedPath, onSuccess, t]);

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <div className="cocursor-modal-overlay" onClick={onClose}>
      <div className="cocursor-modal wide" onClick={e => e.stopPropagation()}>
        <div className="cocursor-modal-header">
          <h2 className="cocursor-modal-title">{t("team.publishSkill")}</h2>
          <button className="cocursor-modal-close" onClick={onClose}>×</button>
        </div>

        <div className="cocursor-modal-body">
          {/* 步骤指示器 */}
          <div className="cocursor-skill-publish-steps">
            <div className={`cocursor-skill-publish-step ${step === "select" ? "active" : step !== "select" ? "completed" : ""}`}>
              <span className="cocursor-skill-publish-step-number">1</span>
              <span className="cocursor-skill-publish-step-label">{t("team.selectDirectory")}</span>
            </div>
            <div className="cocursor-skill-publish-step-connector"></div>
            <div className={`cocursor-skill-publish-step ${step === "validate" ? "active" : step === "publish" ? "completed" : ""}`}>
              <span className="cocursor-skill-publish-step-number">2</span>
              <span className="cocursor-skill-publish-step-label">{t("team.validateSkill")}</span>
            </div>
            <div className="cocursor-skill-publish-step-connector"></div>
            <div className={`cocursor-skill-publish-step ${step === "publish" ? "active" : ""}`}>
              <span className="cocursor-skill-publish-step-number">3</span>
              <span className="cocursor-skill-publish-step-label">{t("team.publish")}</span>
            </div>
          </div>

          {error && (
            <div className="cocursor-form-error">{error}</div>
          )}

          {/* 选择目录 */}
          {step === "select" && (
            <div className="cocursor-skill-publish-select">
              <p className="cocursor-modal-desc">{t("team.selectDirectoryDesc")}</p>

              <div className="cocursor-skill-publish-directory">
                <input
                  type="text"
                  className="cocursor-form-input"
                  value={selectedPath}
                  onChange={e => setSelectedPath(e.target.value)}
                  placeholder={t("team.directoryPlaceholder")}
                  readOnly
                />
                <button
                  type="button"
                  className="cocursor-btn secondary"
                  onClick={handleSelectDirectory}
                >
                  {t("team.browse")}
                </button>
              </div>

              <div className="cocursor-skill-publish-hint">
                <div className="cocursor-skill-publish-hint-item">
                  <span className="cocursor-skill-publish-hint-icon">📄</span>
                  <span>{t("team.skillRequirement1")}</span>
                </div>
                <div className="cocursor-skill-publish-hint-item">
                  <span className="cocursor-skill-publish-hint-icon">📁</span>
                  <span>{t("team.skillRequirement2")}</span>
                </div>
              </div>

              <div className="cocursor-modal-footer">
                <button
                  type="button"
                  className="cocursor-btn secondary"
                  onClick={onClose}
                >
                  {t("common.cancel")}
                </button>
                <button
                  type="button"
                  className="cocursor-btn primary"
                  onClick={handleValidate}
                  disabled={loading || !selectedPath}
                >
                  {loading ? t("team.validating") : t("team.next")}
                </button>
              </div>
            </div>
          )}

          {/* 验证结果 */}
          {step === "validate" && validation && (
            <div className="cocursor-skill-publish-validate">
              <div className="cocursor-skill-publish-preview">
                <div className="cocursor-skill-publish-preview-header">
                  <div className="cocursor-skill-publish-preview-icon">📦</div>
                  <div className="cocursor-skill-publish-preview-info">
                    <h3>{validation.name}</h3>
                    <p>v{validation.version}</p>
                  </div>
                </div>
                <p className="cocursor-skill-publish-preview-desc">{validation.description}</p>
                <div className="cocursor-skill-publish-preview-meta">
                  <span>{t("team.fileCount")}: {validation.files?.length || 0}</span>
                  <span>{t("team.totalSize")}: {formatSize(validation.total_size)}</span>
                </div>
              </div>

              <div className="cocursor-form-group">
                <label className="cocursor-form-label">{t("team.pluginId")}</label>
                <input
                  type="text"
                  className="cocursor-form-input"
                  value={pluginId}
                  onChange={e => setPluginId(e.target.value)}
                  placeholder={t("team.pluginIdPlaceholder")}
                />
                <p className="cocursor-form-help">{t("team.pluginIdHelp")}</p>
              </div>

              <div className="cocursor-modal-footer">
                <button
                  type="button"
                  className="cocursor-btn secondary"
                  onClick={() => setStep("select")}
                >
                  {t("common.back")}
                </button>
                <button
                  type="button"
                  className="cocursor-btn primary"
                  onClick={handlePublish}
                  disabled={loading || !pluginId.trim()}
                >
                  {loading ? t("team.publishing") : t("team.publish")}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
