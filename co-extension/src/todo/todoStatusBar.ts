// 待办事项状态栏管理

import * as vscode from "vscode";
import { TodoItem } from "./types";

/**
 * 待办事项状态栏
 */
export class TodoStatusBar {
  private statusBarItem: vscode.StatusBarItem;

  constructor() {
    // 创建状态栏项目，放在左侧
    this.statusBarItem = vscode.window.createStatusBarItem(
      vscode.StatusBarAlignment.Left,
      100 // 优先级
    );
    
    // 设置命令：点击状态栏打开 Quick Pick
    this.statusBarItem.command = "cocursor.todoQuickPick";
    this.statusBarItem.tooltip = "待办事项";
    
    // 初始显示
    this.update([]);
    this.statusBarItem.show();
  }

  /**
   * 更新状态栏显示
   */
  update(todos: TodoItem[]): void {
    const total = todos.length;
    const completed = todos.filter(t => t.completed).length;
    
    if (total === 0) {
      this.statusBarItem.text = "$(checklist) 0";
    } else {
      this.statusBarItem.text = `$(checklist) ${completed}/${total}`;
    }
  }

  /**
   * 销毁状态栏
   */
  dispose(): void {
    this.statusBarItem.dispose();
  }
}
