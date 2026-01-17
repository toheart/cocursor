// Webview 与 Extension 之间的消息类型定义

// Webview -> Extension
export interface WebviewMessage {
  command: string;
  payload?: unknown;
  messageId?: string;
}

// Extension -> Webview
export interface ExtensionMessage {
  type: string;
  data?: unknown;
}

// 消息命令类型
export type MessageCommand =
  | "openDashboard"
  | "refreshTasks"
  | "addTask"
  | "fetchChats"
  | "fetchChatDetail"
  | "joinTeam"
  | "getPeers";
