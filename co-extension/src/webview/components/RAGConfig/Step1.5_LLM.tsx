/**
 * 步骤 1.5: LLM Chat API 配置
 */

import React, { useState, useEffect, useCallback, useRef } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { useToast } from "../../hooks";
import { PasswordInput } from "./PasswordInput";
import { CONFIG_TEMPLATES, ConfigTemplate } from "./types";

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
      apiKey: '',
      model: templateData.model,
    });

    // 清除错误
    setErrors({});
    setTestResult(null);
  };

  // 验证表单
  const validate = useCallback(() => {
    const newErrors: {
      url?: string;
      apiKey?: string;
      model?: string;
    } = {};

    if (!llm.url.trim()) {
      newErrors.url = t('rag.config.urlRequired');
    }
    if (!llm.apiKey.trim()) {
      newErrors.apiKey = t('rag.config.apiKeyRequired');
    }
    if (!llm.model.trim()) {
      newErrors.model = t('rag.config.modelRequired');
    }

    setErrors(newErrors);
    
    // 标记步骤完成状态
    onStepComplete(Object.keys(newErrors).length === 0);
    
    return Object.keys(newErrors).length === 0;
  }, [llm.url, llm.apiKey, llm.model, t, onStepComplete]);

  // 测试连接
  const handleTestConnection = async () => {
    if (!validate()) {
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
          message: t('rag.config.llmTestSuccess'),
        });
        showToast(t('rag.config.llmTestSuccess'), 'success');

        // 标记步骤完成
        onStepComplete(true);

        // 如果启用了自动前进且测试成功，自动进入下一步
        if (autoAdvance && !autoTestTriggerRef.current) {
          autoTestTriggerRef.current = true;
          setTimeout(() => {
            onStepComplete(true);
          }, 1000);
        }
      } else {
        setTestResult({
          success: false,
          message: response.error || t('rag.config.llmTestFailed'),
        });
        showToast(response.error || t('rag.config.llmTestFailed'), 'error');
        onStepComplete(false);
      }
    } catch (error: any) {
      const errorMessage = error.response?.data?.error || error.message || t('rag.config.llmTestFailed');
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
  };

  // 当所有字段填写完整时自动测试
  useEffect(() => {
    if (autoAdvance && !autoTested && llm.url && llm.apiKey && llm.model && !testing && !testResult) {
      // 延迟自动测试，给用户时间看到输入内容
      const timer = setTimeout(() => {
        handleTestConnection();
      }, 500);
      return () => clearTimeout(timer);
    }
  }, [llm.url, llm.apiKey, llm.model, autoAdvance, autoTested, testing, testResult]);

  // 模型建议列表
  const modelSuggestions = [
    { label: 'gpt-4', value: 'gpt-4' },
    { label: 'gpt-4-turbo-preview', value: 'gpt-4-turbo-preview' },
    { label: 'gpt-3.5-turbo', value: 'gpt-3.5-turbo' },
  ];

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
          className={`cocursor-rag-input ${errors.url ? 'error' : ''}`}
          value={llm.url}
          onChange={(e) => {
            onChange({ ...llm, url: e.target.value });
            setErrors({ ...errors, url: undefined });
            setTestResult(null);
          }}
          placeholder={t('rag.config.llm.urlPlaceholder')}
        />
        {errors.url && <div className="cocursor-rag-error">{errors.url}</div>}
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
            setErrors({ ...errors, apiKey: undefined });
            setTestResult(null);
          }}
          placeholder={t('rag.config.apiKeyPlaceholder')}
          error={errors.apiKey}
        />
      </div>

      {/* Model */}
      <div className="cocursor-rag-field">
        <label className="cocursor-rag-label">
          {t('rag.config.model')} *
        </label>
        <input
          type="text"
          className={`cocursor-rag-input ${errors.model ? 'error' : ''}`}
          value={llm.model}
          onChange={(e) => {
            onChange({ ...llm, model: e.target.value });
            setErrors({ ...errors, model: undefined });
            setTestResult(null);
          }}
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
        {errors.model && <div className="cocursor-rag-error">{errors.model}</div>}
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
