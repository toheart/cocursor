/**
 * WebSocket 连接状态全局管理
 * 统一管理所有团队的 WebSocket 连接，提供响应式状态
 */

import { create } from "zustand";
import {
  teamWebSocket,
  TeamEvent,
  TeamEventHandler,
  MemberStatusChangedEvent,
} from "../../../services/teamWebSocket";
import { useMemberStore } from "./memberStore";

interface WsState {
  /** 各团队的连接状态（响应式） */
  connectionStates: Record<string, boolean>;
  /** 当前活跃连接的团队 ID */
  activeTeamId: string | null;
  /** 延迟断开的定时器 */
  disconnectTimers: Record<string, ReturnType<typeof setTimeout>>;

  /** 连接到团队 WebSocket（进入详情页时调用） */
  connectTeam: (teamId: string, leaderEndpoint: string) => void;
  /** 断开团队 WebSocket（离开详情页时调用，带延迟） */
  disconnectTeam: (teamId: string) => void;
  /** 立即断开团队 WebSocket */
  disconnectTeamImmediate: (teamId: string) => void;
  /** 断开所有连接 */
  disconnectAll: () => void;
  /** 检查连接状态 */
  isConnected: (teamId: string) => boolean;
  /** 更新连接状态 */
  setConnectionState: (teamId: string, connected: boolean) => void;
}

// 延迟断开时间（5秒，防止快速切换时频繁重连）
const DISCONNECT_DELAY = 5000;

export const useWsStore = create<WsState>((set, get) => ({
  connectionStates: {},
  activeTeamId: null,
  disconnectTimers: {},

  connectTeam: (teamId, leaderEndpoint) => {
    const state = get();

    // 如果有延迟断开的定时器，先取消
    if (state.disconnectTimers[teamId]) {
      clearTimeout(state.disconnectTimers[teamId]);
      const { [teamId]: _, ...rest } = state.disconnectTimers;
      set({ disconnectTimers: rest });
    }

    set({ activeTeamId: teamId });

    // 如果已连接，不重复连接
    if (teamWebSocket.isConnected(teamId)) {
      set((s) => ({
        connectionStates: { ...s.connectionStates, [teamId]: true },
      }));
      return;
    }

    // 连接
    teamWebSocket.connect(teamId, leaderEndpoint);

    // 注册连接状态监测
    teamWebSocket.on(teamId, (_event: TeamEvent) => {
      // 收到任何消息说明连接正常
      if (!get().connectionStates[teamId]) {
        set((s) => ({
          connectionStates: { ...s.connectionStates, [teamId]: true },
        }));
      }
    });

    // 轮询检测连接状态（每 5 秒）
    const checkInterval = setInterval(() => {
      const connected = teamWebSocket.isConnected(teamId);
      const current = get().connectionStates[teamId];
      if (connected !== current) {
        set((s) => ({
          connectionStates: { ...s.connectionStates, [teamId]: connected },
        }));
      }
      // 如果已不再是活跃团队，停止检测
      if (get().activeTeamId !== teamId) {
        clearInterval(checkInterval);
      }
    }, 5000);

    // 初始标记为已连接（乐观更新）
    set((s) => ({
      connectionStates: { ...s.connectionStates, [teamId]: true },
    }));
  },

  disconnectTeam: (teamId) => {
    const state = get();

    // 清除活跃标记
    if (state.activeTeamId === teamId) {
      set({ activeTeamId: null });
    }

    // 延迟断开
    const timer = setTimeout(() => {
      get().disconnectTeamImmediate(teamId);
      const { [teamId]: _, ...rest } = get().disconnectTimers;
      set({ disconnectTimers: rest });
    }, DISCONNECT_DELAY);

    set((s) => ({
      disconnectTimers: { ...s.disconnectTimers, [teamId]: timer },
    }));
  },

  disconnectTeamImmediate: (teamId) => {
    teamWebSocket.disconnect(teamId);
    set((s) => ({
      connectionStates: { ...s.connectionStates, [teamId]: false },
    }));
  },

  disconnectAll: () => {
    const state = get();
    // 清除所有延迟断开定时器
    Object.values(state.disconnectTimers).forEach(clearTimeout);
    teamWebSocket.disconnectAll();
    set({
      connectionStates: {},
      activeTeamId: null,
      disconnectTimers: {},
    });
  },

  isConnected: (teamId) => {
    return get().connectionStates[teamId] ?? false;
  },

  setConnectionState: (teamId, connected) => {
    set((s) => ({
      connectionStates: { ...s.connectionStates, [teamId]: connected },
    }));
  },
}));

/**
 * 团队事件处理器：将 WS 事件分发到对应的 Store
 */
export function createTeamEventHandler(teamId: string): TeamEventHandler {
  return (event: TeamEvent) => {
    const memberStore = useMemberStore.getState();

    switch (event.type) {
      case "member_joined":
      case "member_left":
      case "member_online":
      case "member_offline":
        // 成员变化时刷新成员列表
        if (memberStore.currentTeamId === teamId) {
          memberStore.fetchMembers(teamId);
        }
        break;

      case "skill_published":
      case "skill_deleted":
      case "skill_index_updated":
        // 技能变化时刷新技能列表
        if (memberStore.currentTeamId === teamId) {
          memberStore.fetchSkills(teamId);
        }
        break;

      case "member_status_changed": {
        const statusEvent = event as MemberStatusChangedEvent;
        if (statusEvent.payload.status_visible) {
          memberStore.updateMemberStatus(statusEvent.payload.member_id, {
            project: statusEvent.payload.project_name,
            file: statusEvent.payload.current_file,
          });
        } else {
          memberStore.updateMemberStatus(
            statusEvent.payload.member_id,
            null,
          );
        }
        break;
      }

      default:
        break;
    }
  };
}
