/**
 * 步骤 4: 配置确认
 * 优化版本：添加完整性检查、问题提示和启动流程
 */

import React, { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

interface Step4Props {
  embedding: {
    url: string;
    model: string;
    apiKey?: string;
  };
  llm?: {
    url: string;
    model: string;
    apiKey?: string;
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

// 配置检查项
interface ConfigCheck {
  key: string;
  label: string;
  status: 'ok' | 'warning' | 'error';
  message: string;
}

export const Step4_Summary: React.FC<Step4Props> = ({
  embedding,
  llm,
  qdrant,
  scan,
  onSave,
}) => {
  const { t } = useTranslation();
  const [saving, setSaving] = useState(false);

  // 配置完整性检查
  const configChecks = useMemo<ConfigCheck[]>(() => {
    const checks: ConfigCheck[] = [];
    
    // 检查 Embedding API
    if (embedding.url && embedding.model) {
      checks.push({
        key: 'embedding',
        label: 'Embedding API',
        status: 'ok',
        message: '配置完成',
      });
    } else {
      checks.push({
        key: 'embedding',
        label: 'Embedding API',
        status: 'error',
        message: '请完成 Embedding API 配置',
      });
    }
    
    // 检查 LLM API
    if (llm?.url && llm?.model) {
      checks.push({
        key: 'llm',
        label: 'LLM Chat API',
        status: 'ok',
        message: '配置完成',
      });
    } else {
      checks.push({
        key: 'llm',
        label: 'LLM Chat API',
        status: 'error',
        message: '请完成 LLM Chat API 配置（必需）',
      });
    }
    
    // 检查 Qdrant
    if (qdrant.status === 'running') {
      checks.push({
        key: 'qdrant',
        label: 'Qdrant',
        status: 'ok',
        message: `运行中 (${qdrant.version})`,
      });
    } else if (qdrant.status === 'installed') {
      checks.push({
        key: 'qdrant',
        label: 'Qdrant',
        status: 'warning',
        message: `已安装但未运行 (${qdrant.version})，保存后将自动启动`,
      });
    } else {
      checks.push({
        key: 'qdrant',
        label: 'Qdrant',
        status: 'error',
        message: '未安装，请先下载安装 Qdrant',
      });
    }
    
    // 检查扫描配置
    if (scan.enabled) {
      checks.push({
        key: 'scan',
        label: '自动扫描',
        status: 'ok',
        message: `已启用，间隔 ${scan.interval}`,
      });
    } else {
      checks.push({
        key: 'scan',
        label: '自动扫描',
        status: 'warning',
        message: '已禁用，需手动触发索引',
      });
    }
    
    return checks;
  }, [embedding, llm, qdrant, scan]);

  // 检查是否可以保存
  const canSave = useMemo(() => {
    return configChecks.every(check => check.status !== 'error');
  }, [configChecks]);

  // 获取状态图标
  const getStatusIcon = (status: 'ok' | 'warning' | 'error') => {
    switch (status) {
      case 'ok': return '✅';
      case 'warning': return '⚠️';
      case 'error': return '❌';
    }
  };

  // 处理保存
  const handleSave = async () => {
    if (!canSave) return;
    setSaving(true);
    try {
      await onSave();
    } finally {
      setSaving(false);
    }
  };

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

  return (
    <div className="cocursor-rag-step-4">
      <div className="cocursor-rag-step-header">
        <h3 className="cocursor-rag-step-title">{t("rag.config.step4.title")}</h3>
        <p className="cocursor-rag-step-description">
          {t("rag.config.step4.description")}
        </p>
      </div>

      {/* 配置完整性检查 */}
      <div className="cocursor-rag-config-checks">
        <h4>配置检查</h4>
        <ul className="cocursor-rag-check-list">
          {configChecks.map(check => (
            <li key={check.key} className={`cocursor-rag-check-item ${check.status}`}>
              <span className="cocursor-rag-check-icon">{getStatusIcon(check.status)}</span>
              <span className="cocursor-rag-check-label">{check.label}</span>
              <span className="cocursor-rag-check-message">{check.message}</span>
            </li>
          ))}
        </ul>
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

      {/* 错误提示（如果有） */}
      {!canSave && (
        <div className="cocursor-rag-summary-error">
          <span className="cocursor-rag-error-icon">❌</span>
          <span>请先解决上述配置问题后再保存</span>
        </div>
      )}

      {/* 保存按钮 */}
      <div className="cocursor-rag-summary-actions">
        <button
          type="button"
          className={`cocursor-rag-save-button cocursor-rag-save-button-primary ${!canSave ? 'disabled' : ''}`}
          onClick={handleSave}
          disabled={!canSave || saving}
        >
          {saving ? '保存中...' : `✓ ${t("rag.config.wizard.saveAndEnable")}`}
        </button>
      </div>

      {/* 保存后说明 */}
      {canSave && (
        <div className="cocursor-rag-summary-info">
          <h4>保存后将执行：</h4>
          <ol>
            <li>保存 API 配置到本地</li>
            {qdrant.status === 'installed' && <li>自动启动 Qdrant 服务</li>}
            {scan.enabled && <li>按配置的间隔开始自动扫描</li>}
          </ol>
        </div>
      )}
    </div>
  );
};
