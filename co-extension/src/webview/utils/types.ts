/**
 * 共享类型定义
 */

// 会话相关类型
export interface Session {
  composerId: string;
  name: string;
  createdAt: number;
  lastUpdatedAt: number;
  totalLinesAdded: number;
  totalLinesRemoved: number;
  filesChangedCount: number;
}

export interface ChatItem {
  composerId: string;
  name: string;
  lastUpdatedAt: number;
  totalLinesAdded?: number;
  totalLinesRemoved?: number;
  filesChangedCount?: number;
}

// 会话详情相关类型
export interface ToolCall {
  name: string;
  arguments: Record<string, string>;
}

export interface Message {
  type: "user" | "ai";
  text: string;
  timestamp: number;
  code_blocks?: Array<{
    language: string;
    code: string;
  }>;
  files?: string[];
  tools?: ToolCall[];
}

export interface SessionDetailData {
  session: {
    composerId: string;
    name: string;
    createdAt: number;
    lastUpdatedAt: number;
  };
  messages: Message[];
  total_messages: number;
  has_more: boolean;
}

// API 响应类型
export interface ApiResponse<T> {
  data: T;
  page?: {
    page: number;
    pageSize: number;
    total: number;
  };
}

export interface SessionListResponse {
  data: Session[];
  page?: {
    page: number;
    pageSize: number;
    total: number;
  };
}

// 插件相关类型
export interface Plugin {
  id: string;
  name: string;
  description: string;
  author: string;
  version: string;
  icon?: string;
  category: string;
  installed: boolean;
  installed_version?: string;
  skill: {
    skill_name: string;
  };
  mcp?: {
    server_name: string;
    transport: string;
    url: string;
  };
  command?: {
    commands: Array<{
      command_id: string;
    }>;
  };
}

export interface PluginResponse {
  plugins?: Plugin[];
  total?: number;
}

export interface PluginInstallResponse {
  success?: boolean;
  message?: string;
  env_vars?: string[];
  error?: string;
}

// 工作分析相关类型
export interface WorkAnalysisData {
  overview: {
    total_lines_added: number;
    total_lines_removed: number;
    files_changed: number;
    acceptance_rate: number;
    tab_acceptance_rate: number;
    composer_acceptance_rate: number;
    active_sessions: number;
  };
  daily_details: Array<{
    date: string;
    lines_added: number;
    lines_removed: number;
    files_changed: number;
    active_sessions: number;
  }>;
  code_changes_trend: Array<{
    date: string;
    lines_added: number;
    lines_removed: number;
    files_changed: number;
  }>;
  top_files: Array<{
    file_name: string;
    reference_count: number;
    file_type: string;
  }>;
  time_distribution: Array<{
    hour: number;
    day: number;
    count: number;
  }>;
  efficiency_metrics: {
    avg_session_entropy: number;
    avg_context_usage: number;
    entropy_trend: Array<{
      date: string;
      value: number;
    }>;
  };
}

export interface ProjectOption {
  project_name: string;
}

// 窗口对象扩展
declare global {
  interface Window {
    __WORKSPACE_PATH__?: string;
    __VIEW_TYPE__?: "workAnalysis" | "recentSessions" | "marketplace";
    __INITIAL_ROUTE__?: string;
  }
}
