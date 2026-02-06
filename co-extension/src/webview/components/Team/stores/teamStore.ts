/**
 * 团队列表和详情全局状态管理
 */

import { create } from "zustand";
import { apiService } from "../../../services/api";
import { Team } from "../../../types";

interface TeamState {
  /** 团队列表 */
  teams: Team[];
  /** 加载中 */
  loading: boolean;
  /** 错误信息 */
  error: string | null;

  /** 获取团队列表 */
  fetchTeams: () => Promise<void>;
  /** 创建团队 */
  createTeam: (
    name: string,
    iface?: string,
    ip?: string,
  ) => Promise<void>;
  /** 加入团队 */
  joinTeam: (endpoint: string) => Promise<void>;
  /** 离开团队 */
  leaveTeam: (teamId: string) => Promise<void>;
  /** 解散团队 */
  dissolveTeam: (teamId: string) => Promise<void>;
  /** 根据 ID 获取团队 */
  getTeamById: (teamId: string) => Team | undefined;
  /** 重置错误 */
  clearError: () => void;
}

export const useTeamStore = create<TeamState>((set, get) => ({
  teams: [],
  loading: false,
  error: null,

  fetchTeams: async () => {
    set({ loading: true, error: null });
    try {
      const resp = (await apiService.getTeamList()) as {
        teams: Team[];
        total: number;
      };
      set({ teams: resp.teams || [], loading: false });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Failed to fetch teams";
      set({ loading: false, error: message });
    }
  },

  createTeam: async (name, iface, ip) => {
    set({ loading: true, error: null });
    try {
      await apiService.createTeam(name, iface, ip);
      // 创建后刷新列表
      const resp = (await apiService.getTeamList()) as {
        teams: Team[];
        total: number;
      };
      set({ teams: resp.teams || [], loading: false });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Failed to create team";
      set({ loading: false, error: message });
      throw err;
    }
  },

  joinTeam: async (endpoint) => {
    set({ loading: true, error: null });
    try {
      await apiService.joinTeam(endpoint);
      // 加入后刷新列表
      const resp = (await apiService.getTeamList()) as {
        teams: Team[];
        total: number;
      };
      set({ teams: resp.teams || [], loading: false });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Failed to join team";
      set({ loading: false, error: message });
      throw err;
    }
  },

  leaveTeam: async (teamId) => {
    try {
      await apiService.leaveTeam(teamId);
      // 离开后刷新列表
      const resp = (await apiService.getTeamList()) as {
        teams: Team[];
        total: number;
      };
      set({ teams: resp.teams || [] });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Failed to leave team";
      set({ error: message });
      throw err;
    }
  },

  dissolveTeam: async (teamId) => {
    try {
      await apiService.dissolveTeam(teamId);
      // 解散后刷新列表
      const resp = (await apiService.getTeamList()) as {
        teams: Team[];
        total: number;
      };
      set({ teams: resp.teams || [] });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Failed to dissolve team";
      set({ error: message });
      throw err;
    }
  },

  getTeamById: (teamId) => {
    return get().teams.find((t) => t.id === teamId);
  },

  clearError: () => set({ error: null }),
}));
