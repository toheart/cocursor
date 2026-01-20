/**
 * 步骤 2: Qdrant 状态检查
 * 优化版本：添加下载进度显示、版本信息、更好的状态反馈
 */

import React, { useState, useEffect, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { useToast } from "../../hooks";
import { QdrantStatus } from "./types";

// 当前推荐的 Qdrant 版本
const RECOMMENDED_QDRANT_VERSION = "v1.13.0";

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
  const [downloadProgress, setDownloadProgress] = useState<string | null>(null);
  const [downloadError, setDownloadError] = useState<string | null>(null);

  // 检查步骤是否完成
  const isComplete = qdrant.status === 'installed' || qdrant.status === 'running';
  
  useEffect(() => {
    onStepComplete(isComplete);
  }, [isComplete, onStepComplete]);

  // 刷新 Qdrant 状态
  const refreshStatus = useCallback(async () => {
    try {
      const response = await apiService.getQdrantStatus();
      if (response && typeof response === 'object' && 'data' in response) {
        const data = response.data as any;
        const newStatus: QdrantStatus = data.is_running 
          ? 'running' 
          : (data.version ? 'installed' : 'not-installed');
        
        if (newStatus !== qdrant.status || data.version !== qdrant.version) {
          onChange({
            ...qdrant,
            version: data.version || qdrant.version,
            binaryPath: data.binary_path || qdrant.binaryPath,
            status: newStatus,
          });
        }
      }
    } catch (error) {
      console.error('Failed to refresh Qdrant status:', error);
    }
  }, [qdrant, onChange]);

  // 初始化时检查状态
  useEffect(() => {
    refreshStatus();
  }, []);

  // 下载 Qdrant
  const handleDownload = async () => {
    setDownloading(true);
    setDownloadProgress("正在准备下载...");
    setDownloadError(null);
    
    try {
      // 显示下载进度
      setDownloadProgress("正在下载 Qdrant (约 50MB)，请稍候...");
      
      const response = await apiService.downloadQdrant() as { success: boolean; message?: string; error?: string; version?: string; binary_path?: string };
      
      if (response.success) {
        setDownloadProgress("下载完成！");
        showToast(response.message || t("rag.config.qdrantDownloadSuccess"), "success");
        
        // 更新状态为已安装
        onChange({
          ...qdrant,
          version: response.version || RECOMMENDED_QDRANT_VERSION,
          binaryPath: response.binary_path || qdrant.binaryPath,
          status: 'installed',
        });
        
        // 通知父组件刷新配置
        if (onDownloadSuccess) {
          onDownloadSuccess();
        }
      } else {
        const errorMsg = response.error || t("rag.config.qdrantDownloadFailed");
        setDownloadError(errorMsg);
        showToast(errorMsg, "error");
      }
    } catch (error) {
      const errorMsg = t("rag.config.qdrantDownloadFailed") + ": " + (error instanceof Error ? error.message : String(error));
      setDownloadError(errorMsg);
      showToast(errorMsg, "error");
    } finally {
      setDownloading(false);
      // 3秒后清除进度消息
      setTimeout(() => setDownloadProgress(null), 3000);
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
                    showToast(t("rag.config.actions.startNotImplemented"), "success");
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
                    showToast(t("rag.config.actions.stopNotImplemented"), "success");
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
                  showToast(t("rag.config.actions.restartNotImplemented"), "success");
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

      {/* 下载按钮和进度 */}
      {qdrant.status === 'not-installed' && (
        <div className="cocursor-rag-qdrant-download-section">
          <button
            type="button"
            className="cocursor-rag-qdrant-download-button"
            onClick={handleDownload}
            disabled={downloading}
          >
            {downloading ? t("rag.config.downloading") : t("rag.config.downloadQdrant")}
          </button>
          
          {/* 下载进度提示 */}
          {downloadProgress && (
            <div className="cocursor-rag-qdrant-download-progress">
              {downloading && <span className="cocursor-rag-spinner" />}
              <span>{downloadProgress}</span>
            </div>
          )}
          
          {/* 下载错误提示 */}
          {downloadError && (
            <div className="cocursor-rag-qdrant-download-error">
              <span>❌ {downloadError}</span>
              <button
                type="button"
                className="cocursor-rag-retry-button"
                onClick={handleDownload}
              >
                重试
              </button>
            </div>
          )}
          
          {/* 版本信息 */}
          <div className="cocursor-rag-qdrant-version-info">
            <small>将下载 Qdrant {RECOMMENDED_QDRANT_VERSION}</small>
          </div>
        </div>
      )}
      
      {/* 刷新状态按钮 */}
      {qdrant.status !== 'not-installed' && (
        <button
          type="button"
          className="cocursor-rag-refresh-status-button"
          onClick={refreshStatus}
        >
          🔄 刷新状态
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
