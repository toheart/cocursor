import * as vscode from "vscode";
import * as path from "path";
import * as fs from "fs";
import { spawn, ChildProcess } from "child_process";
import axios, { AxiosInstance } from "axios";
import { Logger } from "../utils/logger";

/**
 * DaemonManager 负责启动和管理后端进程
 */
export class DaemonManager {
  private process: ChildProcess | null = null;
  private heartbeatTimer: NodeJS.Timeout | null = null;
  private readonly healthCheckInterval = 5000; // 5秒
  private readonly healthCheckUrl = "http://localhost:19960/health";
  private readonly context: vscode.ExtensionContext;
  private axiosInstance: AxiosInstance;

  constructor(context: vscode.ExtensionContext) {
    this.context = context;
    this.axiosInstance = axios.create({
      timeout: 2000, // 2秒超时
    });
  }

  /**
   * 启动后端服务器
   */
  async start(): Promise<void> {
    if (this.process && !this.process.killed) {
      Logger.backendInfo("后端进程已在运行");
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
          Logger.backendInfo(`后端进程正常退出: code=${code}, signal=${signal}`);
        } else {
          Logger.backendWarn(`后端进程异常退出: code=${code}, signal=${signal}`);
        }
        this.process = null;
        this.stopHeartbeat();
      });

      this.process.on("error", (error) => {
        Logger.error(`启动后端进程失败: ${error.message}`);
        Logger.backendError(`启动后端进程失败: ${error.message}`);
        vscode.window.showErrorMessage(
          `启动后端服务器失败: ${error.message}`
        );
        this.process = null;
        this.stopHeartbeat();
      });

      // 等待进程启动后开始心跳检测
      // 使用 setTimeout 而非 Promise，因为这是延迟启动心跳，不阻塞启动流程
      setTimeout(() => {
        this.startHeartbeat();
      }, 2000); // 等待2秒让进程启动
    } catch (error) {
      const message =
        error instanceof Error ? error.message : String(error);
      Logger.error(`启动后端服务器失败: ${message}`);
      Logger.backendError(`启动后端服务器失败: ${message}`);
      vscode.window.showErrorMessage(`启动后端服务器失败: ${message}`);
      throw error;
    }
  }

  /**
   * 停止后端服务器
   */
  stop(): void {
    this.stopHeartbeat();

    if (this.process && !this.process.killed) {
      Logger.backendInfo("停止后端进程");
      // Windows 上使用 taskkill，Unix 上使用 kill
      if (process.platform === "win32") {
        // Windows: 终止进程树
        spawn("taskkill", ["/F", "/T", "/PID", this.process.pid!.toString()], {
          detached: true,
          stdio: "ignore",
        });
      } else {
        // Unix/Linux/Mac: 发送 SIGTERM
        this.process.kill("SIGTERM");
      }
      this.process = null;
    }
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
      `cocursor${ext}`
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
    const prodPath = path.join(
      this.context.extensionPath,
      "bin",
      binaryName
    );

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
   * 开始心跳检测
   */
  private startHeartbeat(): void {
    if (this.heartbeatTimer) {
      return; // 已经在运行
    }

    Logger.backendDebug("开始心跳检测");
    this.heartbeatTimer = setInterval(async () => {
      await this.checkHealth();
    }, this.healthCheckInterval);
  }

  /**
   * 停止心跳检测
   */
  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
      Logger.backendDebug("停止心跳检测");
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
}
