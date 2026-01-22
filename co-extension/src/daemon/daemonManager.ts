import * as vscode from "vscode";
import * as path from "path";
import * as fs from "fs";
import { spawn, ChildProcess, execSync } from "child_process";
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
  private readonly shutdownUrl = "http://localhost:19960/api/v1/shutdown";
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
   * 停止后端服务器（同步方法，用于 deactivate）
   */
  stop(): void {
    this.stopHeartbeat();

    // 首先尝试通过 HTTP API 优雅关闭
    this.sendShutdownRequest();

    // 然后强制终止进程
    this.killProcess();

    // 清理可能的残留进程（通过端口查找）
    this.killProcessByPort(19960);
  }

  /**
   * 异步停止后端服务器（用于需要等待的场景）
   */
  async stopAsync(): Promise<void> {
    this.stopHeartbeat();

    // 首先尝试通过 HTTP API 优雅关闭
    await this.sendShutdownRequestAsync();

    // 等待进程退出
    if (this.process && !this.process.killed) {
      const exitPromise = new Promise<void>((resolve) => {
        const timeout = setTimeout(() => {
          Logger.backendWarn("等待进程退出超时，强制终止");
          resolve();
        }, 3000); // 最多等待 3 秒

        this.process?.once("exit", () => {
          clearTimeout(timeout);
          resolve();
        });
      });

      // 发送 SIGTERM
      this.process.kill("SIGTERM");
      await exitPromise;

      // 如果进程仍在运行，强制终止
      if (this.process && !this.process.killed) {
        this.killProcess();
      }
    }

    this.process = null;

    // 清理可能的残留进程
    this.killProcessByPort(19960);
  }

  /**
   * 同步发送关闭请求（不等待响应）
   */
  private sendShutdownRequest(): void {
    try {
      // 使用 XMLHttpRequest 同步请求（Node.js 环境下使用 execSync + curl）
      if (process.platform === "win32") {
        execSync(`curl -s -X POST "${this.shutdownUrl}" --max-time 1`, {
          stdio: "ignore",
          timeout: 2000,
        });
      } else {
        execSync(`curl -s -X POST "${this.shutdownUrl}" --max-time 1 2>/dev/null || true`, {
          stdio: "ignore",
          timeout: 2000,
        });
      }
      Logger.backendInfo("已发送关闭请求到后端");
    } catch {
      // 忽略错误，可能后端已经关闭
      Logger.backendDebug("发送关闭请求失败（后端可能已关闭）");
    }
  }

  /**
   * 异步发送关闭请求
   */
  private async sendShutdownRequestAsync(): Promise<void> {
    try {
      await this.axiosInstance.post(this.shutdownUrl, {}, { timeout: 2000 });
      Logger.backendInfo("已发送关闭请求到后端");
      // 等待一小段时间让后端处理关闭
      await new Promise((resolve) => setTimeout(resolve, 500));
    } catch {
      // 忽略错误，可能后端已经关闭
      Logger.backendDebug("发送关闭请求失败（后端可能已关闭）");
    }
  }

  /**
   * 强制终止当前管理的进程
   */
  private killProcess(): void {
    if (!this.process) {
      return;
    }

    const pid = this.process.pid;
    if (!pid) {
      this.process = null;
      return;
    }

    Logger.backendInfo(`强制终止进程 PID=${pid}`);

    try {
      if (process.platform === "win32") {
        // Windows: 使用 taskkill 强制终止进程树（同步执行）
        try {
          execSync(`taskkill /F /T /PID ${pid}`, {
            stdio: "ignore",
            timeout: 5000,
          });
        } catch {
          // 进程可能已经退出
        }
      } else {
        // Unix/Linux/Mac: 先尝试 SIGTERM，再 SIGKILL
        try {
          this.process.kill("SIGTERM");
          // 给进程一点时间响应 SIGTERM
          execSync("sleep 0.5", { stdio: "ignore" });
        } catch {
          // 忽略
        }

        // 检查进程是否仍在运行，如果是则发送 SIGKILL
        try {
          process.kill(pid, 0); // 测试进程是否存在
          this.process.kill("SIGKILL");
          Logger.backendInfo(`已发送 SIGKILL 到进程 PID=${pid}`);
        } catch {
          // 进程已经不存在，这是好事
          Logger.backendDebug(`进程 PID=${pid} 已退出`);
        }
      }
    } catch (error) {
      Logger.backendWarn(`终止进程失败: ${error instanceof Error ? error.message : String(error)}`);
    }

    this.process = null;
  }

  /**
   * 通过端口查找并终止占用该端口的进程
   */
  private killProcessByPort(port: number): void {
    try {
      if (process.platform === "win32") {
        // Windows: 使用 netstat 查找并终止
        try {
          const result = execSync(`netstat -ano | findstr :${port} | findstr LISTENING`, {
            encoding: "utf-8",
            timeout: 5000,
          });
          const lines = result.trim().split("\n");
          for (const line of lines) {
            const parts = line.trim().split(/\s+/);
            const pid = parts[parts.length - 1];
            if (pid && /^\d+$/.test(pid)) {
              Logger.backendInfo(`通过端口 ${port} 发现残留进程 PID=${pid}，正在终止`);
              execSync(`taskkill /F /PID ${pid}`, { stdio: "ignore", timeout: 5000 });
            }
          }
        } catch {
          // 没有找到占用端口的进程，这是正常的
        }
      } else {
        // Unix/Linux/Mac: 使用 lsof 查找并终止
        try {
          const result = execSync(`lsof -ti:${port}`, {
            encoding: "utf-8",
            timeout: 5000,
          });
          const pids = result.trim().split("\n").filter((p) => p && /^\d+$/.test(p));
          for (const pid of pids) {
            Logger.backendInfo(`通过端口 ${port} 发现残留进程 PID=${pid}，正在终止`);
            try {
              execSync(`kill -9 ${pid}`, { stdio: "ignore", timeout: 2000 });
            } catch {
              // 进程可能已经退出
            }
          }
        } catch {
          // 没有找到占用端口的进程，这是正常的
        }
      }
    } catch (error) {
      Logger.backendDebug(`端口清理失败: ${error instanceof Error ? error.message : String(error)}`);
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
