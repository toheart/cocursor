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

export type PluginSource =
  | "builtin"
  | "project"
  | "team_global"
  | "team_project";

export interface Plugin {
  id: string;
  name: string;
  description: string;
  author: string;
  version: string;
  icon?: string;
  category: string;
  category_display?: string; // 本地化的分类显示名称
  installed: boolean;
  installed_version?: string;
  skill?: {
    skill_name: string;
    description?: string; // SKILL.md 中的使用说明
  };
  mcp?: {
    server_name: string;
    transport: string;
    url: string;
  };
  command?: {
    commands: CommandItem[];
  };
  // 团队相关字段
  source?: PluginSource;
  full_id?: string;
  team_id?: string;
  team_name?: string;
  author_id?: string;
  author_name?: string;
  author_endpoint?: string;
  author_online?: boolean;
  published_at?: string;
  is_downloaded?: boolean;
  downloaded_at?: string;
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
    tab_acceptance_rate: number;
    composer_acceptance_rate: number;
    active_sessions: number;
  };
  daily_details: DailyAnalysis[];
  code_changes_trend: CodeChangeTrend[];
  top_files: TopFile[];
  time_distribution: TimeDistribution[];
  efficiency_metrics: EfficiencyMetrics;
}

export interface DailyAnalysis {
  date: string;
  lines_added: number;
  lines_removed: number;
  files_changed: number;
  active_sessions: number;
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

// ========== RAG 相关 ==========

// 旧的搜索结果（兼容）
export interface RAGSearchResult {
  type: "message" | "turn";
  session_id: string;
  score: number;
  content: string;
  user_text?: string;
  ai_text?: string;
  message_id?: string;
  turn_index?: number;
  project_id: string;
  project_name: string;
  timestamp: number;
  message_ids?: string[];
  summary?: string;
}

// 新的知识片段搜索结果
export interface ChunkSearchResult {
  chunk_id: string;
  session_id: string;
  score: number;
  project_id: string;
  project_name: string;
  user_query_preview: string;
  summary?: string;
  main_topic?: string;
  tags?: string[];
  tools_used?: string[];
  files_modified?: string[];
  has_code: boolean;
  timestamp: number;
  is_enriched: boolean;
  turn_index?: number; // 用于定位到具体消息
}

// 知识片段详情
export interface ChunkDetail extends ChunkSearchResult {
  user_query: string;
  ai_response_core: string;
  enrichment_status: "pending" | "processing" | "completed" | "failed";
  enrichment_error?: string;
}

// 增强队列统计
export interface EnrichmentStats {
  pending_count: number;
  processing_count: number;
  completed_count: number;
  failed_count: number;
}

// 索引统计
export interface IndexStats {
  total_files: number;
  total_chunks: number;
  last_scan_time: number;
}

// 搜索过滤器
export interface SearchFilters {
  has_code?: boolean;
  tools_used?: string[];
  time_range?: {
    start: number;
    end: number;
  };
}

// ========== 团队相关 ==========

export interface Team {
  id: string;
  name: string;
  leader_id: string;
  leader_name: string;
  leader_endpoint: string;
  member_count: number;
  skill_count: number;
  created_at: string;
  joined_at: string;
  is_leader: boolean;
  leader_online: boolean;
  last_sync_at?: string;
}

export interface TeamMember {
  id: string;
  name: string;
  endpoint: string;
  is_leader: boolean;
  is_online: boolean;
  joined_at: string;
  work_status?: MemberWorkStatus;
}

// 成员工作状态
export interface MemberWorkStatus {
  project_name: string;
  current_file: string;
  last_active_at: string;
  status_visible: boolean;
}

// 团队日报
export interface TeamDailySummary {
  member_id: string;
  member_name: string;
  date: string;
  summary?: string;
  language?: string;
  shared_at: string;
  total_sessions: number;
  project_count: number;
}

export interface Identity {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface NetworkInterface {
  name: string;
  addresses: string[];
  is_up: boolean;
  is_loopback: boolean;
  is_virtual: boolean;
}

export interface NetworkConfig {
  preferred_interface?: string;
  preferred_ip?: string;
  last_updated?: string;
}

export interface DiscoveredTeam {
  team_id: string;
  name: string;
  leader_name: string;
  endpoint: string;
  member_count: number;
  version: string;
}

export interface TeamSkillEntry {
  plugin_id: string;
  name: string;
  description: string;
  version: string;
  scope: string;
  author_id: string;
  author_name: string;
  author_endpoint: string;
  published_at: string;
  file_count: number;
  total_size: number;
  checksum: string;
}

// 技能元数据预填充
export interface SkillMetadataPrefill {
  name: string;
  name_zh_cn?: string;
  description: string;
  description_zh_cn?: string;
  version: string;
  author: string;
  category: string;
}

// 技能验证结果
export interface SkillValidationResult {
  valid: boolean;
  error?: string;
  source_type: "plugin" | "skill"; // 来源类型
  prefill: SkillMetadataPrefill; // 预填充数据
  missing_fields?: string[]; // 缺失的必填字段
  files: string[];
  total_size: number;
  skill_path: string;

  // 向后兼容字段
  name: string;
  description: string;
  version: string;
}

// 用户提交的技能元数据
export interface SkillMetadata {
  plugin_id: string;
  name: string;
  name_zh_cn?: string;
  description: string;
  description_zh_cn?: string;
  version: string;
  category: string;
  author: string;
}

// ========== 团队周报相关 ==========

// 团队项目配置
export interface TeamProjectConfig {
  team_id: string;
  projects: ProjectMatcher[];
  updated_at: string;
}

// 项目匹配规则
export interface ProjectMatcher {
  id: string; // 规则 ID (UUID)
  name: string; // 显示名称
  repo_url: string; // Git Remote URL
}

// 成员周统计数据
export interface MemberWeeklyStats {
  member_id: string;
  member_name: string;
  week_start: string; // YYYY-MM-DD
  daily_stats: MemberDailyStats[];
  updated_at: string;
}

// 成员每日统计
export interface MemberDailyStats {
  date: string; // YYYY-MM-DD
  git_stats?: GitDailyStats;
  cursor_stats?: CursorDailyStats;
  work_items?: WorkItemSummary[];
  has_report: boolean;
}

// Git 每日统计
export interface GitDailyStats {
  total_commits: number;
  total_added: number;
  total_removed: number;
  projects: ProjectGitStats[];
}

// 项目 Git 统计
export interface ProjectGitStats {
  project_name: string;
  repo_url: string;
  commits: number;
  lines_added: number;
  lines_removed: number;
  commit_messages: CommitSummary[];
}

// 提交摘要
export interface CommitSummary {
  hash: string;
  message: string;
  time: string;
  files_count: number;
}

// Cursor 每日统计
export interface CursorDailyStats {
  session_count: number;
  tokens_used: number;
  lines_added: number;
  lines_removed: number;
}

// 工作条目摘要
export interface WorkItemSummary {
  project: string;
  category: string;
  description: string;
}

// 团队周视图
export interface TeamWeeklyView {
  team_id: string;
  week_start: string; // 周一日期
  week_end: string; // 周日日期
  calendar: TeamDayColumn[];
  project_summary: ProjectWeekStats[];
  updated_at: string;
}

// 日历中的一天
export interface TeamDayColumn {
  date: string; // YYYY-MM-DD
  day_of_week: number; // 1=周一...7=周日
  members: MemberDayCell[];
}

// 日历格子（一个成员一天的数据）
export interface MemberDayCell {
  member_id: string;
  member_name: string;
  activity_level: number; // 0-4
  commits: number;
  lines_changed: number;
  has_report: boolean;
  is_online: boolean;
}

// 项目周统计
export interface ProjectWeekStats {
  project_name: string;
  repo_url: string;
  total_commits: number;
  total_added: number;
  total_removed: number;
  contributors: ContributorStats[];
}

// 贡献者统计
export interface ContributorStats {
  member_id: string;
  member_name: string;
  commits: number;
  lines_added: number;
  lines_removed: number;
}

// 成员日详情
export interface MemberDailyDetail {
  member_id: string;
  member_name: string;
  date: string;
  git_stats?: GitDailyStats;
  cursor_stats?: CursorDailyStats;
  work_items?: WorkItemSummary[];
  has_report: boolean;
  is_online: boolean;
  is_cached: boolean;
}

// ========== VSCode 相关 ==========

declare global {
  interface Window {
    __WORKSPACE_PATH__?: string;
    __VIEW_TYPE__?: "workAnalysis" | "recentSessions" | "marketplace" | "team";
    __INITIAL_ROUTE__?: string;
  }
}
