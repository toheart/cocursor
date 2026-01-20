import * as vscode from "vscode";
import axios from "axios";
import FormData from "form-data";
import { WebviewMessage, ExtensionMessage } from "./types/message";

export type WebviewType = "workAnalysis" | "recentSessions" | "marketplace" | "ragSearch" | "workflow" | "team";

export class WebviewPanel {
  public static workAnalysisPanel: WebviewPanel | undefined;
  public static recentSessionsPanel: WebviewPanel | undefined;
  public static marketplacePanel: WebviewPanel | undefined;
  public static ragSearchPanel: WebviewPanel | undefined;
  public static workflowPanel: WebviewPanel | undefined;
  public static teamPanel: WebviewPanel | undefined;
  private readonly _panel: vscode.WebviewPanel;
  private readonly _extensionUri: vscode.Uri;
  private readonly _viewType: WebviewType;
  private readonly _context: vscode.ExtensionContext;
  private _disposables: vscode.Disposable[] = [];

  private initialRoute: string = "/";
  // 项目列表缓存（避免重复请求）
  private static projectListCache: {
    projects: Array<{ project_name: string; workspaces: Array<{ path: string }> }>;
    timestamp: number;
  } | null = null;
  private static readonly CACHE_TTL = 5000; // 5秒缓存

  private constructor(panel: vscode.WebviewPanel, extensionUri: vscode.Uri, viewType: WebviewType, context: vscode.ExtensionContext, route?: string) {
    this._panel = panel;
    this._extensionUri = extensionUri;
    this._viewType = viewType;
    this._context = context;
    if (route) {
      this.initialRoute = route;
    }

    console.log("WebviewPanel: 创建新面板", extensionUri.toString(), "viewType:", this._viewType, "route:", this.initialRoute);

    // 设置 Webview 内容
    const html = this._getHtmlForWebview(this._panel.webview);
    this._panel.webview.html = html;
    console.log("WebviewPanel: HTML 内容已设置");

    // 监听消息
    this._panel.webview.onDidReceiveMessage(
      (message: WebviewMessage) => {
        console.log("WebviewPanel: 收到消息", message);
        this._handleMessage(message);
      },
      null,
      this._disposables
    );

    // 监听面板关闭
    this._panel.onDidDispose(() => {
      console.log("WebviewPanel: 面板已关闭");
      this.dispose();
    }, null, this._disposables);
  }

  public static createOrShow(extensionUri: vscode.Uri, viewType: WebviewType, context: vscode.ExtensionContext, route?: string): void {
    console.log("WebviewPanel: createOrShow 被调用", extensionUri.toString(), viewType, route);
    
    const column = vscode.window.activeTextEditor
      ? vscode.window.activeTextEditor.viewColumn
      : undefined;

    // 根据类型获取对应的面板
    let currentPanel: WebviewPanel | undefined;
    if (viewType === "workAnalysis") {
      currentPanel = WebviewPanel.workAnalysisPanel;
    } else if (viewType === "recentSessions") {
      currentPanel = WebviewPanel.recentSessionsPanel;
    } else if (viewType === "marketplace") {
      currentPanel = WebviewPanel.marketplacePanel;
    } else if (viewType === "ragSearch") {
      currentPanel = WebviewPanel.ragSearchPanel;
    } else if (viewType === "workflow") {
      currentPanel = WebviewPanel.workflowPanel;
    } else if (viewType === "team") {
      currentPanel = WebviewPanel.teamPanel;
    }

    // 如果已经有对应类型的面板，显示它并导航到指定路由
    if (currentPanel) {
      console.log(`WebviewPanel: 使用现有${viewType}面板`);
      currentPanel._panel.reveal(column);
      // 发送路由导航消息（HashRouter 会自动处理 #）
      if (route) {
        // 移除 # 前缀，HashRouter 会自动添加
        const cleanRoute = route.startsWith("#") ? route.substring(1) : route;
        currentPanel._panel.webview.postMessage({
          type: "navigate",
          route: cleanRoute
        });
      }
      return;
    }

    // 创建新面板
    console.log(`WebviewPanel: 创建新${viewType}面板`);
    const panelTitle = 
      viewType === "workAnalysis" ? "工作分析 - CoCursor" :
      viewType === "recentSessions" ? "最近对话 - CoCursor" :
      viewType === "marketplace" ? "插件市场 - CoCursor" :
      viewType === "ragSearch" ? "RAG 搜索 - CoCursor" :
      viewType === "workflow" ? "OpenSpec 工作流 - CoCursor" :
      "团队 - CoCursor";
    const panelId = 
      viewType === "workAnalysis" ? "cocursorWorkAnalysis" :
      viewType === "recentSessions" ? "cocursorRecentSessions" :
      viewType === "marketplace" ? "cocursorMarketplace" :
      viewType === "ragSearch" ? "cocursorRAGSearch" :
      viewType === "workflow" ? "cocursorWorkflow" :
      "cocursorTeam";
    
    const panel = vscode.window.createWebviewPanel(
      panelId,
      panelTitle,
      column || vscode.ViewColumn.One,
      {
        enableScripts: true,
        localResourceRoots: [
          vscode.Uri.joinPath(extensionUri, "dist"),
          vscode.Uri.joinPath(extensionUri, "src")
        ],
        retainContextWhenHidden: true
      }
    );

    const newPanel = new WebviewPanel(panel, extensionUri, viewType, context, route);
    
    // 根据类型保存到对应的静态变量
    if (viewType === "workAnalysis") {
      WebviewPanel.workAnalysisPanel = newPanel;
    } else if (viewType === "recentSessions") {
      WebviewPanel.recentSessionsPanel = newPanel;
    } else if (viewType === "marketplace") {
      WebviewPanel.marketplacePanel = newPanel;
    } else if (viewType === "ragSearch") {
      WebviewPanel.ragSearchPanel = newPanel;
    } else if (viewType === "workflow") {
      WebviewPanel.workflowPanel = newPanel;
    } else if (viewType === "team") {
      WebviewPanel.teamPanel = newPanel;
    }
    
    console.log(`WebviewPanel: ${viewType}面板创建完成`);
  }

  private _handleMessage(message: WebviewMessage): void {
    switch (message.command) {
      case "fetchChats":
        this._handleFetchChats();
        break;
      case "fetchChatDetail":
        this._handleFetchChatDetail(message.payload as { chatId: string });
        break;
      case "getPeers":
        this._handleGetPeers();
        break;
      case "fetchCurrentSessionHealth":
        this._handleFetchCurrentSessionHealth(message.payload as { projectPath?: string; projectName?: string });
        break;
      case "fetchProjectDetail":
        this._handleFetchProjectDetail(message.payload as { projectName: string });
        break;
      case "fetchProjectStats":
        this._handleFetchProjectStats(message.payload as { projectName: string; startDate?: string; endDate?: string });
        break;
      case "fetchWorkAnalysis":
        this._handleFetchWorkAnalysis(message.payload as { startDate?: string; endDate?: string; projectName?: string });
        break;
      case "fetchSessionList":
        this._handleFetchSessionList(message.payload as { projectName?: string; limit?: number; offset?: number; search?: string });
        break;
      case "fetchSessionDetail":
        this._handleFetchSessionDetail(message.payload as { sessionId: string; limit?: number });
        break;
      case "fetchProjectList":
        this._handleFetchProjectList();
        break;
      case "showEntropyWarning":
        this._handleShowEntropyWarning(message.payload as { entropy: number; message: string });
        break;
      case "updateTitle":
        // 更新 WebView 标题（静默处理，不报错）
        if (message.payload && typeof message.payload === "object" && "title" in message.payload) {
          this._panel.title = String(message.payload.title);
        }
        break;
      case "changeLanguage":
        this._handleChangeLanguage(message.payload as { language: string });
        break;
      case "fetchPlugins":
        this._handleFetchPlugins(message.payload as { category?: string; search?: string; installed?: boolean; lang?: string; source?: string; team_id?: string });
        break;
      case "fetchPlugin":
        this._handleFetchPlugin(message.payload as { id: string });
        break;
      case "fetchInstalledPlugins":
        this._handleFetchInstalledPlugins();
        break;
      case "installPlugin":
        this._handleInstallPlugin(message.payload as { id: string; workspacePath: string; force?: boolean });
        break;
      case "uninstallPlugin":
        this._handleUninstallPlugin(message.payload as { id: string; workspacePath: string });
        break;
      case "checkPluginStatus":
        this._handleCheckPluginStatus(message.payload as { id: string });
        break;
      case "showConfirmDialog":
        this._handleShowConfirmDialog(message.payload as { message: string; confirmText?: string; cancelText?: string });
        break;
      case "fetchWorkflows":
        this._handleFetchWorkflows(message.payload as { projectPath?: string; status?: string });
        break;
      case "fetchWorkflowDetail":
        this._handleFetchWorkflowDetail(message.payload as { changeId: string; projectPath?: string });
        break;
      case "fetchRAGConfig":
        this._handleFetchRAGConfig();
        break;
      case "updateRAGConfig":
        this._handleUpdateRAGConfig(message.payload as { config: unknown });
        break;
      case "testRAGConfig":
        this._handleTestRAGConfig(message.payload as { config: { url: string; api_key: string; model: string } });
        break;
      case "testLLMConnection":
        this._handleTestLLMConnection(message.payload as { config: { url: string; api_key: string; model: string } });
        break;
      case "searchRAG":
        this._handleSearchRAG(message.payload as { query: string; projectIds?: string[]; limit?: number });
        break;
      case "searchRAGChunks":
        this._handleSearchRAGChunks(message.payload as { query: string; projectIds?: string[]; limit?: number });
        break;
      case "triggerRAGIndex":
        this._handleTriggerRAGIndex(message.payload as { sessionId?: string });
        break;
      case "triggerFullIndex":
        this._handleTriggerFullIndex(message.payload as { batch_size?: number; concurrency?: number } | undefined);
        break;
      case "fetchIndexProgress":
        this._handleFetchIndexProgress();
        break;
      case "clearAllData":
        this._handleClearAllData();
        break;
      case "fetchRAGStats":
        this._handleFetchRAGStats();
        break;
      case "fetchIndexedProjects":
        this._handleFetchIndexedProjects();
        break;
      case "fetchQdrantStatus":
        this._handleFetchQdrantStatus();
        break;
      case "downloadQdrant":
        this._handleDownloadQdrant(message.payload as { version?: string });
        break;
      case "uploadQdrantPackage":
        this._handleUploadQdrantPackage(message.payload as { filename: string; fileBase64: string });
        break;
      case "startQdrant":
        this._handleStartQdrant();
        break;
      case "stopQdrant":
        this._handleStopQdrant();
        break;
      case "openRAGSearch":
        this._handleOpenRAGSearch(message.payload as { route?: string } | undefined);
        break;
      // ========== 团队相关命令 ==========
      case "fetchTeamIdentity":
        this._handleFetchTeamIdentity();
        break;
      case "setTeamIdentity":
        this._handleSetTeamIdentity(message.payload as { name: string });
        break;
      case "fetchNetworkInterfaces":
        this._handleFetchNetworkInterfaces();
        break;
      case "createTeam":
        this._handleCreateTeam(message.payload as { name: string; preferred_interface?: string; preferred_ip?: string });
        break;
      case "discoverTeams":
        this._handleDiscoverTeams(message.payload as { timeout?: number });
        break;
      case "joinTeam":
        this._handleJoinTeamByEndpoint(message.payload as { endpoint: string });
        break;
      case "fetchTeamList":
        this._handleFetchTeamList();
        break;
      case "fetchTeamMembers":
        this._handleFetchTeamMembers(message.payload as { teamId: string });
        break;
      case "leaveTeam":
        this._handleLeaveTeam(message.payload as { teamId: string });
        break;
      case "dissolveTeam":
        this._handleDissolveTeam(message.payload as { teamId: string });
        break;
      case "fetchTeamSkillIndex":
        this._handleFetchTeamSkillIndex(message.payload as { teamId: string });
        break;
      case "validateSkillDirectory":
        this._handleValidateSkillDirectory(message.payload as { path: string });
        break;
      case "publishTeamSkill":
        this._handlePublishTeamSkill(message.payload as { teamId: string; pluginId: string; localPath: string });
        break;
      case "publishTeamSkillWithMetadata":
        this._handlePublishTeamSkillWithMetadata(message.payload as { 
          teamId: string; 
          localPath: string; 
          metadata: {
            plugin_id: string;
            name: string;
            name_zh_cn?: string;
            description: string;
            description_zh_cn?: string;
            version: string;
            category: string;
            author: string;
          };
        });
        break;
      case "installTeamSkill":
        this._handleInstallTeamSkill(message.payload as { teamId: string; pluginId: string; version?: string; force?: boolean });
        break;
      case "uninstallTeamSkill":
        this._handleUninstallTeamSkill(message.payload as { teamId: string; pluginId: string });
        break;
      case "downloadTeamSkill":
        this._handleDownloadTeamSkill(message.payload as { teamId: string; pluginId: string; authorEndpoint: string; checksum?: string });
        break;
      case "selectDirectory":
        this._handleSelectDirectory();
        break;
      // ========== 日报相关命令 ==========
      case "fetchDailyReportStatus":
        this._handleFetchDailyReportStatus(message.payload as { startDate: string; endDate: string });
        break;
      case "fetchDailySummary":
        this._handleFetchDailySummary(message.payload as { date: string });
        break;
      // 团队协作相关
      case "shareCode":
        this._handleShareCode(message.payload as { teamId: string; file_name: string; file_path?: string; language?: string; start_line?: number; end_line?: number; code: string; message?: string });
        break;
      case "updateWorkStatus":
        this._handleUpdateWorkStatus(message.payload as { teamId: string; project_name?: string; current_file?: string; status_visible?: boolean });
        break;
      case "shareTeamDailySummary":
        this._handleShareTeamDailySummary(message.payload as { teamId: string; date: string });
        break;
      case "fetchTeamDailySummaries":
        this._handleFetchTeamDailySummaries(message.payload as { teamId: string; date?: string });
        break;
      case "fetchTeamDailySummaryDetail":
        this._handleFetchTeamDailySummaryDetail(message.payload as { teamId: string; memberId: string; date: string });
        break;
      default:
        console.warn(`未知命令: ${message.command}`);
    }
  }

  // 处理语言切换：保存到 globalState 并广播到所有 webview
  private _handleChangeLanguage(payload: { language: string }): void {
    const language = payload.language;
    if (language !== 'zh-CN' && language !== 'en') {
      console.warn(`Invalid language: ${language}`);
      return;
    }

    // 保存到 globalState
    this._context.globalState.update('cocursor-language', language).then(() => {
      // 广播到所有 webview
      WebviewPanel.broadcastLanguageChange(language);
      
      // 通知侧边栏刷新（通过命令）
      vscode.commands.executeCommand('cocursor.refreshSidebarLanguage');
    });
  }

  // 静态方法：广播语言变更到所有已打开的 webview
  public static broadcastLanguageChange(language: string): void {
    const panels = [
      WebviewPanel.workAnalysisPanel,
      WebviewPanel.recentSessionsPanel,
      WebviewPanel.marketplacePanel,
      WebviewPanel.ragSearchPanel,
      WebviewPanel.workflowPanel,
      WebviewPanel.teamPanel
    ].filter(Boolean) as WebviewPanel[];

    panels.forEach(panel => {
      panel._sendMessage({
        type: "languageChanged",
        data: { language }
      });
    });
  }

  private async _handleFetchChats(): Promise<void> {
    try {
      // TODO: 调用后端 API
      // 返回空数组，前端期望直接收到数组数据
      const response: unknown[] = [];
      this._sendMessage({
        type: "fetchChats-response",
        data: response
      });
    } catch (error) {
      this._sendMessage({
        type: "fetchChats-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchChatDetail(payload: { chatId: string }): Promise<void> {
    try {
      // TODO: 调用后端 API
      const response = { code: 0, data: null, message: "success" };
      this._sendMessage({
        type: "fetchChatDetail-response",
        data: response
      });
    } catch (error) {
      this._sendMessage({
        type: "fetchChatDetail-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleGetPeers(): Promise<void> {
    try {
      // TODO: 调用后端 API
      const response = { code: 0, data: [], message: "success" };
      this._sendMessage({
        type: "getPeers-response",
        data: response
      });
    } catch (error) {
      this._sendMessage({
        type: "getPeers-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  // 获取项目列表（带缓存）
  private async _getProjectList(): Promise<Array<{ project_name: string; workspaces: Array<{ path: string }> }>> {
    const now = Date.now();
    // 检查缓存是否有效
    if (WebviewPanel.projectListCache && (now - WebviewPanel.projectListCache.timestamp) < WebviewPanel.CACHE_TTL) {
      return WebviewPanel.projectListCache.projects;
    }

    try {
      const projectsResponse = await axios.get("http://localhost:19960/api/v1/project/list", {
        timeout: 5000
      });
      
      if (projectsResponse.data && projectsResponse.data.data && projectsResponse.data.data.projects) {
        const projects = projectsResponse.data.data.projects as Array<{
          project_name: string;
          workspaces: Array<{ path: string }>;
        }>;
        // 更新缓存
        WebviewPanel.projectListCache = {
          projects,
          timestamp: now
        };
        return projects;
      }
    } catch (error) {
      console.log(`获取项目列表失败: ${error instanceof Error ? error.message : String(error)}`);
    }
    
    // 如果请求失败，返回空数组
    return [];
  }

  // 通过路径查找项目名
  private _findProjectNameByPath(projectPath: string, projects: Array<{ project_name: string; workspaces: Array<{ path: string }> }>): string | null {
    const normalizePathForCompare = (path: string): string => {
      return path.replace(/\\/g, "/").toLowerCase().replace(/\/$/, "");
    };
    const normalizedTargetPath = normalizePathForCompare(projectPath);
    
    for (const project of projects) {
      for (const ws of project.workspaces) {
        if (normalizePathForCompare(ws.path) === normalizedTargetPath) {
          return project.project_name;
        }
      }
    }
    return null;
  }

  private async _handleFetchCurrentSessionHealth(payload: { projectPath?: string; projectName?: string }): Promise<void> {
    try {
      // 优先使用 project_name，其次通过 project_path 查找项目名
      const workspaceFolders = vscode.workspace.workspaceFolders;
      let projectPath = payload.projectPath;
      let projectName = payload.projectName;
      
      if (!projectName && !projectPath && workspaceFolders && workspaceFolders.length > 0) {
        projectPath = workspaceFolders[0].uri.fsPath;
      }

      // 如果没有 project_name，通过路径查找项目名（使用缓存）
      if (!projectName && projectPath) {
        try {
          const projects = await this._getProjectList();
          projectName = this._findProjectNameByPath(projectPath, projects) || undefined;
        } catch (error) {
          console.log(`通过路径查找项目名失败: ${error instanceof Error ? error.message : String(error)}`);
          // 继续执行，后端会尝试自动检测
        }
      }

      // 调用后端 API（统一使用 project_name 参数）
      let apiUrl = "http://localhost:19960/api/v1/stats/current-session";
      if (projectName) {
        apiUrl += `?project_name=${encodeURIComponent(projectName)}`;
      }
      // 如果没有 project_name，后端会尝试从当前工作目录自动检测

      const response = await axios.get(apiUrl, {
        timeout: 5000
      });
      
      if (response.data.code === 0 && response.data.data) {
        this._sendMessage({
          type: "fetchCurrentSessionHealth-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取会话健康状态失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchCurrentSessionHealth-response",
        data: { 
          error: error instanceof Error ? error.message : "未知错误",
          entropy: 0,
          status: "healthy",
          warning: ""
        }
      });
    }
  }

  private async _handleFetchProjectDetail(payload: { projectName: string }): Promise<void> {
    try {
      const response = await axios.get(
        `http://localhost:19960/api/v1/project/${encodeURIComponent(payload.projectName)}/sessions`,
        { timeout: 5000 }
      );
      
      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchProjectDetail-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取项目详情失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchProjectDetail-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchProjectStats(payload: { projectName: string; startDate?: string; endDate?: string }): Promise<void> {
    try {
      const params = new URLSearchParams();
      if (payload.startDate) {
        params.append("start_date", payload.startDate);
      }
      if (payload.endDate) {
        params.append("end_date", payload.endDate);
      }

      const apiUrl = `http://localhost:19960/api/v1/project/${encodeURIComponent(payload.projectName)}/stats/acceptance${params.toString() ? `?${params.toString()}` : ""}`;
      const response = await axios.get(apiUrl, {
        timeout: 5000
      });
      
      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchProjectStats-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取项目统计失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchProjectStats-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchWorkAnalysis(payload: { startDate?: string; endDate?: string }): Promise<void> {
    try {
      let apiUrl = "http://localhost:19960/api/v1/stats/work-analysis";
      const params = new URLSearchParams();
      if (payload.startDate) {
        params.append("start_date", payload.startDate);
      }
      if (payload.endDate) {
        params.append("end_date", payload.endDate);
      }
      if (params.toString()) {
        apiUrl += `?${params.toString()}`;
      }

      // 工作分析接口需要遍历多个工作区和日期，处理时间较长
      const response = await axios.get(apiUrl, { timeout: 60000 });
      
      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchWorkAnalysis-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取工作分析数据失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchWorkAnalysis-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchSessionList(payload: { projectName?: string; limit?: number; offset?: number; search?: string }): Promise<void> {
    try {
      const params = new URLSearchParams();
      if (payload.projectName) {
        params.append("project_name", payload.projectName);
      }
      if (payload.limit) {
        params.append("limit", payload.limit.toString());
      }
      if (payload.offset) {
        params.append("offset", payload.offset.toString());
      }
      if (payload.search) {
        params.append("search", payload.search);
      }

      const apiUrl = `http://localhost:19960/api/v1/sessions/list${params.toString() ? `?${params.toString()}` : ""}`;
      const response = await axios.get(apiUrl, { timeout: 10000 });
      
      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchSessionList-response",
          data: {
            data: response.data.data,
            page: response.data.page
          }
        });
      } else {
        throw new Error(response.data.message || "获取会话列表失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchSessionList-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchSessionDetail(payload: { sessionId: string; limit?: number }): Promise<void> {
    try {
      const params = new URLSearchParams();
      if (payload.limit) {
        params.append("limit", payload.limit.toString());
      }

      const apiUrl = `http://localhost:19960/api/v1/sessions/${encodeURIComponent(payload.sessionId)}/detail${params.toString() ? `?${params.toString()}` : ""}`;
      const response = await axios.get(apiUrl, { timeout: 10000 });
      
      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchSessionDetail-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取会话详情失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchSessionDetail-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchProjectList(): Promise<void> {
    try {
      const projects = await this._getProjectList();
      this._sendMessage({
        type: "fetchProjectList-response",
        data: { projects: projects.map(p => ({ project_name: p.project_name })) }
      });
    } catch (error) {
      this._sendMessage({
        type: "fetchProjectList-response",
        data: { 
          error: error instanceof Error ? error.message : "未知错误",
          projects: []
        }
      });
    }
  }

  private _handleShowEntropyWarning(payload: { entropy: number; message: string }): void {
    // 显示 VS Code 警告通知
    vscode.window.showWarningMessage(
      `⚠️ ${payload.message} (熵值: ${payload.entropy.toFixed(2)})`,
      "查看详情"
    ).then((selection) => {
      if (selection === "查看详情") {
        // 显示面板（如果已关闭则创建）
        WebviewPanel.createOrShow(this._extensionUri, "workAnalysis", this._context);
      }
    });
  }

  private async _handleFetchPlugins(payload: { category?: string; search?: string; installed?: boolean; lang?: string; source?: string; team_id?: string }): Promise<void> {
    try {
      const params = new URLSearchParams();
      if (payload.category) {
        params.append("category", payload.category);
      }
      if (payload.search) {
        params.append("search", payload.search);
      }
      if (payload.installed !== undefined) {
        params.append("installed", payload.installed.toString());
      }
      if (payload.lang) {
        params.append("lang", payload.lang);
      }
      if (payload.source) {
        params.append("source", payload.source);
      }
      if (payload.team_id) {
        params.append("team_id", payload.team_id);
      }

      const apiUrl = `http://localhost:19960/api/v1/marketplace/plugins${params.toString() ? `?${params.toString()}` : ""}`;
      const response = await axios.get(apiUrl, { timeout: 10000 });

      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchPlugins-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "Failed to fetch plugins");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchPlugins-response",
        data: { error: error instanceof Error ? error.message : "Unknown error" }
      });
    }
  }

  private async _handleFetchPlugin(payload: { id: string; lang?: string }): Promise<void> {
    try {
      const apiUrl = `http://localhost:19960/api/v1/marketplace/plugins/${encodeURIComponent(payload.id)}`;
      const params = new URLSearchParams();
      if (payload.lang) {
        params.append("lang", payload.lang);
      }
      const urlWithParams = params.toString() ? `${apiUrl}?${params.toString()}` : apiUrl;
      
      const response = await axios.get(urlWithParams, { timeout: 10000 });

      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchPlugin-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "Failed to fetch plugin");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchPlugin-response",
        data: { error: error instanceof Error ? error.message : "Unknown error" }
      });
    }
  }

  private async _handleFetchInstalledPlugins(payload?: { lang?: string }): Promise<void> {
    try {
      const apiUrl = "http://localhost:19960/api/v1/marketplace/installed";
      const params = new URLSearchParams();
      if (payload && payload.lang) {
        params.append("lang", payload.lang);
      }
      const urlWithParams = params.toString() ? `${apiUrl}?${params.toString()}` : apiUrl;
      
      const response = await axios.get(urlWithParams, { timeout: 10000 });

      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchInstalledPlugins-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "Failed to fetch installed plugins");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchInstalledPlugins-response",
        data: { error: error instanceof Error ? error.message : "Unknown error" }
      });
    }
  }

  private async _handleInstallPlugin(payload: { id: string; workspacePath: string; force?: boolean }): Promise<void> {
    try {
      const apiUrl = `http://localhost:19960/api/v1/marketplace/plugins/${encodeURIComponent(payload.id)}/install`;
      const response = await axios.post(apiUrl, {
        workspace_path: payload.workspacePath,
        force: payload.force || false
      }, { timeout: 30000 });

      if (response.data.code === 0) {
        const result = response.data.data as { success: boolean; message: string; env_vars?: string[] };
        
        // 如果有环境变量需要配置，显示提示
        if (result.env_vars && result.env_vars.length > 0) {
          const envVarsList = result.env_vars.join(", ");
          vscode.window.showWarningMessage(
            `Plugin installed successfully. Please configure the following environment variables: ${envVarsList}`,
            "OK"
          );
        } else {
          vscode.window.showInformationMessage("Plugin installed successfully", "OK");
        }

        this._sendMessage({
          type: "installPlugin-response",
          data: result
        });
      } else {
        throw new Error(response.data.message || "Failed to install plugin");
      }
    } catch (error) {
      // 处理 409 冲突响应
      if (axios.isAxiosError(error) && error.response?.status === 409) {
        const conflictData = error.response.data?.data as {
          skill_name: string;
          plugin_id: string;
          message: string;
          conflict_type: string;
        };
        
        // 返回冲突信息给前端，让前端决定是否强制覆盖
        this._sendMessage({
          type: "installPlugin-response",
          data: {
            conflict: true,
            conflict_type: conflictData?.conflict_type || "unknown",
            skill_name: conflictData?.skill_name || "",
            message: conflictData?.message || "Skill conflict detected"
          }
        });
        return;
      }
      
      const errorMessage = error instanceof Error ? error.message : "Unknown error";
      vscode.window.showErrorMessage(`Failed to install plugin: ${errorMessage}`);
      this._sendMessage({
        type: "installPlugin-response",
        data: { error: errorMessage }
      });
    }
  }

  private async _handleUninstallPlugin(payload: { id: string; workspacePath: string }): Promise<void> {
    try {
      const apiUrl = `http://localhost:19960/api/v1/marketplace/plugins/${encodeURIComponent(payload.id)}/uninstall`;
      const response = await axios.post(apiUrl, {
        workspace_path: payload.workspacePath
      }, { timeout: 30000 });

      if (response.data.code === 0) {
        vscode.window.showInformationMessage("Plugin uninstalled successfully", "OK");
        this._sendMessage({
          type: "uninstallPlugin-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "Failed to uninstall plugin");
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : "Unknown error";
      vscode.window.showErrorMessage(`Failed to uninstall plugin: ${errorMessage}`);
      this._sendMessage({
        type: "uninstallPlugin-response",
        data: { error: errorMessage }
      });
    }
  }

  private async _handleCheckPluginStatus(payload: { id: string }): Promise<void> {
    try {
      const apiUrl = `http://localhost:19960/api/v1/marketplace/plugins/${encodeURIComponent(payload.id)}/status`;
      const response = await axios.get(apiUrl, { timeout: 10000 });

      if (response.data.code === 0) {
        this._sendMessage({
          type: "checkPluginStatus-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "Failed to check plugin status");
      }
    } catch (error) {
      this._sendMessage({
        type: "checkPluginStatus-response",
        data: { error: error instanceof Error ? error.message : "Unknown error" }
      });
    }
  }

  private async _handleShowConfirmDialog(payload: { message: string; confirmText?: string; cancelText?: string }): Promise<void> {
    const confirmText = payload.confirmText || "Yes";
    const cancelText = payload.cancelText || "No";
    
    const result = await vscode.window.showWarningMessage(
      payload.message,
      { modal: true },
      confirmText,
      cancelText
    );
    
    this._sendMessage({
      type: "showConfirmDialog-response",
      data: result === confirmText
    });
  }

  private async _handleFetchWorkflows(payload: { projectPath?: string; status?: string }): Promise<void> {
    try {
      const params = new URLSearchParams();
      if (payload.projectPath) {
        params.append("project_path", payload.projectPath);
      }
      if (payload.status) {
        params.append("status", payload.status);
      }

      const apiUrl = `http://localhost:19960/api/v1/workflows${params.toString() ? `?${params.toString()}` : ""}`;
      console.log("[WebviewPanel] Fetching workflows from:", apiUrl);
      const response = await axios.get(apiUrl, { timeout: 10000 });

      console.log("[WebviewPanel] API response:", {
        code: response.data.code,
        message: response.data.message,
        dataType: Array.isArray(response.data.data) ? "array" : typeof response.data.data,
        dataLength: Array.isArray(response.data.data) ? response.data.data.length : "N/A"
      });

      if (response.data.code === 0) {
        const workflows = response.data.data || [];
        console.log("[WebviewPanel] Sending workflows to frontend:", workflows.length, "items");
        this._sendMessage({
          type: "fetchWorkflows-response",
          data: workflows
        });
      } else {
        throw new Error(response.data.message || "获取工作流列表失败");
      }
    } catch (error) {
      console.error("[WebviewPanel] Error fetching workflows:", error);
      this._sendMessage({
        type: "fetchWorkflows-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchWorkflowDetail(payload: { changeId: string; projectPath?: string }): Promise<void> {
    try {
      const params = new URLSearchParams();
      if (payload.projectPath) {
        params.append("project_path", payload.projectPath);
      }

      const apiUrl = `http://localhost:19960/api/v1/workflows/${encodeURIComponent(payload.changeId)}${params.toString() ? `?${params.toString()}` : ""}`;
      const response = await axios.get(apiUrl, { timeout: 10000 });

      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchWorkflowDetail-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取工作流详情失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchWorkflowDetail-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  // ========== RAG 相关处理 ==========

  private async _handleFetchRAGConfig(): Promise<void> {
    try {
      const response = await axios.get("http://localhost:19960/api/v1/rag/config", { timeout: 10000 });
      this._sendMessage({
        type: "fetchRAGConfig-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "fetchRAGConfig-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleUpdateRAGConfig(payload: { config: unknown }): Promise<void> {
    try {
      const response = await axios.post("http://localhost:19960/api/v1/rag/config", payload.config, { timeout: 10000 });
      this._sendMessage({
        type: "updateRAGConfig-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "updateRAGConfig-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleTestRAGConfig(payload: { config: { url: string; api_key: string; model: string } }): Promise<void> {
    try {
      const response = await axios.post("http://localhost:19960/api/v1/rag/config/test", payload.config, { timeout: 30000 });
      this._sendMessage({
        type: "testRAGConfig-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "testRAGConfig-response",
        data: { 
          success: false,
          error: error instanceof Error ? error.message : "未知错误" 
        }
      });
    }
  }

  private async _handleSearchRAG(payload: { query: string; projectIds?: string[]; limit?: number }): Promise<void> {
    try {
      const requestBody: any = {
        query: payload.query,
        limit: payload.limit || 10
      };
      // 只有当 projectIds 有值时才添加到请求体
      if (payload.projectIds && payload.projectIds.length > 0) {
        requestBody.project_ids = payload.projectIds;
      }

      const response = await axios.post("http://localhost:19960/api/v1/rag/search", requestBody, { timeout: 30000 });
      this._sendMessage({
        type: "searchRAG-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "searchRAG-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleSearchRAGChunks(payload: { query: string; projectIds?: string[]; limit?: number }): Promise<void> {
    try {
      const requestBody: any = {
        query: payload.query,
        limit: payload.limit || 20
      };
      // 只有当 projectIds 有值时才添加到请求体
      if (payload.projectIds && payload.projectIds.length > 0) {
        requestBody.project_ids = payload.projectIds;
      }

      const response = await axios.post("http://localhost:19960/api/v1/rag/search/chunks", requestBody, { timeout: 30000 });
      this._sendMessage({
        type: "searchRAGChunks-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "searchRAGChunks-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleTriggerRAGIndex(payload: { sessionId?: string }): Promise<void> {
    try {
      const response = await axios.post("http://localhost:19960/api/v1/rag/index", {
        session_id: payload.sessionId || ""
      }, { timeout: 10000 });
      this._sendMessage({
        type: "triggerRAGIndex-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "triggerRAGIndex-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleTestLLMConnection(payload: { config: { url: string; api_key: string; model: string } }): Promise<void> {
    try {
      const response = await axios.post("http://localhost:19960/api/v1/rag/config/llm/test", payload.config, { timeout: 30000 });
      this._sendMessage({
        type: "testLLMConnection-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "testLLMConnection-response",
        data: {
          success: false,
          error: error instanceof Error ? error.message : "未知错误"
        }
      });
    }
  }

  private async _handleTriggerFullIndex(payload?: { batch_size?: number; concurrency?: number }): Promise<void> {
    try {
      const response = await axios.post("http://localhost:19960/api/v1/rag/index/full", {
        batch_size: payload?.batch_size,
        concurrency: payload?.concurrency
      }, { timeout: 10000 });
      this._sendMessage({
        type: "triggerFullIndex-response",
        data: response.data
      });
    } catch (error) {
      // 尝试从 axios 响应中获取后端错误消息
      let errorMessage = "未知错误";
      if (axios.isAxiosError(error) && error.response?.data?.error) {
        errorMessage = error.response.data.error;
      } else if (error instanceof Error) {
        errorMessage = error.message;
      }
      this._sendMessage({
        type: "triggerFullIndex-response",
        data: { error: errorMessage }
      });
    }
  }

  private async _handleFetchIndexProgress(): Promise<void> {
    try {
      const response = await axios.get("http://localhost:19960/api/v1/rag/index/progress", { timeout: 10000 });
      this._sendMessage({
        type: "fetchIndexProgress-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "fetchIndexProgress-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleClearAllData(): Promise<void> {
    try {
      const response = await axios.delete("http://localhost:19960/api/v1/rag/data", { timeout: 10000 });
      this._sendMessage({
        type: "clearAllData-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "clearAllData-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchRAGStats(): Promise<void> {
    try {
      const response = await axios.get("http://localhost:19960/api/v1/rag/stats", { timeout: 10000 });
      this._sendMessage({
        type: "fetchRAGStats-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "fetchRAGStats-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchIndexedProjects(): Promise<void> {
    try {
      const response = await axios.get("http://localhost:19960/api/v1/rag/projects", { timeout: 10000 });
      this._sendMessage({
        type: "fetchIndexedProjects-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "fetchIndexedProjects-response",
        data: { projects: [], total: 0, error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleDownloadQdrant(payload: { version?: string }): Promise<void> {
    try {
      // 异步下载模式：接口立即返回，前端通过轮询 status 获取进度
      const response = await axios.post("http://localhost:19960/api/v1/rag/qdrant/download", {
        version: payload.version || ""
      }, { timeout: 30000 }); // 30秒超时（只是启动下载任务，不需要等待下载完成）
      this._sendMessage({
        type: "downloadQdrant-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "downloadQdrant-response",
        data: { 
          success: false,
          error: error instanceof Error ? error.message : "未知错误" 
        }
      });
    }
  }

  private _handleOpenRAGSearch(payload?: { route?: string }): void {
    const route = payload?.route || "/";
    WebviewPanel.createOrShow(this._extensionUri, "ragSearch", this._context, route);
  }

  private async _handleUploadQdrantPackage(payload: { filename: string; fileBase64: string }): Promise<void> {
    try {
      // 将 base64 转换为 Buffer
      const fileBuffer = Buffer.from(payload.fileBase64, "base64");
      
      // 创建 FormData（使用 form-data 包）
      const form = new FormData();
      form.append("file", fileBuffer, {
        filename: payload.filename,
        contentType: payload.filename.endsWith(".zip") ? "application/zip" : "application/gzip"
      });

      const response = await axios.post("http://localhost:19960/api/v1/rag/qdrant/upload", form, {
        headers: {
          ...form.getHeaders()
        },
        timeout: 120000, // 2分钟超时
        maxContentLength: Infinity,
        maxBodyLength: Infinity
      });

      this._sendMessage({
        type: "uploadQdrantPackage-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "uploadQdrantPackage-response",
        data: {
          success: false,
          error: error instanceof Error ? error.message : "未知错误"
        }
      });
    }
  }

  private async _handleFetchQdrantStatus(): Promise<void> {
    try {
      const response = await axios.get("http://localhost:19960/api/v1/rag/qdrant/status", { timeout: 10000 });
      this._sendMessage({
        type: "fetchQdrantStatus-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "fetchQdrantStatus-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleStartQdrant(): Promise<void> {
    try {
      const response = await axios.post("http://localhost:19960/api/v1/rag/qdrant/start", {}, { timeout: 10000 });
      this._sendMessage({
        type: "startQdrant-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "startQdrant-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleStopQdrant(): Promise<void> {
    try {
      const response = await axios.post("http://localhost:19960/api/v1/rag/qdrant/stop", {}, { timeout: 10000 });
      this._sendMessage({
        type: "stopQdrant-response",
        data: response.data
      });
    } catch (error) {
      this._sendMessage({
        type: "stopQdrant-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  // ========== 团队相关处理函数 ==========

  private async _handleFetchTeamIdentity(): Promise<void> {
    try {
      const response = await axios.get("http://localhost:19960/api/v1/team/identity", { timeout: 10000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchTeamIdentity-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取身份失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchTeamIdentity-response",
        data: { exists: false, error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleSetTeamIdentity(payload: { name: string }): Promise<void> {
    try {
      const response = await axios.post("http://localhost:19960/api/v1/team/identity", payload, { timeout: 10000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "setTeamIdentity-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "设置身份失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "setTeamIdentity-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchNetworkInterfaces(): Promise<void> {
    try {
      const response = await axios.get("http://localhost:19960/api/v1/team/network/interfaces", { timeout: 10000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchNetworkInterfaces-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取网卡列表失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchNetworkInterfaces-response",
        data: { interfaces: [], error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleCreateTeam(payload: { name: string; preferred_interface?: string; preferred_ip?: string }): Promise<void> {
    try {
      const response = await axios.post("http://localhost:19960/api/v1/team/create", payload, { timeout: 30000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "createTeam-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "创建团队失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "createTeam-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleDiscoverTeams(payload: { timeout?: number }): Promise<void> {
    try {
      const timeout = payload.timeout || 5;
      const response = await axios.get(`http://localhost:19960/api/v1/team/discover?timeout=${timeout}`, { timeout: (timeout + 5) * 1000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "discoverTeams-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "发现团队失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "discoverTeams-response",
        data: { teams: [], error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleJoinTeamByEndpoint(payload: { endpoint: string }): Promise<void> {
    try {
      const response = await axios.post("http://localhost:19960/api/v1/team/join", payload, { timeout: 30000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "joinTeam-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "加入团队失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "joinTeam-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchTeamList(): Promise<void> {
    try {
      const response = await axios.get("http://localhost:19960/api/v1/team/list", { timeout: 10000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchTeamList-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取团队列表失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchTeamList-response",
        data: { teams: [], total: 0, error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchTeamMembers(payload: { teamId: string }): Promise<void> {
    try {
      const response = await axios.get(`http://localhost:19960/api/v1/team/${payload.teamId}/members`, { timeout: 10000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchTeamMembers-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取成员列表失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchTeamMembers-response",
        data: { members: [], error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleLeaveTeam(payload: { teamId: string }): Promise<void> {
    try {
      const response = await axios.post(`http://localhost:19960/api/v1/team/${payload.teamId}/leave`, {}, { timeout: 10000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "leaveTeam-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "离开团队失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "leaveTeam-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleDissolveTeam(payload: { teamId: string }): Promise<void> {
    try {
      const response = await axios.post(`http://localhost:19960/api/v1/team/${payload.teamId}/dissolve`, {}, { timeout: 10000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "dissolveTeam-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "解散团队失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "dissolveTeam-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchTeamSkillIndex(payload: { teamId: string }): Promise<void> {
    try {
      const response = await axios.get(`http://localhost:19960/api/v1/team/${payload.teamId}/skills`, { timeout: 10000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchTeamSkillIndex-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取技能索引失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchTeamSkillIndex-response",
        data: { entries: [], error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleValidateSkillDirectory(payload: { path: string }): Promise<void> {
    try {
      const response = await axios.post("http://localhost:19960/api/v1/team/skills/validate", { path: payload.path }, { timeout: 10000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "validateSkillDirectory-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "验证技能目录失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "validateSkillDirectory-response",
        data: { valid: false, error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handlePublishTeamSkill(payload: { teamId: string; pluginId: string; localPath: string }): Promise<void> {
    try {
      const response = await axios.post(`http://localhost:19960/api/v1/team/${payload.teamId}/skills/publish`, {
        plugin_id: payload.pluginId,
        local_path: payload.localPath
      }, { timeout: 60000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "publishTeamSkill-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "发布技能失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "publishTeamSkill-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  // 带元数据发布技能
  private async _handlePublishTeamSkillWithMetadata(payload: { 
    teamId: string; 
    localPath: string; 
    metadata: {
      plugin_id: string;
      name: string;
      name_zh_cn?: string;
      description: string;
      description_zh_cn?: string;
      version: string;
      category: string;
      author: string;
    };
  }): Promise<void> {
    try {
      const response = await axios.post(`http://localhost:19960/api/v1/team/${payload.teamId}/skills/publish-with-metadata`, {
        local_path: payload.localPath,
        metadata: payload.metadata
      }, { timeout: 60000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "publishTeamSkillWithMetadata-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "发布技能失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "publishTeamSkillWithMetadata-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  // 安装团队技能
  private async _handleInstallTeamSkill(payload: { teamId: string; pluginId: string; version?: string; force?: boolean }): Promise<void> {
    try {
      // 获取当前工作区路径
      const workspaceFolders = vscode.workspace.workspaceFolders;
      const workspacePath = workspaceFolders && workspaceFolders.length > 0 
        ? workspaceFolders[0].uri.fsPath 
        : "";
      
      const response = await axios.post(
        `http://localhost:19960/api/v1/team/${payload.teamId}/skills/${payload.pluginId}/install`, 
        {
          workspace_path: workspacePath,
          version: payload.version || "1.0.0",
          force: payload.force || false
        }, 
        { timeout: 60000 }
      );
      if (response.data.code === 0) {
        this._sendMessage({
          type: "installTeamSkill-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "安装技能失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "installTeamSkill-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  // 卸载团队技能
  private async _handleUninstallTeamSkill(payload: { teamId: string; pluginId: string }): Promise<void> {
    try {
      // 获取当前工作区路径
      const workspaceFolders = vscode.workspace.workspaceFolders;
      const workspacePath = workspaceFolders && workspaceFolders.length > 0 
        ? workspaceFolders[0].uri.fsPath 
        : "";
      
      const response = await axios.post(
        `http://localhost:19960/api/v1/team/${payload.teamId}/skills/${payload.pluginId}/uninstall`, 
        {
          workspace_path: workspacePath
        }, 
        { timeout: 60000 }
      );
      if (response.data.code === 0) {
        this._sendMessage({
          type: "uninstallTeamSkill-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "卸载技能失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "uninstallTeamSkill-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleDownloadTeamSkill(payload: { teamId: string; pluginId: string; authorEndpoint: string; checksum?: string }): Promise<void> {
    try {
      const response = await axios.post(`http://localhost:19960/api/v1/team/${payload.teamId}/skills/download`, {
        plugin_id: payload.pluginId,
        author_endpoint: payload.authorEndpoint,
        checksum: payload.checksum
      }, { timeout: 120000 });
      if (response.data.code === 0) {
        this._sendMessage({
          type: "downloadTeamSkill-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "下载技能失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "downloadTeamSkill-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleSelectDirectory(): Promise<void> {
    try {
      const result = await vscode.window.showOpenDialog({
        canSelectFiles: false,
        canSelectFolders: true,
        canSelectMany: false,
        openLabel: "选择技能目录"
      });

      if (result && result.length > 0) {
        this._sendMessage({
          type: "selectDirectory-response",
          data: { path: result[0].fsPath }
        });
      } else {
        this._sendMessage({
          type: "selectDirectory-response",
          data: { cancelled: true }
        });
      }
    } catch (error) {
      this._sendMessage({
        type: "selectDirectory-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  // ========== 日报相关处理 ==========

  private async _handleFetchDailyReportStatus(payload: { startDate: string; endDate: string }): Promise<void> {
    try {
      const response = await axios.get("http://localhost:19960/api/v1/daily-summary/batch-status", {
        params: {
          start_date: payload.startDate,
          end_date: payload.endDate
        },
        timeout: 10000
      });

      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchDailyReportStatus-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取日报状态失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchDailyReportStatus-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchDailySummary(payload: { date: string }): Promise<void> {
    try {
      const response = await axios.get("http://localhost:19960/api/v1/daily-summary", {
        params: {
          date: payload.date
        },
        timeout: 10000
      });

      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchDailySummary-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取日报失败");
      }
    } catch (error) {
      // 404 表示日报不存在，返回 null 而不是错误
      if (axios.isAxiosError(error) && error.response?.status === 404) {
        this._sendMessage({
          type: "fetchDailySummary-response",
          data: null
        });
      } else {
        this._sendMessage({
          type: "fetchDailySummary-response",
          data: { error: error instanceof Error ? error.message : "未知错误" }
        });
      }
    }
  }

  // ========== 团队协作 API 处理器 ==========

  private async _handleShareCode(payload: { 
    teamId: string; 
    file_name: string; 
    file_path?: string; 
    language?: string; 
    start_line?: number; 
    end_line?: number; 
    code: string; 
    message?: string 
  }): Promise<void> {
    try {
      const response = await axios.post(
        `http://localhost:19960/api/v1/team/${payload.teamId}/share-code`,
        {
          file_name: payload.file_name,
          file_path: payload.file_path,
          language: payload.language,
          start_line: payload.start_line,
          end_line: payload.end_line,
          code: payload.code,
          message: payload.message
        },
        { timeout: 10000 }
      );
      if (response.data.code === 0) {
        this._sendMessage({
          type: "shareCode-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "分享代码失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "shareCode-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleUpdateWorkStatus(payload: { 
    teamId: string; 
    project_name?: string; 
    current_file?: string; 
    status_visible?: boolean 
  }): Promise<void> {
    try {
      const response = await axios.post(
        `http://localhost:19960/api/v1/team/${payload.teamId}/status`,
        {
          project_name: payload.project_name,
          current_file: payload.current_file,
          status_visible: payload.status_visible
        },
        { timeout: 5000 }
      );
      if (response.data.code === 0) {
        this._sendMessage({
          type: "updateWorkStatus-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "更新工作状态失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "updateWorkStatus-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleShareTeamDailySummary(payload: { teamId: string; date: string }): Promise<void> {
    try {
      const response = await axios.post(
        `http://localhost:19960/api/v1/team/${payload.teamId}/daily-summaries/share`,
        { date: payload.date },
        { timeout: 30000 }
      );
      if (response.data.code === 0) {
        this._sendMessage({
          type: "shareTeamDailySummary-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "分享日报失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "shareTeamDailySummary-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchTeamDailySummaries(payload: { teamId: string; date?: string }): Promise<void> {
    try {
      const date = payload.date || new Date().toISOString().split("T")[0];
      const response = await axios.get(
        `http://localhost:19960/api/v1/team/${payload.teamId}/daily-summaries`,
        { 
          params: { date },
          timeout: 10000 
        }
      );
      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchTeamDailySummaries-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取团队日报列表失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchTeamDailySummaries-response",
        data: { summaries: [], error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private async _handleFetchTeamDailySummaryDetail(payload: { teamId: string; memberId: string; date: string }): Promise<void> {
    try {
      const response = await axios.get(
        `http://localhost:19960/api/v1/team/${payload.teamId}/daily-summaries/${payload.memberId}`,
        { 
          params: { date: payload.date },
          timeout: 30000 
        }
      );
      if (response.data.code === 0) {
        this._sendMessage({
          type: "fetchTeamDailySummaryDetail-response",
          data: response.data.data
        });
      } else {
        throw new Error(response.data.message || "获取日报详情失败");
      }
    } catch (error) {
      this._sendMessage({
        type: "fetchTeamDailySummaryDetail-response",
        data: { error: error instanceof Error ? error.message : "未知错误" }
      });
    }
  }

  private _sendMessage(message: ExtensionMessage): void {
    this._panel.webview.postMessage(message);
  }

  private _getHtmlForWebview(webview: vscode.Webview): string {
    // 获取资源 URI
    const scriptUri = webview.asWebviewUri(
      vscode.Uri.joinPath(this._extensionUri, "dist", "webview", "index.js")
    );
    const styleUri = webview.asWebviewUri(
      vscode.Uri.joinPath(this._extensionUri, "dist", "webview", "index.css")
    );

    console.log("WebviewPanel: Script URI", scriptUri.toString());
    console.log("WebviewPanel: Style URI", styleUri.toString());

    // 使用 nonce 增强安全性
    const nonce = getNonce();

    // 获取工作区路径
    const workspaceFolders = vscode.workspace.workspaceFolders;
    let workspacePathScript = "";
    if (workspaceFolders && workspaceFolders.length > 0) {
      const path = workspaceFolders[0].uri.fsPath;
      // 使用 JSON.stringify 确保跨平台兼容性和安全性
      workspacePathScript = `window.__WORKSPACE_PATH__ = ${JSON.stringify(path)};`;
    }

    // 从 globalState 获取语言设置
    const savedLanguage = this._context.globalState.get<string>('cocursor-language') || 'zh-CN';
    const languageScript = `window.__INITIAL_LANGUAGE__ = ${JSON.stringify(savedLanguage)};`;

    const html = `<!DOCTYPE html>
      <html lang="zh-CN">
      <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src ${webview.cspSource} 'unsafe-inline'; script-src 'nonce-${nonce}' ${webview.cspSource};">
        <link href="${styleUri}" rel="stylesheet">
        <title>CoCursor 仪表板</title>
      </head>
      <body>
        <div id="root">加载中...</div>
        <script nonce="${nonce}">
          window.__INITIAL_ROUTE__ = "${this.initialRoute}";
          window.__VIEW_TYPE__ = "${this._viewType}";
          ${workspacePathScript}
          ${languageScript}
        </script>
        <script nonce="${nonce}" src="${scriptUri}"></script>
      </body>
      </html>`;
    
    return html;
  }

  public dispose(): void {
    // 根据类型清除对应的静态变量
    if (this._viewType === "workAnalysis") {
      WebviewPanel.workAnalysisPanel = undefined;
    } else if (this._viewType === "recentSessions") {
      WebviewPanel.recentSessionsPanel = undefined;
    } else if (this._viewType === "marketplace") {
      WebviewPanel.marketplacePanel = undefined;
    } else if (this._viewType === "ragSearch") {
      WebviewPanel.ragSearchPanel = undefined;
    } else if (this._viewType === "workflow") {
      WebviewPanel.workflowPanel = undefined;
    } else if (this._viewType === "team") {
      WebviewPanel.teamPanel = undefined;
    }

    // 清理资源
    while (this._disposables.length) {
      const disposable = this._disposables.pop();
      if (disposable) {
        disposable.dispose();
      }
    }

    // 销毁面板
    this._panel.dispose();
  }
}

function getNonce(): string {
  let text = "";
  const possible = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  for (let i = 0; i < 32; i++) {
    text += possible.charAt(Math.floor(Math.random() * possible.length));
  }
  return text;
}
