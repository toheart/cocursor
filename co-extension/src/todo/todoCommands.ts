// 待办事项命令（Quick Pick 交互）

import * as vscode from "vscode";
import { TodoItem } from "./types";
import { todoApi } from "./todoApi";
import { TodoStatusBar } from "./todoStatusBar";

/**
 * Quick Pick 项目类型
 */
interface TodoQuickPickItem extends vscode.QuickPickItem {
  action?: "add" | "clear" | "toggle";
  todo?: TodoItem;
}

/**
 * 待办事项命令管理器
 */
export class TodoCommands {
  private statusBar: TodoStatusBar;
  private todos: TodoItem[] = [];

  constructor(statusBar: TodoStatusBar) {
    this.statusBar = statusBar;
  }

  /**
   * 刷新待办列表
   */
  async refresh(): Promise<void> {
    this.todos = await todoApi.list();
    this.statusBar.update(this.todos);
  }

  /**
   * 打开 Quick Pick
   */
  async showQuickPick(): Promise<void> {
    // 先刷新数据
    await this.refresh();

    // 构建 Quick Pick 项目
    const items: TodoQuickPickItem[] = [];

    // 添加新待办选项
    items.push({
      label: "$(add) 添加新待办...",
      action: "add",
    });

    // 如果有已完成的待办，显示清除选项
    const hasCompleted = this.todos.some(t => t.completed);
    if (hasCompleted) {
      items.push({
        label: "$(trash) 清除已完成",
        action: "clear",
      });
    }

    // 添加分隔线（如果有待办项）
    if (this.todos.length > 0) {
      items.push({
        label: "",
        kind: vscode.QuickPickItemKind.Separator,
      });
    }

    // 添加待办项（未完成在前，已完成在后）
    const sortedTodos = [...this.todos].sort((a, b) => {
      // 未完成在前
      if (a.completed !== b.completed) {
        return a.completed ? 1 : -1;
      }
      // 按创建时间倒序
      return b.createdAt - a.createdAt;
    });

    for (const todo of sortedTodos) {
      const icon = todo.completed ? "$(pass-filled)" : "$(circle-large-outline)";
      items.push({
        label: `${icon} ${todo.content}`,
        description: todo.completed ? "已完成" : undefined,
        action: "toggle",
        todo,
      });
    }

    // 显示 Quick Pick
    const selected = await vscode.window.showQuickPick(items, {
      placeHolder: "待办事项",
      matchOnDescription: true,
    });

    if (!selected) {
      return;
    }

    // 处理选择
    switch (selected.action) {
      case "add":
        await this.addTodo();
        break;
      case "clear":
        await this.clearCompleted();
        break;
      case "toggle":
        if (selected.todo) {
          await this.toggleTodo(selected.todo);
        }
        break;
    }
  }

  /**
   * 添加新待办
   */
  async addTodo(): Promise<void> {
    const content = await vscode.window.showInputBox({
      placeHolder: "输入待办内容",
      prompt: "添加新待办",
    });

    if (!content || content.trim() === "") {
      return;
    }

    const todo = await todoApi.create(content.trim());
    if (todo) {
      vscode.window.showInformationMessage(`已添加待办: ${content.trim()}`);
      await this.refresh();
      // 添加后重新显示 Quick Pick
      await this.showQuickPick();
    } else {
      vscode.window.showErrorMessage("添加待办失败");
    }
  }

  /**
   * 切换待办状态
   */
  async toggleTodo(todo: TodoItem): Promise<void> {
    const updated = await todoApi.toggle(todo);
    if (updated) {
      await this.refresh();
      // 切换后重新显示 Quick Pick
      await this.showQuickPick();
    } else {
      vscode.window.showErrorMessage("更新待办失败");
    }
  }

  /**
   * 清除已完成待办
   */
  async clearCompleted(): Promise<void> {
    const count = await todoApi.deleteCompleted();
    if (count > 0) {
      vscode.window.showInformationMessage(`已清除 ${count} 个已完成待办`);
    }
    await this.refresh();
    // 清除后重新显示 Quick Pick
    await this.showQuickPick();
  }
}
