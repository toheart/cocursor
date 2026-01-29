// 待办事项模块入口

import * as vscode from "vscode";
import { TodoStatusBar } from "./todoStatusBar";
import { TodoCommands } from "./todoCommands";

/**
 * 待办事项模块
 */
export class TodoModule {
  private statusBar: TodoStatusBar;
  private commands: TodoCommands;
  private disposables: vscode.Disposable[] = [];

  constructor() {
    // 创建状态栏
    this.statusBar = new TodoStatusBar();
    
    // 创建命令管理器
    this.commands = new TodoCommands(this.statusBar);
  }

  /**
   * 激活模块
   */
  async activate(context: vscode.ExtensionContext): Promise<void> {
    // 注册命令：打开 Quick Pick
    const quickPickCommand = vscode.commands.registerCommand(
      "cocursor.todoQuickPick",
      () => this.commands.showQuickPick()
    );
    this.disposables.push(quickPickCommand);

    // 注册命令：直接添加待办
    const addCommand = vscode.commands.registerCommand(
      "cocursor.todoAdd",
      () => this.commands.addTodo()
    );
    this.disposables.push(addCommand);

    // 将 disposables 添加到 context
    context.subscriptions.push(...this.disposables);
    context.subscriptions.push(this.statusBar);

    // 初始加载数据
    await this.commands.refresh();
  }

  /**
   * 销毁模块
   */
  dispose(): void {
    this.disposables.forEach(d => d.dispose());
    this.statusBar.dispose();
  }
}

// 导出类型
export * from "./types";
