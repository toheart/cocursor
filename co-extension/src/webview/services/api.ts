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
  async getPlugins(category?: string, search?: string, installed?: boolean, lang?: string, source?: string, teamId?: string): Promise<unknown> {
    return this.postMessage("fetchPlugins", { category, search, installed, lang, source, team_id: teamId });
  }

  // 获取插件详情
  async getPlugin(id: string, lang?: string): Promise<unknown> {
    return this.postMessage("fetchPlugin", { id, lang });
  }

  // 获取已安装插件列表
  async getInstalledPlugins(lang?: string): Promise<unknown> {
    return this.postMessage("fetchInstalledPlugins", { lang });
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
    embedding_api?: { url: string; api_key: string; model: string };
    llm_chat_api?: { url: string; api_key: string; model: string; language: string };
    scan_config?: { enabled: boolean; interval: string; batch_size: number; concurrency: number };
  }): Promise<unknown> {
    return this.postMessage("updateRAGConfig", { config });
  }

  // 测试 RAG 配置连接
  async testRAGConfig(config: { url: string; api_key: string; model: string }): Promise<unknown> {
    return this.postMessage("testRAGConfig", { config });
  }

  // 测试 LLM 连接
  async testLLMConnection(config: { url: string; api_key: string; model: string }): Promise<{ success: boolean; error?: string }> {
    return this.postMessage("testLLMConnection", { config }) as any;
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

  // 获取 Qdrant 状态
  async getQdrantStatus(): Promise<unknown> {
    return this.postMessage("fetchQdrantStatus");
  }

  // 启动 Qdrant
  async startQdrant(): Promise<unknown> {
    return this.postMessage("startQdrant");
  }

  // 停止 Qdrant
  async stopQdrant(): Promise<unknown> {
    return this.postMessage("stopQdrant");
  }

  // 触发全量索引
  async triggerFullIndex(): Promise<unknown> {
    return this.postMessage("triggerFullIndex");
  }

  // 清空所有数据
  async clearAllData(): Promise<unknown> {
    return this.postMessage("clearAllData");
  }

  // ========== 新 RAG API（使用 KnowledgeChunk） ==========

  // 搜索知识片段
  async searchChunks(query: string, projectIds?: string[], limit?: number): Promise<unknown> {
    return this.postMessage("searchRAGChunks", { query, projectIds, limit });
  }

  // 获取知识片段详情
  async getChunkDetail(chunkId: string): Promise<unknown> {
    return this.postMessage("fetchChunkDetail", { chunkId });
  }

  // 获取增强队列统计
  async getEnrichmentStats(): Promise<unknown> {
    return this.postMessage("fetchEnrichmentStats");
  }

  // 重试失败的增强任务
  async retryEnrichment(): Promise<unknown> {
    return this.postMessage("retryEnrichment");
  }

  // 获取索引统计（新）
  async getIndexStats(): Promise<unknown> {
    return this.postMessage("fetchIndexStats");
  }

  // ========== 团队相关 API ==========

  // 获取本机身份
  async getTeamIdentity(): Promise<unknown> {
    return this.postMessage("fetchTeamIdentity");
  }

  // 创建或更新身份
  async setTeamIdentity(name: string): Promise<unknown> {
    return this.postMessage("setTeamIdentity", { name });
  }

  // 获取网卡列表
  async getNetworkInterfaces(): Promise<unknown> {
    return this.postMessage("fetchNetworkInterfaces");
  }

  // 创建团队
  async createTeam(name: string, preferredInterface?: string, preferredIP?: string): Promise<unknown> {
    return this.postMessage("createTeam", { name, preferred_interface: preferredInterface, preferred_ip: preferredIP });
  }

  // 发现团队
  async discoverTeams(timeout?: number): Promise<unknown> {
    return this.postMessage("discoverTeams", { timeout });
  }

  // 加入团队
  async joinTeam(endpoint: string): Promise<unknown> {
    return this.postMessage("joinTeam", { endpoint });
  }

  // 获取已加入团队列表
  async getTeamList(): Promise<unknown> {
    return this.postMessage("fetchTeamList");
  }

  // 获取团队成员列表
  async getTeamMembers(teamId: string): Promise<unknown> {
    return this.postMessage("fetchTeamMembers", { teamId });
  }

  // 离开团队
  async leaveTeam(teamId: string): Promise<unknown> {
    return this.postMessage("leaveTeam", { teamId });
  }

  // 解散团队
  async dissolveTeam(teamId: string): Promise<unknown> {
    return this.postMessage("dissolveTeam", { teamId });
  }

  // 获取团队技能目录
  async getTeamSkillIndex(teamId: string): Promise<unknown> {
    return this.postMessage("fetchTeamSkillIndex", { teamId });
  }

  // 验证技能目录
  async validateSkillDirectory(path: string): Promise<unknown> {
    return this.postMessage("validateSkillDirectory", { path });
  }

  // 发布技能到团队
  async publishTeamSkill(teamId: string, pluginId: string, localPath: string): Promise<unknown> {
    return this.postMessage("publishTeamSkill", { teamId, pluginId, localPath });
  }

  // 下载团队技能
  async downloadTeamSkill(teamId: string, pluginId: string, authorEndpoint: string, checksum?: string): Promise<unknown> {
    return this.postMessage("downloadTeamSkill", { teamId, pluginId, authorEndpoint, checksum });
  }

  // 选择目录（调用 VSCode 文件选择器）
  async selectDirectory(): Promise<unknown> {
    return this.postMessage("selectDirectory");
  }

  // ========== 日报相关 API ==========

  // 获取日报批量状态
  async getDailyReportStatus(startDate: string, endDate: string): Promise<DailyReportStatusResponse> {
    return this.postMessage("fetchDailyReportStatus", { startDate, endDate }) as Promise<DailyReportStatusResponse>;
  }

  // 获取指定日期的日报
  async getDailySummary(date: string): Promise<DailySummary | null> {
    return this.postMessage("fetchDailySummary", { date }) as Promise<DailySummary | null>;
  }
}

// 日报状态响应类型
export interface DailyReportStatusResponse {
  statuses: Record<string, boolean>;
}

// 日报类型
export interface DailySummary {
  id: string;
  date: string;
  summary: string;
  language: string;
  total_sessions: number;
  created_at?: string;
  updated_at?: string;
}

// 会话健康状态类型
export interface SessionHealth {
  entropy: number;
  status: "healthy" | "sub_healthy" | "dangerous";
  warning?: string;
}

export const apiService = new ApiService();
