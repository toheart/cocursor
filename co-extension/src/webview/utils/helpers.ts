/**
 * 共享工具函数
 */

/**
 * 格式化日期时间
 */
export const formatDate = (timestamp: number): string => {
  const date = new Date(timestamp);
  return date.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  });
};

/**
 * 格式化日期（完整格式）
 */
export const formatDateFull = (timestamp: number): string => {
  const date = new Date(timestamp);
  return date.toLocaleString("zh-CN");
};

/**
 * 获取熵值对应的颜色
 */
export const getEntropyColor = (entropy: number): string => {
  if (entropy < 40) {
    return "var(--vscode-testing-iconPassed)"; // 绿色
  } else if (entropy < 70) {
    return "var(--vscode-testing-iconQueued)"; // 黄色
  } else {
    return "var(--vscode-testing-iconFailed)"; // 红色
  }
};

/**
 * 获取熵值状态文本
 */
export const getEntropyStatusText = (status: string): string => {
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
};

/**
 * 获取工作区路径
 */
export const getWorkspacePath = (): string => {
  const workspacePath = window.__WORKSPACE_PATH__;
  if (!workspacePath) {
    console.warn("Workspace path not found, using current directory");
    return "";
  }
  return workspacePath;
};

/**
 * 安全的 JSON 解析
 */
export const safeJsonParse = <T>(json: string, defaultValue: T): T => {
  try {
    return JSON.parse(json) as T;
  } catch (err) {
    console.error("JSON parse error:", err);
    return defaultValue;
  }
};

/**
 * 防抖函数
 */
export const debounce = <T extends (...args: unknown[]) => void>(
  func: T,
  wait: number
): ((...args: Parameters<T>) => void) => {
  let timeout: NodeJS.Timeout | null = null;
  
  return function executedFunction(...args: Parameters<T>) {
    const later = () => {
      timeout = null;
      func(...args);
    };
    
    if (timeout) {
      clearTimeout(timeout);
    }
    timeout = setTimeout(later, wait);
  };
};

/**
 * 计算周的起止日期
 */
export const getWeekRange = (weeksAgo: number): { start: string; end: string } => {
  const today = new Date();
  const dayOfWeek = today.getDay(); // 0 = 周日, 1 = 周一, ...
  const mondayOffset = dayOfWeek === 0 ? -6 : 1 - dayOfWeek; // 调整到周一
  
  const targetDate = new Date(today);
  targetDate.setDate(today.getDate() + mondayOffset - (weeksAgo * 7));
  
  const weekStart = new Date(targetDate);
  const weekEnd = new Date(targetDate);
  weekEnd.setDate(weekStart.getDate() + 6);
  
  return {
    start: weekStart.toISOString().split('T')[0],
    end: weekEnd.toISOString().split('T')[0]
  };
};
