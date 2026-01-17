// API 服务 - 通过 Extension 代理调用后端 API

import { ExtensionMessage } from "../../types/message";

class ApiService {
  private vscode: ReturnType<typeof acquireVsCodeApi>;

  constructor() {
    this.vscode = acquireVsCodeApi();
  }

  // 发送消息到 Extension
  private postMessage(command: string, payload?: unknown): Promise<unknown> {
    return new Promise((resolve, reject) => {
      const messageId = `${command}-${Date.now()}-${Math.random()}`;
      const handler = (event: MessageEvent<ExtensionMessage>) => {
        if (event.data.type === `${command}-response`) {
          window.removeEventListener("message", handler);
          if (event.data.data && typeof event.data.data === "object" && "error" in event.data.data) {
            reject(new Error(String(event.data.data.error)));
          } else {
            resolve(event.data.data);
          }
        }
      };

      window.addEventListener("message", handler);
      this.vscode.postMessage({ command, payload, messageId });

      // 超时处理
      setTimeout(() => {
        window.removeEventListener("message", handler);
        reject(new Error("Request timeout"));
      }, 30000);
    });
  }

  // 获取对话列表
  async getChats(): Promise<unknown> {
    return this.postMessage("fetchChats");
  }

  // 获取对话详情
  async getChatDetail(chatId: string): Promise<unknown> {
    return this.postMessage("fetchChatDetail", { chatId });
  }

  // 获取节点列表
  async getPeers(): Promise<unknown> {
    return this.postMessage("getPeers");
  }

  // 加入团队
  async joinTeam(teamCode: string): Promise<unknown> {
    return this.postMessage("joinTeam", { teamCode });
  }
}

export const apiService = new ApiService();
