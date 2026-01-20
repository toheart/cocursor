/**
 * RAG 配置组件 - 简化版
 * 单页卡片式布局，所有配置一目了然
 */

import React, { useState, useEffect, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../services/api";
import { useToast } from "../hooks";
import { ToastContainer } from "./shared";
import { QdrantCard } from "./RAGConfig/QdrantCard";
import { EmbeddingCard } from "./RAGConfig/EmbeddingCard";
import { LLMCard } from "./RAGConfig/LLMCard";
import { IndexCard } from "./RAGConfig/IndexCard";

// Qdrant 状态类型
type QdrantStatus = "not-installed" | "installed" | "running" | "downloading";

// API 配置
interface APIConfig {
  url: string;
  apiKey: string;
  model: string;
}

// 后端配置响应
interface RAGConfigResponse {
  embedding_api: {
    url: string;
    model: string;
    has_api_key?: boolean;
  };
  llm_chat_api?: {
    url: string;
    model: string;
    has_api_key?: boolean;
  };
  qdrant: {
    version: string;
    binary_path: string;
  };
}

export const RAGConfig: React.FC = () => {
  const { t } = useTranslation();
  const { showToast, toasts } = useToast();

  // 状态
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  
  // Qdrant 状态
  const [qdrantStatus, setQdrantStatus] = useState<QdrantStatus>("not-installed");
  
  // Embedding 配置
  const [embeddingConfig, setEmbeddingConfig] = useState<APIConfig>({
    url: "",
    apiKey: "",
    model: "",
  });
  const [embeddingConfigured, setEmbeddingConfigured] = useState(false);
  
  // LLM 配置（可选）
  const [llmConfig, setLLMConfig] = useState<APIConfig>({
    url: "",
    apiKey: "",
    model: "",
  });
  const [llmConfigured, setLLMConfigured] = useState(false);
  
  // 是否有未保存的更改
  const [hasChanges, setHasChanges] = useState(false);

  // 加载配置
  const loadConfig = useCallback(async () => {
    try {
      const response = (await apiService.getRAGConfig()) as RAGConfigResponse;
      if (response) {
        // 设置 Embedding 配置
        const embedding = response.embedding_api;
        if (embedding) {
          setEmbeddingConfig({
            url: embedding.url || "",
            apiKey: embedding.has_api_key ? "••••••" : "",
            model: embedding.model || "",
          });
          setEmbeddingConfigured(!!(embedding.url && embedding.model && embedding.has_api_key));
        }
        
        // 设置 LLM 配置
        const llm = response.llm_chat_api;
        if (llm) {
          setLLMConfig({
            url: llm.url || "",
            apiKey: llm.has_api_key ? "••••••" : "",
            model: llm.model || "",
          });
          setLLMConfigured(!!(llm.url && llm.model && llm.has_api_key));
        }
      }
    } catch (error) {
      console.error("Failed to load RAG config:", error);
    } finally {
      setLoading(false);
    }
  }, []);

  // 初始化加载
  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  // Qdrant 状态变化处理
  const handleQdrantStatusChange = (status: QdrantStatus) => {
    setQdrantStatus(status);
  };

  // Embedding 配置变化处理
  const handleEmbeddingConfigChange = (config: APIConfig) => {
    setEmbeddingConfig(config);
    setHasChanges(true);
  };

  // LLM 配置变化处理
  const handleLLMConfigChange = (config: APIConfig) => {
    setLLMConfig(config);
    setHasChanges(true);
  };

  // 保存配置
  const handleSave = async () => {
    // 验证必填项
    if (!embeddingConfig.url || !embeddingConfig.model) {
      showToast(t("rag.save.embeddingRequired"), "error");
      return;
    }

    setSaving(true);
    try {
      const configData: any = {
        embedding_api: {
          url: embeddingConfig.url,
          model: embeddingConfig.model,
        },
      };
      
      // 只有当 API Key 不是占位符时才发送
      if (embeddingConfig.apiKey && embeddingConfig.apiKey !== "••••••") {
        configData.embedding_api.api_key = embeddingConfig.apiKey;
      }

      // LLM 配置（可选）
      if (llmConfig.url && llmConfig.model) {
        configData.llm_chat_api = {
          url: llmConfig.url,
          model: llmConfig.model,
        };
        if (llmConfig.apiKey && llmConfig.apiKey !== "••••••") {
          configData.llm_chat_api.api_key = llmConfig.apiKey;
        }
      }

      await apiService.updateRAGConfig(configData);
      showToast(t("rag.save.success"), "success");
      setHasChanges(false);
      setEmbeddingConfigured(true);
      if (llmConfig.url && llmConfig.model) {
        setLLMConfigured(true);
      }
    } catch (error) {
      showToast(t("rag.save.failed"), "error");
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="rag-loading">
        <span className="rag-spinner" />
        <span>{t("common.loading")}</span>
      </div>
    );
  }

  return (
    <div className="rag-config">
      {/* 头部 */}
      <div className="rag-header">
        <h2>
          RAG Config
          <span className="rag-beta-badge">BETA</span>
        </h2>
      </div>

      {/* 状态栏 */}
      <div className="rag-status-bar">
        <div className="rag-status-item">
          <span
            className="rag-status-dot"
            style={{
              backgroundColor:
                qdrantStatus === "running"
                  ? "var(--vscode-terminal-ansiGreen)"
                  : "var(--vscode-descriptionForeground)",
            }}
          />
          <span>Qdrant</span>
        </div>
        <div className="rag-status-item">
          <span
            className="rag-status-dot"
            style={{
              backgroundColor: embeddingConfigured
                ? "var(--vscode-terminal-ansiGreen)"
                : "var(--vscode-descriptionForeground)",
            }}
          />
          <span>Embedding</span>
        </div>
      </div>

      {/* 卡片区域 */}
      <div className="rag-cards">
        {/* Qdrant 卡片 */}
        <QdrantCard onStatusChange={handleQdrantStatusChange} />

        {/* Embedding 卡片 */}
        <EmbeddingCard
          initialConfig={embeddingConfig}
          onConfigChange={handleEmbeddingConfigChange}
          isConfigured={embeddingConfigured}
        />

        {/* LLM 卡片（可选） */}
        <LLMCard
          initialConfig={llmConfig}
          onConfigChange={handleLLMConfigChange}
          isConfigured={llmConfigured}
        />

        {/* 索引卡片 */}
        <IndexCard
          qdrantRunning={qdrantStatus === "running"}
          embeddingConfigured={embeddingConfigured}
        />
      </div>

      {/* 底部保存按钮 */}
      {hasChanges && (
        <div className="rag-footer">
          <button
            className="rag-btn rag-btn-primary rag-btn-save"
            onClick={handleSave}
            disabled={saving}
          >
            {saving ? t("common.loading") : t("rag.save.button")}
          </button>
        </div>
      )}

      {/* Toast 容器 */}
      <ToastContainer toasts={toasts} />
    </div>
  );
};
