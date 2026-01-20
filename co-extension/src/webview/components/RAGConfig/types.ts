/**
 * RAG 配置类型定义
 */

// 配置模式
export type ConfigMode = 'wizard' | 'quick-edit';

// Qdrant 状态
export type QdrantStatus = 'installed' | 'not-installed' | 'running' | 'stopped';

// 步骤类型（4步：Embedding → LLM(可选) → Qdrant → 索引参数）
export type StepNumber = 1 | 2 | 3 | 4;

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
  // 索引配置（仅用于全量索引）
  index: {
    batchSize: number;
    concurrency: number;
  };
  // 保留 scan 字段以兼容旧配置，但不再使用
  scan?: {
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
// 注意：URL 应该是基础地址，后端会自动处理 /v1/embeddings 路径
export const CONFIG_TEMPLATES: Record<ConfigTemplate, ConfigTemplateData> = {
  openai: {
    name: 'rag.config.template.openai',
    url: 'https://api.openai.com',  // 基础地址，不含 /v1
    model: 'text-embedding-3-small', // 更新为新版模型，性能更好且更便宜
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
