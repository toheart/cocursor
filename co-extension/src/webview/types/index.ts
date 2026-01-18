/**
 * 共享类型定义
 */

// ========== 基础类型 ==========

export interface PageResponse<T> {
  data: T[];
  page: {
    total: number;
    page: number;
    pageSize: number;
  };
}

// ========== 会话相关 ==========

export interface Session {
  composerId: string;
  name: string;
  createdAt: number;
  lastUpdatedAt: number;
  totalLinesAdded: number;
  totalLinesRemoved: number;
  filesChangedCount: number;
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

export interface SessionHealth {
  entropy: number;
  status: "healthy" | "sub_healthy" | "dangerous";
  warning?: string;
}

// ========== 消息相关 ==========

export interface ToolCall {
  name: string;
  arguments: Record<string, string>;
}

export interface Message {
  type: "user" | "ai";
  text: string;
  timestamp: number;
  code_blocks?: CodeBlock[];
  files?: string[];
  tools?: ToolCall[];
}

export interface CodeBlock {
  language: string;
  code: string;
}

// ========== 插件相关 ==========

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
  skill?: {
    skill_name: string;
  };
  mcp?: {
    server_name: string;
    transport: string;
    url: string;
  };
  command?: {
    commands: CommandItem[];
  };
}

export interface CommandItem {
  command_id: string;
}

export interface UsageInstruction {
  type: "Skill" | "MCP" | "Command";
  title: string;
  description: string;
}

export interface Toast {
  id: string;
  message: string;
  type: "success" | "error";
}

// ========== 工作分析相关 ==========

export interface WorkAnalysisData {
  overview: {
    total_lines_added: number;
    total_lines_removed: number;
    files_changed: number;
    acceptance_rate: number;
    active_sessions: number;
    total_prompts?: number;
    total_generations?: number;
  };
  code_changes_trend: CodeChangeTrend[];
  top_files: TopFile[];
  time_distribution: TimeDistribution[];
  efficiency_metrics: EfficiencyMetrics;
}

export interface CodeChangeTrend {
  date: string;
  lines_added: number;
  lines_removed: number;
  files_changed: number;
}

export interface TopFile {
  file_name: string;
  reference_count: number;
  file_type: string;
}

export interface TimeDistribution {
  hour: number;
  day: number;
  count: number;
}

export interface EfficiencyMetrics {
  avg_session_entropy: number;
  avg_context_usage: number;
  entropy_trend?: EntropyTrend[];
}

export interface EntropyTrend {
  date: string;
  value: number;
}

export interface ProjectOption {
  project_name: string;
}

// ========== 工作流相关 ==========

export interface Workflow {
  id: string;
  title: string;
  description: string;
  status: "pending" | "approved" | "rejected" | "implemented" | "archived";
  createdAt: number;
  updatedAt: number;
  changeId: string;
}

export interface WorkflowDetail extends Workflow {
  tasks: WorkflowTask[];
  spec: string;
  implementationNotes?: string;
}

export interface WorkflowTask {
  id: string;
  title: string;
  description: string;
  status: "pending" | "in_progress" | "completed";
  assignee?: string;
}

// ========== 状态管理 ==========

export interface LoadingState {
  isLoading: boolean;
  error: string | null;
}

export interface PaginationState {
  page: number;
  limit: number;
  total: number;
  hasMore: boolean;
}

// ========== UI 状态 ==========

export type WeekOption = "thisWeek" | "lastWeek" | "twoWeeksAgo" | "custom";

export interface WeekRange {
  start: string;
  end: string;
}

// ========== VSCode 相关 ==========

declare global {
  interface Window {
    __WORKSPACE_PATH__?: string;
    __VIEW_TYPE__?: "workAnalysis" | "recentSessions" | "marketplace";
    __INITIAL_ROUTE__?: string;
  }
}
