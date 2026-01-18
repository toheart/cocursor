/**
 * 会话健康状态工具函数
 */

import { SessionHealth } from "../types";

/**
 * 获取熵值对应的颜色
 */
export function getEntropyColor(entropy: number): string {
  if (entropy < 40) {
    return "var(--vscode-testing-iconPassed)"; // 绿色
  } else if (entropy < 70) {
    return "var(--vscode-testing-iconQueued)"; // 黄色
  } else {
    return "var(--vscode-testing-iconFailed)"; // 红色
  }
}

/**
 * 获取熵值状态的文本
 */
export function getEntropyStatusText(status: string): string {
  switch (status) {
    case "healthy":
      return "健康";
    case "sub_healthy":
      return "亚健康";
    case "dangerous":
      return "危险";
    default:
      return "未知";
  }
}

/**
 * 检查是否需要警告
 */
export function shouldWarnAboutEntropy(
  previousEntropy: number | null,
  currentEntropy: number
): boolean {
  const wasHealthy = previousEntropy === null || previousEntropy < 70;
  const isDangerous = currentEntropy >= 70;
  return wasHealthy && isDangerous;
}
