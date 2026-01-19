/**
 * 快速编辑模式组件
 * 单页卡片式编辑界面
 */

import React from "react";
import { useTranslation } from "react-i18next";
import { useToast } from "../../hooks";
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
  onEditEmbedding,
  onTestConnection,
  onStartQdrant,
  onStopQdrant,
  onRestartQdrant,
  onEditScan,
  onScanNow,
  onTriggerFullIndex,
  onClearAllData,
  onResetConfig,
}) => {
  const { t } = useTranslation();
  const { showToast } = useToast();

  // 获取服务提供商
  const getProvider = () => {
    if (embedding.url.includes('openai.com')) {
      return 'OpenAI';
    } else if (embedding.url.includes('azure.com')) {
      return 'Azure OpenAI';
    } else {
      return '自定义';
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
    switch (qdrant.status) {
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
              onClick={() => {
                if (onEditEmbedding) {
                  onEditEmbedding();
                } else {
                  showToast(t("rag.config.quickEdit.edit") + t("rag.config.quickEdit.featureNotImplemented"), "success");
                }
              }}
            >
              {t("rag.config.quickEdit.edit")}
            </button>
            <button
              type="button"
              className="cocursor-rag-quick-edit-card-action"
              onClick={() => {
                if (onTestConnection) {
                  onTestConnection();
                } else {
                  showToast(t("rag.config.quickEdit.test") + t("rag.config.quickEdit.featureNotImplemented"), "success");
                }
              }}
            >
              {t("rag.config.quickEdit.test")}
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
              {qdrant.status === 'running'
                ? t("rag.config.qdrantRunning")
                : qdrant.status === 'installed'
                ? t("rag.config.qdrantInstalled")
                : qdrant.status === 'stopped'
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
              onClick={() => {
                if (onStartQdrant) {
                  onStartQdrant();
                } else {
                  showToast(t("rag.config.actions.start") + t("rag.config.quickEdit.featureNotImplemented"), "success");
                }
              }}
              >
                {t("rag.config.start")}
              </button>
            )}
            {qdrant.status === 'running' && (
              <button
                type="button"
                className="cocursor-rag-quick-edit-card-action"
              onClick={() => {
                if (onStopQdrant) {
                  onStopQdrant();
                } else {
                  showToast(t("rag.config.actions.stop") + t("rag.config.quickEdit.featureNotImplemented"), "success");
                }
              }}
              >
                {t("rag.config.stop")}
              </button>
            )}
            <button
              type="button"
              className="cocursor-rag-quick-edit-card-action"
              onClick={() => {
                if (onRestartQdrant) {
                  onRestartQdrant();
                } else {
                  showToast(t("rag.config.actions.restart") + t("rag.config.quickEdit.featureNotImplemented"), "success");
                }
              }}
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
              onClick={() => {
                if (onEditScan) {
                  onEditScan();
                } else {
                  showToast(t("rag.config.quickEdit.edit") + t("rag.config.quickEdit.featureNotImplemented"), "success");
                }
              }}
            >
              {t("rag.config.quickEdit.edit")}
            </button>
            <button
              type="button"
              className="cocursor-rag-quick-edit-card-action"
              onClick={() => {
                if (onScanNow) {
                  onScanNow();
                } else {
                  showToast(t("rag.config.quickEdit.scanNow") + t("rag.config.quickEdit.featureNotImplemented"), "success");
                }
              }}
            >
              {t("rag.config.quickEdit.scanNow")}
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
              onClick={() => {
                showToast(t("rag.config.llm.title") + t("rag.config.quickEdit.featureNotImplemented"), "success");
              }}
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
              onClick={() => {
                showToast(t("rag.config.indexStatus") + t("rag.config.actions.indexStatusDetail"), "success");
              }}
            >
              {t("rag.config.summary.status")}
            </button>
          </div>
        </div>
      </div>

      {/* 快速操作栏 */}
      <div className="cocursor-rag-quick-edit-actions">
        <button
          type="button"
          className="cocursor-rag-quick-edit-action-button secondary"
          onClick={() => {
            if (onTriggerFullIndex) {
              onTriggerFullIndex();
            } else {
              showToast(t("rag.config.actions.fullIndex") + t("rag.config.quickEdit.featureNotImplemented"), "success");
            }
          }}
        >
          🔄 {t("rag.config.actions.fullIndex")}
        </button>
        <button
          type="button"
          className="cocursor-rag-quick-edit-action-button secondary"
          onClick={() => {
            if (onClearAllData) {
              if (window.confirm("此操作将删除所有已索引的数据,包括对话总结和向量。此操作不可撤销,确定要继续吗?")) {
                onClearAllData();
              }
            } else {
              showToast(t("rag.config.actions.clearData") + t("rag.config.quickEdit.featureNotImplemented"), "success");
            }
          }}
        >
          🗑️ {t("rag.config.actions.clearData")}
        </button>
        <button
          type="button"
          className="cocursor-rag-quick-edit-action-button secondary"
          onClick={() => {
            if (onResetConfig) {
              onResetConfig();
            } else {
              if (window.confirm(t("rag.config.quickEdit.resetConfig") + "?")) {
                showToast(t("rag.config.quickEdit.resetConfig") + " 功能待实现", "success");
              }
            }
          }}
        >
          {t("rag.config.quickEdit.resetConfig")}
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
