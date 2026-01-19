/**
 * 步骤 4: 配置确认
 */

import React from "react";
import { useTranslation } from "react-i18next";

interface Step4Props {
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
    status: 'installed' | 'not-installed' | 'running' | 'stopped';
  };
  scan: {
    enabled: boolean;
    interval: string;
    batchSize: number;
    concurrency: number;
  };
  onSave: () => void;
}

export const Step4_Summary: React.FC<Step4Props> = ({
  embedding,
  llm,
  qdrant,
  scan,
  onSave,
}) => {
  const { t } = useTranslation();

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

  return (
    <div className="cocursor-rag-step-4">
      <div className="cocursor-rag-step-header">
        <h3 className="cocursor-rag-step-title">{t("rag.config.step4.title")}</h3>
        <p className="cocursor-rag-step-description">
          {t("rag.config.step4.description")}
        </p>
      </div>

      {/* 配置摘要 */}
      <div className="cocursor-rag-summary">
        {/* Embedding 配置 */}
        <div className="cocursor-rag-summary-section">
          <div className="cocursor-rag-summary-header">
            <h4 className="cocursor-rag-summary-title">
              {t("rag.config.summary.embedding")}
            </h4>
            <span className="cocursor-rag-summary-badge">
              {qdrant.status === 'installed' || qdrant.status === 'running'
                ? t("rag.config.summary.enabled")
                : t("rag.config.summary.disabled")}
            </span>
          </div>
          <div className="cocursor-rag-summary-content">
            <div className="cocursor-rag-summary-item">
              <strong>{t("rag.config.summary.provider")}:</strong>
              <span>{getProvider()}</span>
            </div>
            <div className="cocursor-rag-summary-item">
              <strong>{t("rag.config.apiUrl")}:</strong>
              <span>{embedding.url || t("common.none")}</span>
            </div>
            <div className="cocursor-rag-summary-item">
              <strong>{t("rag.config.model")}:</strong>
              <span>{embedding.model || t("common.none")}</span>
            </div>
          </div>
        </div>

        {/* LLM 配置 */}
        {llm && llm.url && (
          <div className="cocursor-rag-summary-section">
            <div className="cocursor-rag-summary-header">
              <h4 className="cocursor-rag-summary-title">
                {t("rag.config.summary.llm")}
              </h4>
              <span className="cocursor-rag-summary-badge">
                {llm.url ? t("rag.config.summary.enabled") : t("rag.config.summary.disabled")}
              </span>
            </div>
            <div className="cocursor-rag-summary-content">
              <div className="cocursor-rag-summary-item">
                <strong>{t("rag.config.apiUrl")}:</strong>
                <span>{llm.url || t("common.none")}</span>
              </div>
              <div className="cocursor-rag-summary-item">
                <strong>{t("rag.config.model")}:</strong>
                <span>{llm.model || t("common.none")}</span>
              </div>
            </div>
          </div>
        )}

        {/* Qdrant 配置 */}
        <div className="cocursor-rag-summary-section">
          <div className="cocursor-rag-summary-header">
            <h4 className="cocursor-rag-summary-title">
              {t("rag.config.summary.qdrant")}
            </h4>
            <span className={`cocursor-rag-summary-badge status-${qdrant.status}`}>
              {qdrant.status === 'not-installed'
                ? t("rag.config.qdrantNotInstalled")
                : qdrant.status === 'running'
                ? t("rag.config.qdrantRunning")
                : qdrant.status === 'installed'
                ? t("rag.config.qdrantInstalled")
                : t("rag.config.qdrantStopped")}
            </span>
          </div>
          <div className="cocursor-rag-summary-content">
            <div className="cocursor-rag-summary-item">
              <strong>{t("rag.config.qdrantVersion")}:</strong>
              <span>{qdrant.version || t("common.unknown")}</span>
            </div>
            <div className="cocursor-rag-summary-item">
              <strong>{t("rag.config.qdrantPath")}:</strong>
              <span>{qdrant.binaryPath || t("common.unknown")}</span>
            </div>
          </div>
        </div>

        {/* 扫描配置 */}
        <div className="cocursor-rag-summary-section">
          <div className="cocursor-rag-summary-header">
            <h4 className="cocursor-rag-summary-title">
              {t("rag.config.summary.scan")}
            </h4>
            <span className="cocursor-rag-summary-badge">
              {scan.enabled ? t("rag.config.summary.enabled") : t("rag.config.summary.disabled")}
            </span>
          </div>
          <div className="cocursor-rag-summary-content">
            <div className="cocursor-rag-summary-item">
              <strong>{t("rag.config.scanInterval")}:</strong>
              <span>{getIntervalText(scan.interval)}</span>
            </div>
            <div className="cocursor-rag-summary-item">
              <strong>{t("rag.config.batchSize")}:</strong>
              <span>{scan.batchSize}</span>
            </div>
            <div className="cocursor-rag-summary-item">
              <strong>{t("rag.config.concurrency")}:</strong>
              <span>{scan.concurrency}</span>
            </div>
          </div>
        </div>
      </div>

      {/* 警告提示 */}
      <div className="cocursor-rag-summary-warning">
        <span className="cocursor-rag-warning-icon">⚠️</span>
        <span>{t("rag.config.saveWarning")}</span>
      </div>

      {/* 保存按钮 */}
      <div className="cocursor-rag-summary-actions">
        <button
          type="button"
          className="cocursor-rag-save-button cocursor-rag-save-button-primary"
          onClick={onSave}
        >
          ✓ {t("rag.config.wizard.saveAndEnable")}
        </button>
      </div>
    </div>
  );
};
