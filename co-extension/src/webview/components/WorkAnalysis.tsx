import React, { useState, useEffect, useRef, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { apiService, SessionHealth, DailySummary, getVscodeApi } from "../services/api";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from "recharts";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { useDataRefresh } from "../hooks";
import {
  DailyReportStats,
  WorkCategoriesChart,
  TimeDistributionChart,
  CodeChangesStats,
  ProjectDetails,
  useScreenshot,
} from "./DailyReport";
import { ActiveSessionsCard, ActiveSessionsOverview } from "./common/ActiveSessionsCard";

/**
 * 工作分析数据接口
 */
interface WorkAnalysisData {
  overview: {
    total_lines_added: number;
    total_lines_removed: number;
    files_changed: number;
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
    completed_changes?: number; // 当日完成的 OpenSpec 变更数量
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
  
  // 格式化本地日期为 YYYY-MM-DD，避免时区问题
  const formatLocalDate = (date: Date): string => {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  };

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
    
    // 使用本地时间格式化，避免 toISOString() 的 UTC 时区问题
    return {
      start: formatLocalDate(weekStart),
      end: formatLocalDate(weekEnd)
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
  const [activeSessions, setActiveSessions] = useState<ActiveSessionsOverview | null>(null);
  const [activeSessionsLoading, setActiveSessionsLoading] = useState(false);
  const loadDataTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const isMountedRef = useRef(true);
  const healthIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const activeSessionsIntervalRef = useRef<NodeJS.Timeout | null>(null);

  // 日报弹窗相关状态
  const [showReportModal, setShowReportModal] = useState(false);
  const [reportModalType, setReportModalType] = useState<"view" | "generate">("view");
  const [selectedDate, setSelectedDate] = useState<string>("");
  const [dailySummary, setDailySummary] = useState<DailySummary | null>(null);
  const [loadingReport, setLoadingReport] = useState(false);
  
  // 技能安装状态
  const [skillInstalled, setSkillInstalled] = useState<boolean | null>(null);
  const [loadingSkillStatus, setLoadingSkillStatus] = useState(false);
  const [installingSkill, setInstallingSkill] = useState(false);
  
  // 截图相关状态
  const [screenshotMode, setScreenshotMode] = useState(false);
  const reportContentRef = useRef<HTMLDivElement>(null);
  const { takeScreenshot, copyToClipboard, isCapturing } = useScreenshot(reportContentRef, {
    filename: `daily-report-${selectedDate}.png`,
    watermark: `Generated by CoCursor · ${selectedDate}`,
  });

  // 加载活跃会话状态
  const loadActiveSessions = async (): Promise<void> => {
    if (!isMountedRef.current) return;
    
    try {
      setActiveSessionsLoading(true);
      const result = await apiService.getActiveSessions();
      
      if (!isMountedRef.current) return;
      if (result && typeof result === "object" && !("error" in result)) {
        setActiveSessions(result as ActiveSessionsOverview);
      }
    } catch (error) {
      console.error("加载活跃会话失败:", error);
    } finally {
      if (isMountedRef.current) {
        setActiveSessionsLoading(false);
      }
    }
  };

  // 加载健康状态和活跃会话（页面可见时才轮询）
  useEffect(() => {
    isMountedRef.current = true;
    loadSessionHealth();
    loadActiveSessions();
    
    // 监听页面可见性，只在可见时轮询
    const handleVisibilityChange = () => {
      if (document.hidden) {
        // 页面隐藏，停止轮询
        if (healthIntervalRef.current) {
          clearInterval(healthIntervalRef.current);
          healthIntervalRef.current = null;
        }
        if (activeSessionsIntervalRef.current) {
          clearInterval(activeSessionsIntervalRef.current);
          activeSessionsIntervalRef.current = null;
        }
      } else {
        // 页面可见，恢复轮询并立即刷新
        if (isMountedRef.current) {
          loadSessionHealth();
          loadActiveSessions();
        }
        if (!healthIntervalRef.current) {
          healthIntervalRef.current = setInterval(() => {
            if (isMountedRef.current) {
              loadSessionHealth();
            }
          }, 30000);
        }
        // 活跃会话 10 秒轮询
        if (!activeSessionsIntervalRef.current) {
          activeSessionsIntervalRef.current = setInterval(() => {
            if (isMountedRef.current) {
              loadActiveSessions();
            }
          }, 10000);
        }
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);

    // 初始状态：如果页面可见，启动轮询
    if (!document.hidden) {
      healthIntervalRef.current = setInterval(() => {
        if (isMountedRef.current) {
          loadSessionHealth();
        }
      }, 30000);
      // 活跃会话 10 秒轮询
      activeSessionsIntervalRef.current = setInterval(() => {
        if (isMountedRef.current) {
          loadActiveSessions();
        }
      }, 10000);
    }
    
    return () => {
      isMountedRef.current = false;
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      if (healthIntervalRef.current) {
        clearInterval(healthIntervalRef.current);
        healthIntervalRef.current = null;
      }
      if (activeSessionsIntervalRef.current) {
        clearInterval(activeSessionsIntervalRef.current);
        activeSessionsIntervalRef.current = null;
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

  // 监听来自 Extension 的刷新通知（如日报生成后）
  useDataRefresh(
    useCallback(() => {
      console.log("[WorkAnalysis] received refresh notification, reloading data");
      loadData();
      loadSessionHealth();
    }, []),
    { dataType: ["workAnalysis", "dailySummary", "all"] }
  );

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
  const handleGenerateReport = async (date: string) => {
    setSelectedDate(date);
    setReportModalType("generate");
    setShowReportModal(true);
    setSkillInstalled(null);
    setLoadingSkillStatus(true);
    
    // 检查技能安装状态
    try {
      const status = await apiService.checkPluginStatus("daily-summary") as { installed: boolean };
      setSkillInstalled(status.installed);
    } catch (err) {
      console.error("Failed to check skill status:", err);
      // 检查失败时默认认为未安装
      setSkillInstalled(false);
    } finally {
      setLoadingSkillStatus(false);
    }
  };

  // 一键安装技能
  const handleInstallSkill = async () => {
    setInstallingSkill(true);
    try {
      const workspacePath = (window as any).__WORKSPACE_PATH__ || "";
      const response = await apiService.installPlugin("daily-summary", workspacePath) as { error?: string };
      
      if (response.error) {
        console.error("Failed to install skill:", response.error);
        // 可以显示错误提示
        return;
      }
      
      // 安装成功，更新状态
      setSkillInstalled(true);
    } catch (err) {
      console.error("Failed to install skill:", err);
    } finally {
      setInstallingSkill(false);
    }
  };

  // 跳转到技能市场
  const handleGoToMarketplace = () => {
    const vscode = getVscodeApi();
    vscode.postMessage({
      command: "openMarketplace",
      payload: { skillId: "daily-summary" }
    });
    handleCloseModal();
  };

  // 关闭弹窗
  const handleCloseModal = () => {
    setShowReportModal(false);
    setDailySummary(null);
    setSkillInstalled(null);
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
        
        {/* 活跃会话状态卡片 */}
        <ActiveSessionsCard data={activeSessions} loading={activeSessionsLoading} />

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
                <h3>{t("workAnalysis.overview.weekSessions")}</h3>
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
                      // 解析日期字符串，避免时区问题
                      // day.date 格式: "2026-01-19"
                      const [year, month, dayOfMonth] = day.date.split('-').map(Number);
                      const dateObj = new Date(year, month - 1, dayOfMonth);
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
                            
                            {/* 完成的变更数量 */}
                            {(day.completed_changes ?? 0) > 0 && (
                              <div className="cocursor-daily-card-metric-row">
                                <div className="cocursor-daily-card-icon changes">✓</div>
                                <div className="cocursor-daily-card-metric-value">
                                  {day.completed_changes}
                                </div>
                                <div className="cocursor-daily-card-metric-label">
                                  {t("workAnalysis.dailyDetails.completedChanges")}
                                </div>
                              </div>
                            )}
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
            className="cocursor-modal cocursor-work-daily-report-modal"
            onClick={(e) => e.stopPropagation()}
          >
            {reportModalType === "view" ? (
              <>
                <div className="cocursor-work-daily-report-header">
                  <div className="cocursor-work-daily-report-title">
                    <span className="cocursor-work-daily-report-icon">📊</span>
                    <div>
                      <h2>{selectedDate}</h2>
                      <span className="cocursor-work-daily-report-subtitle">{t("workAnalysis.dailyReport.viewTitle")}</span>
                    </div>
                  </div>
                  <div className="cocursor-work-daily-report-actions screenshot-ignore">
                    <button
                      className="cocursor-btn icon-btn"
                      onClick={async () => {
                        setScreenshotMode(true);
                        // 等待 DOM 更新后再截图
                        setTimeout(async () => {
                          await takeScreenshot();
                          setScreenshotMode(false);
                        }, 100);
                      }}
                      disabled={isCapturing || !dailySummary}
                      title={t("dailyReport.saveScreenshot")}
                    >
                      {isCapturing ? "⏳" : "📷"}
                    </button>
                    <button
                      className="cocursor-btn icon-btn"
                      onClick={async () => {
                        setScreenshotMode(true);
                        setTimeout(async () => {
                          await copyToClipboard();
                          setScreenshotMode(false);
                        }, 100);
                      }}
                      disabled={isCapturing || !dailySummary}
                      title={t("dailyReport.copyToClipboard")}
                    >
                      📋
                    </button>
                    <button className="cocursor-modal-close-btn" onClick={handleCloseModal}>×</button>
                  </div>
                </div>
                {loadingReport ? (
                  <div className="cocursor-work-daily-report-loading">
                    <div className="cocursor-loading-spinner"></div>
                    <span>{t("workAnalysis.loading")}</span>
                  </div>
                ) : dailySummary ? (
                  <div 
                    ref={reportContentRef} 
                    className={`cocursor-daily-report-content ${screenshotMode ? "screenshot-mode" : ""}`}
                    data-screenshot-target="true"
                  >
                    {/* 统计卡片 */}
                    <DailyReportStats
                      totalSessions={dailySummary.total_sessions}
                      projectCount={dailySummary.projects?.length || 0}
                      efficiencyMetrics={dailySummary.efficiency_metrics}
                    />
                    
                    {/* 工作分类 */}
                    {dailySummary.work_categories && (
                      <WorkCategoriesChart categories={dailySummary.work_categories} />
                    )}
                    
                    {/* 时间分布 */}
                    {dailySummary.time_distribution && (
                      <TimeDistributionChart distribution={dailySummary.time_distribution} />
                    )}
                    
                    {/* 代码变更 */}
                    {dailySummary.code_changes && (
                      <CodeChangesStats codeChanges={dailySummary.code_changes} />
                    )}
                    
                    {/* 项目详情 */}
                    {dailySummary.projects && dailySummary.projects.length > 0 && (
                      <ProjectDetails projects={dailySummary.projects} screenshotMode={screenshotMode} />
                    )}
                    
                    {/* Markdown 摘要 */}
                    <div className="cocursor-daily-report-section">
                      <h4 className="cocursor-daily-report-section-title">
                        <span className="section-icon">📝</span>
                        {t("dailyReport.summary")}
                      </h4>
                      <div className="cocursor-daily-report-markdown-container">
                        <ReactMarkdown
                          remarkPlugins={[remarkGfm]}
                          components={{
                            h1: ({ children }) => <h1 className="cocursor-md-h1">{children}</h1>,
                            h2: ({ children }) => <h2 className="cocursor-md-h2">{children}</h2>,
                            h3: ({ children }) => <h3 className="cocursor-md-h3">{children}</h3>,
                            h4: ({ children }) => <h4 className="cocursor-md-h4">{children}</h4>,
                            p: ({ children }) => <p className="cocursor-md-p">{children}</p>,
                            ul: ({ children }) => <ul className="cocursor-md-ul">{children}</ul>,
                            ol: ({ children }) => <ol className="cocursor-md-ol">{children}</ol>,
                            li: ({ children }) => <li className="cocursor-md-li">{children}</li>,
                            code: ({ className, children, ...props }) => {
                              const isInline = !className;
                              return isInline ? (
                                <code className="cocursor-md-code-inline" {...props}>{children}</code>
                              ) : (
                                <code className={`cocursor-md-code-block ${className || ""}`} {...props}>{children}</code>
                              );
                            },
                            pre: ({ children }) => <pre className="cocursor-md-pre">{children}</pre>,
                            blockquote: ({ children }) => <blockquote className="cocursor-md-blockquote">{children}</blockquote>,
                            a: ({ href, children }) => (
                              <a href={href} className="cocursor-md-link" target="_blank" rel="noopener noreferrer">{children}</a>
                            ),
                            strong: ({ children }) => <strong className="cocursor-md-strong">{children}</strong>,
                            em: ({ children }) => <em className="cocursor-md-em">{children}</em>,
                            hr: () => <hr className="cocursor-md-hr" />,
                            table: ({ children }) => <table className="cocursor-md-table">{children}</table>,
                            thead: ({ children }) => <thead className="cocursor-md-thead">{children}</thead>,
                            tbody: ({ children }) => <tbody className="cocursor-md-tbody">{children}</tbody>,
                            tr: ({ children }) => <tr className="cocursor-md-tr">{children}</tr>,
                            th: ({ children }) => <th className="cocursor-md-th">{children}</th>,
                            td: ({ children }) => <td className="cocursor-md-td">{children}</td>,
                          }}
                        >
                          {dailySummary.summary}
                        </ReactMarkdown>
                      </div>
                    </div>
                    
                    {/* 截图模式下的水印 */}
                    {screenshotMode && (
                      <div className="cocursor-daily-report-watermark">
                        Generated by CoCursor · {selectedDate}
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="cocursor-work-daily-report-empty">
                    <span className="cocursor-empty-icon">📝</span>
                    <span>{t("workAnalysis.dailyReport.notAvailable")}</span>
                  </div>
                )}
                <div className="cocursor-work-daily-report-footer screenshot-ignore">
                  <button className="cocursor-btn primary" onClick={handleCloseModal}>
                    {t("workAnalysis.dailyReport.close")}
                  </button>
                </div>
              </>
            ) : (
              <>
                {/* 弹框头部 */}
                <div className="cocursor-work-daily-report-header">
                  <div className="cocursor-work-daily-report-title">
                    <span className="cocursor-work-daily-report-icon">
                      {loadingSkillStatus ? "⏳" : skillInstalled ? "✨" : "📦"}
                    </span>
                    <div>
                      <h2>{selectedDate}</h2>
                      <span className="cocursor-work-daily-report-subtitle">
                        {loadingSkillStatus 
                          ? t("workAnalysis.dailyReport.checkingSkill")
                          : skillInstalled 
                            ? t("workAnalysis.dailyReport.generateTitle")
                            : t("workAnalysis.dailyReport.needInstallSkill")}
                      </span>
                    </div>
                  </div>
                  <button className="cocursor-modal-close-btn" onClick={handleCloseModal}>×</button>
                </div>

                {/* 加载中状态 */}
                {loadingSkillStatus && (
                  <div className="cocursor-work-daily-report-loading">
                    <div className="cocursor-loading-spinner"></div>
                    <span>{t("workAnalysis.dailyReport.checkingSkill")}</span>
                  </div>
                )}

                {/* 未安装技能 */}
                {!loadingSkillStatus && !skillInstalled && (
                  <div className="cocursor-skill-install-guide">
                    <p style={{ marginBottom: "16px", lineHeight: "1.6" }}>
                      {t("workAnalysis.dailyReport.skillDescription")}
                    </p>
                    <ul style={{ 
                      margin: "0 0 20px 0", 
                      paddingLeft: "20px",
                      lineHeight: "1.8",
                      color: "var(--vscode-descriptionForeground)"
                    }}>
                      <li>{t("workAnalysis.dailyReport.skillFeature1")}</li>
                      <li>{t("workAnalysis.dailyReport.skillFeature2")}</li>
                      <li>{t("workAnalysis.dailyReport.skillFeature3")}</li>
                    </ul>
                    <div style={{ display: "flex", gap: "12px", justifyContent: "flex-end" }}>
                      <button
                        onClick={handleGoToMarketplace}
                        style={{
                          background: "transparent",
                          border: "1px solid var(--vscode-button-background)",
                          color: "var(--vscode-button-background)",
                          padding: "8px 16px",
                          borderRadius: "4px",
                          cursor: "pointer"
                        }}
                      >
                        {t("workAnalysis.dailyReport.viewInMarketplace")}
                      </button>
                      <button
                        onClick={handleInstallSkill}
                        disabled={installingSkill}
                        style={{
                          background: "var(--vscode-button-background)",
                          color: "var(--vscode-button-foreground)",
                          border: "none",
                          padding: "8px 20px",
                          borderRadius: "4px",
                          cursor: installingSkill ? "not-allowed" : "pointer",
                          opacity: installingSkill ? 0.7 : 1,
                          display: "flex",
                          alignItems: "center",
                          gap: "8px"
                        }}
                      >
                        {installingSkill && <span className="cocursor-loading-spinner-small"></span>}
                        {installingSkill 
                          ? t("workAnalysis.dailyReport.installing")
                          : t("workAnalysis.dailyReport.installNow")}
                      </button>
                    </div>
                  </div>
                )}

                {/* 已安装技能 - 显示使用说明 */}
                {!loadingSkillStatus && skillInstalled && (
                  <div className="cocursor-skill-usage-guide">
                    <p style={{ marginBottom: "16px", fontWeight: 500 }}>
                      {t("workAnalysis.dailyReport.usageTitle")}
                    </p>
                    
                    {/* 方式一：Slash 命令 */}
                    <div style={{ marginBottom: "20px" }}>
                      <div style={{ 
                        fontSize: "13px", 
                        color: "var(--vscode-descriptionForeground)",
                        marginBottom: "8px"
                      }}>
                        {t("workAnalysis.dailyReport.method1")}
                      </div>
                      <div
                        style={{
                          backgroundColor: "var(--vscode-input-background)",
                          border: "1px solid var(--vscode-input-border)",
                          borderRadius: "4px",
                          padding: "12px",
                          fontFamily: "monospace",
                          fontSize: "13px"
                        }}
                      >
                        /daily-summary {selectedDate}
                      </div>
                    </div>

                    {/* 方式二：自然语言 */}
                    <div style={{ marginBottom: "20px" }}>
                      <div style={{ 
                        fontSize: "13px", 
                        color: "var(--vscode-descriptionForeground)",
                        marginBottom: "8px"
                      }}>
                        {t("workAnalysis.dailyReport.method2")}
                      </div>
                      <div style={{
                        backgroundColor: "var(--vscode-input-background)",
                        border: "1px solid var(--vscode-input-border)",
                        borderRadius: "4px",
                        padding: "12px",
                        fontSize: "13px",
                        color: "var(--vscode-foreground)",
                        lineHeight: "1.6"
                      }}>
                        <div style={{ marginBottom: "4px" }}>"{t("workAnalysis.dailyReport.nlExample1")}"</div>
                        <div>"{t("workAnalysis.dailyReport.nlExample2", { date: selectedDate })}"</div>
                      </div>
                    </div>

                    {/* 提示 */}
                    <div style={{
                      backgroundColor: "var(--vscode-inputValidation-infoBackground)",
                      border: "1px solid var(--vscode-inputValidation-infoBorder)",
                      borderRadius: "4px",
                      padding: "12px",
                      fontSize: "13px",
                      display: "flex",
                      alignItems: "flex-start",
                      gap: "8px"
                    }}>
                      <span>💡</span>
                      <span>{t("workAnalysis.dailyReport.autoRefreshTip")}</span>
                    </div>

                    <div style={{ marginTop: "20px", textAlign: "right" }}>
                      <button
                        onClick={handleCloseModal}
                        className="cocursor-btn primary"
                      >
                        {t("workAnalysis.dailyReport.understood")}
                      </button>
                    </div>
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
