/**
 * 步骤 2: Qdrant 状态检查
 */

import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { useToast } from "../../hooks";
import { QdrantStatus } from "./types";

interface Step2Props {
  qdrant: {
    version: string;
    binaryPath: string;
    dataPath: string;
    status: QdrantStatus;
  };
  onChange: (data: { version: string; binaryPath: string; dataPath: string; status: QdrantStatus }) => void;
  onStepComplete: (completed: boolean) => void;
  onDownloadSuccess?: () => void;
}

export const Step2_Qdrant: React.FC<Step2Props> = ({
  qdrant,
  onChange,
  onStepComplete,
  onDownloadSuccess,
}) => {
  const { t } = useTranslation();
  const { showToast } = useToast();

  const [downloading, setDownloading] = useState(false);

  // 检查步骤是否完成
  const isComplete = qdrant.status === 'installed' || qdrant.status === 'running';
  
  useEffect(() => {
    onStepComplete(isComplete);
  }, [isComplete, onStepComplete]);

  // 下载 Qdrant
  const handleDownload = async () => {
    setDownloading(true);
    try {
      const response = await apiService.downloadQdrant() as { success: boolean; message?: string; error?: string };
      if (response.success) {
        showToast(response.message || t("rag.config.qdrantDownloadSuccess"), "success");
        // 更新状态为已安装
        onChange({
          ...qdrant,
          status: 'installed',
        });
        // 通知父组件刷新配置
        if (onDownloadSuccess) {
          onDownloadSuccess();
        }
      } else {
        showToast(response.error || t("rag.config.qdrantDownloadFailed"), "error");
      }
    } catch (error) {
      showToast(t("rag.config.qdrantDownloadFailed") + ": " + (error instanceof Error ? error.message : String(error)), "error");
    } finally {
      setDownloading(false);
    }
  };

  // 获取状态图标
  const getStatusIcon = () => {
    switch (qdrant.status) {
      case 'running':
        return '🟢';
      case 'installed':
        return '⚪';
      case 'stopped':
        return '🔴';
      case 'not-installed':
        return '⚠️';
      default:
        return '❓';
    }
  };

  // 获取状态文本
  const getStatusText = () => {
    switch (qdrant.status) {
      case 'running':
        return t("rag.config.qdrantRunning");
      case 'installed':
        return t("rag.config.qdrantInstalled");
      case 'stopped':
        return t("rag.config.qdrantStopped");
      case 'not-installed':
        return t("rag.config.qdrantNotInstalled");
      default:
        return t("rag.config.qdrantUnknown");
    }
  };

  return (
    <div className="cocursor-rag-step-2">
      <div className="cocursor-rag-step-header">
        <h3 className="cocursor-rag-step-title">{t("rag.config.step2.title")}</h3>
        <p className="cocursor-rag-step-description">
          {t("rag.config.step2.description")}
        </p>
      </div>

      {/* Qdrant 状态卡片 */}
      <div className="cocursor-rag-qdrant-status-card">
        <div className="cocursor-rag-qdrant-status-header">
          <div className="cocursor-rag-qdrant-status-info">
            <span className="cocursor-rag-qdrant-status-icon">{getStatusIcon()}</span>
            <div>
              <div className="cocursor-rag-qdrant-status-title">{getStatusText()}</div>
              {qdrant.version && (
                <div className="cocursor-rag-qdrant-version">
                  {t("rag.config.qdrantVersion")}: {qdrant.version}
                </div>
              )}
            </div>
          </div>
          {qdrant.status !== 'not-installed' && (
            <div className="cocursor-rag-qdrant-actions">
              {qdrant.status === 'stopped' && (
                <button
                  type="button"
                  className="cocursor-rag-qdrant-action-button"
                  onClick={() => {
                    // TODO: 实现启动 Qdrant
                    showToast("启动功能待实现", "info");
                  }}
                >
                  {t("rag.config.start")}
                </button>
              )}
              {qdrant.status === 'running' && (
                <button
                  type="button"
                  className="cocursor-rag-qdrant-action-button"
                  onClick={() => {
                    // TODO: 实现停止 Qdrant
                    showToast("停止功能待实现", "info");
                  }}
                >
                  {t("rag.config.stop")}
                </button>
              )}
              <button
                type="button"
                className="cocursor-rag-qdrant-action-button"
                onClick={() => {
                  // TODO: 实现重启 Qdrant
                  showToast("重启功能待实现", "info");
                }}
              >
                {t("rag.config.restart")}
              </button>
            </div>
          )}
        </div>

        {qdrant.binaryPath && (
          <div className="cocursor-rag-qdrant-detail">
            <strong>{t("rag.config.qdrantPath")}:</strong> {qdrant.binaryPath}
          </div>
        )}
      </div>

      {/* 下载按钮 */}
      {qdrant.status === 'not-installed' && (
        <button
          type="button"
          className="cocursor-rag-qdrant-download-button"
          onClick={handleDownload}
          disabled={downloading}
        >
          {downloading ? t("rag.config.downloading") : t("rag.config.downloadQdrant")}
        </button>
      )}

      {/* 帮助信息 */}
      <div className="cocursor-rag-qdrant-help">
        <div className="cocursor-rag-qdrant-help-item">
          <strong>ℹ️ {t("rag.config.qdrantHelp.title")}:</strong>
        </div>
        <ul className="cocursor-rag-qdrant-help-list">
          <li>{t("rag.config.qdrantHelp.description")}</li>
          <li>{t("rag.config.qdrantHelp.performance")}</li>
          <li>{t("rag.config.qdrantHelp.docs")}</li>
        </ul>
      </div>
    </div>
  );
};
