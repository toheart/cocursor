// API 服务 - 通过 Extension 代理调用后端 API

import { ExtensionMessage } from "../../types/message";

// VS Code API 单例管理器（每个 webview 实例只能获取一次）
let vscodeApiInstance: ReturnType<typeof acquireVsCodeApi> | null = null;

export function getVscodeApi(): ReturnType<typeof acquireVsCodeApi> {
  if (!vscodeApiInstance) {
    vscodeApiInstance = acquireVsCodeApi();
  }
  return vscodeApiInstance;
}

class ApiService {
  private vscode: ReturnType<typeof acquireVsCodeApi>;

  constructor() {
    this.vscode = getVscodeApi();
  }

  // 发送消息到 Extension
  private postMessage(command: string, payload?: unknown): Promise<unknown> {
    return new Promise((resolve, reject) => {
      const messageId = `${command}-${Date.now()}-${Math.random()}`;
      const handler = (event: MessageEvent<ExtensionMessage>) => {
        if (event.data.type === `${command}-response`) {
          window.removeEventListener("message", handler);
          if (event.data.data && typeof event.data.data === "object" && "error" in event.data.data) {
            reject(new Error(String(event.data.data.error)));
          } else {
            resolve(event.data.data);
          }
        }
      };

      window.addEventListener("message", handler);
      this.vscode.postMessage({ command, payload, messageId });

      // 超时处理
      setTimeout(() => {
        window.removeEventListener("message", handler);
        reject(new Error("Request timeout"));
      }, 30000);
    });
  }

  // 获取对话列表
  async getChats(): Promise<unknown> {
    return this.postMessage("fetchChats");
  }

  // 获取对话详情
  async getChatDetail(chatId: string): Promise<unknown> {
    return this.postMessage("fetchChatDetail", { chatId });
  }

  // 获取节点列表
  async getPeers(): Promise<unknown> {
    return this.postMessage("getPeers");
  }

  // 加入团队
  async joinTeam(teamCode: string): Promise<unknown> {
    return this.postMessage("joinTeam", { teamCode });
  }

  // 获取当前会话健康状态
  async getCurrentSessionHealth(projectPath?: string): Promise<SessionHealth> {
    return this.postMessage("fetchCurrentSessionHealth", { projectPath }) as Promise<SessionHealth>;
  }

  // 获取工作分析数据（全局视图）
  async getWorkAnalysis(startDate?: string, endDate?: string): Promise<unknown> {
    return this.postMessage("fetchWorkAnalysis", { startDate, endDate });
  }

  // 获取会话列表
  async getSessionList(projectName?: string, limit?: number, offset?: number, search?: string): Promise<unknown> {
    return this.postMessage("fetchSessionList", { projectName, limit, offset, search });
  }

  // 获取会话详情
  async getSessionDetail(sessionId: string, limit?: number): Promise<unknown> {
    return this.postMessage("fetchSessionDetail", { sessionId, limit });
  }

  // 获取插件列表
  async getPlugins(category?: string, search?: string, installed?: boolean): Promise<unknown> {
    return this.postMessage("fetchPlugins", { category, search, installed });
  }

  // 获取插件详情
  async getPlugin(id: string): Promise<unknown> {
    return this.postMessage("fetchPlugin", { id });
  }

  // 获取已安装插件列表
  async getInstalledPlugins(): Promise<unknown> {
    return this.postMessage("fetchInstalledPlugins");
  }

  // 安装插件
  async installPlugin(id: string, workspacePath: string): Promise<unknown> {
    return this.postMessage("installPlugin", { id, workspacePath });
  }

  // 卸载插件
  async uninstallPlugin(id: string, workspacePath: string): Promise<unknown> {
    return this.postMessage("uninstallPlugin", { id, workspacePath });
  }

  // 检查插件状态
  async checkPluginStatus(id: string): Promise<unknown> {
    return this.postMessage("checkPluginStatus", { id });
  }

  // 获取工作流列表
  async getWorkflows(projectPath?: string, status?: string): Promise<unknown> {
    return this.postMessage("fetchWorkflows", { projectPath, status });
  }

  // 获取工作流详情
  async getWorkflowDetail(changeId: string, projectPath?: string): Promise<unknown> {
    return this.postMessage("fetchWorkflowDetail", { changeId, projectPath });
  }

  // ========== RAG 相关 API ==========
  
  // 获取 RAG 配置
  async getRAGConfig(): Promise<unknown> {
    return this.postMessage("fetchRAGConfig");
  }

  // 更新 RAG 配置
  async updateRAGConfig(config: {
    embedding_api: { url: string; api_key: string; model: string };
    scan_config: { enabled: boolean; interval: string; batch_size: number; concurrency: number };
  }): Promise<unknown> {
    return this.postMessage("updateRAGConfig", { config });
  }

  // 测试 RAG 配置连接
  async testRAGConfig(config: { url: string; api_key: string; model: string }): Promise<unknown> {
    return this.postMessage("testRAGConfig", { config });
  }

  // RAG 语义搜索
  async searchRAG(query: string, projectIds?: string[], limit?: number): Promise<unknown> {
    return this.postMessage("searchRAG", { query, projectIds, limit });
  }

  // 触发 RAG 索引
  async triggerRAGIndex(sessionId?: string): Promise<unknown> {
    return this.postMessage("triggerRAGIndex", { sessionId });
  }

  // 获取 RAG 统计信息
  async getRAGStats(): Promise<unknown> {
    return this.postMessage("fetchRAGStats");
  }

  // 下载 Qdrant
  async downloadQdrant(version?: string): Promise<unknown> {
    return this.postMessage("downloadQdrant", { version });
  }
}

// 会话健康状态类型
export interface SessionHealth {
  entropy: number;
  status: "healthy" | "sub_healthy" | "dangerous";
  warning?: string;
}

export const apiService = new ApiService();
