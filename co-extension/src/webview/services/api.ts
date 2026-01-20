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
  // timeout: 超时时间（毫秒），0 表示不超时
  private postMessage(command: string, payload?: unknown, timeout: number = 30000): Promise<unknown> {
    return new Promise((resolve, reject) => {
      const messageId = `${command}-${Date.now()}-${Math.random()}`;
      let timeoutId: ReturnType<typeof setTimeout> | null = null;
      
      const handler = (event: MessageEvent<ExtensionMessage>) => {
        if (event.data.type === `${command}-response`) {
          window.removeEventListener("message", handler);
          // 清除超时计时器
          if (timeoutId) {
            clearTimeout(timeoutId);
          }
          if (event.data.data && typeof event.data.data === "object" && "error" in event.data.data) {
            reject(new Error(String(event.data.data.error)));
          } else {
            resolve(event.data.data);
          }
        }
      };

      window.addEventListener("message", handler);
      this.vscode.postMessage({ command, payload, messageId });

      // 超时处理（timeout > 0 时启用）
      if (timeout > 0) {
        timeoutId = setTimeout(() => {
          window.removeEventListener("message", handler);
          reject(new Error("Request timeout"));
        }, timeout);
      }
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
  // force: 强制覆盖（当检测到手动安装的同名 skill 时）
  async installPlugin(id: string, workspacePath: string, force?: boolean): Promise<unknown> {
    return this.postMessage("installPlugin", { id, workspacePath, force: force || false });
  }

  // 显示确认对话框（通过 VSCode API）
  async showConfirmDialog(message: string, confirmText?: string, cancelText?: string): Promise<boolean> {
    return this.postMessage("showConfirmDialog", { message, confirmText, cancelText }) as Promise<boolean>;
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

  // 上传 Qdrant 安装包
  // 注意：此方法接收 base64 编码的文件内容，因为 webview 无法直接传输 File 对象
  async uploadQdrantPackage(filename: string, fileBase64: string): Promise<unknown> {
    // 使用较长超时（2分钟），因为上传可能需要一些时间
    return this.postMessage("uploadQdrantPackage", { filename, fileBase64 }, 120000);
  }

  // 启动 Qdrant
  async startQdrant(): Promise<unknown> {
    return this.postMessage("startQdrant");
  }

  // 停止 Qdrant
  async stopQdrant(): Promise<unknown> {
    return this.postMessage("stopQdrant");
  }

  // 触发全量索引（支持配置参数）
  async triggerFullIndex(batchSize?: number, concurrency?: number): Promise<unknown> {
    return this.postMessage("triggerFullIndex", { batch_size: batchSize, concurrency });
  }

  // 获取索引进度
  async getIndexProgress(): Promise<unknown> {
    return this.postMessage("fetchIndexProgress");
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

  // 获取已索引的项目列表
  async getIndexedProjects(): Promise<{ projects: Array<{ project_id: string; project_name: string; chunk_count: number }>; total: number }> {
    return this.postMessage("fetchIndexedProjects") as Promise<{ projects: Array<{ project_id: string; project_name: string; chunk_count: number }>; total: number }>;
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

  // 发布技能到团队（旧版，向后兼容）
  async publishTeamSkill(teamId: string, pluginId: string, localPath: string): Promise<unknown> {
    return this.postMessage("publishTeamSkill", { teamId, pluginId, localPath });
  }

  // 发布技能到团队（带元数据）
  async publishTeamSkillWithMetadata(teamId: string, localPath: string, metadata: {
    plugin_id: string;
    name: string;
    name_zh_cn?: string;
    description: string;
    description_zh_cn?: string;
    version: string;
    category: string;
    author: string;
  }): Promise<unknown> {
    return this.postMessage("publishTeamSkillWithMetadata", { teamId, localPath, metadata });
  }

  // 安装团队技能
  async installTeamSkill(teamId: string, pluginId: string, version?: string, force?: boolean): Promise<unknown> {
    return this.postMessage("installTeamSkill", { teamId, pluginId, version, force });
  }

  // 卸载团队技能
  async uninstallTeamSkill(teamId: string, pluginId: string): Promise<unknown> {
    return this.postMessage("uninstallTeamSkill", { teamId, pluginId });
  }

  // 下载团队技能
  async downloadTeamSkill(teamId: string, pluginId: string, authorEndpoint: string, checksum?: string): Promise<unknown> {
    return this.postMessage("downloadTeamSkill", { teamId, pluginId, authorEndpoint, checksum });
  }

  // 选择目录（调用 VSCode 文件选择器）
  // 用户交互操作，使用较长超时（5分钟）
  async selectDirectory(): Promise<unknown> {
    return this.postMessage("selectDirectory", undefined, 300000);
  }

  // ========== 团队协作 API ==========

  // 分享代码片段
  async shareCode(teamId: string, snippet: {
    file_name: string;
    file_path?: string;
    language?: string;
    start_line?: number;
    end_line?: number;
    code: string;
    message?: string;
  }): Promise<unknown> {
    return this.postMessage("shareCode", { teamId, ...snippet });
  }

  // 更新工作状态
  async updateWorkStatus(teamId: string, status: {
    project_name?: string;
    current_file?: string;
    status_visible?: boolean;
  }): Promise<unknown> {
    return this.postMessage("updateWorkStatus", { teamId, ...status });
  }

  // 分享日报到团队
  async shareTeamDailySummary(teamId: string, date: string): Promise<unknown> {
    return this.postMessage("shareTeamDailySummary", { teamId, date });
  }

  // 获取团队日报列表
  async getTeamDailySummaries(teamId: string, date?: string): Promise<unknown> {
    return this.postMessage("fetchTeamDailySummaries", { teamId, date });
  }

  // 获取团队日报详情
  async getTeamDailySummaryDetail(teamId: string, memberId: string, date: string): Promise<unknown> {
    return this.postMessage("fetchTeamDailySummaryDetail", { teamId, memberId, date });
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

// 项目摘要类型
export interface ProjectSummary {
  project_name: string;
  project_path: string;
  workspace_id: string;
  session_count: number;
}

// 日报类型
export interface DailySummary {
  id: string;
  date: string;
  summary: string;
  language: string;
  total_sessions: number;
  projects?: ProjectSummary[];
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
