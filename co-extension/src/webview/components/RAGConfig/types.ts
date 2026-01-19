/**
 * RAG 配置类型定义
 */

// 配置模式
export type ConfigMode = 'wizard' | 'quick-edit';

// Qdrant 状态
export type QdrantStatus = 'installed' | 'not-installed' | 'running' | 'stopped';

// 步骤类型
export type StepNumber = 1 | 2 | 3 | 4 | 5; // 新增步骤 5

// 配置状态
export interface ConfigState {
  mode: ConfigMode;
  currentStep: StepNumber;
  completedSteps: Set<StepNumber>;
  embedding: {
    url: string;
    apiKey: string;
    model: string;
  };
  llm: {
    url: string;
    apiKey: string;
    model: string;
  };
  qdrant: {
    version: string;
    binaryPath: string;
    dataPath: string;
    status: QdrantStatus;
  };
  scan: {
    enabled: boolean;
    interval: string;
    batchSize: number;
    concurrency: number;
  };
}

// 配置模板类型
export type ConfigTemplate = 'openai' | 'azure' | 'custom';

// 配置模板数据
export interface ConfigTemplateData {
  name: string;
  url: string;
  model: string;
  additionalFields?: Record<string, any>;
}

// 预定义模板
export const CONFIG_TEMPLATES: Record<ConfigTemplate, ConfigTemplateData> = {
  openai: {
    name: 'rag.config.template.openai',
    url: 'https://api.openai.com/v1',
    model: 'text-embedding-ada-002',
  },
  azure: {
    name: 'rag.config.template.azure',
    url: 'https://your-resource.openai.azure.com',
    model: 'text-embedding-ada-002',
    additionalFields: {
      deploymentName: '',
    },
  },
  custom: {
    name: 'rag.config.template.custom',
    url: '',
    model: '',
  },
};
