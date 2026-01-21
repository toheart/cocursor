import * as vscode from "vscode";

/**
 * CoCursor 日志管理器
 * 提供两个独立的 LogOutputChannel：
 * - CoCursor: 扩展前端管理状态
 * - CoCursor Backend: 后端服务日志
 * 
 * 使用 LogOutputChannel 支持日志级别过滤
 * 用户可通过命令面板 "Developer: Set Log Level..." 调整日志级别
 */
class LoggerManager {
  private static instance: LoggerManager;
  private mainChannel: vscode.LogOutputChannel | null = null;
  private backendChannel: vscode.LogOutputChannel | null = null;
  private initialized = false;

  private constructor() {}

  static getInstance(): LoggerManager {
    if (!LoggerManager.instance) {
      LoggerManager.instance = new LoggerManager();
    }
    return LoggerManager.instance;
  }

  /**
   * 初始化日志通道
   */
  init(context: vscode.ExtensionContext): void {
    if (this.initialized) {
      return;
    }

    // 创建主日志通道（扩展状态）
    // LogOutputChannel 支持 trace/debug/info/warn/error 级别
    this.mainChannel = vscode.window.createOutputChannel("CoCursor", { log: true });
    context.subscriptions.push(this.mainChannel);

    // 创建后端日志通道
    this.backendChannel = vscode.window.createOutputChannel("CoCursor Backend", { log: true });
    context.subscriptions.push(this.backendChannel);

    this.initialized = true;
  }

  /**
   * 显示主日志面板
   * @param preserveFocus 是否保持当前焦点（不抢占编辑器焦点）
   */
  showMain(preserveFocus = true): void {
    this.mainChannel?.show(preserveFocus);
  }

  /**
   * 显示后端日志面板
   * @param preserveFocus 是否保持当前焦点（不抢占编辑器焦点）
   */
  showBackend(preserveFocus = true): void {
    this.backendChannel?.show(preserveFocus);
  }

  // ========== 主日志通道方法 ==========

  /**
   * 输出 trace 级别日志到主通道
   */
  trace(message: string): void {
    this.mainChannel?.trace(message);
  }

  /**
   * 输出 debug 级别日志到主通道
   */
  debug(message: string): void {
    this.mainChannel?.debug(message);
  }

  /**
   * 输出 info 级别日志到主通道
   */
  info(message: string): void {
    this.mainChannel?.info(message);
  }

  /**
   * 输出 warn 级别日志到主通道
   */
  warn(message: string): void {
    this.mainChannel?.warn(message);
  }

  /**
   * 输出 error 级别日志到主通道
   */
  error(message: string): void {
    this.mainChannel?.error(message);
  }

  // ========== 后端日志通道方法 ==========

  /**
   * 输出 trace 级别日志到后端通道
   */
  backendTrace(message: string): void {
    this.backendChannel?.trace(message);
  }

  /**
   * 输出 debug 级别日志到后端通道
   */
  backendDebug(message: string): void {
    this.backendChannel?.debug(message);
  }

  /**
   * 输出 info 级别日志到后端通道
   */
  backendInfo(message: string): void {
    this.backendChannel?.info(message);
  }

  /**
   * 输出 warn 级别日志到后端通道
   */
  backendWarn(message: string): void {
    this.backendChannel?.warn(message);
  }

  /**
   * 输出 error 级别日志到后端通道
   */
  backendError(message: string): void {
    this.backendChannel?.error(message);
  }

  /**
   * 解析 slog 格式日志行
   * 格式: time=... level=INFO msg="message" key=value
   */
  private parseSlogLine(line: string): { level: string; message: string } {
    // 提取 level
    const levelMatch = line.match(/level=(\w+)/);
    const level = levelMatch ? levelMatch[1].toLowerCase() : "info";

    // 提取 msg（支持带引号和不带引号的格式）
    const msgMatch = line.match(/msg="([^"]+)"/);
    const message = msgMatch ? msgMatch[1] : line;

    return { level, message };
  }

  /**
   * 处理后端进程 stdout 输出
   * 解析 slog 格式并按正确级别输出
   */
  backendStdout(data: string): void {
    // 按行分割处理
    const lines = data.trim().split("\n");
    for (const line of lines) {
      const trimmedLine = line.trim();
      if (!trimmedLine) {
        continue;
      }

      // 解析 slog 格式
      const { level, message } = this.parseSlogLine(trimmedLine);

      // 根据级别输出到对应方法
      switch (level) {
        case "debug":
          this.backendChannel?.debug(message);
          break;
        case "info":
          this.backendChannel?.info(message);
          break;
        case "warn":
        case "warning":
          this.backendChannel?.warn(message);
          break;
        case "error":
          this.backendChannel?.error(message);
          break;
        default:
          // 未知级别默认为 info
          this.backendChannel?.info(message);
          break;
      }
    }
  }

  /**
   * 处理后端进程 stderr 输出
   * stderr 输出统一作为 error 级别
   */
  backendStderr(data: string): void {
    const lines = data.trim().split("\n");
    for (const line of lines) {
      const trimmedLine = line.trim();
      if (trimmedLine) {
        this.backendChannel?.error(trimmedLine);
      }
    }
  }
}

// 导出单例
export const Logger = LoggerManager.getInstance();
