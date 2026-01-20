/**
 * 团队 WebSocket 服务
 * 用于接收团队事件的实时通知
 */

import { TeamSkillEntry, TeamMember } from "../types";

// WebSocket 事件类型
export type TeamEventType =
  | "member_joined"
  | "member_left"
  | "member_online"
  | "member_offline"
  | "skill_published"
  | "skill_deleted"
  | "skill_index_updated"
  | "team_dissolved"
  | "pong";

// WebSocket 事件
export interface TeamEvent {
  type: TeamEventType;
  team_id: string;
  payload?: unknown;
  timestamp: string;
}

// 事件处理器类型
export type TeamEventHandler = (event: TeamEvent) => void;

// 成员加入/离开事件
export interface MemberEvent extends TeamEvent {
  type: "member_joined" | "member_left" | "member_online" | "member_offline";
  payload: {
    member_id: string;
    member_name: string;
  };
}

// 技能发布/删除事件
export interface SkillEvent extends TeamEvent {
  type: "skill_published" | "skill_deleted";
  payload: {
    plugin_id: string;
    skill?: TeamSkillEntry;
  };
}

// 技能目录更新事件
export interface SkillIndexEvent extends TeamEvent {
  type: "skill_index_updated";
  payload: {
    skills: TeamSkillEntry[];
  };
}

// 团队解散事件
export interface TeamDissolvedEvent extends TeamEvent {
  type: "team_dissolved";
  payload: {
    reason?: string;
  };
}

/**
 * 团队 WebSocket 管理器
 */
class TeamWebSocketManager {
  private connections: Map<string, WebSocket> = new Map();
  private handlers: Map<string, Set<TeamEventHandler>> = new Map();
  private reconnectAttempts: Map<string, number> = new Map();
  private maxReconnectAttempts = 5;
  private reconnectDelay = 3000;
  private pingInterval = 30000;
  private pingTimers: Map<string, NodeJS.Timer> = new Map();

  /**
   * 连接到团队 WebSocket
   */
  connect(teamId: string, leaderEndpoint: string): void {
    if (this.connections.has(teamId)) {
      console.log(`[TeamWS] Already connected to team ${teamId}`);
      return;
    }

    const wsUrl = `ws://${leaderEndpoint.replace(/^https?:\/\//, "")}/team/${teamId}/ws`;
    console.log(`[TeamWS] Connecting to ${wsUrl}`);

    try {
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log(`[TeamWS] Connected to team ${teamId}`);
        this.reconnectAttempts.set(teamId, 0);
        this.startPing(teamId, ws);
      };

      ws.onmessage = (event) => {
        try {
          const teamEvent: TeamEvent = JSON.parse(event.data);
          this.handleEvent(teamId, teamEvent);
        } catch (error) {
          console.error(`[TeamWS] Failed to parse message:`, error);
        }
      };

      ws.onerror = (error) => {
        console.error(`[TeamWS] Error for team ${teamId}:`, error);
      };

      ws.onclose = (event) => {
        console.log(`[TeamWS] Disconnected from team ${teamId}, code: ${event.code}`);
        this.connections.delete(teamId);
        this.stopPing(teamId);

        // 尝试重连（除非是正常关闭）
        if (event.code !== 1000 && event.code !== 1001) {
          this.attemptReconnect(teamId, leaderEndpoint);
        }
      };

      this.connections.set(teamId, ws);
    } catch (error) {
      console.error(`[TeamWS] Failed to connect to team ${teamId}:`, error);
    }
  }

  /**
   * 断开团队 WebSocket
   */
  disconnect(teamId: string): void {
    const ws = this.connections.get(teamId);
    if (ws) {
      ws.close(1000, "User disconnected");
      this.connections.delete(teamId);
      this.stopPing(teamId);
      this.reconnectAttempts.delete(teamId);
    }
  }

  /**
   * 断开所有连接
   */
  disconnectAll(): void {
    for (const teamId of this.connections.keys()) {
      this.disconnect(teamId);
    }
    this.handlers.clear();
  }

  /**
   * 注册事件处理器
   */
  on(teamId: string, handler: TeamEventHandler): () => void {
    if (!this.handlers.has(teamId)) {
      this.handlers.set(teamId, new Set());
    }
    this.handlers.get(teamId)!.add(handler);

    // 返回取消注册函数
    return () => {
      const handlers = this.handlers.get(teamId);
      if (handlers) {
        handlers.delete(handler);
      }
    };
  }

  /**
   * 注册全局事件处理器（接收所有团队事件）
   */
  onGlobal(handler: TeamEventHandler): () => void {
    return this.on("*", handler);
  }

  /**
   * 检查是否已连接
   */
  isConnected(teamId: string): boolean {
    const ws = this.connections.get(teamId);
    return ws !== undefined && ws.readyState === WebSocket.OPEN;
  }

  /**
   * 处理事件
   */
  private handleEvent(teamId: string, event: TeamEvent): void {
    console.log(`[TeamWS] Event received:`, event.type, event);

    // 调用特定团队的处理器
    const teamHandlers = this.handlers.get(teamId);
    if (teamHandlers) {
      teamHandlers.forEach((handler) => handler(event));
    }

    // 调用全局处理器
    const globalHandlers = this.handlers.get("*");
    if (globalHandlers) {
      globalHandlers.forEach((handler) => handler(event));
    }
  }

  /**
   * 尝试重连
   */
  private attemptReconnect(teamId: string, leaderEndpoint: string): void {
    const attempts = this.reconnectAttempts.get(teamId) || 0;
    if (attempts >= this.maxReconnectAttempts) {
      console.log(`[TeamWS] Max reconnect attempts reached for team ${teamId}`);
      return;
    }

    this.reconnectAttempts.set(teamId, attempts + 1);
    const delay = this.reconnectDelay * Math.pow(2, attempts);

    console.log(`[TeamWS] Reconnecting to team ${teamId} in ${delay}ms (attempt ${attempts + 1})`);

    setTimeout(() => {
      this.connect(teamId, leaderEndpoint);
    }, delay);
  }

  /**
   * 启动心跳
   */
  private startPing(teamId: string, ws: WebSocket): void {
    const timer = setInterval(() => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "ping" }));
      }
    }, this.pingInterval);

    this.pingTimers.set(teamId, timer);
  }

  /**
   * 停止心跳
   */
  private stopPing(teamId: string): void {
    const timer = this.pingTimers.get(teamId);
    if (timer) {
      clearInterval(timer);
      this.pingTimers.delete(teamId);
    }
  }
}

// 导出单例
export const teamWebSocket = new TeamWebSocketManager();
