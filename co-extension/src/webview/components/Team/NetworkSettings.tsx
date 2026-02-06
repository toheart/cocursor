/**
 * 网络设置组件
 * 允许用户查看和修改当前使用的网卡/IP
 */

import React, { useState, useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { NetworkInterface } from "../../types";
import { useApi } from "../../hooks";

interface NetworkSettingsProps {
  onClose: () => void;
  onSuccess?: () => void;
}

interface NetworkConfigResponse {
  interfaces: NetworkInterface[];
  config: {
    preferred_interface?: string;
    preferred_ip?: string;
  };
  current_endpoint: string;
}

export const NetworkSettings: React.FC<NetworkSettingsProps> = ({
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
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
      // 如果有配置，使用配置的值
      if (networkConfig.config?.preferred_interface) {
        setSelectedInterface(networkConfig.config.preferred_interface);
      }
      if (networkConfig.config?.preferred_ip) {
        setSelectedIP(networkConfig.config.preferred_ip);
      } else if (networkConfig.current_endpoint) {
        // 从当前端点提取 IP
        const ip = networkConfig.current_endpoint.split(":")[0];
        setSelectedIP(ip);
        // 找到对应的网卡
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
        // 如果当前选中的 IP 不在这个网卡中，选择第一个
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
        onSuccess?.();
        onClose();
      } catch (err: unknown) {
        const errorMessage =
          err instanceof Error ? err.message : t("network.updateFailed");
        setError(errorMessage);
      } finally {
        setLoading(false);
      }
    },
    [selectedInterface, selectedIP, onSuccess, onClose, t],
  );

  // 过滤可用的网卡（排除 loopback 和未启用的）
  const availableInterfaces =
    networkConfig?.interfaces?.filter((i) => i.is_up && !i.is_loopback) || [];

  // 获取选中网卡的 IP 列表
  const selectedInterfaceIPs = selectedInterface
    ? networkConfig?.interfaces?.find((i) => i.name === selectedInterface)
        ?.addresses || []
    : [];

  return (
    <div className="cocursor-modal-overlay" onClick={onClose}>
      <div className="cocursor-modal" onClick={(e) => e.stopPropagation()}>
        <div className="cocursor-modal-header">
          <h2 className="cocursor-modal-title">{t("network.settings")}</h2>
          <button className="cocursor-modal-close" onClick={onClose}>
            ×
          </button>
        </div>

        <form className="cocursor-modal-body" onSubmit={handleSubmit}>
          {loadingConfig ? (
            <div className="cocursor-team-loading">
              <div className="cocursor-team-loading-spinner"></div>
            </div>
          ) : (
            <>
              {/* 当前端点 */}
              <div className="cocursor-form-group">
                <label className="cocursor-form-label">
                  {t("network.currentEndpoint")}
                </label>
                <div className="cocursor-network-current">
                  <span className="cocursor-network-endpoint">
                    {networkConfig?.current_endpoint || "-"}
                  </span>
                </div>
                <p className="cocursor-form-help">
                  {t("network.currentEndpointHelp")}
                </p>
              </div>

              {/* 网卡选择 */}
              <div className="cocursor-form-group">
                <label className="cocursor-form-label">
                  {t("network.selectInterface")}
                </label>
                <select
                  className="cocursor-form-select"
                  value={selectedInterface}
                  onChange={(e) => setSelectedInterface(e.target.value)}
                >
                  <option value="">
                    {t("network.selectInterfacePlaceholder")}
                  </option>
                  {availableInterfaces.map((iface) => (
                    <option key={iface.name} value={iface.name}>
                      {iface.is_virtual ? "⚠️ " : ""}
                      {iface.name} ({iface.addresses.join(", ")})
                      {iface.is_virtual
                        ? ` - ${t("network.virtualInterface")}`
                        : ""}
                    </option>
                  ))}
                </select>
                {/* 虚拟网卡提示 */}
                {selectedInterface &&
                  networkConfig?.interfaces?.find(
                    (i) => i.name === selectedInterface,
                  )?.is_virtual && (
                    <div className="cocursor-network-virtual-warning">
                      <span className="cocursor-network-warning-icon">⚠️</span>
                      <span>{t("network.virtualInterfaceWarning")}</span>
                    </div>
                  )}
              </div>

              {/* IP 选择 */}
              {selectedInterface && selectedInterfaceIPs.length > 0 && (
                <div className="cocursor-form-group">
                  <label className="cocursor-form-label">
                    {t("network.selectIP")}
                  </label>
                  <select
                    className="cocursor-form-select"
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

              {/* 警告提示 */}
              <div className="cocursor-network-warning">
                <span className="cocursor-network-warning-icon">⚠️</span>
                <span>{t("network.changeWarning")}</span>
              </div>

              {error && <div className="cocursor-form-error">{error}</div>}
            </>
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
              disabled={
                loading || loadingConfig || !selectedInterface || !selectedIP
              }
            >
              {loading ? t("common.loading") : t("common.save")}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
