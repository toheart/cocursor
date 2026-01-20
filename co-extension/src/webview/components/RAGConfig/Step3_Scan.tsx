/**
 * 步骤 3: 扫描策略配置
 * 优化版本：高级选项现在可以正确保存到配置中
 */

import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";

interface ScanConfig {
  enabled: boolean;
  interval: string;
  batchSize: number;
  concurrency: number;
  // 高级选项
  incrementalScan?: boolean;
  maxFileSize?: number;
  ignorePatterns?: string;
}

interface Step3Props {
  scan: ScanConfig;
  onChange: (data: ScanConfig) => void;
  onStepComplete: (completed: boolean) => void;
}

export const Step3_Scan: React.FC<Step3Props> = ({
  scan,
  onChange,
  onStepComplete,
}) => {
  const { t } = useTranslation();
  const [showAdvanced, setShowAdvanced] = useState(false);

  // 推荐配置（包含高级选项的默认值）
  const recommendedConfig: ScanConfig = {
    enabled: true,
    interval: "1h",
    batchSize: 10,
    concurrency: 3,
    incrementalScan: true,
    maxFileSize: 10,
    ignorePatterns: "node_modules/**, .git/**, .cursor/**, dist/**, build/**",
  };

  // 检查步骤是否完成
  const isComplete = !!scan.interval && scan.batchSize > 0 && scan.concurrency > 0;
  useEffect(() => {
    onStepComplete(isComplete);
  }, [isComplete, onStepComplete]);

  // 应用推荐配置
  const handleUseRecommended = () => {
    onChange(recommendedConfig);
  };

  return (
    <div className="cocursor-rag-step-3">
      <div className="cocursor-rag-step-header">
        <h3 className="cocursor-rag-step-title">{t("rag.config.step3.title")}</h3>
        <p className="cocursor-rag-step-description">
          {t("rag.config.step3.description")}
        </p>
      </div>

      {/* 推荐配置按钮 */}
      <button
        type="button"
        className="cocursor-rag-recommended-button"
        onClick={handleUseRecommended}
      >
        📋 {t("rag.config.useRecommended")}
      </button>

      {/* 基础配置表单 */}
      <div className="cocursor-rag-scan-form">
        {/* 启用自动扫描 */}
        <div className="cocursor-rag-form-field">
          <label className="cocursor-rag-checkbox-label">
            <input
              type="checkbox"
              checked={scan.enabled}
              onChange={(e) => onChange({ ...scan, enabled: e.target.checked })}
            />
            <span>{t("rag.config.enableAutoScan")}</span>
          </label>
          <div className="cocursor-rag-form-helper">
            {t("rag.config.enableAutoScanHelper")}
          </div>
        </div>

        {/* 扫描间隔 */}
        <div className="cocursor-rag-form-field">
          <label className="cocursor-rag-form-label">
            {t("rag.config.scanInterval")}
          </label>
          <select
            className="cocursor-rag-form-select"
            value={scan.interval}
            onChange={(e) => onChange({ ...scan, interval: e.target.value })}
          >
            <option value="30m">{t("rag.config.interval30m")}</option>
            <option value="1h">{t("rag.config.interval1h")}</option>
            <option value="2h">{t("rag.config.interval2h")}</option>
            <option value="6h">{t("rag.config.interval6h")}</option>
            <option value="24h">{t("rag.config.interval24h")}</option>
            <option value="manual">{t("rag.config.intervalManual")}</option>
          </select>
        </div>

        {/* 批次大小 */}
        <div className="cocursor-rag-form-field">
          <label className="cocursor-rag-form-label">
            {t("rag.config.batchSize")}: {scan.batchSize}
          </label>
          <input
            type="range"
            className="cocursor-rag-slider"
            min="1"
            max="100"
            value={scan.batchSize}
            onChange={(e) => onChange({ ...scan, batchSize: parseInt(e.target.value) || 10 })}
          />
          <div className="cocursor-rag-form-helper">
            {t("rag.config.batchSizeHelper")}
          </div>
        </div>

        {/* 并发数 */}
        <div className="cocursor-rag-form-field">
          <label className="cocursor-rag-form-label">
            {t("rag.config.concurrency")}: {scan.concurrency}
          </label>
          <input
            type="range"
            className="cocursor-rag-slider"
            min="1"
            max="10"
            value={scan.concurrency}
            onChange={(e) => onChange({ ...scan, concurrency: parseInt(e.target.value) || 3 })}
          />
          <div className="cocursor-rag-form-helper">
            {t("rag.config.concurrencyHelper")}
          </div>
        </div>
      </div>

      {/* 高级选项 */}
      <div className="cocursor-rag-advanced-section">
        <button
          type="button"
          className="cocursor-rag-advanced-toggle"
          onClick={() => setShowAdvanced(!showAdvanced)}
        >
          {showAdvanced ? "▼" : "▶"} {t("rag.config.advanced.title")}
        </button>

        {showAdvanced && (
          <div className="cocursor-rag-advanced-content">
            <div className="cocursor-rag-form-field">
              <label className="cocursor-rag-checkbox-label">
                <input
                  type="checkbox"
                  checked={scan.incrementalScan ?? true}
                  onChange={(e) => onChange({ ...scan, incrementalScan: e.target.checked })}
                />
                <span>{t("rag.config.advanced.incrementalScan")}</span>
              </label>
              <div className="cocursor-rag-form-helper">
                启用后仅扫描变更的文件，提高扫描效率
              </div>
            </div>

            <div className="cocursor-rag-form-field">
              <label className="cocursor-rag-form-label">
                {t("rag.config.advanced.maxFileSize")} (MB)
              </label>
              <input
                type="number"
                className="cocursor-rag-form-input"
                value={scan.maxFileSize ?? 10}
                onChange={(e) => onChange({ ...scan, maxFileSize: parseInt(e.target.value) || 10 })}
                min="1"
                max="1000"
              />
              <div className="cocursor-rag-form-helper">
                超过此大小的文件将被忽略
              </div>
            </div>

            <div className="cocursor-rag-form-field">
              <label className="cocursor-rag-form-label">
                {t("rag.config.advanced.ignorePatterns")}
              </label>
              <input
                type="text"
                className="cocursor-rag-form-input"
                value={scan.ignorePatterns ?? "node_modules/**, .git/**, .cursor/**"}
                onChange={(e) => onChange({ ...scan, ignorePatterns: e.target.value })}
                placeholder="node_modules/**, .git/**, dist/**"
              />
              <div className="cocursor-rag-form-helper">
                使用逗号分隔多个模式，支持 glob 语法
              </div>
            </div>
          </div>
        )}
      </div>

      {/* 配置预览 */}
      <div className="cocursor-rag-config-preview">
        <h4>当前配置预览</h4>
        <ul>
          <li>自动扫描: {scan.enabled ? '✓ 已启用' : '✗ 已禁用'}</li>
          <li>扫描间隔: {scan.interval}</li>
          <li>批次大小: {scan.batchSize} 个文件/批</li>
          <li>并发数: {scan.concurrency}</li>
        </ul>
      </div>
    </div>
  );
};
