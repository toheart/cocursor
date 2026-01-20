/**
 * 发布技能组件 - 支持元数据填写
 */

import React, { useState, useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { SkillValidationResult, SkillMetadata, SkillMetadataPrefill } from "../../types";

interface SkillPublishProps {
  teamId: string;
  onClose: () => void;
  onSuccess: () => void;
}

// 分类选项
const CATEGORY_OPTIONS = [
  { value: "productivity", labelKey: "marketplace.categoryProductivity" },
  { value: "creative", labelKey: "marketplace.categoryCreative" },
  { value: "design", labelKey: "marketplace.categoryDesign" },
  { value: "tools", labelKey: "marketplace.categoryTools" },
  { value: "other", labelKey: "marketplace.categoryOther" },
];

export const SkillPublish: React.FC<SkillPublishProps> = ({
  teamId,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const [step, setStep] = useState<"select" | "metadata" | "confirm">("select");
  const [selectedPath, setSelectedPath] = useState("");
  const [validation, setValidation] = useState<SkillValidationResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 元数据表单状态
  const [metadata, setMetadata] = useState<SkillMetadata>({
    plugin_id: "",
    name: "",
    name_zh_cn: "",
    description: "",
    description_zh_cn: "",
    version: "1.0.0",
    category: "other",
    author: "",
  });

  // 从验证结果中预填充元数据
  useEffect(() => {
    if (validation?.prefill) {
      const prefill = validation.prefill;
      setMetadata(prev => ({
        ...prev,
        plugin_id: selectedPath.split("/").pop()?.replace(/[^a-zA-Z0-9-_]/g, "-") || "my-skill",
        name: prefill.name || prev.name,
        name_zh_cn: prefill.name_zh_cn || prev.name_zh_cn,
        description: prefill.description || prev.description,
        description_zh_cn: prefill.description_zh_cn || prev.description_zh_cn,
        version: prefill.version || prev.version,
        category: prefill.category || prev.category,
        author: prefill.author || prev.author,
      }));
    }
  }, [validation, selectedPath]);

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
      setStep("metadata");
    } catch (err: any) {
      setError(err.message || t("team.validationFailed"));
    } finally {
      setLoading(false);
    }
  }, [selectedPath, t]);

  const handleMetadataChange = useCallback((field: keyof SkillMetadata, value: string) => {
    setMetadata(prev => ({ ...prev, [field]: value }));
  }, []);

  const validateMetadata = useCallback((): boolean => {
    if (!metadata.plugin_id.trim()) {
      setError(t("team.pluginIdRequired"));
      return false;
    }
    if (!metadata.name.trim()) {
      setError(t("team.nameRequired"));
      return false;
    }
    if (!metadata.description.trim()) {
      setError(t("team.descriptionRequired"));
      return false;
    }
    if (!metadata.version.trim()) {
      setError(t("team.versionRequired"));
      return false;
    }
    if (!metadata.author.trim()) {
      setError(t("team.authorRequired"));
      return false;
    }
    return true;
  }, [metadata, t]);

  const handleNext = useCallback(() => {
    if (!validateMetadata()) return;
    setError(null);
    setStep("confirm");
  }, [validateMetadata]);

  const handlePublish = useCallback(async () => {
    if (!validation || !validateMetadata()) return;

    setLoading(true);
    setError(null);

    try {
      await apiService.publishTeamSkillWithMetadata(teamId, selectedPath, metadata);
      onSuccess();
    } catch (err: any) {
      setError(err.message || t("team.publishFailed"));
    } finally {
      setLoading(false);
    }
  }, [validation, metadata, teamId, selectedPath, onSuccess, validateMetadata, t]);

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
            <div className={`cocursor-skill-publish-step ${step === "select" ? "active" : "completed"}`}>
              <span className="cocursor-skill-publish-step-number">1</span>
              <span className="cocursor-skill-publish-step-label">{t("team.selectDirectory")}</span>
            </div>
            <div className="cocursor-skill-publish-step-connector"></div>
            <div className={`cocursor-skill-publish-step ${step === "metadata" ? "active" : step === "confirm" ? "completed" : ""}`}>
              <span className="cocursor-skill-publish-step-number">2</span>
              <span className="cocursor-skill-publish-step-label">{t("team.fillMetadata")}</span>
            </div>
            <div className="cocursor-skill-publish-step-connector"></div>
            <div className={`cocursor-skill-publish-step ${step === "confirm" ? "active" : ""}`}>
              <span className="cocursor-skill-publish-step-number">3</span>
              <span className="cocursor-skill-publish-step-label">{t("team.confirmPublish")}</span>
            </div>
          </div>

          {error && (
            <div className="cocursor-form-error">{error}</div>
          )}

          {/* 步骤 1: 选择目录 */}
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

          {/* 步骤 2: 填写元数据 */}
          {step === "metadata" && (
            <div className="cocursor-skill-publish-metadata">
              <p className="cocursor-modal-desc">{t("team.fillMetadataDesc")}</p>

              {/* 来源提示 */}
              {validation?.source_type && (
                <div className="cocursor-skill-publish-source-hint">
                  {validation.source_type === "plugin" 
                    ? t("team.sourceFromPlugin")
                    : t("team.sourceFromSkillMD")}
                </div>
              )}

              <div className="cocursor-form-row">
                <div className="cocursor-form-group">
                  <label className="cocursor-form-label">{t("team.pluginId")} *</label>
                  <input
                    type="text"
                    className="cocursor-form-input"
                    value={metadata.plugin_id}
                    onChange={e => handleMetadataChange("plugin_id", e.target.value)}
                    placeholder={t("team.pluginIdPlaceholder")}
                  />
                  <p className="cocursor-form-help">{t("team.pluginIdHelp")}</p>
                </div>
                <div className="cocursor-form-group">
                  <label className="cocursor-form-label">{t("team.version")} *</label>
                  <input
                    type="text"
                    className="cocursor-form-input"
                    value={metadata.version}
                    onChange={e => handleMetadataChange("version", e.target.value)}
                    placeholder="1.0.0"
                  />
                </div>
              </div>

              <div className="cocursor-form-row">
                <div className="cocursor-form-group">
                  <label className="cocursor-form-label">{t("team.nameEn")} *</label>
                  <input
                    type="text"
                    className="cocursor-form-input"
                    value={metadata.name}
                    onChange={e => handleMetadataChange("name", e.target.value)}
                    placeholder={t("team.nameEnPlaceholder")}
                  />
                </div>
                <div className="cocursor-form-group">
                  <label className="cocursor-form-label">{t("team.nameZh")}</label>
                  <input
                    type="text"
                    className="cocursor-form-input"
                    value={metadata.name_zh_cn}
                    onChange={e => handleMetadataChange("name_zh_cn", e.target.value)}
                    placeholder={t("team.nameZhPlaceholder")}
                  />
                </div>
              </div>

              <div className="cocursor-form-group">
                <label className="cocursor-form-label">{t("team.descriptionEn")} *</label>
                <textarea
                  className="cocursor-form-input cocursor-form-textarea"
                  value={metadata.description}
                  onChange={e => handleMetadataChange("description", e.target.value)}
                  placeholder={t("team.descriptionEnPlaceholder")}
                  rows={2}
                />
              </div>

              <div className="cocursor-form-group">
                <label className="cocursor-form-label">{t("team.descriptionZh")}</label>
                <textarea
                  className="cocursor-form-input cocursor-form-textarea"
                  value={metadata.description_zh_cn}
                  onChange={e => handleMetadataChange("description_zh_cn", e.target.value)}
                  placeholder={t("team.descriptionZhPlaceholder")}
                  rows={2}
                />
              </div>

              <div className="cocursor-form-row">
                <div className="cocursor-form-group">
                  <label className="cocursor-form-label">{t("team.category")} *</label>
                  <select
                    className="cocursor-form-input"
                    value={metadata.category}
                    onChange={e => handleMetadataChange("category", e.target.value)}
                  >
                    {CATEGORY_OPTIONS.map(opt => (
                      <option key={opt.value} value={opt.value}>
                        {t(opt.labelKey)}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="cocursor-form-group">
                  <label className="cocursor-form-label">{t("team.author")} *</label>
                  <input
                    type="text"
                    className="cocursor-form-input"
                    value={metadata.author}
                    onChange={e => handleMetadataChange("author", e.target.value)}
                    placeholder={t("team.authorPlaceholder")}
                  />
                </div>
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
                  onClick={handleNext}
                >
                  {t("team.next")}
                </button>
              </div>
            </div>
          )}

          {/* 步骤 3: 确认发布 */}
          {step === "confirm" && validation && (
            <div className="cocursor-skill-publish-confirm">
              <div className="cocursor-skill-publish-preview">
                <div className="cocursor-skill-publish-preview-header">
                  <div className="cocursor-skill-publish-preview-icon">📦</div>
                  <div className="cocursor-skill-publish-preview-info">
                    <h3>{metadata.name}</h3>
                    {metadata.name_zh_cn && <p className="subtitle">{metadata.name_zh_cn}</p>}
                    <p>v{metadata.version}</p>
                  </div>
                </div>
                <p className="cocursor-skill-publish-preview-desc">{metadata.description}</p>
                {metadata.description_zh_cn && (
                  <p className="cocursor-skill-publish-preview-desc secondary">{metadata.description_zh_cn}</p>
                )}
                <div className="cocursor-skill-publish-preview-meta">
                  <span>{t("team.author")}: {metadata.author}</span>
                  <span>{t("marketplace.category")}: {t(`marketplace.category${metadata.category.charAt(0).toUpperCase() + metadata.category.slice(1)}`)}</span>
                </div>
                <div className="cocursor-skill-publish-preview-meta">
                  <span>{t("team.fileCount")}: {validation.files?.length || 0}</span>
                  <span>{t("team.totalSize")}: {formatSize(validation.total_size)}</span>
                </div>
              </div>

              <div className="cocursor-skill-publish-confirm-note">
                <p>{t("team.publishConfirmNote")}</p>
              </div>

              <div className="cocursor-modal-footer">
                <button
                  type="button"
                  className="cocursor-btn secondary"
                  onClick={() => setStep("metadata")}
                >
                  {t("common.back")}
                </button>
                <button
                  type="button"
                  className="cocursor-btn primary"
                  onClick={handlePublish}
                  disabled={loading}
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
