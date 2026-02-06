import * as vscode from "vscode";
import * as path from "path";
import * as fs from "fs";
import { spawn, ChildProcess } from "child_process";
import axios, { AxiosInstance } from "axios";
import { Logger } from "../utils/logger";

/**
 * DaemonManager 负责启动和管理后端进程
 *
 * 生命周期管理策略：
 * - 使用心跳机制维护窗口活跃状态
 * - 窗口关闭时只停止心跳，不主动关闭后端
 * - 后端通过心跳超时自动退出（所有窗口关闭 5 分钟后）
 */
export class DaemonManager {
  private process: ChildProcess | null = null;
  private healthCheckTimer: NodeJS.Timeout | null = null;
  private heartbeatTimer: NodeJS.Timeout | null = null;
  private readonly healthCheckInterval = 5000; // 5秒健康检查
  private readonly heartbeatInterval = 30000; // 30秒心跳间隔
  private readonly healthCheckUrl = "http://localhost:19960/health";
  private readonly heartbeatUrl = "http://localhost:19960/api/v1/heartbeat";
  private readonly context: vscode.ExtensionContext;
  private axiosInstance: AxiosInstance;
  private readonly windowId: string; // 唯一窗口标识

  constructor(context: vscode.ExtensionContext) {
    this.context = context;
    this.axiosInstance = axios.create({
      timeout: 2000, // 2秒超时
    });
    // 使用 vscode.env.sessionId 作为窗口唯一标识
    this.windowId = vscode.env.sessionId;
    Logger.backendDebug(`Window ID: ${this.windowId}`);
  }

  /**
   * 启动后端服务器
   */
  async start(): Promise<void> {
    if (this.process && !this.process.killed) {
      Logger.backendInfo("后端进程已在运行");
      // 确保心跳仍在运行
      this.startHeartbeat();
      return;
    }

    try {
      const binaryPath = this.findBinary();
      if (!binaryPath) {
        // 同时输出到两个通道，确保用户能看到
        Logger.error("找不到后端二进制文件");
        Logger.backendError("找不到后端二进制文件，请检查安装");
        throw new Error("找不到后端二进制文件");
      }

      Logger.backendInfo(`启动后端进程: ${binaryPath}`);
      this.process = spawn(binaryPath, [], {
        detached: false,
        stdio: "pipe",
      });

      // 处理进程输出（解析 slog 格式）
      this.process.stdout?.on("data", (data) => {
        Logger.backendStdout(data.toString());
      });

      this.process.stderr?.on("data", (data) => {
        Logger.backendStderr(data.toString());
      });

      // 处理进程退出
      this.process.on("exit", (code, signal) => {
        if (code === 0) {
          Logger.backendInfo(
            `后端进程正常退出: code=${code}, signal=${signal}`,
          );
        } else {
          Logger.backendWarn(
            `后端进程异常退出: code=${code}, signal=${signal}`,
          );
        }
        this.process = null;
        this.stopHeartbeat();
      });

      this.process.on("error", (error) => {
        Logger.error(`启动后端进程失败: ${error.message}`);
        Logger.backendError(`启动后端进程失败: ${error.message}`);
        vscode.window.showErrorMessage(`启动后端服务器失败: ${error.message}`);
        this.process = null;
        this.stopHeartbeat();
      });

      // 等待进程启动后开始健康检查和心跳
      // 使用 setTimeout 而非 Promise，因为这是延迟启动，不阻塞启动流程
      setTimeout(() => {
        this.startHealthCheck();
        this.startHeartbeat();
      }, 2000); // 等待2秒让进程启动
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      Logger.error(`启动后端服务器失败: ${message}`);
      Logger.backendError(`启动后端服务器失败: ${message}`);
      vscode.window.showErrorMessage(`启动后端服务器失败: ${message}`);
      throw error;
    }
  }

  /**
   * 停止后端服务器（同步方法，用于 deactivate）
   *
   * 重要：窗口关闭时只停止心跳，不主动关闭后端
   * 后端通过心跳超时机制自动管理生命周期：
   * - 所有窗口心跳超时后 5 分钟自动退出
   * - 避免多窗口场景下误关其他窗口正在使用的后端
   */
  stop(): void {
    Logger.backendInfo("窗口关闭，停止心跳（后端将在空闲后自动退出）");
    this.stopHeartbeat();
    this.stopHealthCheck();

    // 清理本地进程引用（不终止后端进程）
    // 后端会通过心跳超时机制自动退出
    this.process = null;
  }

  /**
   * 异步停止后端服务器（用于需要等待的场景）
   *
   * 重要：窗口关闭时只停止心跳，不主动关闭后端
   * 后端通过心跳超时机制自动管理生命周期
   */
  async stopAsync(): Promise<void> {
    Logger.backendInfo("窗口关闭，停止心跳（后端将在空闲后自动退出）");
    this.stopHeartbeat();
    this.stopHealthCheck();

    // 清理本地进程引用（不终止后端进程）
    this.process = null;
  }

  /**
   * 查找后端二进制文件
   * 开发环境：backend/bin/cocursor(.exe)
   * 生产环境：extension/bin/cocursor-{platform}-{arch}(.exe)
   */
  private findBinary(): string | null {
    const platform = process.platform;
    const arch = process.arch;
    const isWindows = platform === "win32";
    const ext = isWindows ? ".exe" : "";

    // 1. 开发环境：查找 backend/bin/cocursor(.exe)
    const devPath = path.join(
      this.context.extensionPath,
      "..",
      "backend",
      "bin",
      `cocursor${ext}`,
    );
    if (fs.existsSync(devPath)) {
      Logger.backendDebug(`找到开发环境二进制: ${devPath}`);
      return devPath;
    }

    // 2. 生产环境：查找 extension/bin/cocursor-{platform}-{arch}(.exe)
    const platformMap: Record<string, string> = {
      win32: "windows",
      darwin: "darwin",
      linux: "linux",
    };
    const archMap: Record<string, string> = {
      x64: "amd64",
      arm64: "arm64",
    };

    const platformName = platformMap[platform] || platform;
    const archName = archMap[arch] || arch;
    const binaryName = `cocursor-${platformName}-${archName}${ext}`;
    const prodPath = path.join(this.context.extensionPath, "bin", binaryName);

    if (fs.existsSync(prodPath)) {
      Logger.backendDebug(`找到生产环境二进制: ${prodPath}`);
      return prodPath;
    }

    // 3. 尝试查找 bin 目录下的其他可能名称
    const binDir = path.join(this.context.extensionPath, "bin");
    if (fs.existsSync(binDir)) {
      const possibleNames = [
        `cocursor${ext}`,
        `cocursordaemon${ext}`,
        binaryName,
      ];

      for (const name of possibleNames) {
        const fullPath = path.join(binDir, name);
        if (fs.existsSync(fullPath)) {
          Logger.backendDebug(`找到二进制文件: ${fullPath}`);
          return fullPath;
        }
      }
    }

    Logger.backendError("找不到后端二进制文件");
    Logger.backendError(`开发路径: ${devPath}`);
    Logger.backendError(`生产路径: ${prodPath}`);
    return null;
  }

  /**
   * 开始健康检查（检测后端是否存活）
   */
  private startHealthCheck(): void {
    if (this.healthCheckTimer) {
      return; // 已经在运行
    }

    Logger.backendDebug("开始健康检查");
    this.healthCheckTimer = setInterval(async () => {
      await this.checkHealth();
    }, this.healthCheckInterval);
  }

  /**
   * 停止健康检查
   */
  private stopHealthCheck(): void {
    if (this.healthCheckTimer) {
      clearInterval(this.healthCheckTimer);
      this.healthCheckTimer = null;
      Logger.backendDebug("停止健康检查");
    }
  }

  /**
   * 开始心跳发送（向后端注册窗口活跃状态）
   */
  private startHeartbeat(): void {
    if (this.heartbeatTimer) {
      return; // 已经在运行
    }

    Logger.backendDebug("开始心跳发送");

    // 立即发送一次心跳
    this.sendHeartbeat();

    // 然后定期发送
    this.heartbeatTimer = setInterval(async () => {
      await this.sendHeartbeat();
    }, this.heartbeatInterval);
  }

  /**
   * 停止心跳发送
   */
  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
      Logger.backendDebug("停止心跳发送");
    }
  }

  /**
   * 发送心跳到后端
   */
  private async sendHeartbeat(): Promise<void> {
    try {
      const projectPath =
        vscode.workspace.workspaceFolders?.[0]?.uri.fsPath || "";
      const response = await this.axiosInstance.post(this.heartbeatUrl, {
        window_id: this.windowId,
        project_path: projectPath,
      });

      if (response.status === 200) {
        const data = response.data?.data;
        Logger.backendTrace(
          `心跳发送成功，活跃窗口数: ${data?.active_windows || "unknown"}`,
        );
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      Logger.backendWarn(`心跳发送失败: ${message}`);
    }
  }

  /**
   * 检查后端健康状态
   */
  private async checkHealth(): Promise<void> {
    try {
      const response = await this.axiosInstance.get(this.healthCheckUrl);
      if (response.status === 200) {
        Logger.backendTrace("后端健康检查成功");
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      Logger.backendWarn(`后端健康检查失败: ${message}`);
      // 如果进程已退出，清理资源
      if (this.process && this.process.killed) {
        this.stopHeartbeat();
        this.stopHealthCheck();
        this.process = null;
      }
    }
  }

  /**
   * 检查后端是否正在运行
   */
  async isRunning(): Promise<boolean> {
    try {
      const response = await this.axiosInstance.get(this.healthCheckUrl);
      return response.status === 200;
    } catch {
      return false;
    }
  }

  /**
   * 确保心跳正在运行（用于后端已存在的场景）
   * 当后端由其他窗口/进程启动时，当前窗口也需要发送心跳
   */
  ensureHeartbeat(): void {
    this.startHeartbeat();
    this.startHealthCheck();
    Logger.backendDebug("确保心跳运行（后端已存在）");
  }
}
