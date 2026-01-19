/**
 * 步骤 3: 扫描策略配置
 */

import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";

interface Step3Props {
  scan: {
    enabled: boolean;
    interval: string;
    batchSize: number;
    concurrency: number;
  };
  onChange: (data: { enabled: boolean; interval: string; batchSize: number; concurrency: number }) => void;
  onStepComplete: (completed: boolean) => void;
}

export const Step3_Scan: React.FC<Step3Props> = ({
  scan,
  onChange,
  onStepComplete,
}) => {
  const { t } = useTranslation();
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [advanced, setAdvanced] = useState({
    incrementalScan: false,
    maxFileSize: 10,
    ignorePatterns: "node_modules/**, .git/**, .cursor/**",
  });

  // 推荐配置
  const recommendedConfig = {
    enabled: true,
    interval: "1h",
    batchSize: 10,
    concurrency: 3,
  };

  // 检查步骤是否完成
  const isComplete = !!scan.interval && scan.batchSize > 0 && scan.concurrency > 0;
  useEffect(() => {
    onStepComplete(isComplete);
  }, [isComplete, onStepComplete]);

  // 应用推荐配置
  const handleUseRecommended = () => {
    onChange({
      ...scan,
      ...recommendedConfig,
    });
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
                  checked={advanced.incrementalScan}
                  onChange={(e) => setAdvanced({ ...advanced, incrementalScan: e.target.checked })}
                />
                <span>{t("rag.config.advanced.incrementalScan")}</span>
              </label>
            </div>

            <div className="cocursor-rag-form-field">
              <label className="cocursor-rag-form-label">
                {t("rag.config.advanced.maxFileSize")}
              </label>
              <input
                type="number"
                className="cocursor-rag-form-input"
                value={advanced.maxFileSize}
                onChange={(e) => setAdvanced({ ...advanced, maxFileSize: parseInt(e.target.value) || 10 })}
                min="1"
                max="1000"
              />
            </div>

            <div className="cocursor-rag-form-field">
              <label className="cocursor-rag-form-label">
                {t("rag.config.advanced.ignorePatterns")}
              </label>
              <input
                type="text"
                className="cocursor-rag-form-input"
                value={advanced.ignorePatterns}
                onChange={(e) => setAdvanced({ ...advanced, ignorePatterns: e.target.value })}
                placeholder="node_modules/**, .git/**"
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
