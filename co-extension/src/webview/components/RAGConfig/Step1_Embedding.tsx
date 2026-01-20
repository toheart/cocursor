/**
 * 步骤 1: Embedding API 配置
 * 优化版本：改进 UX 体验，统一验证逻辑，添加模型建议
 */

import React, { useState, useEffect, useCallback, useRef, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { useToast } from "../../hooks";
import { PasswordInput } from "./PasswordInput";
import { CONFIG_TEMPLATES, ConfigTemplate } from "./types";

// Embedding 模型建议列表
const EMBEDDING_MODEL_SUGGESTIONS = [
  { label: 'text-embedding-3-small (推荐)', value: 'text-embedding-3-small', provider: 'openai' },
  { label: 'text-embedding-3-large', value: 'text-embedding-3-large', provider: 'openai' },
  { label: 'text-embedding-ada-002', value: 'text-embedding-ada-002', provider: 'openai' },
];

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
  const autoTestTriggerRef = useRef(false); // 防止自动测试重复触发

  // 验证错误
  const [errors, setErrors] = useState<{
    url?: string;
    apiKey?: string;
    model?: string;
  }>({});

  // 追踪哪些字段被用户"触摸"过
  const [touched, setTouched] = useState<{
    url: boolean;
    apiKey: boolean;
    model: boolean;
  }>({
    url: false,
    apiKey: false,
    model: false,
  });

  // 是否已尝试提交（点击测试连接或下一步）
  const [submitted, setSubmitted] = useState(false);

  // 根据选择的模板过滤模型建议
  const filteredModelSuggestions = useMemo(() => {
    if (selectedTemplate === 'openai') {
      return EMBEDDING_MODEL_SUGGESTIONS.filter(m => m.provider === 'openai');
    }
    return EMBEDDING_MODEL_SUGGESTIONS;
  }, [selectedTemplate]);

  // 检测当前 URL 对应的模板类型
  useEffect(() => {
    if (embedding.url.includes('openai.com')) {
      setSelectedTemplate('openai');
    } else if (embedding.url.includes('azure.com')) {
      setSelectedTemplate('azure');
    } else if (embedding.url) {
      setSelectedTemplate('custom');
    }
  }, []);

  // 选择模板
  const handleTemplateSelect = (template: ConfigTemplate) => {
    setSelectedTemplate(template);
    const templateData = CONFIG_TEMPLATES[template];

    onChange({
      url: templateData.url,
      apiKey: embedding.apiKey, // 保留 API Key
      model: templateData.model,
    });

    // 重置错误状态
    setErrors({});
    setTouched({ url: false, apiKey: false, model: false });
    setSubmitted(false);
  };

  // URL 验证
  const isValidUrl = (url: string) => {
    try {
      new URL(url);
      return true;
    } catch {
      return false;
    }
  };

  // 验证单个字段
  const validateField = useCallback((field: 'url' | 'apiKey' | 'model', value: string): string | undefined => {
    switch (field) {
      case 'url':
        if (!value) {
          return t("rag.config.urlRequired");
        } else if (!isValidUrl(value)) {
          return t("rag.config.invalidUrl");
        }
        return undefined;
      case 'apiKey':
        // 占位符表示已配置，通过验证
        if (!value && value !== '••••••') {
          return t("rag.config.apiKeyRequired");
        }
        return undefined;
      case 'model':
        if (!value) {
          return t("rag.config.modelRequired");
        }
        return undefined;
      default:
        return undefined;
    }
  }, [t]);

  // 验证所有字段（用于提交时）
  const validateAllFields = useCallback(() => {
    const newErrors: typeof errors = {};
    
    const urlError = validateField('url', embedding.url);
    if (urlError) newErrors.url = urlError;
    
    const apiKeyError = validateField('apiKey', embedding.apiKey);
    if (apiKeyError) newErrors.apiKey = apiKeyError;
    
    const modelError = validateField('model', embedding.model);
    if (modelError) newErrors.model = modelError;
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [embedding.url, embedding.apiKey, embedding.model, validateField]);

  // 检查表单是否完整（不触发错误显示）
  const isFormComplete = useCallback(() => {
    return (
      embedding.url &&
      isValidUrl(embedding.url) &&
      (embedding.apiKey || embedding.apiKey === '••••••') &&
      embedding.model
    );
  }, [embedding.url, embedding.apiKey, embedding.model]);

  // 字段 blur 时验证
  const handleBlur = (field: 'url' | 'apiKey' | 'model') => {
    setTouched(prev => ({ ...prev, [field]: true }));
    
    const value = embedding[field];
    const error = validateField(field, value);
    setErrors(prev => ({ ...prev, [field]: error }));
  };

  // 测试连接
  const handleTestConnection = useCallback(async () => {
    setSubmitted(true);
    
    // 验证所有字段
    if (!validateAllFields()) {
      return;
    }

    // 如果 API Key 是占位符，说明已配置，不需要测试
    if (embedding.apiKey === '••••••') {
      setTestResult({
        success: true,
        message: t("rag.config.apiKeyConfigured"),
      });
      showToast(t("rag.config.apiKeyConfigured"), "success");
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
  }, [embedding.url, embedding.apiKey, embedding.model, showToast, t, validateAllFields]);

  // 更新步骤完成状态
  useEffect(() => {
    const formComplete = isFormComplete();
    const testSuccess = testResult?.success;
    onStepComplete(formComplete && !!testSuccess);
  }, [isFormComplete, testResult, onStepComplete]);

  // 自动测试：只在 autoAdvance 模式下且表单完整时触发一次
  useEffect(() => {
    if (
      autoAdvance &&
      isFormComplete() &&
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
  }, [autoAdvance, isFormComplete, testResult, testing, handleTestConnection, embedding.apiKey, t]);

  // 判断是否显示某个字段的错误
  const shouldShowError = (field: 'url' | 'apiKey' | 'model') => {
    return (touched[field] || submitted) && errors[field];
  };

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
          {t("rag.config.templateLabel")}
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
              {t(CONFIG_TEMPLATES[template].name)}
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
              shouldShowError('url') ? "error" : ""
            }`}
            value={embedding.url}
            onChange={(e) => 
              onChange({ ...embedding, url: e.target.value })
            }
            onBlur={() => handleBlur('url')}
            placeholder="https://api.openai.com"
          />
          <div className="cocursor-rag-form-hint">
            {t("rag.config.apiUrlHint")}
          </div>
          {shouldShowError('url') && (
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
            error={shouldShowError('apiKey') ? errors.apiKey : undefined}
          />
          {embedding.apiKey === '••••••' && (
            <div className="cocursor-rag-form-hint">
              {t("rag.config.apiKeyConfigured")}
            </div>
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
              shouldShowError('model') ? "error" : ""
            }`}
            value={embedding.model}
            onChange={(e) => 
              onChange({ ...embedding, model: e.target.value })
            }
            onBlur={() => handleBlur('model')}
            placeholder="text-embedding-3-small"
            list="embedding-model-suggestions"
          />
          <datalist id="embedding-model-suggestions">
            {filteredModelSuggestions.map((suggestion) => (
              <option key={suggestion.value} value={suggestion.value}>
                {suggestion.label}
              </option>
            ))}
          </datalist>
          <div className="cocursor-rag-form-hint">
            {t("rag.config.embeddingModelHint")}
          </div>
          {shouldShowError('model') && (
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
            disabled={testing}
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
