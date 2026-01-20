/**
 * 创建团队组件
 */

import React, { useState, useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { NetworkInterface } from "../../types";
import { useApi } from "../../hooks";

interface TeamCreateProps {
  onClose: () => void;
  onSuccess: () => void;
}

export const TeamCreate: React.FC<TeamCreateProps> = ({ onClose, onSuccess }) => {
  const { t } = useTranslation();
  const [teamName, setTeamName] = useState("");
  const [selectedInterface, setSelectedInterface] = useState<string>("");
  const [selectedIP, setSelectedIP] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 获取网卡列表
  const fetchInterfaces = useCallback(async () => {
    const resp = await apiService.getNetworkInterfaces() as { interfaces: NetworkInterface[] };
    return resp.interfaces || [];
  }, []);

  const { data: interfaces } = useApi<NetworkInterface[]>(fetchInterfaces);

  // 当选中网卡变化时，自动选择第一个IP
  useEffect(() => {
    if (selectedInterface && interfaces) {
      const iface = interfaces.find(i => i.name === selectedInterface);
      if (iface && iface.addresses.length > 0) {
        setSelectedIP(iface.addresses[0]);
      } else {
        setSelectedIP("");
      }
    }
  }, [selectedInterface, interfaces]);

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
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
        selectedIP || undefined
      );
      onSuccess();
    } catch (err: any) {
      setError(err.message || t("team.createFailed"));
    } finally {
      setLoading(false);
    }
  }, [teamName, selectedInterface, selectedIP, onSuccess, t]);

  // 筛选出可用的网卡（排除 loopback）
  const availableInterfaces = interfaces?.filter(i => i.is_up && !i.is_loopback) || [];

  return (
    <div className="cocursor-modal-overlay" onClick={onClose}>
      <div className="cocursor-modal wide" onClick={e => e.stopPropagation()}>
        <div className="cocursor-modal-header">
          <h2 className="cocursor-modal-title">{t("team.createTeam")}</h2>
          <button className="cocursor-modal-close" onClick={onClose}>×</button>
        </div>

        <form className="cocursor-modal-body" onSubmit={handleSubmit}>
          <p className="cocursor-modal-desc">{t("team.createTeamDesc")}</p>

          <div className="cocursor-form-group">
            <label className="cocursor-form-label">{t("team.teamName")}</label>
            <input
              type="text"
              className="cocursor-form-input"
              value={teamName}
              onChange={e => setTeamName(e.target.value)}
              placeholder={t("team.teamNamePlaceholder")}
              autoFocus
            />
          </div>

          <div className="cocursor-form-group">
            <label className="cocursor-form-label">
              {t("team.networkInterface")}
              <span className="cocursor-form-label-hint">({t("team.optional")})</span>
            </label>
            <select
              className="cocursor-form-select"
              value={selectedInterface}
              onChange={e => setSelectedInterface(e.target.value)}
            >
              <option value="">{t("team.autoSelect")}</option>
              {availableInterfaces.map(iface => (
                <option key={iface.name} value={iface.name}>
                  {iface.name} ({iface.addresses.join(", ")})
                </option>
              ))}
            </select>
            <p className="cocursor-form-help">{t("team.networkInterfaceHelp")}</p>
          </div>

          {selectedInterface && (
            <div className="cocursor-form-group">
              <label className="cocursor-form-label">{t("team.ipAddress")}</label>
              <select
                className="cocursor-form-select"
                value={selectedIP}
                onChange={e => setSelectedIP(e.target.value)}
              >
                {interfaces
                  ?.find(i => i.name === selectedInterface)
                  ?.addresses.map(addr => (
                    <option key={addr} value={addr}>{addr}</option>
                  ))}
              </select>
            </div>
          )}

          {error && (
            <div className="cocursor-form-error">{error}</div>
          )}

          <div className="cocursor-team-create-info">
            <div className="cocursor-team-create-info-item">
              <span className="cocursor-team-create-info-icon">👑</span>
              <span>{t("team.createInfo1")}</span>
            </div>
            <div className="cocursor-team-create-info-item">
              <span className="cocursor-team-create-info-icon">📡</span>
              <span>{t("team.createInfo2")}</span>
            </div>
            <div className="cocursor-team-create-info-item">
              <span className="cocursor-team-create-info-icon">🔗</span>
              <span>{t("team.createInfo3")}</span>
            </div>
          </div>

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
              disabled={loading || !teamName.trim()}
            >
              {loading ? t("common.loading") : t("team.create")}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
