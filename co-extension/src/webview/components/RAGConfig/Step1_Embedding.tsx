/**
 * 步骤 1: Embedding API 配置
 */

import React, { useState, useEffect, useCallback, useRef } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { useToast } from "../../hooks";
import { PasswordInput } from "./PasswordInput";
import { CONFIG_TEMPLATES, ConfigTemplate } from "./types";

interface Step1Props {
  embedding: {
    url: string;
    apiKey: string;
    model: string;
  };
  onChange: (data: { url: string; apiKey: string; model: string }) => void;
  onStepComplete: (completed: boolean) => void;
  autoAdvance?: boolean; // 是否自动进入下一步
}

export const Step1_Embedding: React.FC<Step1Props> = ({
  embedding,
  onChange,
  onStepComplete,
  autoAdvance = false,
}) => {
  const { t } = useTranslation();
  const { showToast } = useToast();

  const [selectedTemplate, setSelectedTemplate] = useState<ConfigTemplate>('custom');
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);
  const [autoTested, setAutoTested] = useState(false); // 是否已自动测试
  const autoTestTriggerRef = useRef(false); // 防止自动测试重复触发

  // 验证错误
  const [errors, setErrors] = useState<{
    url?: string;
    apiKey?: string;
    model?: string;
  }>({});

  // 选择模板
  const handleTemplateSelect = (template: ConfigTemplate) => {
    setSelectedTemplate(template);
    const templateData = CONFIG_TEMPLATES[template];

    onChange({
      url: templateData.url,
      apiKey: embedding.apiKey, // 保留 API Key
      model: templateData.model,
    });
  };

  // 表单验证 - 使用 useCallback 避免循环依赖
  const validateForm = useCallback(() => {
    const newErrors: typeof errors = {};

    // 验证 URL
    if (!embedding.url) {
      newErrors.url = t("rag.config.urlRequired");
    } else if (embedding.url && !isValidUrl(embedding.url)) {
      newErrors.url = t("rag.config.invalidUrl");
    }

    // 验证 API Key（占位符表示已配置，通过验证）
    if (!embedding.apiKey && embedding.apiKey !== '••••••') {
      newErrors.apiKey = t("rag.config.apiKeyRequired");
    }

    // 验证模型
    if (!embedding.model) {
      newErrors.model = t("rag.config.modelRequired");
    }

    setErrors(newErrors);

    // 检查步骤是否完成
    const isComplete =
      !newErrors.url &&
      !newErrors.apiKey &&
      !newErrors.model;

    return isComplete;
  }, [embedding.url, embedding.apiKey, embedding.model, t]);

  // URL 验证
  const isValidUrl = (url: string) => {
    try {
      new URL(url);
      return true;
    } catch {
      return false;
    }
  };

  // 测试连接
  const handleTestConnection = useCallback(async () => {
    // 如果 API Key 是占位符，说明已配置，不需要测试
    if (embedding.apiKey === '••••••') {
      setTestResult({
        success: true,
        message: t("rag.config.apiKeyConfigured"),
      });
      showToast(t("rag.config.apiKeyConfigured"), "success");
      return;
    }

    if (!embedding.url || !embedding.apiKey || !embedding.model) {
      showToast(t("rag.config.testRequired"), "error");
      return;
    }

    setTesting(true);
    setTestResult(null);

    try {
      const response = await apiService.testRAGConfig({
        url: embedding.url,
        api_key: embedding.apiKey,
        model: embedding.model,
      }) as { success: boolean; error?: string };

      if (response.success) {
        setTestResult({
          success: true,
          message: t("rag.config.testSuccess"),
        });
        showToast(t("rag.config.testSuccess"), "success");
      } else {
        setTestResult({
          success: false,
          message: t("rag.config.testFailed") + ": " + (response.error || ""),
        });
        showToast(t("rag.config.testFailed") + ": " + (response.error || ""), "error");
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      setTestResult({
        success: false,
        message: t("rag.config.testFailed") + ": " + errorMessage,
      });
      showToast(t("rag.config.testFailed") + ": " + errorMessage, "error");
    } finally {
      setTesting(false);
    }
  }, [embedding.url, embedding.apiKey, embedding.model, showToast, t]);

  // 自动测试：只触发一次
  useEffect(() => {
    const isComplete = validateForm();
    const testSuccess = testResult?.success;

    onStepComplete(isComplete && testSuccess);

    // 自动测试：如果表单完整且未测试过，触发一次测试
    // 如果 API Key 是占位符，直接标记为成功，不测试
    if (
      autoAdvance &&
      isComplete &&
      !autoTestTriggerRef.current &&
      !testResult &&
      !testing
    ) {
      autoTestTriggerRef.current = true;
      if (embedding.apiKey === '••••••') {
        // API Key 是占位符，直接标记为已配置
        setTestResult({
          success: true,
          message: t("rag.config.apiKeyConfigured"),
        });
      } else {
        // 需要测试连接
        handleTestConnection();
      }
    }
  }, [validateForm, onStepComplete, testResult, autoAdvance, testing, handleTestConnection, embedding.apiKey, t]);

  return (
    <div className="cocursor-rag-step-1">
      <div className="cocursor-rag-step-header">
        <h3 className="cocursor-rag-step-title">{t("rag.config.step1.title")}</h3>
        <p className="cocursor-rag-step-description">
          {t("rag.config.step1.description")}
        </p>
      </div>

      {/* 模板选择器 */}
      <div className="cocursor-rag-template-selector">
        <label className="cocursor-rag-template-label">
          {t("rag.config.template")}
        </label>
        <div className="cocursor-rag-template-buttons">
          {(["openai", "azure", "custom"] as ConfigTemplate[]).map((template) => (
            <button
              key={template}
              type="button"
              className={`cocursor-rag-template-button ${
                selectedTemplate === template ? "active" : ""
              }`}
              onClick={() => handleTemplateSelect(template)}
            >
              {CONFIG_TEMPLATES[template].name}
            </button>
          ))}
        </div>
      </div>

      {/* 表单字段 */}
      <div className="cocursor-rag-form">
        {/* API URL */}
        <div className="cocursor-rag-form-field">
          <label className="cocursor-rag-form-label">
            {t("rag.config.apiUrl")} *
          </label>
          <input
            type="text"
            className={`cocursor-rag-form-input ${
              errors.url ? "error" : ""
            }`}
            value={embedding.url}
            onChange={(e) => 
              onChange({ ...embedding, url: e.target.value })
            }
            placeholder="https://api.openai.com"
          />
          {errors.url && (
            <div className="cocursor-rag-form-error">{errors.url}</div>
          )}
        </div>

        {/* API Key */}
        <div className="cocursor-rag-form-field">
          <label className="cocursor-rag-form-label">
            {t("rag.config.apiKey")} *
          </label>
          <PasswordInput
            value={embedding.apiKey}
            onChange={(value) => onChange({ ...embedding, apiKey: value })}
            placeholder={t("rag.config.apiKeyPlaceholder")}
            label=""
            error={errors.apiKey}
          />
          {embedding.apiKey === '••••••' && (
            <div className="cocursor-rag-form-hint">
              {t("rag.config.apiKeyConfigured")}
            </div>
          )}
          {errors.apiKey && (
            <div className="cocursor-rag-form-error">{errors.apiKey}</div>
          )}
        </div>

        {/* 模型 */}
        <div className="cocursor-rag-form-field">
          <label className="cocursor-rag-form-label">
            {t("rag.config.model")} *
          </label>
          <input
            type="text"
            className={`cocursor-rag-form-input ${
              errors.model ? "error" : ""
            }`}
            value={embedding.model}
            onChange={(e) => 
              onChange({ ...embedding, model: e.target.value })
            }
            placeholder="text-embedding-ada-002"
          />
          {errors.model && (
            <div className="cocursor-rag-form-error">{errors.model}</div>
          )}
        </div>
      </div>

      {/* 手动测试按钮 - 自动模式时隐藏 */}
      {!autoAdvance && (
        <div className="cocursor-rag-test-section">
          <button
            type="button"
            className="cocursor-rag-test-button"
            onClick={handleTestConnection}
            disabled={
              testing ||
              !embedding.url ||
              !embedding.apiKey ||
              !embedding.model
            }
          >
            {testing ? t("rag.config.testing") : t("rag.config.testConnection")}
          </button>

          {testResult && (
            <div
              className={`cocursor-rag-test-result ${
                testResult.success ? "success" : "error"
              }`}
            >
              {testResult.success ? "✅" : "❌"} {testResult.message}
            </div>
          )}
        </div>
      )}

      {/* 自动模式下显示测试状态 */}
      {autoAdvance && testing && (
        <div className="cocursor-rag-test-result">
          <span className="cocursor-rag-test-indicator" />
          <span>{t("rag.config.autoTesting")}</span>
        </div>
      )}

      {autoAdvance && testResult && (
        <div
              className={`cocursor-rag-test-result ${
                testResult.success ? "success" : "error"
              }`}
        >
          {testResult.success ? "✅" : "❌"} {testResult.message}
        </div>
      )}
    </div>
  );
};
