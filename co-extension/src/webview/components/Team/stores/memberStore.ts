/**
 * 团队成员全局状态管理
 */

import { create } from "zustand";
import { apiService } from "../../../services/api";
import { TeamMember, TeamSkillEntry } from "../../../types";

interface MemberState {
  /** 当前团队 ID */
  currentTeamId: string | null;
  /** 成员列表 */
  members: TeamMember[];
  /** 成员加载中 */
  loadingMembers: boolean;
  /** 技能列表 */
  skills: TeamSkillEntry[];
  /** 技能加载中 */
  loadingSkills: boolean;
  /** 成员实时工作状态 */
  memberStatuses: Record<string, { project: string; file: string }>;
  /** 错误信息 */
  error: string | null;

  /** 设置当前团队（切换团队时调用） */
  setCurrentTeam: (teamId: string) => void;
  /** 获取成员列表 */
  fetchMembers: (teamId: string) => Promise<void>;
  /** 获取技能列表 */
  fetchSkills: (teamId: string) => Promise<void>;
  /** 更新成员工作状态 */
  updateMemberStatus: (
    memberId: string,
    status: { project: string; file: string } | null,
  ) => void;
  /** 重置（离开团队详情时） */
  reset: () => void;
  /** 重置错误 */
  clearError: () => void;
}

export const useMemberStore = create<MemberState>((set) => ({
  currentTeamId: null,
  members: [],
  loadingMembers: false,
  skills: [],
  loadingSkills: false,
  memberStatuses: {},
  error: null,

  setCurrentTeam: (teamId) => {
    set({
      currentTeamId: teamId,
      members: [],
      skills: [],
      memberStatuses: {},
      error: null,
    });
  },

  fetchMembers: async (teamId) => {
    set({ loadingMembers: true });
    try {
      const resp = (await apiService.getTeamMembers(teamId)) as {
        members: TeamMember[];
      };
      set({ members: resp.members || [], loadingMembers: false });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Failed to fetch members";
      set({ loadingMembers: false, error: message });
    }
  },

  fetchSkills: async (teamId) => {
    set({ loadingSkills: true });
    try {
      const resp = (await apiService.getTeamSkillIndex(teamId)) as {
        entries: TeamSkillEntry[];
      };
      set({ skills: resp.entries || [], loadingSkills: false });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Failed to fetch skills";
      set({ loadingSkills: false, error: message });
    }
  },

  updateMemberStatus: (memberId, status) => {
    set((state) => {
      if (status === null) {
        const { [memberId]: _, ...rest } = state.memberStatuses;
        return { memberStatuses: rest };
      }
      return {
        memberStatuses: {
          ...state.memberStatuses,
          [memberId]: status,
        },
      };
    });
  },

  reset: () =>
    set({
      currentTeamId: null,
      members: [],
      skills: [],
      memberStatuses: {},
      error: null,
    }),

  clearError: () => set({ error: null }),
}));
