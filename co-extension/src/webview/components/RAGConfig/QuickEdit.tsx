/**
 * 快速编辑模式组件
 * 单页卡片式编辑界面
 */

import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useToast } from "../../hooks";
import { apiService } from "../../services/api";
import { QdrantStatus } from "./types";

interface RAGStats {
  total_indexed: number;
  last_full_scan: number;
  last_incremental_scan: number;
}

interface QuickEditProps {
  embedding: {
    url: string;
    model: string;
  };
  llm?: {
    url: string;
    model: string;
  };
  qdrant: {
    version: string;
    binaryPath: string;
    status: QdrantStatus;
  };
  scan: {
    enabled: boolean;
    interval: string;
    batchSize: number;
    concurrency: number;
  };
  onSwitchToWizard: () => void;
  onSave: () => void;
  stats?: RAGStats | null;
  onEditEmbedding?: () => void;
  onTestConnection?: () => void;
  onStartQdrant?: () => void;
  onStopQdrant?: () => void;
  onRestartQdrant?: () => void;
  onEditScan?: () => void;
  onScanNow?: () => void;
  onTriggerFullIndex?: () => void;
  onClearAllData?: () => void;
  onResetConfig?: () => void;
}

export const QuickEdit: React.FC<QuickEditProps> = ({
  embedding,
  llm,
  qdrant,
  scan,
  onSwitchToWizard,
  onSave,
  stats,
}) => {
  const { t } = useTranslation();
  const { showToast } = useToast();

  // Qdrant 状态管理
  const [qdrantStatus, setQdrantStatus] = useState<QdrantStatus>(qdrant.status);

  // 定期刷新 Qdrant 状态
  useEffect(() => {
    const refreshInterval = setInterval(() => {
      apiService.getQdrantStatus().then((response) => {
        if (response && typeof response === 'object' && 'data' in response) {
          const data = response.data as any;
          if (data.is_running !== undefined) {
            setQdrantStatus(data.is_running ? 'running' : (qdrant.version ? 'installed' : 'not-installed'));
          }
        }
      }).catch(err => {
        console.error('Failed to fetch Qdrant status:', err);
      });
    }, 5000); // 每5秒刷新一次

    return () => clearInterval(refreshInterval);
  }, [qdrant.version]);

  // 获取服务提供商
  const getProvider = () => {
    if (embedding.url.includes('openai.com')) {
      return t("rag.config.template.openai");
    } else if (embedding.url.includes('azure.com')) {
      return t("rag.config.template.azure");
    } else {
      return t("rag.config.template.custom");
    }
  };

  // 获取间隔文本
  const getIntervalText = (interval: string) => {
    const map: Record<string, string> = {
      '30m': t('rag.config.interval30m'),
      '1h': t('rag.config.interval1h'),
      '2h': t('rag.config.interval2h'),
      '6h': t('rag.config.interval6h'),
      '24h': t('rag.config.interval24h'),
      'manual': t('rag.config.intervalManual'),
    };
    return map[interval] || interval;
  };

  // 获取 Qdrant 状态类
  const getQdrantStatusClass = () => {
    switch (qdrantStatus) {
      case 'running':
        return 'running';
      case 'installed':
        return 'success';
      case 'stopped':
        return 'error';
      case 'not-installed':
        return 'error';
      default:
        return 'unknown';
    }
  };

  // Qdrant 操作
  const handleStartQdrant = async () => {
    try {
      await apiService.startQdrant();
      showToast(t("rag.actions.start") + t("rag.success"), "success");
      // 立即刷新状态
      setTimeout(() => {
        apiService.getQdrantStatus();
      }, 1000);
    } catch (error) {
      showToast(t("rag.actions.start") + t("rag.failed") + ": " + (error instanceof Error ? error.message : String(error)), "error");
    }
  };

  const handleStopQdrant = async () => {
    try {
      await apiService.stopQdrant();
      showToast(t("rag.actions.stop") + t("rag.success"), "success");
      // 立即刷新状态
      setTimeout(() => {
        apiService.getQdrantStatus();
      }, 1000);
    } catch (error) {
      showToast(t("rag.actions.stop") + t("rag.failed") + ": " + (error instanceof Error ? error.message : String(error)), "error");
    }
  };

  const handleTriggerFullIndex = async () => {
    try {
      await apiService.triggerFullIndex();
      showToast(t("rag.triggerFullIndexSuccess"), "success");
    } catch (error) {
      showToast(t("rag.triggerFullIndexFailed") + ": " + (error instanceof Error ? error.message : String(error)), "error");
    }
  };

  const handleClearAllData = async () => {
    if (window.confirm(t("rag.clearDataWarning"))) {
      try {
        await apiService.clearAllData();
        showToast(t("rag.dataCleared"), "success");
      } catch (error) {
        showToast(t("rag.clearDataFailed") + ": " + (error instanceof Error ? error.message : String(error)), "error");
      }
    }
  };

  return (
    <div className="cocursor-rag-quick-edit">
      {/* 头部和模式切换 */}
      <div className="cocursor-rag-quick-edit-header">
        <h2 className="cocursor-rag-quick-edit-title">{t("rag.config.quickEdit.title")}</h2>
        <button
          type="button"
          className="cocursor-rag-quick-edit-switch"
          onClick={onSwitchToWizard}
        >
          📋 {t("rag.config.quickEdit.switchToWizard")}
        </button>
      </div>

      {/* 配置卡片网格 */}
      <div className="cocursor-rag-quick-edit-grid">
        {/* Embedding API 配置卡片 */}
        <div className="cocursor-rag-quick-edit-card">
          <div className="cocursor-rag-quick-edit-card-header">
            <h3 className="cocursor-rag-quick-edit-card-title">
              🔌 {t("rag.config.summary.embedding")}
            </h3>
            <span className="cocursor-rag-quick-edit-card-status success">
              {embedding.url && embedding.model ? t("rag.config.summary.enabled") : t("rag.config.summary.disabled")}
            </span>
          </div>
          <div className="cocursor-rag-quick-edit-card-content">
            <div className="cocursor-rag-quick-edit-card-item">
              <strong>{t("rag.config.summary.provider")}:</strong>
              <span>{getProvider()}</span>
            </div>
            <div className="cocursor-rag-quick-edit-card-item">
              <strong>{t("rag.config.apiUrl")}:</strong>
              <span>{embedding.url || t("common.none")}</span>
            </div>
            <div className="cocursor-rag-quick-edit-card-item">
              <strong>{t("rag.config.model")}:</strong>
              <span>{embedding.model || t("common.none")}</span>
            </div>
          </div>
          <div className="cocursor-rag-quick-edit-card-actions">
            <button
              type="button"
              className="cocursor-rag-quick-edit-card-action"
              onClick={onSwitchToWizard}
            >
              {t("rag.config.quickEdit.edit")}
            </button>
          </div>
        </div>

        {/* Qdrant 配置卡片 */}
        <div className="cocursor-rag-quick-edit-card">
          <div className="cocursor-rag-quick-edit-card-header">
            <h3 className="cocursor-rag-quick-edit-card-title">
              🗄️ {t("rag.config.summary.qdrant")}
            </h3>
            <span className={`cocursor-rag-quick-edit-card-status ${getQdrantStatusClass()}`}>
              {qdrantStatus === 'running'
                ? t("rag.config.qdrantRunning")
                : qdrantStatus === 'installed'
                ? t("rag.config.qdrantInstalled")
                : qdrantStatus === 'stopped'
                ? t("rag.config.qdrantStopped")
                : t("rag.config.qdrantNotInstalled")}
            </span>
          </div>
          <div className="cocursor-rag-quick-edit-card-content">
            <div className="cocursor-rag-quick-edit-card-item">
              <strong>{t("rag.config.qdrantVersion")}:</strong>
              <span>{qdrant.version || t("common.unknown")}</span>
            </div>
            <div className="cocursor-rag-quick-edit-card-item">
              <strong>{t("rag.config.qdrantPath")}:</strong>
              <span>{qdrant.binaryPath || t("common.unknown")}</span>
            </div>
          </div>
          <div className="cocursor-rag-quick-edit-card-actions">
            {qdrant.status === 'stopped' && (
              <button
                type="button"
                className="cocursor-rag-quick-edit-card-action"
                onClick={handleStartQdrant}
              >
                {t("rag.config.start")}
              </button>
            )}
            {qdrant.status === 'running' && (
              <button
                type="button"
                className="cocursor-rag-quick-edit-card-action"
                onClick={handleStopQdrant}
              >
                {t("rag.config.stop")}
              </button>
            )}
            <button
              type="button"
              className="cocursor-rag-quick-edit-card-action"
              onClick={onSwitchToWizard}
            >
              {t("rag.config.restart")}
            </button>
          </div>
        </div>

        {/* 扫描配置卡片 */}
        <div className="cocursor-rag-quick-edit-card">
          <div className="cocursor-rag-quick-edit-card-header">
            <h3 className="cocursor-rag-quick-edit-card-title">
              🔍 {t("rag.config.summary.scan")}
            </h3>
            <span className="cocursor-rag-quick-edit-card-status success">
              {scan.enabled ? t("rag.config.summary.enabled") : t("rag.config.summary.disabled")}
            </span>
          </div>
          <div className="cocursor-rag-quick-edit-card-content">
            <div className="cocursor-rag-quick-edit-card-item">
              <strong>{t("rag.config.scanInterval")}:</strong>
              <span>{getIntervalText(scan.interval)}</span>
            </div>
            <div className="cocursor-rag-quick-edit-card-item">
              <strong>{t("rag.config.batchSize")}:</strong>
              <span>{scan.batchSize}</span>
            </div>
            <div className="cocursor-rag-quick-edit-card-item">
              <strong>{t("rag.config.concurrency")}:</strong>
              <span>{scan.concurrency}</span>
            </div>
          </div>
          <div className="cocursor-rag-quick-edit-card-actions">
            <button
              type="button"
              className="cocursor-rag-quick-edit-card-action"
              onClick={onSwitchToWizard}
            >
              {t("rag.config.quickEdit.edit")}
            </button>
          </div>
        </div>

        {/* LLM 配置卡片 */}
        {llm && llm.url && (
          <div className="cocursor-rag-quick-edit-card">
            <div className="cocursor-rag-quick-edit-card-header">
              <h3 className="cocursor-rag-quick-edit-card-title">
                🤖 {t("rag.config.summary.llm")}
              </h3>
              <span className="cocursor-rag-quick-edit-card-status success">
                {t("rag.config.summary.enabled")}
              </span>
            </div>
            <div className="cocursor-rag-quick-edit-card-content">
              <div className="cocursor-rag-quick-edit-card-item">
                <strong>{t("rag.config.apiUrl")}:</strong>
                <span>{llm.url || t("common.none")}</span>
              </div>
              <div className="cocursor-rag-quick-edit-card-item">
                <strong>{t("rag.config.model")}:</strong>
                <span>{llm.model || t("common.none")}</span>
              </div>
            </div>
            <div className="cocursor-rag-quick-edit-card-actions">
              <button
                type="button"
                className="cocursor-rag-quick-edit-card-action"
                onClick={onSwitchToWizard}
              >
                {t("rag.config.quickEdit.edit")}
              </button>
            </div>
          </div>
        )}

        {/* 索引状态卡片 */}
        <div className="cocursor-rag-quick-edit-card">
          <div className="cocursor-rag-quick-edit-card-header">
            <h3 className="cocursor-rag-quick-edit-card-title">
              📊 {t("rag.config.indexStatus")}
            </h3>
          </div>
          <div className="cocursor-rag-quick-edit-card-content">
            {stats ? (
              <>
                <div className="cocursor-rag-quick-edit-card-item">
                  <strong>{t("rag.config.totalIndexed")}:</strong>
                  <span>{stats.total_indexed.toLocaleString()}</span>
                </div>
                {stats.last_full_scan > 0 && (
                  <div className="cocursor-rag-quick-edit-card-item">
                    <strong>{t("rag.config.lastFullScan")}:</strong>
                    <span>{new Date(stats.last_full_scan * 1000).toLocaleString()}</span>
                  </div>
                )}
                {stats.last_incremental_scan > 0 && (
                  <div className="cocursor-rag-quick-edit-card-item">
                    <strong>{t("rag.config.lastIncrementalScan")}:</strong>
                    <span>{new Date(stats.last_incremental_scan * 1000).toLocaleString()}</span>
                  </div>
                )}
              </>
            ) : (
              <div className="cocursor-rag-quick-edit-card-item">
                <span>{t("common.loading")}</span>
              </div>
            )}
          </div>
          <div className="cocursor-rag-quick-edit-card-actions">
            <button
              type="button"
              className="cocursor-rag-quick-edit-card-action"
              onClick={handleTriggerFullIndex}
            >
              {t("rag.config.actions.fullIndex")}
            </button>
          </div>
        </div>
      </div>

      {/* 快速操作栏 */}
      <div className="cocursor-rag-quick-edit-actions">
        <button
          type="button"
          className="cocursor-rag-quick-edit-action-button secondary"
          onClick={handleTriggerFullIndex}
        >
          🔄 {t("rag.config.actions.fullIndex")}
        </button>
        <button
          type="button"
          className="cocursor-rag-quick-edit-action-button secondary"
          onClick={handleClearAllData}
        >
          🗑️ {t("rag.config.actions.clearData")}
        </button>
        <button
          type="button"
          className="cocursor-rag-quick-edit-action-button primary"
          onClick={onSave}
        >
          ✓ {t("rag.config.quickEdit.saveChanges")}
        </button>
      </div>
    </div>
  );
};
