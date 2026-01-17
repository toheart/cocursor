import * as vscode from "vscode";
import axios from "axios";
import { WebviewMessage, ExtensionMessage } from "./types/message";

export class WebviewPanel {
  public static currentPanel: WebviewPanel | undefined;
  private readonly _panel: vscode.WebviewPanel;
  private readonly _extensionUri: vscode.Uri;
  private _disposables: vscode.Disposable[] = [];

  private constructor(panel: vscode.WebviewPanel, extensionUri: vscode.Uri) {
    this._panel = panel;
    this._extensionUri = extensionUri;

    console.log("WebviewPanel: 创建新面板", extensionUri.toString());

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

  public static createOrShow(extensionUri: vscode.Uri): void {
    console.log("WebviewPanel: createOrShow 被调用", extensionUri.toString());
    
    const column = vscode.window.activeTextEditor
      ? vscode.window.activeTextEditor.viewColumn
      : undefined;

    // 如果已经有面板，显示它
    if (WebviewPanel.currentPanel) {
      console.log("WebviewPanel: 使用现有面板");
      WebviewPanel.currentPanel._panel.reveal(column);
      return;
    }

    // 创建新面板
    console.log("WebviewPanel: 创建新面板");
    const panel = vscode.window.createWebviewPanel(
      "cocursorDashboard",
      "CoCursor 仪表板",
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

    WebviewPanel.currentPanel = new WebviewPanel(panel, extensionUri);
    console.log("WebviewPanel: 面板创建完成");
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
        this._handleFetchCurrentSessionHealth(message.payload as { projectPath?: string });
        break;
      case "showEntropyWarning":
        this._handleShowEntropyWarning(message.payload as { entropy: number; message: string });
        break;
      default:
        console.warn(`未知命令: ${message.command}`);
    }
  }

  private async _handleFetchChats(): Promise<void> {
    try {
      // TODO: 调用后端 API
      const response = { code: 0, data: [], message: "success" };
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

  private async _handleFetchCurrentSessionHealth(payload: { projectPath?: string }): Promise<void> {
    try {
      // 获取当前工作区路径
      const workspaceFolders = vscode.workspace.workspaceFolders;
      let projectPath = payload.projectPath;
      
      if (!projectPath && workspaceFolders && workspaceFolders.length > 0) {
        projectPath = workspaceFolders[0].uri.fsPath;
      }

      // 调用后端 API
      const apiUrl = `http://localhost:19960/api/v1/stats/current-session${projectPath ? `?project_path=${encodeURIComponent(projectPath)}` : ""}`;
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

  private _handleShowEntropyWarning(payload: { entropy: number; message: string }): void {
    // 显示 VS Code 警告通知
    vscode.window.showWarningMessage(
      `⚠️ ${payload.message} (熵值: ${payload.entropy.toFixed(2)})`,
      "查看详情"
    ).then((selection) => {
      if (selection === "查看详情") {
        // 显示面板（如果已关闭则创建）
        WebviewPanel.createOrShow(this._extensionUri);
      }
    });
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
        <script nonce="${nonce}" src="${scriptUri}"></script>
      </body>
      </html>`;
    
    return html;
  }

  public dispose(): void {
    WebviewPanel.currentPanel = undefined;

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
