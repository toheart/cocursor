import React, { useState, useEffect, useRef, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { apiService, SessionHealth, DailySummary, getVscodeApi } from "../services/api";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from "recharts";
import ReactMarkdown from "react-markdown";

/**
 * 工作分析数据接口
 */
interface WorkAnalysisData {
  overview: {
    total_lines_added: number;
    total_lines_removed: number;
    files_changed: number;
    acceptance_rate: number;
    tab_acceptance_rate: number;
    composer_acceptance_rate: number;
    active_sessions: number;
    total_tokens: number;
    token_trend: string;
  };
  daily_details: Array<{
    date: string;
    lines_added: number;
    lines_removed: number;
    files_changed: number;
    active_sessions: number;
    token_usage: number;
    has_daily_report: boolean;
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


export const WorkAnalysis: React.FC = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<WorkAnalysisData | null>(null);

  const handleRAGSearchClick = useCallback(() => {
    // 通过 vscode API 打开独立的 RAG 搜索 webview
    const vscode = getVscodeApi();
    vscode.postMessage({
      command: "openRAGSearch",
    });
  }, []);

  const handleRAGConfigClick = useCallback(() => {
    // 通过 vscode API 打开独立的 RAG 搜索 webview，并导航到配置页面
    const vscode = getVscodeApi();
    vscode.postMessage({
      command: "openRAGSearch",
      payload: { route: "/config" },
    });
  }, []);
  
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
  const [sessionHealth, setSessionHealth] = useState<SessionHealth | null>(null);
  const loadDataTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const isMountedRef = useRef(true);
  const healthIntervalRef = useRef<NodeJS.Timeout | null>(null);

  // 日报弹窗相关状态
  const [showReportModal, setShowReportModal] = useState(false);
  const [reportModalType, setReportModalType] = useState<"view" | "generate">("view");
  const [selectedDate, setSelectedDate] = useState<string>("");
  const [dailySummary, setDailySummary] = useState<DailySummary | null>(null);
  const [loadingReport, setLoadingReport] = useState(false);
  const [copied, setCopied] = useState(false);

  // 加载健康状态
  useEffect(() => {
    isMountedRef.current = true;
    loadSessionHealth();
    
    // 定时刷新健康状态（每 30 秒）
    healthIntervalRef.current = setInterval(() => {
      if (isMountedRef.current) {
        loadSessionHealth();
      }
    }, 30000);
    
    return () => {
      isMountedRef.current = false;
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
  }, [startDate, endDate]);

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
      const result = await apiService.getWorkAnalysis(startDate, endDate);
      
      // 检查组件是否已卸载
      if (!isMountedRef.current) return;
      
      // 确保返回的数据结构完整，避免 null 值
      if (result && typeof result === "object") {
        const workData = result as WorkAnalysisData;
        // 确保数组字段不为 null
        if (!workData.daily_details) workData.daily_details = [];
        if (!workData.top_files) workData.top_files = [];
        if (!workData.code_changes_trend) workData.code_changes_trend = [];
        if (!workData.time_distribution) workData.time_distribution = [];
        if (workData.efficiency_metrics && !workData.efficiency_metrics.entropy_trend) {
          workData.efficiency_metrics.entropy_trend = [];
        }
        // 确保接受率字段存在
        if (workData.overview) {
          if (typeof workData.overview.acceptance_rate === "undefined") workData.overview.acceptance_rate = 0;
          if (typeof workData.overview.tab_acceptance_rate === "undefined") workData.overview.tab_acceptance_rate = 0;
          if (typeof workData.overview.composer_acceptance_rate === "undefined") workData.overview.composer_acceptance_rate = 0;
        }
        setData(workData);
      } else {
        setError(t("common.error"));
      }
    } catch (err) {
      // 组件已卸载，不更新状态
      if (!isMountedRef.current) return;
      setError(err instanceof Error ? err.message : t("common.error"));
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
    return t(`workAnalysis.sessionHealth.${status}`) || t("common.unknown");
  };

  // 格式化 Token 数量
  const formatTokenCount = (count: number): string => {
    if (count >= 1000000) {
      return `${(count / 1000000).toFixed(1)}M`;
    } else if (count >= 1000) {
      return `${(count / 1000).toFixed(1)}K`;
    }
    return count.toString();
  };

  // 格式化接受率（处理 -1 表示 N/A）
  const formatAcceptanceRate = (rate: number): string => {
    if (rate < 0) {
      return t("workAnalysis.dailyReport.notAvailable");
    }
    return `${rate.toFixed(1)}%`;
  };

  // 打开日报查看弹窗
  const handleViewReport = async (date: string) => {
    setSelectedDate(date);
    setReportModalType("view");
    setShowReportModal(true);
    setLoadingReport(true);
    
    try {
      const summary = await apiService.getDailySummary(date);
      setDailySummary(summary);
    } catch (err) {
      console.error("Failed to load daily summary:", err);
      setDailySummary(null);
    } finally {
      setLoadingReport(false);
    }
  };

  // 打开日报生成引导弹窗
  const handleGenerateReport = (date: string) => {
    setSelectedDate(date);
    setReportModalType("generate");
    setShowReportModal(true);
    setCopied(false);
  };

  // 复制命令到剪贴板
  const handleCopyCommand = () => {
    const command = `/daily-summary ${selectedDate}`;
    navigator.clipboard.writeText(command).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };

  // 关闭弹窗
  const handleCloseModal = () => {
    setShowReportModal(false);
    setDailySummary(null);
  };

  return (
    <div className="cocursor-work-analysis">
      <div className="cocursor-filters" style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "16px" }}>
        <div style={{ display: "flex", gap: "8px", alignItems: "center" }}>
          <select
            value={weekOption}
            onChange={(e) => handleWeekChange(e.target.value as WeekOption)}
          >
            <option value="thisWeek">{t("workAnalysis.week.thisWeek")}</option>
            <option value="lastWeek">{t("workAnalysis.week.lastWeek")}</option>
            <option value="twoWeeksAgo">{t("workAnalysis.week.twoWeeksAgo")}</option>
            <option value="custom">{t("workAnalysis.week.custom")}</option>
          </select>
          {weekOption === "custom" && (
            <>
              <input
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                placeholder={t("workAnalysis.startDate")}
                style={{ minWidth: "140px" }}
              />
              <input
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
                placeholder={t("workAnalysis.endDate")}
                style={{ minWidth: "140px" }}
              />
            </>
          )}
          {weekOption !== "custom" && (
            <span className="cocursor-week-range">
              {startDate} {t("workAnalysis.to")} {endDate}
            </span>
          )}
        </div>
      </div>

      <main className="cocursor-main" style={{ padding: "16px" }}>
        {loading && <div className="cocursor-loading">{t("workAnalysis.loading")}</div>}
        {error && <div className="cocursor-error">{t("workAnalysis.error")}: {error}</div>}
        
        {/* 会话健康状态 */}
        {sessionHealth && (
          <div className="cocursor-session-health" style={{ marginBottom: "24px" }}>
            <div className="cocursor-session-health-header">
              <h2>{t("workAnalysis.sessionHealth.title")}</h2>
            </div>
            <div className="cocursor-entropy-display">
              <div className="cocursor-entropy-value">
                <span className="cocursor-entropy-label">{t("workAnalysis.sessionHealth.entropy")}:</span>
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
                <span className="cocursor-entropy-label">{t("workAnalysis.sessionHealth.status")}:</span>
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
              {/* Token 统计卡片 */}
              <div className="cocursor-card">
                <h3>{t("workAnalysis.tokenStats.title")}</h3>
                <div className="cocursor-stat-large">
                  {formatTokenCount(data.overview.total_tokens || 0)}
                </div>
                {data.overview.token_trend && (
                  <div style={{ marginTop: "8px", fontSize: "12px" }}>
                    <span style={{ 
                      color: data.overview.token_trend.startsWith("+") 
                        ? "var(--vscode-testing-iconPassed)" 
                        : data.overview.token_trend.startsWith("-") 
                          ? "var(--vscode-testing-iconFailed)" 
                          : "var(--vscode-foreground)" 
                    }}>
                      {data.overview.token_trend.startsWith("+") ? "↑" : data.overview.token_trend.startsWith("-") ? "↓" : ""} {data.overview.token_trend}
                    </span>
                    <span style={{ opacity: 0.6, marginLeft: "4px" }}>{t("workAnalysis.tokenStats.trend")}</span>
                  </div>
                )}
              </div>
              <div className="cocursor-card">
                <h3>{t("workAnalysis.overview.codeChanges")}</h3>
                <div className="cocursor-stat">
                  <span className="cocursor-stat-label">{t("workAnalysis.overview.added")}:</span>
                  <span className="cocursor-stat-value">{data.overview.total_lines_added}</span>
                </div>
                <div className="cocursor-stat">
                  <span className="cocursor-stat-label">{t("workAnalysis.overview.removed")}:</span>
                  <span className="cocursor-stat-value">{data.overview.total_lines_removed}</span>
                </div>
                <div className="cocursor-stat">
                  <span className="cocursor-stat-label">{t("workAnalysis.overview.files")}:</span>
                  <span className="cocursor-stat-value">{data.overview.files_changed}</span>
                </div>
              </div>
              <div className="cocursor-card">
                <h3>{t("workAnalysis.overview.acceptanceRate")}</h3>
                <div className="cocursor-stat-large">
                  {formatAcceptanceRate(data.overview.acceptance_rate)}
                </div>
                <div style={{ marginTop: "8px", fontSize: "12px", opacity: 0.8 }}>
                  <div>Tab: {formatAcceptanceRate(data.overview.tab_acceptance_rate)}</div>
                  <div>Composer: {formatAcceptanceRate(data.overview.composer_acceptance_rate)}</div>
                </div>
              </div>
              <div className="cocursor-card">
                <h3>{t("workAnalysis.overview.activeSessions")}</h3>
                <div className="cocursor-stat-large">{data.overview.active_sessions}</div>
              </div>
            </div>

            {/* 每日详情卡片网格 */}
            {data.daily_details && Array.isArray(data.daily_details) && data.daily_details.length > 0 && (
              <div className="cocursor-section" style={{ marginTop: "24px" }}>
                <h2>{t("workAnalysis.dailyDetails.title")}</h2>
                <div className="cocursor-daily-cards-grid">
                  {(() => {
                    // 计算当周最大 Token 用量，用于进度条
                    const maxToken = Math.max(...data.daily_details.map(d => d.token_usage || 0), 1);
                    const today = new Date().toISOString().split('T')[0];
                    
                    return data.daily_details.map((day, index) => {
                      const dateObj = new Date(day.date);
                      const dayNum = dateObj.getDate();
                      const weekdays = ['日', '一', '二', '三', '四', '五', '六'];
                      const weekdaysEn = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
                      const isToday = day.date === today;
                      const hasActivity = day.lines_added > 0 || day.lines_removed > 0 || day.token_usage > 0;
                      const tokenPercent = maxToken > 0 ? ((day.token_usage || 0) / maxToken) * 100 : 0;
                      
                      return (
                        <div 
                          key={index}
                          className={`cocursor-daily-card ${day.has_daily_report ? 'has-report' : 'no-report'} ${isToday ? 'is-today' : ''} ${!hasActivity ? 'no-activity' : ''}`}
                        >
                          {/* 日期头部 */}
                          <div className="cocursor-daily-card-date">
                            <div className="cocursor-daily-card-day">{dayNum}</div>
                            <div className="cocursor-daily-card-weekday">
                              {t("common.unknown") === "未知" ? `周${weekdays[dateObj.getDay()]}` : weekdaysEn[dateObj.getDay()]}
                              {isToday && <span className="cocursor-daily-card-today"> · {t("workAnalysis.dailyDetails.today")}</span>}
                            </div>
                          </div>
                          
                          {/* 指标区域 */}
                          <div className="cocursor-daily-card-metrics">
                            {/* Token 用量 */}
                            <div className="cocursor-daily-card-metric-row">
                              <div className="cocursor-daily-card-icon token">⚡</div>
                              <div className="cocursor-daily-card-metric-content">
                                <div className="cocursor-daily-card-metric-value">
                                  {hasActivity ? formatTokenCount(day.token_usage || 0) : '—'}
                                </div>
                                <div className="cocursor-daily-card-mini-bar">
                                  <div 
                                    className="cocursor-daily-card-mini-bar-fill token"
                                    style={{ width: `${tokenPercent}%` }}
                                  />
                                </div>
                              </div>
                            </div>
                            
                            {/* 代码变更 */}
                            <div className="cocursor-daily-card-metric-row">
                              <div className="cocursor-daily-card-icon code">±</div>
                              {hasActivity ? (
                                <div className="cocursor-daily-card-code-changes">
                                  <span className="cocursor-daily-card-added">+{day.lines_added}</span>
                                  <span className="cocursor-daily-card-removed">-{day.lines_removed}</span>
                                </div>
                              ) : (
                                <span className="cocursor-daily-card-no-data">—</span>
                              )}
                            </div>
                            
                            {/* 会话数 */}
                            <div className="cocursor-daily-card-metric-row">
                              <div className="cocursor-daily-card-icon session">◉</div>
                              <div className="cocursor-daily-card-metric-value">
                                {hasActivity ? day.active_sessions : '0'}
                              </div>
                              <div className="cocursor-daily-card-metric-label">
                                {t("workAnalysis.dailyDetails.sessions")}
                              </div>
                            </div>
                          </div>
                          
                          {/* 底部操作区 */}
                          {hasActivity && (
                            <div className="cocursor-daily-card-action">
                              {day.has_daily_report ? (
                                <button
                                  className="cocursor-daily-card-view-btn"
                                  onClick={() => handleViewReport(day.date)}
                                >
                                  {t("workAnalysis.dailyDetails.viewReport")} →
                                </button>
                              ) : (
                                <button
                                  className="cocursor-daily-card-generate-btn"
                                  onClick={() => handleGenerateReport(day.date)}
                                >
                                  + {t("workAnalysis.dailyDetails.generateReport")}
                                </button>
                              )}
                            </div>
                          )}
                        </div>
                      );
                    });
                  })()}
                </div>
              </div>
            )}

            {/* 代码变更趋势图表 */}
            {data.code_changes_trend && Array.isArray(data.code_changes_trend) && data.code_changes_trend.length > 0 && (
              <div className="cocursor-chart-section">
                <h2>{t("workAnalysis.charts.codeChangesTrend")}</h2>
                <div className="cocursor-chart-container">
                  <ResponsiveContainer width="100%" height={300}>
                    <AreaChart
                      data={data.code_changes_trend.map(item => ({
                        date: item.date,
                        [t("workAnalysis.charts.addedLines")]: item.lines_added,
                        [t("workAnalysis.charts.removedLines")]: item.lines_removed,
                        [t("workAnalysis.charts.fileChanges")]: item.files_changed
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
                        dataKey={t("workAnalysis.charts.addedLines")} 
                        stroke="var(--vscode-textLink-foreground)" 
                        fillOpacity={1} 
                        fill="url(#colorAdded)" 
                      />
                      <Area 
                        type="monotone" 
                        dataKey={t("workAnalysis.charts.removedLines")} 
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
                <h2>{t("workAnalysis.topFiles.title")}</h2>
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
                        {file.reference_count} {t("workAnalysis.topFiles.edits")}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* 效率指标 */}
            {data.efficiency_metrics && (
              <div className="cocursor-section">
                <h2>{t("workAnalysis.efficiency.title")}</h2>
                <div className="cocursor-efficiency-metrics">
                  {data.efficiency_metrics.avg_session_entropy !== undefined && (
                    <div className="cocursor-metric">
                      <span className="cocursor-metric-label">{t("workAnalysis.efficiency.avgEntropy")}:</span>
                      <span className="cocursor-metric-value">
                        {data.efficiency_metrics.avg_session_entropy.toFixed(2)}
                      </span>
                    </div>
                  )}
                  {data.efficiency_metrics.avg_context_usage !== undefined && (
                    <div className="cocursor-metric">
                      <span className="cocursor-metric-label">{t("workAnalysis.efficiency.avgContextUsage")}:</span>
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

      {/* 日报弹窗 */}
      {showReportModal && (
        <div
          className="cocursor-modal-overlay"
          style={{
            position: "fixed",
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: "rgba(0, 0, 0, 0.5)",
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            zIndex: 1000
          }}
          onClick={handleCloseModal}
        >
          <div
            className="cocursor-modal"
            style={{
              backgroundColor: "var(--vscode-editor-background)",
              borderRadius: "8px",
              padding: "24px",
              maxWidth: "600px",
              width: "90%",
              maxHeight: "80vh",
              overflow: "auto",
              boxShadow: "0 4px 20px rgba(0, 0, 0, 0.3)"
            }}
            onClick={(e) => e.stopPropagation()}
          >
            {reportModalType === "view" ? (
              <>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "16px" }}>
                  <h2 style={{ margin: 0 }}>{selectedDate} {t("workAnalysis.dailyReport.viewTitle")}</h2>
                  <button
                    onClick={handleCloseModal}
                    style={{
                      background: "transparent",
                      border: "none",
                      color: "var(--vscode-foreground)",
                      cursor: "pointer",
                      fontSize: "18px"
                    }}
                  >
                    ✕
                  </button>
                </div>
                {loadingReport ? (
                  <div style={{ textAlign: "center", padding: "20px" }}>
                    {t("workAnalysis.loading")}
                  </div>
                ) : dailySummary ? (
                  <div className="cocursor-markdown-content" style={{ lineHeight: "1.6" }}>
                    <ReactMarkdown>{dailySummary.summary}</ReactMarkdown>
                  </div>
                ) : (
                  <div style={{ textAlign: "center", padding: "20px", color: "var(--vscode-descriptionForeground)" }}>
                    {t("workAnalysis.dailyReport.notAvailable")}
                  </div>
                )}
                <div style={{ marginTop: "16px", textAlign: "right" }}>
                  <button
                    onClick={handleCloseModal}
                    style={{
                      background: "var(--vscode-button-background)",
                      color: "var(--vscode-button-foreground)",
                      border: "none",
                      padding: "8px 16px",
                      borderRadius: "4px",
                      cursor: "pointer"
                    }}
                  >
                    {t("workAnalysis.dailyReport.close")}
                  </button>
                </div>
              </>
            ) : (
              <>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "16px" }}>
                  <h2 style={{ margin: 0 }}>{t("workAnalysis.dailyReport.generateTitle")}</h2>
                  <button
                    onClick={handleCloseModal}
                    style={{
                      background: "transparent",
                      border: "none",
                      color: "var(--vscode-foreground)",
                      cursor: "pointer",
                      fontSize: "18px"
                    }}
                  >
                    ✕
                  </button>
                </div>
                <p>{t("workAnalysis.dailyReport.generatePrompt")}</p>
                <div
                  style={{
                    backgroundColor: "var(--vscode-input-background)",
                    border: "1px solid var(--vscode-input-border)",
                    borderRadius: "4px",
                    padding: "12px",
                    marginTop: "12px",
                    fontFamily: "monospace"
                  }}
                >
                  /daily-summary {selectedDate}
                </div>
                <div style={{ marginTop: "12px", display: "flex", justifyContent: "flex-end" }}>
                  <button
                    onClick={handleCopyCommand}
                    style={{
                      background: copied ? "var(--vscode-testing-iconPassed)" : "var(--vscode-button-background)",
                      color: "var(--vscode-button-foreground)",
                      border: "none",
                      padding: "8px 16px",
                      borderRadius: "4px",
                      cursor: "pointer"
                    }}
                  >
                    {copied ? t("workAnalysis.dailyReport.copied") : t("workAnalysis.dailyReport.copyCommand")}
                  </button>
                </div>
                <p style={{ marginTop: "16px", color: "var(--vscode-descriptionForeground)", fontSize: "13px" }}>
                  {t("workAnalysis.dailyReport.afterGenerate")}
                </p>
                <div style={{ marginTop: "16px", textAlign: "right" }}>
                  <button
                    onClick={handleCloseModal}
                    style={{
                      background: "transparent",
                      border: "1px solid var(--vscode-button-background)",
                      color: "var(--vscode-button-background)",
                      padding: "8px 16px",
                      borderRadius: "4px",
                      cursor: "pointer"
                    }}
                  >
                    {t("workAnalysis.dailyReport.close")}
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
