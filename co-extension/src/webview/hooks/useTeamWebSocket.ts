/**
 * 团队 WebSocket Hook
 * 用于在 React 组件中管理团队 WebSocket 连接
 */

import { useEffect, useCallback, useRef } from "react";
import { teamWebSocket, TeamEvent, TeamEventHandler } from "../services/teamWebSocket";

interface UseTeamWebSocketOptions {
  teamId: string;
  leaderEndpoint?: string;
  onEvent?: TeamEventHandler;
  enabled?: boolean;
}

/**
 * 管理单个团队的 WebSocket 连接
 */
export function useTeamWebSocket({
  teamId,
  leaderEndpoint,
  onEvent,
  enabled = true,
}: UseTeamWebSocketOptions) {
  const handlerRef = useRef(onEvent);

  // 保持 handler 引用最新
  useEffect(() => {
    handlerRef.current = onEvent;
  }, [onEvent]);

  // 连接/断开 WebSocket
  useEffect(() => {
    if (!enabled || !teamId || !leaderEndpoint) {
      return;
    }

    // 连接
    teamWebSocket.connect(teamId, leaderEndpoint);

    // 注册事件处理器
    const unsubscribe = teamWebSocket.on(teamId, (event: TeamEvent) => {
      handlerRef.current?.(event);
    });

    // 清理
    return () => {
      unsubscribe();
      // 注意：这里不断开连接，因为其他组件可能还在使用
      // 如果需要断开，调用 disconnect 方法
    };
  }, [teamId, leaderEndpoint, enabled]);

  // 手动断开连接
  const disconnect = useCallback(() => {
    if (teamId) {
      teamWebSocket.disconnect(teamId);
    }
  }, [teamId]);

  // 检查连接状态
  const isConnected = teamId ? teamWebSocket.isConnected(teamId) : false;

  return {
    isConnected,
    disconnect,
  };
}

/**
 * 管理多个团队的 WebSocket 连接
 */
export function useTeamWebSocketGlobal(onEvent?: TeamEventHandler) {
  const handlerRef = useRef(onEvent);

  // 保持 handler 引用最新
  useEffect(() => {
    handlerRef.current = onEvent;
  }, [onEvent]);

  // 注册全局事件处理器
  useEffect(() => {
    if (!handlerRef.current) {
      return;
    }

    const unsubscribe = teamWebSocket.onGlobal((event: TeamEvent) => {
      handlerRef.current?.(event);
    });

    return unsubscribe;
  }, []);

  // 连接到指定团队
  const connect = useCallback((teamId: string, leaderEndpoint: string) => {
    teamWebSocket.connect(teamId, leaderEndpoint);
  }, []);

  // 断开指定团队
  const disconnect = useCallback((teamId: string) => {
    teamWebSocket.disconnect(teamId);
  }, []);

  // 断开所有连接
  const disconnectAll = useCallback(() => {
    teamWebSocket.disconnectAll();
  }, []);

  return {
    connect,
    disconnect,
    disconnectAll,
    isConnected: (teamId: string) => teamWebSocket.isConnected(teamId),
  };
}
