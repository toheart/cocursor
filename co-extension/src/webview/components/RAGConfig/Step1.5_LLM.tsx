/**
 * 步骤 1.5: LLM Chat API 配置
 * 优化版本：添加更多模型选项，改进 UX 体验
 */

import React, { useState, useEffect, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { useToast } from "../../hooks";
import { PasswordInput } from "./PasswordInput";
import { CONFIG_TEMPLATES, ConfigTemplate } from "./types";

// LLM 模型建议列表（扩展版）
const LLM_MODEL_SUGGESTIONS = [
  { label: 'gpt-4o (推荐)', value: 'gpt-4o', provider: 'openai' },
  { label: 'gpt-4o-mini', value: 'gpt-4o-mini', provider: 'openai' },
  { label: 'gpt-4-turbo', value: 'gpt-4-turbo', provider: 'openai' },
  { label: 'gpt-4', value: 'gpt-4', provider: 'openai' },
  { label: 'gpt-3.5-turbo', value: 'gpt-3.5-turbo', provider: 'openai' },
  { label: 'claude-3-5-sonnet', value: 'claude-3-5-sonnet-20241022', provider: 'anthropic' },
  { label: 'claude-3-5-haiku', value: 'claude-3-5-haiku-20241022', provider: 'anthropic' },
];

interface Step1_5Props {
  llm: {
    url: string;
    apiKey: string;
    model: string;
  };
  onChange: (data: { url: string; apiKey: string; model: string }) => void;
  onStepComplete: (completed: boolean) => void;
  autoAdvance?: boolean; // 是否自动进入下一步
}

export const Step1_5_LLM: React.FC<Step1_5Props> = ({
  llm,
  onChange,
  onStepComplete,
  autoAdvance = false,
}) => {
  const { t } = useTranslation();
  const { showToast } = useToast();

  const [selectedTemplate, setSelectedTemplate] = useState<ConfigTemplate>('custom');
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);
  const [autoTested, setAutoTested] = useState(false);

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

  // 是否已尝试提交
  const [submitted, setSubmitted] = useState(false);

  // 选择模板
  const handleTemplateSelect = (template: ConfigTemplate) => {
    setSelectedTemplate(template);
    const templateData = CONFIG_TEMPLATES[template];

    onChange({
      url: templateData.url,
      apiKey: '',
      model: templateData.model,
    });

    // 清除错误和状态
    setErrors({});
    setTouched({ url: false, apiKey: false, model: false });
    setSubmitted(false);
    setTestResult(null);
  };

  // 验证单个字段
  const validateField = useCallback((field: 'url' | 'apiKey' | 'model', value: string): string | undefined => {
    switch (field) {
      case 'url':
        if (!value.trim()) {
          return t('rag.config.urlRequired');
        }
        return undefined;
      case 'apiKey':
        // 占位符表示已配置，通过验证
        if (!value.trim() && value !== '••••••') {
          return t('rag.config.apiKeyRequired');
        }
        return undefined;
      case 'model':
        if (!value.trim()) {
          return t('rag.config.modelRequired');
        }
        return undefined;
      default:
        return undefined;
    }
  }, [t]);

  // 验证所有字段（用于提交时）
  const validateAllFields = useCallback(() => {
    const newErrors: typeof errors = {};
    
    const urlError = validateField('url', llm.url);
    if (urlError) newErrors.url = urlError;
    
    const apiKeyError = validateField('apiKey', llm.apiKey);
    if (apiKeyError) newErrors.apiKey = apiKeyError;
    
    const modelError = validateField('model', llm.model);
    if (modelError) newErrors.model = modelError;
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [llm.url, llm.apiKey, llm.model, validateField]);

  // 检查表单是否完整（不触发错误显示）
  const isFormComplete = useCallback(() => {
    return (
      llm.url.trim() &&
      (llm.apiKey.trim() || llm.apiKey === '••••••') &&
      llm.model.trim()
    );
  }, [llm.url, llm.apiKey, llm.model]);

  // 字段 blur 时验证
  const handleBlur = (field: 'url' | 'apiKey' | 'model') => {
    setTouched(prev => ({ ...prev, [field]: true }));
    
    const value = llm[field];
    const error = validateField(field, value);
    setErrors(prev => ({ ...prev, [field]: error }));
  };

  // 测试连接
  const handleTestConnection = useCallback(async () => {
    setSubmitted(true);
    
    // 如果 API Key 是占位符，说明已配置，不需要测试
    if (llm.apiKey === '••••••') {
      setTestResult({
        success: true,
        message: t("rag.config.apiKeyConfigured"),
      });
      showToast(t("rag.config.apiKeyConfigured"), "success");
      onStepComplete(true);
      return;
    }

    // 验证所有字段
    if (!validateAllFields()) {
      return;
    }

    setTesting(true);
    setTestResult(null);

    try {
      const response = await apiService.testLLMConnection({
        url: llm.url,
        api_key: llm.apiKey,
        model: llm.model,
      });

      if (response.success) {
        setTestResult({
          success: true,
          message: t('rag.config.llm.testSuccess'),
        });
        showToast(t('rag.config.llm.testSuccess'), 'success');
        onStepComplete(true);
      } else {
        setTestResult({
          success: false,
          message: response.error || t('rag.config.testFailed'),
        });
        showToast(response.error || t('rag.config.testFailed'), 'error');
        onStepComplete(false);
      }
    } catch (error: any) {
      const errorMessage = error.response?.data?.error || error.message || t('rag.config.testFailed');
      setTestResult({
        success: false,
        message: errorMessage,
      });
      showToast(errorMessage, 'error');
      onStepComplete(false);
    } finally {
      setTesting(false);
      setAutoTested(true);
    }
  }, [llm.url, llm.apiKey, llm.model, showToast, t, validateAllFields, onStepComplete]);

  // 更新步骤完成状态
  useEffect(() => {
    const formComplete = isFormComplete();
    const testSuccess = testResult?.success;
    onStepComplete(formComplete && !!testSuccess);
  }, [isFormComplete, testResult, onStepComplete]);

  // 自动测试
  useEffect(() => {
    if (autoAdvance && !autoTested && isFormComplete() && !testing && !testResult) {
      if (llm.apiKey === '••••••') {
        setTestResult({
          success: true,
          message: t("rag.config.apiKeyConfigured"),
        });
        onStepComplete(true);
        setAutoTested(true);
      } else if (llm.apiKey) {
        const timer = setTimeout(() => {
          handleTestConnection();
        }, 500);
        return () => clearTimeout(timer);
      }
    }
  }, [autoAdvance, autoTested, isFormComplete, testing, testResult, llm.apiKey, handleTestConnection, onStepComplete, t]);

  // 判断是否显示某个字段的错误
  const shouldShowError = (field: 'url' | 'apiKey' | 'model') => {
    return (touched[field] || submitted) && errors[field];
  };

  // 根据选择的模板过滤模型建议
  const modelSuggestions = useMemo(() => {
    if (selectedTemplate === 'openai') {
      return LLM_MODEL_SUGGESTIONS.filter(m => m.provider === 'openai');
    }
    return LLM_MODEL_SUGGESTIONS;
  }, [selectedTemplate]);

  // 检测当前 URL 对应的模板类型
  useEffect(() => {
    if (llm.url.includes('openai.com')) {
      setSelectedTemplate('openai');
    } else if (llm.url.includes('azure.com')) {
      setSelectedTemplate('azure');
    } else if (llm.url) {
      setSelectedTemplate('custom');
    }
  }, []);

  return (
    <div className="cocursor-rag-step">
      <h2 className="cocursor-rag-step-title">
        {t('rag.config.llm.title')}
      </h2>
      <p className="cocursor-rag-step-description">
        {t('rag.config.llm.description')}
      </p>

      {/* 模板选择 */}
      <div className="cocursor-rag-template-selector">
        <label className="cocursor-rag-label">{t('rag.config.selectTemplate')}:</label>
        <div className="cocursor-rag-template-buttons">
          <button
            type="button"
            className={`cocursor-rag-template-button ${selectedTemplate === 'openai' ? 'active' : ''}`}
            onClick={() => handleTemplateSelect('openai')}
          >
            {t('rag.config.template.openai')}
          </button>
          <button
            type="button"
            className={`cocursor-rag-template-button ${selectedTemplate === 'azure' ? 'active' : ''}`}
            onClick={() => handleTemplateSelect('azure')}
          >
            {t('rag.config.template.azure')}
          </button>
          <button
            type="button"
            className={`cocursor-rag-template-button ${selectedTemplate === 'custom' ? 'active' : ''}`}
            onClick={() => handleTemplateSelect('custom')}
          >
            {t('rag.config.template.custom')}
          </button>
        </div>
      </div>

      {/* API URL */}
      <div className="cocursor-rag-field">
        <label className="cocursor-rag-label">
          {t('rag.config.apiUrl')} *
        </label>
        <input
          type="text"
          className={`cocursor-rag-input ${shouldShowError('url') ? 'error' : ''}`}
          value={llm.url}
          onChange={(e) => {
            onChange({ ...llm, url: e.target.value });
            setTestResult(null);
          }}
          onBlur={() => handleBlur('url')}
          placeholder={t('rag.config.llm.urlPlaceholder')}
        />
        {shouldShowError('url') && <div className="cocursor-rag-error">{errors.url}</div>}
      </div>

      {/* API Key */}
      <div className="cocursor-rag-field">
        <label className="cocursor-rag-label">
          {t('rag.config.apiKey')} *
        </label>
        <PasswordInput
          value={llm.apiKey}
          onChange={(value) => {
            onChange({ ...llm, apiKey: value });
            setTestResult(null);
          }}
          placeholder={t('rag.config.apiKeyPlaceholder')}
          error={shouldShowError('apiKey') ? errors.apiKey : undefined}
        />
        {llm.apiKey === '••••••' && (
          <div className="cocursor-rag-form-hint">
            {t('rag.config.apiKeyConfigured')}
          </div>
        )}
      </div>

      {/* Model */}
      <div className="cocursor-rag-field">
        <label className="cocursor-rag-label">
          {t('rag.config.model')} *
        </label>
        <input
          type="text"
          className={`cocursor-rag-input ${shouldShowError('model') ? 'error' : ''}`}
          value={llm.model}
          onChange={(e) => {
            onChange({ ...llm, model: e.target.value });
            setTestResult(null);
          }}
          onBlur={() => handleBlur('model')}
          placeholder={t('rag.config.llm.modelPlaceholder')}
          list="llm-model-suggestions"
        />
        <datalist id="llm-model-suggestions">
          {modelSuggestions.map((suggestion) => (
            <option key={suggestion.value} value={suggestion.value}>
              {suggestion.label}
            </option>
          ))}
        </datalist>
        {shouldShowError('model') && <div className="cocursor-rag-error">{errors.model}</div>}
      </div>

      {/* 测试连接按钮 */}
      <div className="cocursor-rag-test-section">
        <button
          type="button"
          className="cocursor-rag-button cocursor-rag-button-primary"
          onClick={handleTestConnection}
          disabled={testing}
        >
          {testing ? t('rag.config.testing') : t('rag.config.testConnection')}
        </button>

        {/* 测试结果 */}
        {testResult && (
          <div className={`cocursor-rag-test-result ${testResult.success ? 'success' : 'error'}`}>
            {testResult.success ? '✓ ' : '✗ '}
            {testResult.message}
          </div>
        )}
      </div>

      {/* 帮助文本 */}
      <div className="cocursor-rag-help-text">
        <h4>{t('rag.config.llm.helpTitle')}</h4>
        <p>{t('rag.config.llm.helpText')}</p>
        <ul>
          <li>{t('rag.config.llm.help1')}</li>
          <li>{t('rag.config.llm.help2')}</li>
          <li>{t('rag.config.llm.help3')}</li>
        </ul>
      </div>
    </div>
  );
};
