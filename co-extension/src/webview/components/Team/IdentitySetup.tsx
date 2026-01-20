/**
 * 身份设置组件
 */

import React, { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { Identity } from "../../types";

interface IdentitySetupProps {
  identity?: Identity;
  onClose: () => void;
  onSuccess: () => void;
}

export const IdentitySetup: React.FC<IdentitySetupProps> = ({
  identity,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const [name, setName] = useState(identity?.name || "");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!name.trim()) {
      setError(t("team.identityNameRequired"));
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await apiService.setTeamIdentity(name.trim());
      onSuccess();
    } catch (err: any) {
      setError(err.message || t("team.identitySetFailed"));
    } finally {
      setLoading(false);
    }
  }, [name, onSuccess, t]);

  return (
    <div className="cocursor-modal-overlay" onClick={onClose}>
      <div className="cocursor-modal" onClick={e => e.stopPropagation()}>
        <div className="cocursor-modal-header">
          <h2 className="cocursor-modal-title">
            {identity ? t("team.editIdentity") : t("team.setupIdentity")}
          </h2>
          <button className="cocursor-modal-close" onClick={onClose}>×</button>
        </div>

        <form className="cocursor-modal-body" onSubmit={handleSubmit}>
          <p className="cocursor-modal-desc">{t("team.identityDesc")}</p>

          <div className="cocursor-form-group">
            <label className="cocursor-form-label">{t("team.yourName")}</label>
            <input
              type="text"
              className="cocursor-form-input"
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder={t("team.namePlaceholder")}
              autoFocus
            />
          </div>

          {error && (
            <div className="cocursor-form-error">{error}</div>
          )}

          {identity && (
            <div className="cocursor-form-hint">
              <span className="cocursor-form-hint-label">{t("team.yourId")}：</span>
              <code className="cocursor-form-hint-value">{identity.id}</code>
            </div>
          )}

          <div className="cocursor-modal-footer">
            <button
              type="button"
              className="cocursor-btn secondary"
              onClick={onClose}
              disabled={loading}
            >
              {t("common.cancel")}
            </button>
            <button
              type="submit"
              className="cocursor-btn primary"
              disabled={loading || !name.trim()}
            >
              {loading ? t("common.loading") : t("common.save")}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
