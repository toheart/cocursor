/**
 * RAG 配置组件
 */

import React, { useState, useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../services/api";
import { useApi, useToast } from "../hooks";
import { ToastContainer } from "./shared";

interface RAGConfig {
  embedding_api: {
    url: string;
    model: string;
  };
  qdrant: {
    version: string;
    binary_path: string;
    data_path: string;
  };
  scan_config: {
    enabled: boolean;
    interval: string;
    batch_size: number;
    concurrency: number;
  };
}

interface RAGStats {
  total_indexed: number;
  last_full_scan: number;
  last_incremental_scan: number;
  scan_config: {
    enabled: boolean;
    interval: string;
    batch_size: number;
    concurrency: number;
  };
}

export const RAGConfig: React.FC = () => {
  const { t } = useTranslation();
  const { showToast, toasts } = useToast();

  const [formData, setFormData] = useState({
    url: "",
    apiKey: "",
    model: "",
    enabled: false,
    interval: "1h",
    batchSize: 10,
    concurrency: 3,
    qdrant: {} as { version?: string; binary_path?: string; data_path?: string },
  });

  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [downloading, setDownloading] = useState(false);
  const [stats, setStats] = useState<RAGStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(false);

  // 获取配置
  const fetchConfig = useCallback(async () => {
    try {
      const response = await apiService.getRAGConfig() as RAGConfig;
      if (response) {
        setFormData({
          url: response.embedding_api?.url || "",
          apiKey: "", // API Key 不返回
          model: response.embedding_api?.model || "",
          enabled: response.scan_config?.enabled || false,
          interval: response.scan_config?.interval || "1h",
          batchSize: response.scan_config?.batch_size || 10,
          concurrency: response.scan_config?.concurrency || 3,
          qdrant: response.qdrant || {},
        } as any);
      }
    } catch (error) {
      console.error("Failed to fetch RAG config:", error);
    }
  }, []);

  const { loading, refetch: loadConfig } = useApi(fetchConfig, { initialData: null });

  useEffect(() => {
    loadConfig();
    loadStats();
    // 定期刷新统计信息
    const interval = setInterval(loadStats, 5000);
    return () => clearInterval(interval);
  }, []);

  // 获取统计信息
  const loadStats = useCallback(async () => {
    setStatsLoading(true);
    try {
      const response = await apiService.getRAGStats() as RAGStats;
      if (response) {
        setStats(response);
      }
    } catch (error) {
      console.error("Failed to fetch RAG stats:", error);
    } finally {
      setStatsLoading(false);
    }
  }, []);

  // 下载 Qdrant
  const handleDownloadQdrant = useCallback(async () => {
    setDownloading(true);
    try {
      const response = await apiService.downloadQdrant() as { success: boolean; message?: string; error?: string };
      if (response.success) {
        showToast(response.message || t("rag.config.qdrantDownloadSuccess"), "success");
        await loadConfig();
      } else {
        showToast(response.error || t("rag.config.qdrantDownloadFailed"), "error");
      }
    } catch (error) {
      showToast(t("rag.config.qdrantDownloadFailed") + ": " + (error instanceof Error ? error.message : String(error)), "error");
    } finally {
      setDownloading(false);
    }
  }, [showToast, loadConfig, t]);

  // 测试连接
  const handleTest = useCallback(async () => {
    if (!formData.url || !formData.apiKey || !formData.model) {
      showToast(t("rag.config.testRequired"), "error");
      return;
    }

    setTesting(true);
    try {
      const response = await apiService.testRAGConfig({
        url: formData.url,
        api_key: formData.apiKey,
        model: formData.model,
      }) as { success: boolean; error?: string };

      if (response.success) {
        showToast(t("rag.config.testSuccess"), "success");
      } else {
        showToast(t("rag.config.testFailed") + ": " + (response.error || ""), "error");
      }
    } catch (error) {
      showToast(t("rag.config.testFailed") + ": " + (error instanceof Error ? error.message : String(error)), "error");
    } finally {
      setTesting(false);
    }
  }, [formData, showToast, t]);

  // 保存配置
  const handleSave = useCallback(async () => {
    if (!formData.url || !formData.apiKey || !formData.model) {
      showToast(t("rag.config.saveRequired"), "error");
      return;
    }

    setSaving(true);
    try {
      await apiService.updateRAGConfig({
        embedding_api: {
          url: formData.url,
          api_key: formData.apiKey,
          model: formData.model,
        },
        scan_config: {
          enabled: formData.enabled,
          interval: formData.interval,
          batch_size: formData.batchSize,
          concurrency: formData.concurrency,
        },
      });

      showToast(t("rag.config.saveSuccess"), "success");
      await loadConfig();
    } catch (error) {
      showToast(t("rag.config.saveFailed") + ": " + (error instanceof Error ? error.message : String(error)), "error");
    } finally {
      setSaving(false);
    }
  }, [formData, showToast, loadConfig, t]);

  if (loading) {
    return (
      <div style={{ padding: "20px", textAlign: "center" }}>
        {t("common.loading")}
      </div>
    );
  }

  return (
    <div style={{ padding: "20px", maxWidth: "800px", margin: "0 auto" }}>
      <h2 style={{ marginBottom: "24px" }}>{t("rag.config.title")}</h2>

      <div style={{ marginBottom: "24px" }}>
        <h3 style={{ marginBottom: "16px" }}>{t("rag.config.embeddingApi")}</h3>
        
        <div style={{ marginBottom: "16px" }}>
          <label style={{ display: "block", marginBottom: "8px" }}>
            {t("rag.config.apiUrl")} *
          </label>
          <input
            type="text"
            value={formData.url}
            onChange={(e) => setFormData({ ...formData, url: e.target.value })}
            placeholder="https://api.openai.com"
            style={{
              width: "100%",
              padding: "8px",
              fontSize: "14px",
              border: "1px solid var(--vscode-input-border)",
              backgroundColor: "var(--vscode-input-background)",
              color: "var(--vscode-input-foreground)",
            }}
          />
        </div>

        <div style={{ marginBottom: "16px" }}>
          <label style={{ display: "block", marginBottom: "8px" }}>
            {t("rag.config.apiKey")} *
          </label>
          <input
            type="password"
            value={formData.apiKey}
            onChange={(e) => setFormData({ ...formData, apiKey: e.target.value })}
            placeholder={t("rag.config.apiKeyPlaceholder")}
            style={{
              width: "100%",
              padding: "8px",
              fontSize: "14px",
              border: "1px solid var(--vscode-input-border)",
              backgroundColor: "var(--vscode-input-background)",
              color: "var(--vscode-input-foreground)",
            }}
          />
        </div>

        <div style={{ marginBottom: "16px" }}>
          <label style={{ display: "block", marginBottom: "8px" }}>
            {t("rag.config.model")} *
          </label>
          <input
            type="text"
            value={formData.model}
            onChange={(e) => setFormData({ ...formData, model: e.target.value })}
            placeholder="text-embedding-ada-002"
            style={{
              width: "100%",
              padding: "8px",
              fontSize: "14px",
              border: "1px solid var(--vscode-input-border)",
              backgroundColor: "var(--vscode-input-background)",
              color: "var(--vscode-input-foreground)",
            }}
          />
        </div>

        <button
          onClick={handleTest}
          disabled={testing || !formData.url || !formData.apiKey || !formData.model}
          style={{
            padding: "8px 16px",
            backgroundColor: "var(--vscode-button-secondaryBackground)",
            color: "var(--vscode-button-secondaryForeground)",
            border: "1px solid var(--vscode-button-border)",
            cursor: testing ? "not-allowed" : "pointer",
            marginRight: "8px",
          }}
        >
          {testing ? t("rag.config.testing") : t("rag.config.testConnection")}
        </button>
      </div>

      {/* Qdrant 状态 */}
      <div style={{ marginBottom: "24px", padding: "16px", backgroundColor: "var(--vscode-editor-background)", borderRadius: "4px" }}>
        <h3 style={{ marginBottom: "16px" }}>{t("rag.config.qdrantStatus")}</h3>
        {formData.qdrant?.version ? (
          <div style={{ marginBottom: "12px" }}>
            <p style={{ margin: "4px 0" }}>
              <strong>{t("rag.config.qdrantVersion")}:</strong> {formData.qdrant.version}
            </p>
            <p style={{ margin: "4px 0" }}>
              <strong>{t("rag.config.qdrantPath")}:</strong> {formData.qdrant.binary_path || "Not installed"}
            </p>
          </div>
        ) : (
          <div style={{ marginBottom: "12px" }}>
            <p style={{ margin: "4px 0", color: "var(--vscode-errorForeground)" }}>
              {t("rag.config.qdrantNotInstalled")}
            </p>
          </div>
        )}
        <button
          onClick={handleDownloadQdrant}
          disabled={downloading}
          style={{
            padding: "8px 16px",
            backgroundColor: "var(--vscode-button-secondaryBackground)",
            color: "var(--vscode-button-secondaryForeground)",
            border: "1px solid var(--vscode-button-border)",
            cursor: downloading ? "not-allowed" : "pointer",
          }}
        >
          {downloading ? t("rag.config.downloading") : t("rag.config.downloadQdrant")}
        </button>
      </div>

      {/* 索引状态 */}
      {stats && (
        <div style={{ marginBottom: "24px", padding: "16px", backgroundColor: "var(--vscode-editor-background)", borderRadius: "4px" }}>
          <h3 style={{ marginBottom: "16px" }}>{t("rag.config.indexStatus")}</h3>
          <div style={{ marginBottom: "8px" }}>
            <p style={{ margin: "4px 0" }}>
              <strong>{t("rag.config.totalIndexed")}:</strong> {stats.total_indexed.toLocaleString()}
            </p>
            {stats.last_full_scan > 0 && (
              <p style={{ margin: "4px 0" }}>
                <strong>{t("rag.config.lastFullScan")}:</strong> {new Date(stats.last_full_scan * 1000).toLocaleString()}
              </p>
            )}
            {stats.last_incremental_scan > 0 && (
              <p style={{ margin: "4px 0" }}>
                <strong>{t("rag.config.lastIncrementalScan")}:</strong> {new Date(stats.last_incremental_scan * 1000).toLocaleString()}
              </p>
            )}
          </div>
        </div>
      )}

      <div style={{ marginBottom: "24px" }}>
        <h3 style={{ marginBottom: "16px" }}>{t("rag.config.scanConfig")}</h3>
        
        <div style={{ marginBottom: "16px" }}>
          <label style={{ display: "flex", alignItems: "center", gap: "8px" }}>
            <input
              type="checkbox"
              checked={formData.enabled}
              onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
            />
            {t("rag.config.enableAutoScan")}
          </label>
        </div>

        <div style={{ marginBottom: "16px" }}>
          <label style={{ display: "block", marginBottom: "8px" }}>
            {t("rag.config.scanInterval")}
          </label>
          <select
            value={formData.interval}
            onChange={(e) => setFormData({ ...formData, interval: e.target.value })}
            style={{
              width: "100%",
              padding: "8px",
              fontSize: "14px",
              border: "1px solid var(--vscode-input-border)",
              backgroundColor: "var(--vscode-input-background)",
              color: "var(--vscode-input-foreground)",
            }}
          >
            <option value="30m">{t("rag.config.interval30m")}</option>
            <option value="1h">{t("rag.config.interval1h")}</option>
            <option value="2h">{t("rag.config.interval2h")}</option>
            <option value="6h">{t("rag.config.interval6h")}</option>
            <option value="24h">{t("rag.config.interval24h")}</option>
            <option value="manual">{t("rag.config.intervalManual")}</option>
          </select>
        </div>

        <div style={{ marginBottom: "16px" }}>
          <label style={{ display: "block", marginBottom: "8px" }}>
            {t("rag.config.batchSize")}
          </label>
          <input
            type="number"
            value={formData.batchSize}
            onChange={(e) => setFormData({ ...formData, batchSize: parseInt(e.target.value) || 10 })}
            min="1"
            max="100"
            style={{
              width: "100%",
              padding: "8px",
              fontSize: "14px",
              border: "1px solid var(--vscode-input-border)",
              backgroundColor: "var(--vscode-input-background)",
              color: "var(--vscode-input-foreground)",
            }}
          />
        </div>

        <div style={{ marginBottom: "16px" }}>
          <label style={{ display: "block", marginBottom: "8px" }}>
            {t("rag.config.concurrency")}
          </label>
          <input
            type="number"
            value={formData.concurrency}
            onChange={(e) => setFormData({ ...formData, concurrency: parseInt(e.target.value) || 3 })}
            min="1"
            max="10"
            style={{
              width: "100%",
              padding: "8px",
              fontSize: "14px",
              border: "1px solid var(--vscode-input-border)",
              backgroundColor: "var(--vscode-input-background)",
              color: "var(--vscode-input-foreground)",
            }}
          />
        </div>
      </div>

      <div style={{ display: "flex", gap: "8px" }}>
        <button
          onClick={handleSave}
          disabled={saving || !formData.url || !formData.apiKey || !formData.model}
          style={{
            padding: "10px 20px",
            backgroundColor: "var(--vscode-button-background)",
            color: "var(--vscode-button-foreground)",
            border: "none",
            cursor: saving ? "not-allowed" : "pointer",
          }}
        >
          {saving ? t("common.loading") : t("common.save")}
        </button>
      </div>

      <ToastContainer toasts={toasts} />
    </div>
  );
};
