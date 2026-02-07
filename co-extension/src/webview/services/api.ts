// API 服务 - 通过 Extension 代理调用后端 API

import { ExtensionMessage } from "../../types/message";
import type {
  TeamProjectConfig,
  ProjectMatcher,
  TeamWeeklyView,
  MemberDailyDetail,
  TeamMemberSummariesView,
} from "../types";

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
  private postMessage(
    command: string,
    payload?: unknown,
    timeout: number = 30000,
  ): Promise<unknown> {
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
          if (
            event.data.data &&
            typeof event.data.data === "object" &&
            "error" in event.data.data
          ) {
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
    return this.postMessage("fetchCurrentSessionHealth", {
      projectPath,
    }) as Promise<SessionHealth>;
  }

  // 获取工作分析数据（全局视图）
  async getWorkAnalysis(
    startDate?: string,
    endDate?: string,
  ): Promise<unknown> {
    return this.postMessage("fetchWorkAnalysis", { startDate, endDate });
  }

  // 获取活跃会话概览
  async getActiveSessions(workspaceId?: string): Promise<unknown> {
    return this.postMessage("fetchActiveSessions", { workspaceId });
  }

  // 获取会话列表
  async getSessionList(
    projectName?: string,
    limit?: number,
    offset?: number,
    search?: string,
  ): Promise<unknown> {
    return this.postMessage("fetchSessionList", {
      projectName,
      limit,
      offset,
      search,
    });
  }

  // 获取会话详情
  async getSessionDetail(sessionId: string, limit?: number): Promise<unknown> {
    return this.postMessage("fetchSessionDetail", { sessionId, limit });
  }

  // 获取插件列表
  async getPlugins(
    category?: string,
    search?: string,
    installed?: boolean,
    lang?: string,
    source?: string,
    teamId?: string,
  ): Promise<unknown> {
    return this.postMessage("fetchPlugins", {
      category,
      search,
      installed,
      lang,
      source,
      team_id: teamId,
    });
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
  async installPlugin(
    id: string,
    workspacePath: string,
    force?: boolean,
  ): Promise<unknown> {
    return this.postMessage("installPlugin", {
      id,
      workspacePath,
      force: force || false,
    });
  }

  // 显示确认对话框（通过 VSCode API）
  async showConfirmDialog(
    message: string,
    confirmText?: string,
    cancelText?: string,
  ): Promise<boolean> {
    return this.postMessage("showConfirmDialog", {
      message,
      confirmText,
      cancelText,
    }) as Promise<boolean>;
  }

  // 卸载插件
  async uninstallPlugin(id: string, workspacePath: string): Promise<unknown> {
    return this.postMessage("uninstallPlugin", { id, workspacePath });
  }

  // 检查插件状态
  async checkPluginStatus(id: string): Promise<unknown> {
    return this.postMessage("checkPluginStatus", { id });
  }

  // ========== RAG 相关 API ==========

  // 获取 RAG 配置
  async getRAGConfig(): Promise<unknown> {
    return this.postMessage("fetchRAGConfig");
  }

  // 更新 RAG 配置
  async updateRAGConfig(config: {
    embedding_api?: { url: string; api_key: string; model: string };
    llm_chat_api?: {
      url: string;
      api_key: string;
      model: string;
      language: string;
    };
    scan_config?: {
      enabled: boolean;
      interval: string;
      batch_size: number;
      concurrency: number;
    };
  }): Promise<unknown> {
    return this.postMessage("updateRAGConfig", { config });
  }

  // 测试 RAG 配置连接
  async testRAGConfig(config: {
    url: string;
    api_key: string;
    model: string;
  }): Promise<unknown> {
    return this.postMessage("testRAGConfig", { config });
  }

  // 测试 LLM 连接
  async testLLMConnection(config: {
    url: string;
    api_key: string;
    model: string;
  }): Promise<{ success: boolean; error?: string }> {
    return this.postMessage("testLLMConnection", { config }) as any;
  }

  // RAG 语义搜索
  async searchRAG(
    query: string,
    projectIds?: string[],
    limit?: number,
  ): Promise<unknown> {
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
  async uploadQdrantPackage(
    filename: string,
    fileBase64: string,
  ): Promise<unknown> {
    // 使用较长超时（2分钟），因为上传可能需要一些时间
    return this.postMessage(
      "uploadQdrantPackage",
      { filename, fileBase64 },
      120000,
    );
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
  async triggerFullIndex(
    batchSize?: number,
    concurrency?: number,
  ): Promise<unknown> {
    return this.postMessage("triggerFullIndex", {
      batch_size: batchSize,
      concurrency,
    });
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
  async searchChunks(
    query: string,
    projectIds?: string[],
    limit?: number,
  ): Promise<unknown> {
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
  async getIndexedProjects(): Promise<{
    projects: Array<{
      project_id: string;
      project_name: string;
      chunk_count: number;
    }>;
    total: number;
  }> {
    return this.postMessage("fetchIndexedProjects") as Promise<{
      projects: Array<{
        project_id: string;
        project_name: string;
        chunk_count: number;
      }>;
      total: number;
    }>;
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

  // 更新网络配置
  async updateNetworkConfig(
    preferredInterface: string,
    preferredIP: string,
  ): Promise<unknown> {
    return this.postMessage("updateNetworkConfig", {
      preferred_interface: preferredInterface,
      preferred_ip: preferredIP,
    });
  }

  // 创建团队
  async createTeam(
    name: string,
    preferredInterface?: string,
    preferredIP?: string,
  ): Promise<unknown> {
    return this.postMessage("createTeam", {
      name,
      preferred_interface: preferredInterface,
      preferred_ip: preferredIP,
    });
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
  async publishTeamSkill(
    teamId: string,
    pluginId: string,
    localPath: string,
  ): Promise<unknown> {
    return this.postMessage("publishTeamSkill", {
      teamId,
      pluginId,
      localPath,
    });
  }

  // 发布技能到团队（带元数据）
  async publishTeamSkillWithMetadata(
    teamId: string,
    localPath: string,
    metadata: {
      plugin_id: string;
      name: string;
      name_zh_cn?: string;
      description: string;
      description_zh_cn?: string;
      version: string;
      category: string;
      author: string;
    },
  ): Promise<unknown> {
    return this.postMessage("publishTeamSkillWithMetadata", {
      teamId,
      localPath,
      metadata,
    });
  }

  // 安装团队技能
  async installTeamSkill(
    teamId: string,
    pluginId: string,
    version?: string,
    force?: boolean,
  ): Promise<unknown> {
    return this.postMessage("installTeamSkill", {
      teamId,
      pluginId,
      version,
      force,
    });
  }

  // 卸载团队技能
  async uninstallTeamSkill(teamId: string, pluginId: string): Promise<unknown> {
    return this.postMessage("uninstallTeamSkill", { teamId, pluginId });
  }

  // 下载团队技能
  async downloadTeamSkill(
    teamId: string,
    pluginId: string,
    authorEndpoint: string,
    checksum?: string,
  ): Promise<unknown> {
    return this.postMessage("downloadTeamSkill", {
      teamId,
      pluginId,
      authorEndpoint,
      checksum,
    });
  }

  // 选择目录（调用 VSCode 文件选择器）
  // 用户交互操作，使用较长超时（5分钟）
  async selectDirectory(): Promise<unknown> {
    return this.postMessage("selectDirectory", undefined, 300000);
  }

  // ========== 团队协作 API ==========

  // 更新工作状态
  async updateWorkStatus(
    teamId: string,
    status: {
      project_name?: string;
      current_file?: string;
      status_visible?: boolean;
    },
  ): Promise<unknown> {
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
  async getTeamDailySummaryDetail(
    teamId: string,
    memberId: string,
    date: string,
  ): Promise<unknown> {
    return this.postMessage("fetchTeamDailySummaryDetail", {
      teamId,
      memberId,
      date,
    });
  }

  // ========== 日报相关 API ==========

  // 获取日报批量状态
  async getDailyReportStatus(
    startDate: string,
    endDate: string,
  ): Promise<DailyReportStatusResponse> {
    return this.postMessage("fetchDailyReportStatus", {
      startDate,
      endDate,
    }) as Promise<DailyReportStatusResponse>;
  }

  // 获取指定日期的日报
  async getDailySummary(date: string): Promise<DailySummary | null> {
    return this.postMessage("fetchDailySummary", {
      date,
    }) as Promise<DailySummary | null>;
  }

  // ========== 团队周报相关 API ==========

  // 获取团队项目配置
  async getTeamProjectConfig(teamId: string): Promise<TeamProjectConfig> {
    return this.postMessage("fetchTeamProjectConfig", {
      teamId,
    }) as Promise<TeamProjectConfig>;
  }

  // 更新团队项目配置
  async updateTeamProjectConfig(
    teamId: string,
    projects: ProjectMatcher[],
  ): Promise<unknown> {
    return this.postMessage("updateTeamProjectConfig", { teamId, projects });
  }

  // 添加团队项目
  async addTeamProject(
    teamId: string,
    project: { name: string; repo_url: string },
  ): Promise<ProjectMatcher> {
    return this.postMessage("addTeamProject", {
      teamId,
      ...project,
    }) as Promise<ProjectMatcher>;
  }

  // 移除团队项目
  async removeTeamProject(teamId: string, projectId: string): Promise<unknown> {
    return this.postMessage("removeTeamProject", { teamId, projectId });
  }

  // 获取团队周报
  // 周报需要收集所有成员数据，使用较长超时（90秒）
  async getTeamWeeklyReport(
    teamId: string,
    weekStart: string,
  ): Promise<TeamWeeklyView> {
    return this.postMessage(
      "fetchTeamWeeklyReport",
      { teamId, weekStart },
      90000,
    ) as Promise<TeamWeeklyView>;
  }

  // 获取成员日详情
  async getMemberDailyDetail(
    teamId: string,
    memberId: string,
    date: string,
  ): Promise<MemberDailyDetail> {
    return this.postMessage("fetchMemberDailyDetail", {
      teamId,
      memberId,
      date,
    }) as Promise<MemberDailyDetail>;
  }

  // 获取团队成员周报汇总
  // 需要从各成员 P2P 拉取，使用较长超时（90秒）
  async getTeamMemberSummaries(
    teamId: string,
    weekStart: string,
  ): Promise<TeamMemberSummariesView> {
    return this.postMessage(
      "fetchTeamMemberSummaries",
      { teamId, weekStart },
      90000,
    ) as Promise<TeamMemberSummariesView>;
  }

  // 刷新团队周统计
  async refreshTeamWeeklyStats(
    teamId: string,
    weekStart: string,
  ): Promise<unknown> {
    return this.postMessage("refreshTeamWeeklyStats", { teamId, weekStart });
  }

  // 选择文件夹（调用 VSCode 文件选择对话框）
  async selectFolder(): Promise<{ path: string } | null> {
    return this.postMessage("selectFolder", {}) as Promise<{
      path: string;
    } | null>;
  }

  // 通过路径添加项目（自动读取 Git 信息）
  async addTeamProjectByPath(
    teamId: string,
    path: string,
    name?: string,
  ): Promise<ProjectMatcher> {
    return this.postMessage("addTeamProjectByPath", {
      teamId,
      path,
      name,
    }) as Promise<ProjectMatcher>;
  }

  // ========== 会话分享相关 API ==========

  // 获取共享会话列表
  async getSharedSessions(
    teamId: string,
    page?: number,
    pageSize?: number,
  ): Promise<unknown> {
    return this.postMessage("fetchSharedSessions", {
      teamId,
      page: page || 1,
      page_size: pageSize || 20,
    });
  }

  // 获取共享会话详情
  async getSharedSessionDetail(
    teamId: string,
    shareId: string,
  ): Promise<unknown> {
    return this.postMessage("fetchSharedSessionDetail", { teamId, shareId });
  }

  // 添加会话评论
  async addSessionComment(
    teamId: string,
    shareId: string,
    comment: { content: string; mentions?: string[] },
  ): Promise<unknown> {
    return this.postMessage("addSessionComment", {
      teamId,
      shareId,
      ...comment,
    });
  }

  // ========== 代码分析相关 API ==========

  // 扫描入口函数
  async scanEntryPoints(projectPath: string): Promise<ScanEntryPointsResponse> {
    return this.postMessage("scanEntryPoints", {
      project_path: projectPath,
    }) as Promise<ScanEntryPointsResponse>;
  }

  // 注册项目
  async registerProject(request: RegisterProjectRequest): Promise<unknown> {
    return this.postMessage("registerProject", request);
  }

  // 获取项目配置
  async getProjectConfig(projectPath: string): Promise<ProjectConfig> {
    return this.postMessage("getProjectConfig", {
      project_path: projectPath,
    }) as Promise<ProjectConfig>;
  }

  // 检查调用图状态
  async checkCallGraphStatus(
    projectPath: string,
    commit?: string,
  ): Promise<CallGraphStatus> {
    return this.postMessage("checkCallGraphStatus", {
      project_path: projectPath,
      commit,
    }) as Promise<CallGraphStatus>;
  }

  // 生成调用图（同步）
  async generateCallGraph(
    projectPath: string,
    commit?: string,
  ): Promise<GenerateResponse> {
    // 生成调用图可能需要较长时间，设置 5 分钟超时
    return this.postMessage(
      "generateCallGraph",
      { project_path: projectPath, commit },
      300000,
    ) as Promise<GenerateResponse>;
  }

  // 生成调用图（异步）
  async generateCallGraphAsync(
    projectPath: string,
    commit?: string,
  ): Promise<{ task_id: string; status: string }> {
    return this.postMessage("generateCallGraphAsync", {
      project_path: projectPath,
      commit,
    }) as Promise<{ task_id: string; status: string }>;
  }

  // 生成调用图（异步，包含配置）
  async generateCallGraphWithConfig(
    request: GenerateWithConfigRequest,
  ): Promise<{ task_id: string; status: string }> {
    return this.postMessage("generateCallGraphWithConfig", request) as Promise<{
      task_id: string;
      status: string;
    }>;
  }

  // 获取生成进度
  async getGenerationProgress(taskId: string): Promise<unknown> {
    return this.postMessage("getGenerationProgress", { task_id: taskId });
  }
}

// 日报状态响应类型
export interface DailyReportStatusResponse {
  statuses: Record<string, boolean>;
}

// 工作分类统计
export interface WorkCategories {
  requirements_discussion: number; // 需求讨论
  coding: number; // 编码
  problem_solving: number; // 问题排查
  refactoring: number; // 重构
  code_review: number; // 代码审查
  documentation: number; // 文档编写
  testing: number; // 测试
  other: number; // 其他
}

// 时段统计
export interface TimeSlotStats {
  sessions: number; // 会话数
  hours: number; // 总时长（小时）
}

// 时间分布汇总
export interface TimeDistributionSummary {
  morning: TimeSlotStats; // 上午（9-12）
  afternoon: TimeSlotStats; // 下午（14-18）
  evening: TimeSlotStats; // 晚上（19-22）
  night: TimeSlotStats; // 夜间（22-2）
}

// 效率指标汇总
export interface EfficiencyMetricsSummary {
  avg_session_duration: number; // 平均会话时长（分钟）
  avg_messages_per_session: number; // 平均消息数
  total_active_time: number; // 总活跃时长（小时）
}

// 代码变更统计
export interface CodeChangeSummary {
  lines_added: number; // 新增行数
  lines_removed: number; // 删除行数
  files_changed: number; // 变更文件数
}

// 工作项
export interface WorkItem {
  category: string; // 工作类型
  description: string; // 工作描述
  session_id: string; // 关联的会话ID
}

// 会话摘要（用于每日总结）
export interface DailySessionSummary {
  session_id: string;
  name: string;
  project_name: string;
  created_at: number;
  updated_at: number;
  message_count: number;
  duration: number; // 持续时长（毫秒）
}

// 项目摘要类型
export interface ProjectSummary {
  project_name: string;
  project_path: string;
  workspace_id: string;
  session_count: number;
  // 详细信息
  work_items?: WorkItem[];
  sessions?: DailySessionSummary[];
  code_changes?: CodeChangeSummary;
  active_hours?: number[];
}

// 日报类型
export interface DailySummary {
  id: string;
  date: string;
  summary: string;
  language: string;
  total_sessions: number;
  projects?: ProjectSummary[];
  // 结构化统计数据
  work_categories?: WorkCategories;
  code_changes?: CodeChangeSummary;
  time_distribution?: TimeDistributionSummary;
  efficiency_metrics?: EfficiencyMetricsSummary;
  created_at?: string;
  updated_at?: string;
}

// 会话健康状态类型
export interface SessionHealth {
  entropy: number;
  status: "healthy" | "sub_healthy" | "dangerous";
  warning?: string;
}

// 代码分析相关类型
export interface EntryPointCandidate {
  file: string;
  function: string;
  type: string;
  priority: number;
  recommended: boolean;
}

export interface ScanEntryPointsResponse {
  project_name: string;
  remote_url: string;
  candidates: EntryPointCandidate[];
  default_exclude: string[];
}

export interface CallGraphStatus {
  exists: boolean;
  up_to_date: boolean;
  current_commit?: string;
  head_commit?: string;
  commits_behind?: number;
  project_registered: boolean;
  db_path?: string;
  created_at?: string;
  func_count?: number;
  valid_go_module?: boolean;
  go_module_error?: string;
}

export interface RegisterProjectRequest {
  project_path: string;
  entry_points: string[];
  exclude: string[];
  algorithm: string;
}

export interface ProjectConfig {
  id: string;
  name: string;
  remote_url: string;
  local_paths: string[];
  entry_points: string[];
  exclude: string[];
  algorithm: string;
  integration_test_dir?: string;
  integration_test_tag?: string;
  created_at?: string;
  updated_at?: string;
}

export interface GenerateWithConfigRequest {
  project_path: string;
  entry_points: string[];
  exclude: string[];
  algorithm: string;
  commit?: string;
  integration_test_dir?: string;
  integration_test_tag?: string;
}

export interface GenerateResponse {
  commit: string;
  func_count: number;
  edge_count: number;
  generation_time_ms: number;
  db_path: string;
}

export const apiService = new ApiService();
