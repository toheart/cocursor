import React, { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { apiService, SessionHealth, getVscodeApi } from "../services/api";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from "recharts";

/**
 * 工作分析数据接口
 */
interface WorkAnalysisData {
  overview: {
    total_lines_added: number;
    total_lines_removed: number;
    files_changed: number;
    acceptance_rate: number;
    active_sessions: number;
    total_prompts?: number;        // 总 Prompts 数（用户输入）
    total_generations?: number;    // 总 Generations 数（AI 回复）
  };
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

interface ProjectOption {
  project_name: string;
}

export const WorkAnalysis: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<WorkAnalysisData | null>(null);
  const [projectName, setProjectName] = useState<string>("");
  
  // 周选择器相关
  type WeekOption = "thisWeek" | "lastWeek" | "twoWeeksAgo" | "custom";
  const [weekOption, setWeekOption] = useState<WeekOption>("thisWeek");
  const [startDate, setStartDate] = useState<string>("");
  const [endDate, setEndDate] = useState<string>("");
  
  // 计算周的起止日期
  const getWeekRange = (weeksAgo: number): { start: string; end: string } => {
    const today = new Date();
    const dayOfWeek = today.getDay(); // 0 = 周日, 1 = 周一, ...
    const mondayOffset = dayOfWeek === 0 ? -6 : 1 - dayOfWeek; // 调整到周一
    
    const targetDate = new Date(today);
    targetDate.setDate(today.getDate() + mondayOffset - (weeksAgo * 7));
    
    const weekStart = new Date(targetDate);
    const weekEnd = new Date(targetDate);
    weekEnd.setDate(weekStart.getDate() + 6);
    
    return {
      start: weekStart.toISOString().split('T')[0],
      end: weekEnd.toISOString().split('T')[0]
    };
  };
  
  // 初始化周选择
  useEffect(() => {
    const range = getWeekRange(0); // 本周
    setStartDate(range.start);
    setEndDate(range.end);
  }, []);
  
  // 处理周选择变化
  const handleWeekChange = (value: WeekOption) => {
    setWeekOption(value);
    if (value === "custom") {
      // 自定义周：保持当前日期，用户可以手动调整
      return;
    }
    
    let weeksAgo = 0;
    if (value === "lastWeek") weeksAgo = 1;
    else if (value === "twoWeeksAgo") weeksAgo = 2;
    
    const range = getWeekRange(weeksAgo);
    setStartDate(range.start);
    setEndDate(range.end);
  };
  const [projects, setProjects] = useState<ProjectOption[]>([]);
  const [loadingProjects, setLoadingProjects] = useState(false);
  const [sessionHealth, setSessionHealth] = useState<SessionHealth | null>(null);
  const loadDataTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const isMountedRef = useRef(true);
  const loadProjectsTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const healthIntervalRef = useRef<NodeJS.Timeout | null>(null);

  // 加载项目列表和健康状态
  useEffect(() => {
    isMountedRef.current = true;
    loadProjects();
    loadSessionHealth();
    
    // 定时刷新健康状态（每 30 秒）
    healthIntervalRef.current = setInterval(() => {
      if (isMountedRef.current) {
        loadSessionHealth();
      }
    }, 30000);
    
    return () => {
      isMountedRef.current = false;
      if (loadProjectsTimeoutRef.current) {
        clearTimeout(loadProjectsTimeoutRef.current);
        loadProjectsTimeoutRef.current = null;
      }
      if (healthIntervalRef.current) {
        clearInterval(healthIntervalRef.current);
        healthIntervalRef.current = null;
      }
    };
  }, []);

  // 加载数据（带防抖）
  useEffect(() => {
    // 清除之前的定时器
    if (loadDataTimeoutRef.current) {
      clearTimeout(loadDataTimeoutRef.current);
    }
    
    // 设置新的定时器，300ms 防抖
    loadDataTimeoutRef.current = setTimeout(() => {
      loadData();
    }, 300);

    return () => {
      if (loadDataTimeoutRef.current) {
        clearTimeout(loadDataTimeoutRef.current);
      }
    };
  }, [projectName, startDate, endDate]);

  const loadProjects = async (): Promise<void> => {
    if (!isMountedRef.current) return;
    
    try {
      setLoadingProjects(true);
      // 通过 Extension 获取项目列表
      const vscode = getVscodeApi();
      
      const result = await new Promise<{ projects: ProjectOption[] }>((resolve, reject) => {
        const messageId = `fetchProjectList-${Date.now()}-${Math.random()}`;
        const handler = (event: MessageEvent) => {
          if (event.data.type === "fetchProjectList-response") {
            window.removeEventListener("message", handler);
            if (event.data.data && typeof event.data.data === "object" && "error" in event.data.data) {
              reject(new Error(String(event.data.data.error)));
            } else {
              resolve(event.data.data as { projects: ProjectOption[] });
            }
          }
        };
        window.addEventListener("message", handler);
        vscode.postMessage({ command: "fetchProjectList", payload: {}, messageId });
        
        loadProjectsTimeoutRef.current = setTimeout(() => {
          window.removeEventListener("message", handler);
          reject(new Error("Request timeout"));
        }, 5000) as unknown as NodeJS.Timeout;
      });
      
      // 检查组件是否已卸载
      if (!isMountedRef.current) return;
      
      if (result && result.projects) {
        setProjects(result.projects);
      }
    } catch (err) {
      // 组件已卸载，不更新状态
      if (!isMountedRef.current) return;
      console.error("加载项目列表失败:", err);
      // 静默失败，不影响主功能
    } finally {
      if (isMountedRef.current) {
        setLoadingProjects(false);
      }
    }
  };

  const loadSessionHealth = async (): Promise<void> => {
    if (!isMountedRef.current) return;
    
    try {
      const workspacePath = (window as any).__WORKSPACE_PATH__;
      const health = await apiService.getCurrentSessionHealth(workspacePath);
      
      if (!isMountedRef.current) return;
      setSessionHealth(health);
    } catch (error) {
      // 静默失败
      console.error("加载会话健康状态失败:", error);
    }
  };

  const loadData = async (): Promise<void> => {
    if (!isMountedRef.current) return;
    
    try {
      setLoading(true);
      setError(null);
      const result = await apiService.getWorkAnalysis(startDate, endDate, projectName || undefined);
      
      // 检查组件是否已卸载
      if (!isMountedRef.current) return;
      
      // 确保返回的数据结构完整，避免 null 值
      if (result && typeof result === "object") {
        const workData = result as WorkAnalysisData;
        // 确保数组字段不为 null
        if (!workData.top_files) workData.top_files = [];
        if (!workData.code_changes_trend) workData.code_changes_trend = [];
        if (!workData.time_distribution) workData.time_distribution = [];
        if (workData.efficiency_metrics && !workData.efficiency_metrics.entropy_trend) {
          workData.efficiency_metrics.entropy_trend = [];
        }
        setData(workData);
      } else {
        setError("返回数据格式错误");
      }
    } catch (err) {
      // 组件已卸载，不更新状态
      if (!isMountedRef.current) return;
      setError(err instanceof Error ? err.message : "未知错误");
      setData(null);
    } finally {
      if (isMountedRef.current) {
        setLoading(false);
      }
    }
  };

  const getEntropyColor = (entropy: number): string => {
    if (entropy < 40) {
      return "var(--vscode-testing-iconPassed)";
    } else if (entropy < 70) {
      return "var(--vscode-testing-iconQueued)";
    } else {
      return "var(--vscode-testing-iconFailed)";
    }
  };

  const getEntropyStatusText = (status: string): string => {
    switch (status) {
      case "healthy":
        return "健康";
      case "sub_healthy":
        return "亚健康";
      case "dangerous":
        return "危险";
      default:
        return "未知";
    }
  };


  return (
    <div className="cocursor-work-analysis">
      <div className="cocursor-filters">
        <select
          value={projectName}
          onChange={(e) => setProjectName(e.target.value)}
          disabled={loadingProjects}
        >
          <option value="">所有项目</option>
          {Array.isArray(projects) && projects.map((project) => (
            <option key={project.project_name} value={project.project_name}>
              {project.project_name}
            </option>
          ))}
        </select>
        <select
          value={weekOption}
          onChange={(e) => handleWeekChange(e.target.value as WeekOption)}
        >
          <option value="thisWeek">本周</option>
          <option value="lastWeek">上周</option>
          <option value="twoWeeksAgo">上上周</option>
          <option value="custom">自定义周</option>
        </select>
        {weekOption === "custom" && (
          <>
            <input
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              placeholder="开始日期"
              style={{ minWidth: "140px" }}
            />
            <input
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              placeholder="结束日期"
              style={{ minWidth: "140px" }}
            />
          </>
        )}
        {weekOption !== "custom" && (
          <span className="cocursor-week-range">
            {startDate} 至 {endDate}
          </span>
        )}
      </div>

      <main className="cocursor-main" style={{ padding: "16px" }}>
        {loading && <div className="cocursor-loading">加载中...</div>}
        {error && <div className="cocursor-error">错误: {error}</div>}
        
        {/* 会话健康状态 */}
        {sessionHealth && (
          <div className="cocursor-session-health" style={{ marginBottom: "24px" }}>
            <div className="cocursor-session-health-header">
              <h2>会话健康状态</h2>
            </div>
            <div className="cocursor-entropy-display">
              <div className="cocursor-entropy-value">
                <span className="cocursor-entropy-label">熵值:</span>
                <span
                  className="cocursor-entropy-number"
                  style={{
                    color: getEntropyColor(sessionHealth.entropy),
                    fontWeight: "bold"
                  }}
                >
                  {sessionHealth.entropy.toFixed(2)}
                </span>
              </div>
              <div className="cocursor-entropy-status">
                <span className="cocursor-entropy-label">状态:</span>
                <span
                  className="cocursor-entropy-status-text"
                  style={{
                    color: getEntropyColor(sessionHealth.entropy)
                  }}
                >
                  {getEntropyStatusText(sessionHealth.status)}
                </span>
              </div>
              <div className="cocursor-entropy-progress">
                <div
                  className="cocursor-entropy-progress-bar"
                  style={{
                    width: `${Math.min((sessionHealth.entropy / 100) * 100, 100)}%`,
                    backgroundColor: getEntropyColor(sessionHealth.entropy),
                    animation:
                      sessionHealth.entropy >= 70
                        ? "pulse 1s ease-in-out infinite"
                        : "none"
                  }}
                />
              </div>
              {sessionHealth.warning && (
                <div
                  className="cocursor-entropy-warning"
                  style={{
                    color: "var(--vscode-errorForeground)",
                    marginTop: "8px",
                    padding: "8px",
                    backgroundColor: "var(--vscode-inputValidation-errorBackground)",
                    borderRadius: "4px"
                  }}
                >
                  ⚠️ {sessionHealth.warning}
                </div>
              )}
            </div>
          </div>
        )}

        {data && (
          <>
            {/* 概览卡片 */}
            <div className="cocursor-overview-cards">
              <div className="cocursor-card">
                <h3>代码变更</h3>
                <div className="cocursor-stat">
                  <span className="cocursor-stat-label">添加:</span>
                  <span className="cocursor-stat-value">{data.overview.total_lines_added}</span>
                </div>
                <div className="cocursor-stat">
                  <span className="cocursor-stat-label">删除:</span>
                  <span className="cocursor-stat-value">{data.overview.total_lines_removed}</span>
                </div>
                <div className="cocursor-stat">
                  <span className="cocursor-stat-label">文件:</span>
                  <span className="cocursor-stat-value">{data.overview.files_changed}</span>
                </div>
              </div>
              <div className="cocursor-card">
                <h3>接受率</h3>
                <div className="cocursor-stat-large">
                  {data.overview.acceptance_rate.toFixed(1)}%
                </div>
              </div>
              <div className="cocursor-card">
                <h3>活跃会话</h3>
                <div className="cocursor-stat-large">{data.overview.active_sessions}</div>
              </div>
              {data.overview.total_prompts !== undefined && (
                <div className="cocursor-card">
                  <h3>AI 交互</h3>
                  <div className="cocursor-stat">
                    <span className="cocursor-stat-label">Prompts:</span>
                    <span className="cocursor-stat-value">{data.overview.total_prompts}</span>
                  </div>
                  <div className="cocursor-stat">
                    <span className="cocursor-stat-label">Generations:</span>
                    <span className="cocursor-stat-value">{data.overview.total_generations || 0}</span>
                  </div>
                </div>
              )}
            </div>

            {/* 代码变更趋势图表 */}
            {data.code_changes_trend && Array.isArray(data.code_changes_trend) && data.code_changes_trend.length > 0 && (
              <div className="cocursor-chart-section">
                <h2>代码变更趋势</h2>
                <div className="cocursor-chart-container">
                  <ResponsiveContainer width="100%" height={300}>
                    <AreaChart
                      data={data.code_changes_trend.map(item => ({
                        date: item.date,
                        添加行数: item.lines_added,
                        删除行数: item.lines_removed,
                        文件变更: item.files_changed
                      }))}
                      margin={{ top: 10, right: 30, left: 0, bottom: 0 }}
                    >
                      <defs>
                        <linearGradient id="colorAdded" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="var(--vscode-textLink-foreground)" stopOpacity={0.8}/>
                          <stop offset="95%" stopColor="var(--vscode-textLink-foreground)" stopOpacity={0.1}/>
                        </linearGradient>
                        <linearGradient id="colorRemoved" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="var(--vscode-errorForeground)" stopOpacity={0.8}/>
                          <stop offset="95%" stopColor="var(--vscode-errorForeground)" stopOpacity={0.1}/>
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" stroke="var(--vscode-panel-border)" />
                      <XAxis 
                        dataKey="date" 
                        stroke="var(--vscode-foreground)"
                        tick={{ fill: "var(--vscode-foreground)", fontSize: 12 }}
                      />
                      <YAxis 
                        stroke="var(--vscode-foreground)"
                        tick={{ fill: "var(--vscode-foreground)", fontSize: 12 }}
                      />
                      <Tooltip 
                        contentStyle={{
                          backgroundColor: "var(--vscode-sideBar-background)",
                          border: "1px solid var(--vscode-panel-border)",
                          borderRadius: "6px",
                          color: "var(--vscode-foreground)"
                        }}
                      />
                      <Legend 
                        wrapperStyle={{ paddingTop: "20px" }}
                        iconType="circle"
                      />
                      <Area 
                        type="monotone" 
                        dataKey="添加行数" 
                        stroke="var(--vscode-textLink-foreground)" 
                        fillOpacity={1} 
                        fill="url(#colorAdded)" 
                      />
                      <Area 
                        type="monotone" 
                        dataKey="删除行数" 
                        stroke="var(--vscode-errorForeground)" 
                        fillOpacity={1} 
                        fill="url(#colorRemoved)" 
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              </div>
            )}

            {/* Top 文件 - 紧凑横向卡片布局 */}
            {data.top_files && Array.isArray(data.top_files) && data.top_files.length > 0 && (
              <div className="cocursor-section">
                <h2>最常编辑文件</h2>
                <div className="cocursor-file-cards">
                  {data.top_files.slice(0, 5).map((file, index) => (
                    <div key={index} className="cocursor-file-card">
                      <div className="cocursor-file-card-header">
                        <span className="cocursor-file-card-index">#{index + 1}</span>
                        <span className="cocursor-file-card-type">{file.file_type || "file"}</span>
                      </div>
                      <div className="cocursor-file-card-name" title={file.file_name}>
                        {file.file_name.length > 30 
                          ? file.file_name.substring(0, 30) + "..." 
                          : file.file_name}
                      </div>
                      <div className="cocursor-file-card-count">
                        {file.reference_count} 次编辑
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* 效率指标 */}
            {data.efficiency_metrics && (
              <div className="cocursor-section">
                <h2>效率指标</h2>
                <div className="cocursor-efficiency-metrics">
                  {data.efficiency_metrics.avg_session_entropy !== undefined && (
                    <div className="cocursor-metric">
                      <span className="cocursor-metric-label">平均熵值:</span>
                      <span className="cocursor-metric-value">
                        {data.efficiency_metrics.avg_session_entropy.toFixed(2)}
                      </span>
                    </div>
                  )}
                  {data.efficiency_metrics.avg_context_usage !== undefined && (
                    <div className="cocursor-metric">
                      <span className="cocursor-metric-label">平均上下文使用率:</span>
                      <span className="cocursor-metric-value">
                        {data.efficiency_metrics.avg_context_usage.toFixed(2)}%
                      </span>
                    </div>
                  )}
                </div>
              </div>
            )}
          </>
        )}
      </main>
    </div>
  );
};
