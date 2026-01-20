import * as vscode from "vscode";
import axios from "axios";
import { WebviewPanel } from "./webviewPanel";
import { SidebarProvider } from "./sidebar/sidebarProvider";
import { DaemonManager } from "./daemon/daemonManager";
import { checkAndReportProject } from "./utils/projectReporter";
import { watchWorkspaceChanges } from "./utils/workspaceDetector";
import { initI18n } from "./utils/i18n";

let statusBarItem: vscode.StatusBarItem;
let sidebarProvider: SidebarProvider;
let daemonManager: DaemonManager | null = null;
let windowStateListener: vscode.Disposable | null = null;
let activeEditorListener: vscode.Disposable | null = null;
let statusReportThrottle: NodeJS.Timeout | null = null;
let lastReportedStatus: { project: string; file: string } | null = null;

export function activate(context: vscode.ExtensionContext): void {
  // 初始化 i18n
  initI18n(context);
  
  // 使用多个日志输出，确保能看到
  console.log("========================================");
  console.log("CoCursor Extension 已激活！");
  console.log("Extension URI:", context.extensionUri.toString());
  console.log("========================================");
  
  // 只在首次激活时显示通知
  const isFirstActivation = context.globalState.get<boolean>("cocursor.firstActivation", true);
  if (isFirstActivation) {
    vscode.window.showInformationMessage("CoCursor 扩展已激活！", "打开仪表板").then((selection) => {
      if (selection === "打开仪表板") {
        vscode.commands.executeCommand("cocursor.openDashboard");
      }
    });
    // 标记为已激活过
    context.globalState.update("cocursor.firstActivation", false);
  }

  // 创建状态栏项
  statusBarItem = vscode.window.createStatusBarItem(
    vscode.StatusBarAlignment.Right,
    100
  );
  statusBarItem.text = "$(check) CoCursor";
  statusBarItem.tooltip = "点击打开 CoCursor 仪表板";
  statusBarItem.command = "cocursor.openDashboard";
  statusBarItem.show();
  context.subscriptions.push(statusBarItem);

  // 初始化 DaemonManager
  daemonManager = new DaemonManager(context);

  // 读取配置
  const config = vscode.workspace.getConfiguration("cocursor");
  const autoStartServer = config.get<boolean>("autoStartServer", true);

  // MCP 服务器配置已移至后端，后端启动时会自动配置
  // 不再需要在前端配置

  if (autoStartServer) {
    startBackendServer(context).then(() => {
      // 后端启动后注册工作区
      registerWorkspace();
      // 检测并上报当前项目
      checkAndReportProject();
    });
  } else {
    // 即使不自动启动，也尝试注册（可能后端已手动启动）
    registerWorkspace();
    // 检测并上报当前项目
    checkAndReportProject();
  }

  // 监听工作区变化
  const workspaceWatcher = watchWorkspaceChanges((newPath) => {
    console.log(`CoCursor: 检测到工作区变化: ${newPath}`);
    checkAndReportProject();
  });
  context.subscriptions.push(workspaceWatcher);

  // 监听窗口焦点变化
  windowStateListener = vscode.window.onDidChangeWindowState((e) => {
    if (e.focused) {
      updateWorkspaceFocus();
      // 焦点变化时也上报项目（确保状态同步）
      checkAndReportProject();
    }
  });
  context.subscriptions.push(windowStateListener);

  // 注册命令
  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openDashboard", () => {
      console.log("命令 cocursor.openDashboard 被调用");
      try {
        WebviewPanel.createOrShow(context.extensionUri, "workAnalysis", context, "/");
        console.log("WebviewPanel.createOrShow 调用成功");
      } catch (error) {
        console.error("打开工作分析失败:", error);
        vscode.window.showErrorMessage(`打开工作分析失败: ${error instanceof Error ? error.message : String(error)}`);
      }
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.refreshTasks", () => {
      vscode.window.showInformationMessage("刷新任务列表");
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.addTask", () => {
      vscode.window
        .showInputBox({
          prompt: "输入任务名称",
          placeHolder: "例如：完成项目文档"
        })
        .then((taskName) => {
          if (taskName) {
            vscode.window.showInformationMessage(`添加任务: ${taskName}`);
          }
        });
    })
  );

  // 注册侧边栏提供者
  sidebarProvider = new SidebarProvider(context);
  context.subscriptions.push(
    vscode.window.registerTreeDataProvider("cocursor.sidebar", sidebarProvider)
  );

  // 注册侧边栏相关命令
  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.refreshSidebar", () => {
      sidebarProvider.refresh();
    })
  );

  // 注册侧边栏语言刷新命令
  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.refreshSidebarLanguage", () => {
      sidebarProvider.refreshLanguage();
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openWorkAnalysis", () => {
      WebviewPanel.createOrShow(context.extensionUri, "workAnalysis", context, "/");
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openSessions", () => {
      WebviewPanel.createOrShow(context.extensionUri, "recentSessions", context);
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openMarketplace", () => {
      WebviewPanel.createOrShow(context.extensionUri, "marketplace", context);
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openRAGSearch", () => {
      WebviewPanel.createOrShow(context.extensionUri, "ragSearch", context, "/");
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openWorkflows", () => {
      WebviewPanel.createOrShow(context.extensionUri, "workflow", context, "/");
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openTeam", () => {
      WebviewPanel.createOrShow(context.extensionUri, "team", context, "/");
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.showPeers", () => {
      vscode.window.showInformationMessage("显示节点列表功能开发中...");
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.showStats", () => {
      vscode.window.showInformationMessage("显示使用统计功能开发中...");
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.showCurrentTeam", () => {
      vscode.window.showInformationMessage("显示当前团队功能开发中...");
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.joinTeam", () => {
      vscode.window
        .showInputBox({
          prompt: "输入团队码",
          placeHolder: "例如: ABC123"
        })
        .then((teamCode) => {
          if (teamCode) {
            vscode.window.showInformationMessage(`加入团队: ${teamCode}`);
          }
        });
    })
  );

  // 注册代码分享命令（右键菜单和命令面板）
  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.shareCodeToTeam", () => {
      shareSelectedCode();
    })
  );

  // 监听活动编辑器变化，用于工作状态上报
  const statusSharingEnabled = context.globalState.get<boolean>("cocursor.statusSharingEnabled", false);
  if (statusSharingEnabled) {
    activeEditorListener = vscode.window.onDidChangeActiveTextEditor((editor) => {
      if (editor) {
        throttledReportWorkStatus(editor);
      }
    });
    context.subscriptions.push(activeEditorListener);
  }

  // 注册状态分享开关命令
  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.toggleStatusSharing", () => {
      const current = context.globalState.get<boolean>("cocursor.statusSharingEnabled", false);
      const newValue = !current;
      context.globalState.update("cocursor.statusSharingEnabled", newValue);

      if (newValue && !activeEditorListener) {
        activeEditorListener = vscode.window.onDidChangeActiveTextEditor((editor) => {
          if (editor) {
            throttledReportWorkStatus(editor);
          }
        });
        context.subscriptions.push(activeEditorListener);
        vscode.window.showInformationMessage("工作状态分享已开启");
      } else if (!newValue && activeEditorListener) {
        activeEditorListener.dispose();
        activeEditorListener = null;
        vscode.window.showInformationMessage("工作状态分享已关闭");
      }
    })
  );

  console.log("CoCursor: 所有命令已注册完成");
}

// 分享选中的代码到团队
async function shareSelectedCode(): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showWarningMessage("请先打开一个文件并选择代码");
    return;
  }

  const selection = editor.selection;
  if (selection.isEmpty) {
    vscode.window.showWarningMessage("请先选择要分享的代码");
    return;
  }

  const selectedText = editor.document.getText(selection);
  if (!selectedText.trim()) {
    vscode.window.showWarningMessage("选中的代码为空");
    return;
  }

  // 检查代码大小（10KB 限制）
  if (selectedText.length > 10 * 1024) {
    vscode.window.showWarningMessage("选中的代码过大（超过 10KB），请选择较小的片段");
    return;
  }

  // 获取文件信息
  const fileName = editor.document.fileName.split(/[\\/]/).pop() || "unknown";
  const workspaceFolder = vscode.workspace.getWorkspaceFolder(editor.document.uri);
  const relativePath = workspaceFolder
    ? vscode.workspace.asRelativePath(editor.document.uri)
    : fileName;
  const language = editor.document.languageId;
  const startLine = selection.start.line + 1;
  const endLine = selection.end.line + 1;

  // 询问附加消息（可选）
  const message = await vscode.window.showInputBox({
    prompt: "添加说明（可选）",
    placeHolder: "如：这段代码有问题，帮我看看",
  });

  try {
    // 调用后端 API 分享代码
    // 需要先获取当前加入的团队 ID
    const teamsResp = await axios.get("http://localhost:19960/api/v1/team/list", { timeout: 5000 });
    const teams = teamsResp.data?.teams || [];

    if (teams.length === 0) {
      vscode.window.showWarningMessage("您尚未加入任何团队，请先加入或创建团队");
      return;
    }

    // 如果加入了多个团队，让用户选择
    let teamId: string;
    if (teams.length === 1) {
      teamId = teams[0].id;
    } else {
      const selected = await vscode.window.showQuickPick(
        teams.map((t: { id: string; name: string }) => ({ label: t.name, id: t.id })),
        { placeHolder: "选择要分享到的团队" }
      );
      if (!selected) {
        return;
      }
      teamId = selected.id;
    }

    // 调用分享 API
    await axios.post(
      `http://localhost:19960/api/v1/team/${teamId}/share-code`,
      {
        file_name: fileName,
        file_path: relativePath,
        language: language,
        start_line: startLine,
        end_line: endLine,
        code: selectedText,
        message: message || "",
      },
      { timeout: 10000 }
    );

    vscode.window.showInformationMessage(`代码已分享到团队 (${fileName}:${startLine}-${endLine})`);
  } catch (error) {
    const errorMsg = error instanceof Error ? error.message : String(error);
    vscode.window.showErrorMessage(`分享代码失败: ${errorMsg}`);
  }
}

// 节流上报工作状态（30 秒）
function throttledReportWorkStatus(editor: vscode.TextEditor): void {
  const workspaceFolder = vscode.workspace.getWorkspaceFolder(editor.document.uri);
  const projectName = workspaceFolder?.name || "unknown";
  const relativePath = workspaceFolder
    ? vscode.workspace.asRelativePath(editor.document.uri)
    : editor.document.fileName.split(/[\\/]/).pop() || "unknown";

  // 检查是否有变化
  if (
    lastReportedStatus &&
    lastReportedStatus.project === projectName &&
    lastReportedStatus.file === relativePath
  ) {
    return;
  }

  // 清除之前的节流定时器
  if (statusReportThrottle) {
    clearTimeout(statusReportThrottle);
  }

  // 设置节流
  statusReportThrottle = setTimeout(async () => {
    try {
      // 获取当前加入的团队
      const teamsResp = await axios.get("http://localhost:19960/api/v1/team/list", { timeout: 5000 });
      const teams = teamsResp.data?.teams || [];

      // 向所有加入的团队上报状态
      for (const team of teams) {
        await axios.post(
          `http://localhost:19960/api/v1/team/${team.id}/status`,
          {
            project_name: projectName,
            current_file: relativePath,
            status_visible: true,
          },
          { timeout: 5000 }
        );
      }

      lastReportedStatus = { project: projectName, file: relativePath };
    } catch (error) {
      // 静默失败
      console.log("CoCursor: 工作状态上报失败:", error instanceof Error ? error.message : String(error));
    }
  }, 30000); // 30 秒节流
}

async function startBackendServer(_context: vscode.ExtensionContext): Promise<void> {
  if (!daemonManager) {
    console.error("DaemonManager 未初始化");
    return;
  }

  try {
    // 先检查是否已有实例运行
    const isRunning = await daemonManager.isRunning();
    if (isRunning) {
      console.log("后端服务器已在运行");
      return;
    }

    // 启动后端服务器
    await daemonManager.start();
    console.log("后端服务器启动成功");
    
    // 等待服务器完全启动（给一点时间）
    await new Promise((resolve) => setTimeout(resolve, 1000));
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.error(`启动后端服务器失败: ${message}`);
    vscode.window.showErrorMessage(`启动后端服务器失败: ${message}`);
  }
}

// 注册工作区
async function registerWorkspace(): Promise<void> {
  try {
    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (!workspaceFolders || workspaceFolders.length === 0) {
      console.log("CoCursor: 没有打开的工作区，跳过注册");
      return;
    }

    const fsPath = workspaceFolders[0].uri.fsPath;
    console.log(`CoCursor: 注册工作区: ${fsPath}`);

    // 调用后端 API 注册工作区
    const response = await axios.post(
      "http://localhost:19960/api/v1/workspace/register",
      { path: fsPath },
      { timeout: 5000 }
    );

    if (response.data.workspaceID) {
      console.log(`CoCursor: 工作区注册成功，WorkspaceID: ${response.data.workspaceID}`);
    }
  } catch (error) {
    // 静默失败，不阻塞扩展激活
    const message = error instanceof Error ? error.message : String(error);
    console.log(`CoCursor: 工作区注册失败（可能后端未启动）: ${message}`);
  }
}

// 更新工作区焦点
async function updateWorkspaceFocus(): Promise<void> {
  try {
    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (!workspaceFolders || workspaceFolders.length === 0) {
      return;
    }

    const fsPath = workspaceFolders[0].uri.fsPath;
    console.log(`CoCursor: 更新工作区焦点: ${fsPath}`);

      // 调用后端 API 更新焦点
      await axios.post(
        "http://localhost:19960/api/v1/workspace/focus",
        { path: fsPath },
        { timeout: 5000 }
      );
      
  } catch (error) {
    // 静默失败
    const message = error instanceof Error ? error.message : String(error);
    console.log(`CoCursor: 更新工作区焦点失败: ${message}`);
  }
}

export function deactivate(): void {
  // 清理资源
  if (statusBarItem) {
    statusBarItem.dispose();
  }

  // 清理窗口状态监听器
  if (windowStateListener) {
    windowStateListener.dispose();
    windowStateListener = null;
  }

  // 清理编辑器监听器
  if (activeEditorListener) {
    activeEditorListener.dispose();
    activeEditorListener = null;
  }

  // 清理节流定时器
  if (statusReportThrottle) {
    clearTimeout(statusReportThrottle);
    statusReportThrottle = null;
  }

  // 停止后端服务器
  if (daemonManager) {
    daemonManager.stop();
    daemonManager = null;
  }
}
