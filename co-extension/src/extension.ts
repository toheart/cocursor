import * as vscode from "vscode";
import axios from "axios";
import { WebviewPanel } from "./webviewPanel";
import { SidebarProvider } from "./sidebar/sidebarProvider";
import { DaemonManager } from "./daemon/daemonManager";
import { ReminderService } from "./reminder/reminderService";
import { TodoModule } from "./todo";
import { checkAndReportProject } from "./utils/projectReporter";
import { watchWorkspaceChanges } from "./utils/workspaceDetector";
import { initI18n } from "./utils/i18n";
import { Logger } from "./utils/logger";

let statusBarItem: vscode.StatusBarItem;
let sidebarProvider: SidebarProvider;
let daemonManager: DaemonManager | null = null;
let reminderService: ReminderService | null = null;
let todoModule: TodoModule | null = null;
let windowStateListener: vscode.Disposable | null = null;
let activeEditorListener: vscode.Disposable | null = null;
let statusReportThrottle: NodeJS.Timeout | null = null;
let lastReportedStatus: { project: string; file: string } | null = null;

export function activate(context: vscode.ExtensionContext): void {
  // 初始化 Logger（优先初始化，确保后续日志可见）
  Logger.init(context);

  // 自动显示两个 Output 面板（不抢占焦点）
  Logger.showMain(true);
  Logger.showBackend(true);

  // 初始化 i18n
  initI18n(context);

  // 输出扩展激活日志
  Logger.info("========================================");
  Logger.info("CoCursor Extension 已激活！");
  Logger.info(`Extension URI: ${context.extensionUri.toString()}`);
  Logger.info("========================================");

  // 只在首次激活时显示通知
  const isFirstActivation = context.globalState.get<boolean>(
    "cocursor.firstActivation",
    true,
  );
  if (isFirstActivation) {
    vscode.window
      .showInformationMessage("CoCursor 扩展已激活！", "打开仪表板")
      .then((selection) => {
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
    100,
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
    Logger.debug(`检测到工作区变化: ${newPath}`);
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
      Logger.debug("命令 cocursor.openDashboard 被调用");
      try {
        WebviewPanel.createOrShow(
          context.extensionUri,
          "workAnalysis",
          context,
          "/",
        );
        Logger.debug("WebviewPanel.createOrShow 调用成功");
      } catch (error) {
        Logger.error(
          `打开工作分析失败: ${error instanceof Error ? error.message : String(error)}`,
        );
        vscode.window.showErrorMessage(
          `打开工作分析失败: ${error instanceof Error ? error.message : String(error)}`,
        );
      }
    }),
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.refreshTasks", () => {
      // 刷新所有 Webview 面板的数据
      WebviewPanel.notifyRefresh();
      Logger.debug("已发送刷新通知到所有 Webview 面板");
    }),
  );

  // 刷新 Webview 数据的命令（支持指定数据类型）
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "cocursor.refreshWebview",
      (dataType?: string) => {
        WebviewPanel.notifyRefresh(undefined, dataType);
        Logger.debug(`已发送刷新通知 (dataType: ${dataType || "all"})`);
      },
    ),
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.addTask", () => {
      vscode.window
        .showInputBox({
          prompt: "输入任务名称",
          placeHolder: "例如：完成项目文档",
        })
        .then((taskName) => {
          if (taskName) {
            vscode.window.showInformationMessage(`添加任务: ${taskName}`);
          }
        });
    }),
  );

  // 注册侧边栏提供者
  sidebarProvider = new SidebarProvider(context);
  context.subscriptions.push(
    vscode.window.registerTreeDataProvider("cocursor.sidebar", sidebarProvider),
  );

  // 注册侧边栏相关命令
  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.refreshSidebar", () => {
      sidebarProvider.refresh();
    }),
  );

  // 注册侧边栏语言刷新命令
  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.refreshSidebarLanguage", () => {
      sidebarProvider.refreshLanguage();
    }),
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openWorkAnalysis", () => {
      WebviewPanel.createOrShow(
        context.extensionUri,
        "workAnalysis",
        context,
        "/",
      );
    }),
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openSessions", () => {
      WebviewPanel.createOrShow(
        context.extensionUri,
        "recentSessions",
        context,
      );
    }),
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openMarketplace", () => {
      WebviewPanel.createOrShow(context.extensionUri, "marketplace", context);
    }),
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openRAGSearch", () => {
      WebviewPanel.createOrShow(
        context.extensionUri,
        "ragSearch",
        context,
        "/",
      );
    }),
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openTeam", () => {
      WebviewPanel.createOrShow(context.extensionUri, "team", context, "/");
    }),
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openCodeAnalysis", () => {
      WebviewPanel.createOrShow(context.extensionUri, "codeAnalysis", context, "/");
    }),
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.showPeers", () => {
      vscode.window.showInformationMessage("显示节点列表功能开发中...");
    }),
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.showStats", () => {
      vscode.window.showInformationMessage("显示使用统计功能开发中...");
    }),
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.showCurrentTeam", () => {
      vscode.window.showInformationMessage("显示当前团队功能开发中...");
    }),
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.joinTeam", () => {
      vscode.window
        .showInputBox({
          prompt: "输入团队码",
          placeHolder: "例如: ABC123",
        })
        .then((teamCode) => {
          if (teamCode) {
            vscode.window.showInformationMessage(`加入团队: ${teamCode}`);
          }
        });
    }),
  );

  // 监听活动编辑器变化，用于工作状态上报
  const statusSharingEnabled = context.globalState.get<boolean>(
    "cocursor.statusSharingEnabled",
    false,
  );
  if (statusSharingEnabled) {
    activeEditorListener = vscode.window.onDidChangeActiveTextEditor(
      (editor) => {
        if (editor) {
          throttledReportWorkStatus(editor);
        }
      },
    );
    context.subscriptions.push(activeEditorListener);
  }

  // 注册状态分享开关命令
  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.toggleStatusSharing", () => {
      const current = context.globalState.get<boolean>(
        "cocursor.statusSharingEnabled",
        false,
      );
      const newValue = !current;
      context.globalState.update("cocursor.statusSharingEnabled", newValue);

      if (newValue && !activeEditorListener) {
        activeEditorListener = vscode.window.onDidChangeActiveTextEditor(
          (editor) => {
            if (editor) {
              throttledReportWorkStatus(editor);
            }
          },
        );
        context.subscriptions.push(activeEditorListener);
        vscode.window.showInformationMessage("工作状态分享已开启");
      } else if (!newValue && activeEditorListener) {
        activeEditorListener.dispose();
        activeEditorListener = null;
        vscode.window.showInformationMessage("工作状态分享已关闭");
      }
    }),
  );

  // 初始化并启动提醒服务
  reminderService = new ReminderService(context);
  reminderService.start();

  // 监听提醒配置变化
  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration("cocursor.reminder")) {
        Logger.debug("提醒配置已变更");
      }
    }),
  );

  // 初始化待办事项模块
  todoModule = new TodoModule();
  todoModule.activate(context).catch((err) => {
    Logger.warn(`待办模块初始化失败: ${err instanceof Error ? err.message : String(err)}`);
  });

  Logger.info("所有命令已注册完成");
}

// 节流上报工作状态（30 秒）
function throttledReportWorkStatus(editor: vscode.TextEditor): void {
  const workspaceFolder = vscode.workspace.getWorkspaceFolder(
    editor.document.uri,
  );
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
      const teamsResp = await axios.get(
        "http://localhost:19960/api/v1/team/list",
        { timeout: 5000 },
      );
      // API 返回格式: { code: 0, message: "success", data: { teams: [...], total: N } }
      const teams = teamsResp.data?.data?.teams || [];

      // 向所有加入的团队上报状态
      for (const team of teams) {
        await axios.post(
          `http://localhost:19960/api/v1/team/${team.id}/status`,
          {
            project_name: projectName,
            current_file: relativePath,
            status_visible: true,
          },
          { timeout: 5000 },
        );
      }

      lastReportedStatus = { project: projectName, file: relativePath };
    } catch (error) {
      // 静默失败
      Logger.debug(
        `工作状态上报失败: ${error instanceof Error ? error.message : String(error)}`,
      );
    }
  }, 30000); // 30 秒节流
}

async function startBackendServer(
  _context: vscode.ExtensionContext,
): Promise<void> {
  if (!daemonManager) {
    Logger.error("DaemonManager 未初始化");
    return;
  }

  try {
    // 先检查是否已有实例运行
    const isRunning = await daemonManager.isRunning();
    if (isRunning) {
      Logger.info("后端服务器已在运行");
      return;
    }

    // 启动后端服务器
    await daemonManager.start();
    Logger.info("后端服务器启动成功");

    // 等待服务器完全启动（给一点时间）
    await new Promise((resolve) => setTimeout(resolve, 1000));
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    Logger.error(`启动后端服务器失败: ${message}`);
    vscode.window.showErrorMessage(`启动后端服务器失败: ${message}`);
  }
}

// 注册工作区
async function registerWorkspace(): Promise<void> {
  try {
    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (!workspaceFolders || workspaceFolders.length === 0) {
      Logger.debug("没有打开的工作区，跳过注册");
      return;
    }

    const fsPath = workspaceFolders[0].uri.fsPath;
    Logger.info(`注册工作区: ${fsPath}`);

    // 调用后端 API 注册工作区
    const response = await axios.post(
      "http://localhost:19960/api/v1/workspace/register",
      { path: fsPath },
      { timeout: 5000 },
    );

    if (response.data.workspaceID) {
      Logger.info(`工作区注册成功，WorkspaceID: ${response.data.workspaceID}`);
    }
  } catch (error) {
    // 静默失败，不阻塞扩展激活
    const message = error instanceof Error ? error.message : String(error);
    Logger.warn(`工作区注册失败（可能后端未启动）: ${message}`);
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
    Logger.debug(`更新工作区焦点: ${fsPath}`);

    // 调用后端 API 更新焦点
    await axios.post(
      "http://localhost:19960/api/v1/workspace/focus",
      { path: fsPath },
      { timeout: 5000 },
    );
  } catch (error) {
    // 静默失败
    const message = error instanceof Error ? error.message : String(error);
    Logger.debug(`更新工作区焦点失败: ${message}`);
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

  // 停止提醒服务
  if (reminderService) {
    reminderService.stop();
    reminderService = null;
  }

  // 销毁待办模块
  if (todoModule) {
    todoModule.dispose();
    todoModule = null;
  }

  // 停止后端服务器
  if (daemonManager) {
    daemonManager.stop();
    daemonManager = null;
  }
}
