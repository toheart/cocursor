/**
 * 身份信息全局状态管理
 */

import { create } from "zustand";
import { apiService } from "../../../services/api";
import { Identity } from "../../../types";

interface IdentityState {
  /** 身份信息 */
  identity: Identity | null;
  /** 是否已设置身份 */
  hasIdentity: boolean;
  /** 加载中 */
  loading: boolean;
  /** 错误信息 */
  error: string | null;

  /** 获取身份信息 */
  fetchIdentity: () => Promise<void>;
  /** 设置身份（创建或更新） */
  setIdentity: (name: string) => Promise<void>;
  /** 重置错误 */
  clearError: () => void;
}

export const useIdentityStore = create<IdentityState>((set) => ({
  identity: null,
  hasIdentity: false,
  loading: false,
  error: null,

  fetchIdentity: async () => {
    set({ loading: true, error: null });
    try {
      const resp = (await apiService.getTeamIdentity()) as {
        exists: boolean;
        identity?: Identity;
      };
      set({
        identity: resp.identity || null,
        hasIdentity: resp.exists,
        loading: false,
      });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Failed to fetch identity";
      set({ loading: false, error: message });
    }
  },

  setIdentity: async (name: string) => {
    set({ loading: true, error: null });
    try {
      await apiService.setTeamIdentity(name);
      // 设置成功后重新获取
      const resp = (await apiService.getTeamIdentity()) as {
        exists: boolean;
        identity?: Identity;
      };
      set({
        identity: resp.identity || null,
        hasIdentity: resp.exists,
        loading: false,
      });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Failed to set identity";
      set({ loading: false, error: message });
      throw err;
    }
  },

  clearError: () => set({ error: null }),
}));
