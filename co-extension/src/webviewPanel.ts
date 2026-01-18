import * as vscode from "vscode";
import axios from "axios";
import { WebviewMessage, ExtensionMessage } from "./types/message";

export type WebviewType = "workAnalysis" | "recentSessions" | "marketplace";

export class WebviewPanel {
  public static workAnalysisPanel: WebviewPanel | undefined;
  public static recentSessionsPanel: WebviewPanel | undefined;
  public static marketplacePanel: WebviewPanel | undefined;
  private readonly _panel: vscode.WebviewPanel;
  private readonly _extensionUri: vscode.Uri;
  private readonly _viewType: WebviewType;
  private _disposables: vscode.Disposable[] = [];

  private initialRoute: string = "/";
  // 项目列表缓存（避免重复请求）
  private static projectListCache: {
    projects: Array<{ project_name: string; workspaces: Array<{ path: string }> }>;
    timestamp: number;
  } | null = null;
  private static readonly CACHE_TTL = 5000; // 5秒缓存

  private constructor(panel: vscode.WebviewPanel, extensionUri: vscode.Uri, viewType: WebviewType, route?: string) {
    this._panel = panel;
    this._extensionUri = extensionUri;
    this._viewType = viewType;
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

  public static createOrShow(extensionUri: vscode.Uri, viewType: WebviewType, route?: string): void {
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
      "插件市场 - CoCursor";
    const panelId = 
      viewType === "workAnalysis" ? "cocursorWorkAnalysis" :
      viewType === "recentSessions" ? "cocursorRecentSessions" :
      "cocursorMarketplace";
    
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

    const newPanel = new WebviewPanel(panel, extensionUri, viewType, route);
    
    // 根据类型保存到对应的静态变量
    if (viewType === "workAnalysis") {
      WebviewPanel.workAnalysisPanel = newPanel;
    } else if (viewType === "recentSessions") {
      WebviewPanel.recentSessionsPanel = newPanel;
    } else if (viewType === "marketplace") {
      WebviewPanel.marketplacePanel = newPanel;
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
      case "joinTeam":
        this._handleJoinTeam(message.payload as { teamCode: string });
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
      case "fetchPlugins":
        this._handleFetchPlugins(message.payload as { category?: string; search?: string; installed?: boolean });
        break;
      case "fetchPlugin":
        this._handleFetchPlugin(message.payload as { id: string });
        break;
      case "fetchInstalledPlugins":
        this._handleFetchInstalledPlugins();
        break;
      case "installPlugin":
        this._handleInstallPlugin(message.payload as { id: string; workspacePath: string });
        break;
      case "uninstallPlugin":
        this._handleUninstallPlugin(message.payload as { id: string; workspacePath: string });
        break;
      case "checkPluginStatus":
        this._handleCheckPluginStatus(message.payload as { id: string });
        break;
      default:
        console.warn(`未知命令: ${message.command}`);
    }
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

  private async _handleJoinTeam(payload: { teamCode: string }): Promise<void> {
    try {
      // TODO: 调用后端 API
      const response = { code: 0, data: null, message: "success" };
      this._sendMessage({
        type: "joinTeam-response",
        data: response
      });
    } catch (error) {
      this._sendMessage({
        type: "joinTeam-response",
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

  private async _handleFetchWorkAnalysis(payload: { startDate?: string; endDate?: string; projectName?: string }): Promise<void> {
    try {
      let apiUrl = "http://localhost:19960/api/v1/stats/work-analysis";
      const params = new URLSearchParams();
      if (payload.startDate) {
        params.append("start_date", payload.startDate);
      }
      if (payload.endDate) {
        params.append("end_date", payload.endDate);
      }
      if (payload.projectName) {
        params.append("project_name", payload.projectName);
      }
      if (params.toString()) {
        apiUrl += `?${params.toString()}`;
      }

      const response = await axios.get(apiUrl, { timeout: 10000 });
      
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
        WebviewPanel.createOrShow(this._extensionUri, "workAnalysis");
      }
    });
  }

  private async _handleFetchPlugins(payload: { category?: string; search?: string; installed?: boolean }): Promise<void> {
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

  private async _handleFetchPlugin(payload: { id: string }): Promise<void> {
    try {
      const apiUrl = `http://localhost:19960/api/v1/marketplace/plugins/${encodeURIComponent(payload.id)}`;
      const response = await axios.get(apiUrl, { timeout: 10000 });

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

  private async _handleFetchInstalledPlugins(): Promise<void> {
    try {
      const apiUrl = "http://localhost:19960/api/v1/marketplace/installed";
      const response = await axios.get(apiUrl, { timeout: 10000 });

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

  private async _handleInstallPlugin(payload: { id: string; workspacePath: string }): Promise<void> {
    try {
      const apiUrl = `http://localhost:19960/api/v1/marketplace/plugins/${encodeURIComponent(payload.id)}/install`;
      const response = await axios.post(apiUrl, {
        workspace_path: payload.workspacePath
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
