import * as vscode from "vscode";
import { WebviewPanel } from "./webviewPanel";
import { SidebarProvider } from "./sidebar/sidebarProvider";

let statusBarItem: vscode.StatusBarItem;
let sidebarProvider: SidebarProvider;

export function activate(context: vscode.ExtensionContext): void {
  // 使用多个日志输出，确保能看到
  console.log("========================================");
  console.log("CoCursor Extension 已激活！");
  console.log("Extension URI:", context.extensionUri.toString());
  console.log("========================================");
  
  // 显示通知确认激活
  vscode.window.showInformationMessage("CoCursor 扩展已激活！", "打开仪表板").then((selection) => {
    if (selection === "打开仪表板") {
      vscode.commands.executeCommand("cocursor.openDashboard");
    }
  });

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

  // 读取配置
  const config = vscode.workspace.getConfiguration("cocursor");
  const autoStartServer = config.get<boolean>("autoStartServer", true);

  if (autoStartServer) {
    startBackendServer(context);
  }

  // 注册命令
  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.openDashboard", () => {
      console.log("命令 cocursor.openDashboard 被调用");
      try {
        WebviewPanel.createOrShow(context.extensionUri);
        console.log("WebviewPanel.createOrShow 调用成功");
      } catch (error) {
        console.error("打开仪表板失败:", error);
        vscode.window.showErrorMessage(`打开仪表板失败: ${error instanceof Error ? error.message : String(error)}`);
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

  console.log("CoCursor: 所有命令已注册完成");
}

function startBackendServer(context: vscode.ExtensionContext): void {
  // TODO: 启动后端服务器
  console.log("启动后端服务器...");
}

export function deactivate(): void {
  // 清理资源
  if (statusBarItem) {
    statusBarItem.dispose();
  }
}
