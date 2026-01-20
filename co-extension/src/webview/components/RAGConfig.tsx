/**
 * RAG 配置组件 - 重构版
 * 支持分步引导模式和快速编辑模式
 */

import React, { useState, useEffect, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../services/api";
import { useApi, useToast } from "../hooks";
import { ToastContainer } from "./shared";
import { WizardProgress } from "./RAGConfig/WizardProgress";
import { Step1_Embedding } from "./RAGConfig/Step1_Embedding";
import { Step1_5_LLM } from "./RAGConfig/Step1.5_LLM";
import { Step2_Qdrant } from "./RAGConfig/Step2_Qdrant";
import { Step3_Scan } from "./RAGConfig/Step3_Scan";
import { Step4_Summary } from "./RAGConfig/Step4_Summary";
import { QuickEdit } from "./RAGConfig/QuickEdit";
import { ConfigState, StepNumber, QdrantStatus } from "./RAGConfig/types";

interface RAGConfig {
  embedding_api: {
    url: string;
    model: string;
    has_api_key?: boolean;
  };
  llm_chat_api: {
    url: string;
    model: string;
    has_api_key?: boolean;
  };
  qdrant: {
    version: string;
    binary_path: string;
    data_path: string;
  };
  scan_config: {
    enabled: boolean;
    interval: string;
    batch_size: number;
    concurrency: number;
    // 高级选项
    incremental_scan?: boolean;
    max_file_size?: number;
    ignore_patterns?: string;
  };
}

interface RAGStats {
  total_indexed: number;
  last_full_scan: number;
  last_incremental_scan: number;
  scan_config: {
    enabled: boolean;
    interval: string;
    batch_size: number;
    concurrency: number;
  };
}

export const RAGConfig: React.FC = () => {
  const { t } = useTranslation();
  const { showToast, toasts } = useToast();

  // 配置状态
  const [configState, setConfigState] = useState<ConfigState>({
    mode: 'wizard',
    currentStep: 1,
    completedSteps: new Set<StepNumber>(),
    embedding: {
      url: '',
      apiKey: '',
      model: '',
    },
    llm: {
      url: '',
      apiKey: '',
      model: '',
    },
    qdrant: {
      version: '',
      binaryPath: '',
      dataPath: '',
      status: 'not-installed',
    },
    scan: {
      enabled: false,
      interval: '1h',
      batchSize: 10,
      concurrency: 3,
      // 高级选项默认值
      incrementalScan: true,
      maxFileSize: 10,
      ignorePatterns: 'node_modules/**, .git/**, .cursor/**, dist/**, build/**',
    },
  });

  const [saving, setSaving] = useState(false);
  const [stats, setStats] = useState<RAGStats | null>(null);

  // 获取配置
  const fetchConfig = useCallback(async () => {
    try {
      const response = await apiService.getRAGConfig() as RAGConfig;
      if (response) {
        // 检测是否首次访问或已有配置
        const hasConfig = response.embedding_api?.url && response.qdrant?.version;
        setConfigState(prev => ({
          ...prev,
          mode: hasConfig ? 'quick-edit' : 'wizard',
          embedding: {
            url: response.embedding_api?.url || '',
            apiKey: response.embedding_api?.has_api_key ? '••••••' : '', // 如果已配置，显示占位符
            model: response.embedding_api?.model || '',
          },
          llm: {
            url: response.llm_chat_api?.url || '',
            apiKey: response.llm_chat_api?.has_api_key ? '••••••' : '', // 如果已配置，显示占位符
            model: response.llm_chat_api?.model || '',
          },
          qdrant: {
            version: response.qdrant?.version || '',
            binaryPath: response.qdrant?.binary_path || '',
            dataPath: response.qdrant?.data_path || '',
            status: response.qdrant?.version ? 'installed' : 'not-installed',
          },
          scan: {
            enabled: response.scan_config?.enabled || false,
            interval: response.scan_config?.interval || '1h',
            batchSize: response.scan_config?.batch_size || 10,
            concurrency: response.scan_config?.concurrency || 3,
            // 高级选项
            incrementalScan: response.scan_config?.incremental_scan ?? true,
            maxFileSize: response.scan_config?.max_file_size || 10,
            ignorePatterns: response.scan_config?.ignore_patterns || 'node_modules/**, .git/**, .cursor/**',
          },
        }));
      }
    } catch (error) {
      console.error("Failed to fetch RAG config:", error);
    }
  }, []);

  const { loading: configLoading, refetch: loadConfig } = useApi(fetchConfig, { initialData: null });

  useEffect(() => {
    loadConfig();
    loadStats();
    // 定期刷新统计信息
    const interval = setInterval(() => {
      loadStats();
    }, 5000);
    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 获取统计信息
  const loadStats = useCallback(async () => {
    try {
      const response = await apiService.getRAGStats() as RAGStats;
      if (response) {
        setStats(response);
      }
    } catch (error) {
      console.error("Failed to fetch RAG stats:", error);
    }
  }, []);

  // 步骤切换
  const handleNextStep = () => {
    if (configState.currentStep < 5) {
      setConfigState(prev => ({ ...prev, currentStep: prev.currentStep + 1 as StepNumber }));
    }
  };

  const handlePreviousStep = () => {
    if (configState.currentStep > 1) {
      setConfigState(prev => ({ ...prev, currentStep: prev.currentStep - 1 as StepNumber }));
    }
  };

  const handleSkipStep = () => {
    if (configState.currentStep < 5) {
      setConfigState(prev => ({
        ...prev,
        currentStep: prev.currentStep + 1 as StepNumber,
      }));
    }
  };

  const handleStepComplete = (step: StepNumber, completed: boolean) => {
    setConfigState(prev => {
      const newCompleted = new Set(prev.completedSteps);
      if (completed) {
        newCompleted.add(step);
      } else {
        newCompleted.delete(step);
      }
      return { ...prev, completedSteps: newCompleted };
    });
  };

  const handleEmbeddingChange = (data: { url: string; apiKey: string; model: string }) => {
    setConfigState(prev => {
      // 检查是否真正发生变化，避免不必要的重新渲染
      if (
        prev.embedding.url === data.url &&
        prev.embedding.apiKey === data.apiKey &&
        prev.embedding.model === data.model
      ) {
        return prev;
      }
      return { ...prev, embedding: data };
    });
  };

  const handleLLMChange = (data: { url: string; apiKey: string; model: string }) => {
    setConfigState(prev => ({ ...prev, llm: data }));
  };

  const handleQdrantChange = (data: { version: string; binaryPath: string; dataPath: string; status: QdrantStatus }) => {
    setConfigState(prev => ({ ...prev, qdrant: data }));
  };

  const handleScanChange = (data: { 
    enabled: boolean; 
    interval: string; 
    batchSize: number; 
    concurrency: number;
    incrementalScan?: boolean;
    maxFileSize?: number;
    ignorePatterns?: string;
  }) => {
    setConfigState(prev => ({ ...prev, scan: { ...prev.scan, ...data } }));
  };

  const handleSwitchMode = () => {
    setConfigState(prev => ({ ...prev, mode: prev.mode === 'wizard' ? 'quick-edit' : 'wizard' }));
  };


  // 保存配置
  const handleSave = async () => {
    const { embedding, llm, scan } = configState;
    
    // 检查 Embedding API 配置
    if (!embedding.url || !embedding.model) {
      showToast(t("rag.config.saveRequired"), "error");
      return;
    }
    
    // 检查 LLM Chat API 配置（必需）
    if (!llm.url || !llm.apiKey || !llm.model) {
      showToast(t("rag.config.llm.saveRequired"), "error");
      return;
    }

    setSaving(true);
    try {
      const configData: any = {
        embedding_api: {
          url: embedding.url,
          model: embedding.model,
          // 如果 API Key 是占位符，说明没有修改，不发送到后端
          ...(embedding.apiKey !== '••••••' && { api_key: embedding.apiKey }),
        },
        llm_chat_api: {
          url: llm.url,
          model: llm.model,
          // 如果 API Key 是占位符，说明没有修改，不发送到后端
          ...(llm.apiKey !== '••••••' && { api_key: llm.apiKey }),
        },
        scan_config: {
          enabled: scan.enabled,
          interval: scan.interval,
          batch_size: scan.batchSize,
          concurrency: scan.concurrency,
          // 高级选项
          incremental_scan: scan.incrementalScan ?? true,
          max_file_size: scan.maxFileSize ?? 10,
          ignore_patterns: scan.ignorePatterns ?? 'node_modules/**, .git/**, .cursor/**',
        },
      };

      await apiService.updateRAGConfig(configData);

      showToast(t("rag.config.saveSuccess"), "success");
      
      // 保存成功后切换到快速编辑模式
      setConfigState(prev => ({ ...prev, mode: 'quick-edit' }));
    } catch (error) {
      showToast(t("rag.config.saveFailed") + `: ${error instanceof Error ? error.message : String(error)}`, "error");
    } finally {
      setSaving(false);
    }
  };

  // 引导模式：检查是否可以进入下一步
  const canGoNext = () => {
    switch (configState.currentStep) {
      case 1:
        return configState.completedSteps.has(1);
      case 2:
        return configState.completedSteps.has(2); // LLM 配置是必需的
      case 3:
        return configState.completedSteps.has(3);
      case 4:
        return configState.completedSteps.has(4);
      case 5:
        return true;
      default:
        return false;
    }
  };

  if (configLoading) {
    return (
      <div style={{ padding: "20px", textAlign: "center" }}>
        {t("common.loading")}
      </div>
    );
  }

  return (
    <div className="cocursor-rag-config">
      {/* 模式切换按钮 */}
      <div className="cocursor-rag-config-header">
        <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
          <h2>{t("rag.config.title")}</h2>
          <span
            className="cocursor-beta-badge"
            title={t("rag.betaTooltip")}
          >
            {t("rag.beta")}
          </span>
        </div>
        {configState.mode === 'quick-edit' && (
          <button
            type="button"
            className="cocursor-rag-config-mode-switch"
            onClick={handleSwitchMode}
          >
            📋 {t("rag.config.wizard.title")}
          </button>
        )}
      </div>

      {/* 引导模式 */}
      {configState.mode === 'wizard' && (
        <>
          <WizardProgress
            currentStep={configState.currentStep}
            completedSteps={configState.completedSteps}
          />

          {/* 步骤 1: Embedding 配置 */}
          {configState.currentStep === 1 && (
            <Step1_Embedding
              embedding={configState.embedding}
              onChange={handleEmbeddingChange}
              onStepComplete={(completed: boolean) => handleStepComplete(1, completed)}
              // 只在有完整配置时才启用自动测试模式
              autoAdvance={!!configState.embedding.url && !!configState.embedding.apiKey && !!configState.embedding.model}
            />
          )}

          {/* 步骤 1.5: LLM 配置 */}
          {configState.currentStep === 2 && (
            <Step1_5_LLM
              llm={configState.llm}
              onChange={handleLLMChange}
              onStepComplete={(completed: boolean) => handleStepComplete(2, completed)}
              autoAdvance={!!configState.llm.url && !!configState.llm.apiKey && !!configState.llm.model}
            />
          )}

          {/* 步骤 2: Qdrant 状态 */}
          {configState.currentStep === 3 && (
            <Step2_Qdrant
              qdrant={configState.qdrant}
              onChange={handleQdrantChange}
              onStepComplete={(completed: boolean) => handleStepComplete(3, completed)}
              onDownloadSuccess={loadConfig}
            />
          )}

          {/* 步骤 3: 扫描配置 */}
          {configState.currentStep === 4 && (
            <Step3_Scan
              scan={configState.scan}
              onChange={handleScanChange}
              onStepComplete={(completed: boolean) => handleStepComplete(4, completed)}
            />
          )}

          {/* 步骤 4: 配置确认 */}
          {configState.currentStep === 5 && (
            <Step4_Summary
              embedding={configState.embedding}
              llm={configState.llm}
              qdrant={configState.qdrant}
              scan={configState.scan}
              onSave={handleSave}
            />
          )}

          {/* 底部导航 */}
          <div className="cocursor-rag-config-footer">
            <button
              type="button"
              className="cocursor-rag-config-button secondary"
              onClick={handlePreviousStep}
              disabled={configState.currentStep === 1}
            >
              {t("rag.config.wizard.previous")}
            </button>

            {configState.currentStep < 5 && (
              <button
                type="button"
                className="cocursor-rag-config-button secondary"
                onClick={handleSkipStep}
                disabled={configState.currentStep === 1 && !configState.completedSteps.has(1)}
              >
                {t("rag.config.wizard.skip")}
              </button>
            )}

            <button
              type="button"
              className={`cocursor-rag-config-button primary ${configState.currentStep === 5 ? 'final' : ''}`}
              onClick={configState.currentStep === 4 ? handleSave : handleNextStep}
              disabled={!canGoNext()}
            >
              {configState.currentStep === 4
                ? t("rag.config.wizard.saveAndEnable")
                : t("rag.config.wizard.next")}
            </button>
          </div>
        </>
      )}

      {/* 快速编辑模式 */}
      {configState.mode === 'quick-edit' && (
        <QuickEdit
          embedding={configState.embedding}
          llm={configState.llm}
          qdrant={configState.qdrant}
          scan={configState.scan}
          onSwitchToWizard={handleSwitchMode}
          onSave={handleSave}
          stats={stats}
        />
      )}

      {/* Toast 容器 */}
      <ToastContainer toasts={toasts} />
    </div>
  );
};
