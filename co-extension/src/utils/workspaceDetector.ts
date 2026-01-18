import * as vscode from "vscode";

/**
 * 获取当前 VSCode 工作区路径
 * @returns 工作区路径，如果没有打开的工作区则返回 null
 */
export function getCurrentWorkspacePath(): string | null {
  const workspaceFolders = vscode.workspace.workspaceFolders;

  if (!workspaceFolders || workspaceFolders.length === 0) {
    return null;
  }

  // 使用第一个工作区的路径（支持多工作区可扩展）
  const workspaceUri = workspaceFolders[0].uri;
  return workspaceUri.fsPath; // 转换为文件系统路径
}

/**
 * 监听工作区变化
 * @param callback 工作区变化时的回调函数
 * @returns Disposable，用于清理监听器
 */
export function watchWorkspaceChanges(
  callback: (path: string) => void
): vscode.Disposable {
  return vscode.workspace.onDidChangeWorkspaceFolders(() => {
    const newPath = getCurrentWorkspacePath();
    if (newPath) {
      callback(newPath);
    }
  });
}

/**
 * 规范化路径（统一分隔符、去除尾部斜杠等）
 * @param path 原始路径
 * @returns 规范化后的路径
 */
export function normalizePath(path: string): string {
  // 统一使用正斜杠（跨平台）
  const normalized = path.replace(/\\/g, "/");
  // 移除尾部斜杠
  return normalized.replace(/\/+$/, "");
}
