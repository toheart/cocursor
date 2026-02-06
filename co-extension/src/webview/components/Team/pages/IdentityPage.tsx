/**
 * 身份设置页面（路由页面，替代弹窗）
 */

import React, { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { useIdentityStore } from "../stores";
import { PageHeader } from "../shared";
import { ToastContainer } from "../../shared/ToastContainer";
import { useToast } from "../../../hooks";

export const IdentityPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { showToast, toasts } = useToast();
  const { identity, hasIdentity, setIdentity } = useIdentityStore();

  const [name, setName] = useState(identity?.name || "");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();

      if (!name.trim()) {
        setError(t("team.identityNameRequired"));
        return;
      }

      setLoading(true);
      setError(null);

      try {
        await setIdentity(name.trim());
        showToast(
          hasIdentity ? t("team.identityUpdated") : t("team.identitySet"),
          "success",
        );
        navigate("/");
      } catch (err: unknown) {
        const message =
          err instanceof Error ? err.message : t("team.identitySetFailed");
        setError(message);
      } finally {
        setLoading(false);
      }
    },
    [name, hasIdentity, setIdentity, showToast, navigate, t],
  );

  return (
    <div className="ct-page">
      <ToastContainer toasts={toasts} />
      <PageHeader
        title={hasIdentity ? t("team.editIdentity") : t("team.setupIdentity")}
        backTo="/"
      />

      <form className="ct-page-body" onSubmit={handleSubmit}>
        <div className="ct-form-section">
          <p className="ct-form-desc">{t("team.identityDesc")}</p>

          <div className="ct-form-group">
            <label className="ct-form-label">{t("team.yourName")}</label>
            <input
              type="text"
              className="ct-form-input"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("team.namePlaceholder")}
              autoFocus
            />
          </div>

          {error && <div className="ct-form-error">{error}</div>}

          {identity && (
            <div className="ct-form-hint-box">
              <span className="ct-form-hint-label">{t("team.yourId")}:</span>
              <code className="ct-form-hint-value">{identity.id}</code>
            </div>
          )}
        </div>

        <div className="ct-page-actions">
          <button
            type="button"
            className="ct-btn secondary"
            onClick={() => navigate("/")}
            disabled={loading}
          >
            {t("common.cancel")}
          </button>
          <button
            type="submit"
            className="ct-btn primary"
            disabled={loading || !name.trim()}
          >
            {loading && <span className="ct-btn-spinner" />}
            {loading ? t("common.loading") : t("common.save")}
          </button>
        </div>
      </form>
    </div>
  );
};
