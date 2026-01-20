/**
 * Embedding API 配置卡片
 * 简化的配置表单，支持模板选择和连接测试
 */

import React, { useState, useEffect, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { useToast } from "../../hooks";

// 预设模板
const TEMPLATES = {
  siliconflow: {
    name: "SiliconFlow",
    url: "https://api.siliconflow.cn/v1",
    model: "BAAI/bge-m3",
  },
  openai: {
    name: "OpenAI",
    url: "https://api.openai.com/v1",
    model: "text-embedding-3-small",
  },
  custom: {
    name: "自定义",
    url: "",
    model: "",
  },
};

type TemplateKey = keyof typeof TEMPLATES;

interface EmbeddingConfig {
  url: string;
  apiKey: string;
  model: string;
}

interface EmbeddingCardProps {
  // 初始配置
  initialConfig?: EmbeddingConfig;
  // 配置变化回调
  onConfigChange?: (config: EmbeddingConfig) => void;
  // 是否已配置（用于显示状态）
  isConfigured?: boolean;
}

export const EmbeddingCard: React.FC<EmbeddingCardProps> = ({
  initialConfig,
  onConfigChange,
  isConfigured = false,
}) => {
  const { t } = useTranslation();
  const { showToast } = useToast();

  // 是否展开编辑
  const [isEditing, setIsEditing] = useState(!isConfigured);
  
  // 配置状态
  const [config, setConfig] = useState<EmbeddingConfig>({
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
  const handleConfigChange = (field: keyof EmbeddingConfig, value: string) => {
    setConfig((prev) => ({ ...prev, [field]: value }));
    setTestResult(null);
  };

  // 测试连接
  const handleTest = async () => {
    if (!config.url || !config.model) {
      showToast(t("rag.embedding.fillRequired"), "error");
      return;
    }
    
    // API Key 为占位符时跳过测试
    if (config.apiKey === "••••••") {
      setTestResult({ success: true, message: t("rag.embedding.configured") });
      return;
    }
    
    if (!config.apiKey) {
      showToast(t("rag.embedding.apiKeyRequired"), "error");
      return;
    }

    setTesting(true);
    setTestResult(null);

    try {
      const response = (await apiService.testRAGConfig({
        url: config.url,
        api_key: config.apiKey,
        model: config.model,
      })) as { success: boolean; error?: string };

      if (response.success) {
        setTestResult({ success: true, message: t("rag.embedding.testSuccess") });
        showToast(t("rag.embedding.testSuccess"), "success");
        onConfigChange?.(config);
      } else {
        setTestResult({
          success: false,
          message: response.error || t("rag.embedding.testFailed"),
        });
      }
    } catch (error) {
      setTestResult({
        success: false,
        message: error instanceof Error ? error.message : t("rag.embedding.testFailed"),
      });
    } finally {
      setTesting(false);
    }
  };

  // 保存并关闭
  const handleSave = () => {
    if (testResult?.success) {
      onConfigChange?.(config);
      setIsEditing(false);
    } else {
      showToast(t("rag.embedding.testFirst"), "error");
    }
  };

  return (
    <div className={`rag-card rag-embedding-card ${isEditing ? "editing" : ""}`}>
      <div className="rag-card-header">
        <div className="rag-card-title">
          <span
            className="rag-status-dot"
            style={{
              backgroundColor: isConfigured || testResult?.success
                ? "var(--vscode-terminal-ansiGreen)"
                : "var(--vscode-descriptionForeground)",
            }}
          />
          <span>Embedding API</span>
          {config.model && !isEditing && (
            <span className="rag-card-version">{config.model}</span>
          )}
        </div>
        {!isEditing && (
          <button
            className="rag-btn-link"
            onClick={() => setIsEditing(true)}
          >
            {t("common.edit")}
          </button>
        )}
      </div>

      {isEditing && (
        <div className="rag-card-content">
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
              placeholder={t("rag.embedding.apiKeyPlaceholder")}
            />
          </div>

          {/* Model */}
          <div className="rag-form-field">
            <label>Model</label>
            <input
              type="text"
              value={config.model}
              onChange={(e) => handleConfigChange("model", e.target.value)}
              placeholder="text-embedding-3-small"
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
              onClick={() => setIsEditing(false)}
            >
              {t("common.cancel")}
            </button>
            <button
              className="rag-btn rag-btn-primary"
              onClick={handleTest}
              disabled={testing}
            >
              {testing ? t("common.loading") : t("rag.embedding.test")}
            </button>
          </div>
        </div>
      )}

      {/* 折叠状态显示配置摘要 */}
      {!isEditing && config.url && (
        <div className="rag-card-summary">
          <span className="rag-summary-url" title={config.url}>
            {config.url}
          </span>
        </div>
      )}
    </div>
  );
};
