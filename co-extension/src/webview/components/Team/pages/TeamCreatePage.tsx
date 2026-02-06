/**
 * 创建团队页面（路由页面，替代弹窗）
 */

import React, { useState, useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { apiService } from "../../../services/api";
import { NetworkInterface } from "../../../types";
import { useApi, useToast } from "../../../hooks";
import { useTeamStore } from "../stores";
import { PageHeader } from "../shared";
import { ToastContainer } from "../../shared/ToastContainer";

export const TeamCreatePage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { showToast, toasts } = useToast();
  const { fetchTeams } = useTeamStore();

  const [teamName, setTeamName] = useState("");
  const [selectedInterface, setSelectedInterface] = useState<string>("");
  const [selectedIP, setSelectedIP] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 获取网卡列表
  const fetchInterfaces = useCallback(async () => {
    const resp = (await apiService.getNetworkInterfaces()) as {
      interfaces: NetworkInterface[];
    };
    return resp.interfaces || [];
  }, []);

  const { data: interfaces } = useApi<NetworkInterface[]>(fetchInterfaces);

  // 当选中网卡变化时，自动选择第一个 IP
  useEffect(() => {
    if (selectedInterface && interfaces) {
      const iface = interfaces.find((i) => i.name === selectedInterface);
      if (iface && iface.addresses.length > 0) {
        setSelectedIP(iface.addresses[0]);
      } else {
        setSelectedIP("");
      }
    }
  }, [selectedInterface, interfaces]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();

      if (!teamName.trim()) {
        setError(t("team.teamNameRequired"));
        return;
      }

      setLoading(true);
      setError(null);

      try {
        await apiService.createTeam(
          teamName.trim(),
          selectedInterface || undefined,
          selectedIP || undefined,
        );
        await fetchTeams();
        showToast(t("team.createSuccess"), "success");
        navigate("/");
      } catch (err: unknown) {
        const message =
          err instanceof Error ? err.message : t("team.createFailed");
        setError(message);
      } finally {
        setLoading(false);
      }
    },
    [teamName, selectedInterface, selectedIP, fetchTeams, showToast, navigate, t],
  );

  // 筛选出可用的网卡（排除 loopback）
  const availableInterfaces =
    interfaces?.filter((i) => i.is_up && !i.is_loopback) || [];

  return (
    <div className="ct-page">
      <ToastContainer toasts={toasts} />
      <PageHeader title={t("team.createTeam")} backTo="/" />

      <form className="ct-page-body" onSubmit={handleSubmit}>
        <div className="ct-form-section">
          <div className="ct-form-group">
            <label className="ct-form-label">{t("team.teamName")}</label>
            <input
              type="text"
              className="ct-form-input"
              value={teamName}
              onChange={(e) => setTeamName(e.target.value)}
              placeholder={t("team.teamNamePlaceholder")}
              autoFocus
            />
          </div>

          <div className="ct-form-group">
            <label className="ct-form-label">
              {t("team.networkInterface")}
              <span className="ct-form-hint-inline">
                ({t("team.optional")})
              </span>
            </label>
            <select
              className="ct-form-select"
              value={selectedInterface}
              onChange={(e) => setSelectedInterface(e.target.value)}
            >
              <option value="">{t("team.autoSelect")}</option>
              {availableInterfaces.map((iface) => (
                <option key={iface.name} value={iface.name}>
                  {iface.name} ({iface.addresses.join(", ")})
                </option>
              ))}
            </select>
            <p className="ct-form-help">{t("team.networkInterfaceHelp")}</p>
          </div>

          {selectedInterface && (
            <div className="ct-form-group">
              <label className="ct-form-label">{t("team.ipAddress")}</label>
              <select
                className="ct-form-select"
                value={selectedIP}
                onChange={(e) => setSelectedIP(e.target.value)}
              >
                {interfaces
                  ?.find((i) => i.name === selectedInterface)
                  ?.addresses.map((addr) => (
                    <option key={addr} value={addr}>
                      {addr}
                    </option>
                  ))}
              </select>
            </div>
          )}
        </div>

        {error && <div className="ct-form-error">{error}</div>}

        <div className="ct-info-box">
          <div className="ct-info-item">
            <span className="codicon codicon-star-full" />
            <span>{t("team.createInfo1")}</span>
          </div>
          <div className="ct-info-item">
            <span className="codicon codicon-broadcast" />
            <span>{t("team.createInfo2")}</span>
          </div>
          <div className="ct-info-item">
            <span className="codicon codicon-link" />
            <span>{t("team.createInfo3")}</span>
          </div>
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
            disabled={loading || !teamName.trim()}
          >
            {loading && <span className="ct-btn-spinner" />}
            {loading ? t("common.loading") : t("team.create")}
          </button>
        </div>
      </form>
    </div>
  );
};
