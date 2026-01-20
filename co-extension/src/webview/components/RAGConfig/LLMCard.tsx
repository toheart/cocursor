/**
 * LLM Chat API 配置卡片（可选）
 * 用于配置对话总结的 LLM API
 */

import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { useToast } from "../../hooks";

// 预设模板
const TEMPLATES = {
  openai: {
    name: "OpenAI",
    url: "https://api.openai.com/v1",
    model: "gpt-3.5-turbo",
  },
  siliconflow: {
    name: "SiliconFlow",
    url: "https://api.siliconflow.cn/v1",
    model: "Qwen/Qwen2.5-7B-Instruct",
  },
  custom: {
    name: "自定义",
    url: "",
    model: "",
  },
};

type TemplateKey = keyof typeof TEMPLATES;

interface LLMConfig {
  url: string;
  apiKey: string;
  model: string;
}

interface LLMCardProps {
  // 初始配置
  initialConfig?: LLMConfig;
  // 配置变化回调
  onConfigChange?: (config: LLMConfig) => void;
  // 是否已配置
  isConfigured?: boolean;
}

export const LLMCard: React.FC<LLMCardProps> = ({
  initialConfig,
  onConfigChange,
  isConfigured = false,
}) => {
  const { t } = useTranslation();
  const { showToast } = useToast();

  // 是否展开
  const [isExpanded, setIsExpanded] = useState(false);
  // 是否编辑中
  const [isEditing, setIsEditing] = useState(false);

  // 配置状态
  const [config, setConfig] = useState<LLMConfig>({
    url: initialConfig?.url || "",
    apiKey: initialConfig?.apiKey || "",
    model: initialConfig?.model || "",
  });

  // 当前模板
  const [template, setTemplate] = useState<TemplateKey>("custom");

  // 测试状态
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{
    success: boolean;
    message: string;
  } | null>(null);

  // 初始化时检测模板
  useEffect(() => {
    if (initialConfig?.url) {
      if (initialConfig.url.includes("siliconflow")) {
        setTemplate("siliconflow");
      } else if (initialConfig.url.includes("openai.com")) {
        setTemplate("openai");
      } else {
        setTemplate("custom");
      }
      setConfig(initialConfig);
    }
  }, [initialConfig]);

  // 选择模板
  const handleTemplateChange = (key: TemplateKey) => {
    setTemplate(key);
    const t = TEMPLATES[key];
    setConfig((prev) => ({
      ...prev,
      url: t.url,
      model: t.model,
    }));
    setTestResult(null);
  };

  // 更新配置
  const handleConfigChange = (field: keyof LLMConfig, value: string) => {
    setConfig((prev) => ({ ...prev, [field]: value }));
    setTestResult(null);
  };

  // 测试连接
  const handleTest = async () => {
    if (!config.url || !config.model) {
      showToast(t("rag.llm.fillRequired"), "error");
      return;
    }

    // API Key 为占位符时跳过测试
    if (config.apiKey === "••••••") {
      setTestResult({ success: true, message: t("rag.llm.configured") });
      return;
    }

    if (!config.apiKey) {
      showToast(t("rag.llm.apiKeyRequired"), "error");
      return;
    }

    setTesting(true);
    setTestResult(null);

    try {
      const response = (await apiService.testLLMConfig({
        url: config.url,
        api_key: config.apiKey,
        model: config.model,
      })) as { success: boolean; error?: string };

      if (response.success) {
        setTestResult({ success: true, message: t("rag.llm.testSuccess") });
        showToast(t("rag.llm.testSuccess"), "success");
        onConfigChange?.(config);
      } else {
        setTestResult({
          success: false,
          message: response.error || t("rag.llm.testFailed"),
        });
      }
    } catch (error) {
      setTestResult({
        success: false,
        message: error instanceof Error ? error.message : t("rag.llm.testFailed"),
      });
    } finally {
      setTesting(false);
    }
  };

  // 清除配置
  const handleClear = () => {
    setConfig({ url: "", apiKey: "", model: "" });
    setTestResult(null);
    onConfigChange?.({ url: "", apiKey: "", model: "" });
    setIsEditing(false);
  };

  return (
    <div className={`rag-card rag-llm-card ${isExpanded ? "expanded" : ""}`}>
      <div
        className="rag-card-header rag-card-header-clickable"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="rag-card-title">
          <span
            className="rag-status-dot"
            style={{
              backgroundColor: isConfigured
                ? "var(--vscode-terminal-ansiGreen)"
                : "var(--vscode-descriptionForeground)",
            }}
          />
          <span>LLM Chat API</span>
          <span className="rag-card-optional">{t("rag.llm.optional")}</span>
          {config.model && !isExpanded && (
            <span className="rag-card-version">{config.model}</span>
          )}
        </div>
        <div className="rag-card-header-right">
          <span className="rag-expand-icon">{isExpanded ? "▼" : "▶"}</span>
        </div>
      </div>

      {isExpanded && (
        <div className="rag-card-content">
          {/* 说明文字 */}
          <div className="rag-card-description">
            {t("rag.llm.description")}
          </div>

          {!isEditing && !config.url ? (
            // 未配置状态
            <div className="rag-card-empty">
              <span>{t("rag.llm.notConfigured")}</span>
              <button
                className="rag-btn rag-btn-primary"
                onClick={() => setIsEditing(true)}
              >
                {t("rag.llm.configure")}
              </button>
            </div>
          ) : !isEditing && config.url ? (
            // 已配置，显示摘要
            <div className="rag-config-summary">
              <div className="rag-summary-row">
                <span className="rag-summary-label">URL</span>
                <span className="rag-summary-value">{config.url}</span>
              </div>
              <div className="rag-summary-row">
                <span className="rag-summary-label">Model</span>
                <span className="rag-summary-value">{config.model}</span>
              </div>
              <div className="rag-card-actions">
                <button
                  className="rag-btn-link"
                  onClick={() => setIsEditing(true)}
                >
                  {t("common.edit")}
                </button>
                <button className="rag-btn-link rag-btn-danger-link" onClick={handleClear}>
                  {t("rag.llm.clear")}
                </button>
              </div>
            </div>
          ) : (
            // 编辑模式
            <>
              {/* 模板选择 */}
              <div className="rag-template-selector">
                {(Object.keys(TEMPLATES) as TemplateKey[]).map((key) => (
                  <button
                    key={key}
                    className={`rag-template-btn ${template === key ? "active" : ""}`}
                    onClick={() => handleTemplateChange(key)}
                  >
                    {TEMPLATES[key].name}
                  </button>
                ))}
              </div>

              {/* URL */}
              <div className="rag-form-field">
                <label>API URL</label>
                <input
                  type="text"
                  value={config.url}
                  onChange={(e) => handleConfigChange("url", e.target.value)}
                  placeholder="https://api.openai.com/v1"
                />
              </div>

              {/* API Key */}
              <div className="rag-form-field">
                <label>API Key</label>
                <input
                  type="password"
                  value={config.apiKey}
                  onChange={(e) => handleConfigChange("apiKey", e.target.value)}
                  placeholder={t("rag.llm.apiKeyPlaceholder")}
                />
              </div>

              {/* Model */}
              <div className="rag-form-field">
                <label>Model</label>
                <input
                  type="text"
                  value={config.model}
                  onChange={(e) => handleConfigChange("model", e.target.value)}
                  placeholder="gpt-3.5-turbo"
                />
              </div>

              {/* 测试结果 */}
              {testResult && (
                <div
                  className={`rag-test-result ${testResult.success ? "success" : "error"}`}
                >
                  {testResult.success ? "✅" : "❌"} {testResult.message}
                </div>
              )}

              {/* 操作按钮 */}
              <div className="rag-card-actions">
                <button
                  className="rag-btn rag-btn-secondary"
                  onClick={() => {
                    setIsEditing(false);
                    // 恢复原始配置
                    if (initialConfig) {
                      setConfig(initialConfig);
                    }
                  }}
                >
                  {t("common.cancel")}
                </button>
                <button
                  className="rag-btn rag-btn-primary"
                  onClick={handleTest}
                  disabled={testing}
                >
                  {testing ? t("common.loading") : t("rag.llm.test")}
                </button>
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
};
