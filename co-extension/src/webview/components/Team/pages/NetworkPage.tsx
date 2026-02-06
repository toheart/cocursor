/**
 * 网络设置页面（路由页面，替代弹窗）
 */

import React, { useState, useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { apiService } from "../../../services/api";
import { NetworkInterface } from "../../../types";
import { useApi, useToast } from "../../../hooks";
import { PageHeader, LoadingState } from "../shared";
import { ToastContainer } from "../../shared/ToastContainer";

interface NetworkConfigResponse {
  interfaces: NetworkInterface[];
  config: {
    preferred_interface?: string;
    preferred_ip?: string;
  };
  current_endpoint: string;
}

export const NetworkPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { showToast, toasts } = useToast();

  const [selectedInterface, setSelectedInterface] = useState<string>("");
  const [selectedIP, setSelectedIP] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 获取网络配置
  const fetchNetworkConfig = useCallback(async () => {
    const resp =
      (await apiService.getNetworkInterfaces()) as NetworkConfigResponse;
    return resp;
  }, []);

  const { data: networkConfig, loading: loadingConfig } =
    useApi<NetworkConfigResponse>(fetchNetworkConfig);

  // 初始化选中的网卡和 IP
  useEffect(() => {
    if (networkConfig) {
      if (networkConfig.config?.preferred_interface) {
        setSelectedInterface(networkConfig.config.preferred_interface);
      }
      if (networkConfig.config?.preferred_ip) {
        setSelectedIP(networkConfig.config.preferred_ip);
      } else if (networkConfig.current_endpoint) {
        const ip = networkConfig.current_endpoint.split(":")[0];
        setSelectedIP(ip);
        const iface = networkConfig.interfaces?.find((i) =>
          i.addresses.includes(ip),
        );
        if (iface) {
          setSelectedInterface(iface.name);
        }
      }
    }
  }, [networkConfig]);

  // 当选中网卡变化时，自动选择第一个 IP
  useEffect(() => {
    if (selectedInterface && networkConfig?.interfaces) {
      const iface = networkConfig.interfaces.find(
        (i) => i.name === selectedInterface,
      );
      if (iface && iface.addresses.length > 0) {
        if (!iface.addresses.includes(selectedIP)) {
          setSelectedIP(iface.addresses[0]);
        }
      }
    }
  }, [selectedInterface, networkConfig, selectedIP]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();

      if (!selectedInterface || !selectedIP) {
        setError(t("network.selectRequired"));
        return;
      }

      setLoading(true);
      setError(null);

      try {
        await apiService.updateNetworkConfig(selectedInterface, selectedIP);
        showToast(t("network.updateSuccess"), "success");
        navigate("/");
      } catch (err: unknown) {
        const message =
          err instanceof Error ? err.message : t("network.updateFailed");
        setError(message);
      } finally {
        setLoading(false);
      }
    },
    [selectedInterface, selectedIP, showToast, navigate, t],
  );

  const availableInterfaces =
    networkConfig?.interfaces?.filter((i) => i.is_up && !i.is_loopback) || [];

  const selectedInterfaceIPs = selectedInterface
    ? networkConfig?.interfaces?.find((i) => i.name === selectedInterface)
        ?.addresses || []
    : [];

  return (
    <div className="ct-page">
      <ToastContainer toasts={toasts} />
      <PageHeader title={t("network.settings")} backTo="/" />

      <form className="ct-page-body" onSubmit={handleSubmit}>
        {loadingConfig ? (
          <LoadingState />
        ) : (
          <div className="ct-form-section">
            {/* 当前端点 */}
            <div className="ct-form-group">
              <label className="ct-form-label">
                {t("network.currentEndpoint")}
              </label>
              <div className="ct-network-endpoint">
                <code>{networkConfig?.current_endpoint || "-"}</code>
              </div>
              <p className="ct-form-help">
                {t("network.currentEndpointHelp")}
              </p>
            </div>

            {/* 网卡选择 */}
            <div className="ct-form-group">
              <label className="ct-form-label">
                {t("network.selectInterface")}
              </label>
              <select
                className="ct-form-select"
                value={selectedInterface}
                onChange={(e) => setSelectedInterface(e.target.value)}
              >
                <option value="">
                  {t("network.selectInterfacePlaceholder")}
                </option>
                {availableInterfaces.map((iface) => (
                  <option key={iface.name} value={iface.name}>
                    {iface.is_virtual ? "[virtual] " : ""}
                    {iface.name} ({iface.addresses.join(", ")})
                  </option>
                ))}
              </select>
              {/* 虚拟网卡警告 */}
              {selectedInterface &&
                networkConfig?.interfaces?.find(
                  (i) => i.name === selectedInterface,
                )?.is_virtual && (
                  <div className="ct-warning-box">
                    <span className="codicon codicon-warning" />
                    <span>{t("network.virtualInterfaceWarning")}</span>
                  </div>
                )}
            </div>

            {/* IP 选择 */}
            {selectedInterface && selectedInterfaceIPs.length > 0 && (
              <div className="ct-form-group">
                <label className="ct-form-label">
                  {t("network.selectIP")}
                </label>
                <select
                  className="ct-form-select"
                  value={selectedIP}
                  onChange={(e) => setSelectedIP(e.target.value)}
                >
                  {selectedInterfaceIPs.map((ip) => (
                    <option key={ip} value={ip}>
                      {ip}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {/* 修改警告 */}
            <div className="ct-warning-box">
              <span className="codicon codicon-warning" />
              <span>{t("network.changeWarning")}</span>
            </div>

            {error && <div className="ct-form-error">{error}</div>}
          </div>
        )}

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
            disabled={
              loading || loadingConfig || !selectedInterface || !selectedIP
            }
          >
            {loading && <span className="ct-btn-spinner" />}
            {loading ? t("common.loading") : t("common.save")}
          </button>
        </div>
      </form>
    </div>
  );
};
