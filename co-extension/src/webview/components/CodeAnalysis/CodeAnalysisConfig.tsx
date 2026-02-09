/**
 * 代码分析配置组件
 * 一站式面板设计：配置 + 生成合并为紧凑的操作界面
 */

import React, { useState, useEffect, useCallback, useRef } from "react";
import { useTranslation } from "react-i18next";
import {
  apiService,
  type DiffAnalysisResult,
  type ImpactAnalysisResult,
} from "../../services/api";
import { useToast } from "../../hooks";
import { ToastContainer, Button, Loading } from "../shared";

// 入口函数候选
interface EntryPointCandidate {
  file: string;
  function: string;
  type: string;
  priority: number;
  recommended: boolean;
}

// 扫描入口函数响应
interface ScanEntryPointsResponse {
  project_name: string;
  remote_url: string;
  candidates: EntryPointCandidate[];
  default_exclude: string[];
}

// 调用图状态
interface CallGraphStatus {
  exists: boolean;
  up_to_date: boolean;
  current_commit?: string;
  head_commit?: string;
  commits_behind?: number;
  project_registered: boolean;
  db_path?: string;
  created_at?: string;
  func_count?: number;
  valid_go_module: boolean;
  go_module_error?: string;
}

// 项目配置
interface ProjectConfig {
  entry_points: string[];
  exclude: string[];
  algorithm: string;
  integration_test_dir?: string;
  integration_test_tag?: string;
}

// 生成结果
interface GenerateResponse {
  commit: string;
  func_count: number;
  edge_count: number;
  generation_time_ms: number;
  db_path: string;
  actual_algorithm?: string;
  fallback?: boolean;
  fallback_reason?: string;
}

// 生成任务进度
interface GenerationTask {
  task_id: string;
  project_id: string;
  project_path: string;
  commit: string;
  status: "pending" | "running" | "completed" | "failed";
  progress: number;
  message: string;
  result?: {
    commit: string;
    func_count: number;
    edge_count: number;
    generation_time_ms: number;
    db_path: string;
    actual_algorithm?: string;
    fallback?: boolean;
    fallback_reason?: string;
  };
  error?: string;
  error_code?: string;
  suggestion?: string;
  details?: string;
  started_at?: string;
  completed_at?: string;
}

// 算法选项
const ALGORITHM_OPTIONS = [
  { value: "static", label: "Static", descKey: "codeAnalysis.algorithmDesc.static" },
  { value: "cha", label: "CHA", descKey: "codeAnalysis.algorithmDesc.cha" },
  { value: "rta", label: "RTA", descKey: "codeAnalysis.algorithmDesc.rta", recommended: true },
  { value: "vta", label: "VTA", descKey: "codeAnalysis.algorithmDesc.vta", warning: true },
];

export const CodeAnalysisConfig: React.FC = () => {
  const { t } = useTranslation();
  const { showToast, toasts } = useToast();

  // 状态
  const [loading, setLoading] = useState(true);
  const [scanning, setScanning] = useState(false);
  const [generating, setGenerating] = useState(false);

  // 项目信息
  const [projectPath, setProjectPath] = useState<string>("");
  const [projectName, setProjectName] = useState<string>("");

  // 配置
  const [candidates, setCandidates] = useState<EntryPointCandidate[]>([]);
  const [selectedEntryPoints, setSelectedEntryPoints] = useState<string[]>([]);
  const [excludePaths, setExcludePaths] =
    useState<string>("vendor/\n*_test.go");
  const [algorithm, setAlgorithm] = useState<string>("rta");

  // Base commit / 分支
  const [baseCommit, setBaseCommit] = useState<string>("HEAD");
  // 集成测试配置
  const [integrationTestDir, setIntegrationTestDir] = useState<string>("");
  const [integrationTestTag, setIntegrationTestTag] = useState<string>("");

  // 展开/折叠状态
  const [showAdvanced, setShowAdvanced] = useState(false);

  // 影响面分析状态
  const [impactCommitRange, setImpactCommitRange] = useState<string>("");
  const [impactDepth, setImpactDepth] = useState<number>(3);
  const [analyzingImpact, setAnalyzingImpact] = useState(false);
  const [diffResult, setDiffResult] = useState<DiffAnalysisResult | null>(null);
  const [impactResult, setImpactResult] =
    useState<ImpactAnalysisResult | null>(null);
  const [expandedImpacts, setExpandedImpacts] = useState<Set<number>>(
    new Set(),
  );

  // 调用图状态
  const [callGraphStatus, setCallGraphStatus] =
    useState<CallGraphStatus | null>(null);
  const [generateResult, setGenerateResult] = useState<GenerateResponse | null>(
    null,
  );

  // Go 模块验证错误
  const [moduleError, setModuleError] = useState<string | null>(null);
  // 算法失败错误
  const [lastError, setLastError] = useState<{
    message: string;
    code?: string;
    suggestion?: string;
    details?: string;
  } | null>(null);

  // 异步生成任务状态
  const [currentTask, setCurrentTask] = useState<GenerationTask | null>(null);
  const pollingIntervalRef = useRef<ReturnType<typeof setInterval> | null>(
    null,
  );
  // 用于防止重复处理完成状态
  const taskCompletedRef = useRef<boolean>(false);
  // 轮询超时保护（10 分钟）
  const pollStartTimeRef = useRef<number>(0);
  // 进度停滞检测
  const lastProgressRef = useRef<{ progress: number; time: number }>({ progress: 0, time: 0 });
  // 记录已初始化的项目路径
  const initializedProjectPathRef = useRef<string>("");

  // 检查调用图状态
  const checkStatus = useCallback(
    async (path: string): Promise<boolean> => {
      try {
        setLoading(true);
        setModuleError(null);
        const status = await apiService.checkCallGraphStatus(path);
        const callGraphStatus = status as CallGraphStatus;
        setCallGraphStatus(callGraphStatus);

        // 检查是否为有效的 Go 模块
        if (!callGraphStatus.valid_go_module) {
          setModuleError(
            callGraphStatus.go_module_error || t("codeAnalysis.error.noGoMod"),
          );
          return false;
        }
        return true;
      } catch (error) {
        console.error("Failed to check status:", error);
        showToast(t("codeAnalysis.error.checkStatus"), "error");
        return false;
      } finally {
        setLoading(false);
      }
    },
    [t, showToast],
  );

  // 扫描入口函数
  const scanEntryPoints = useCallback(
    async (path: string, config?: ProjectConfig | null) => {
      try {
        setScanning(true);
        setModuleError(null);
        const result = await apiService.scanEntryPoints(path);
        const response = result as ScanEntryPointsResponse;

        setProjectName(response.project_name);
        setCandidates(response.candidates);

        if (!config && response.default_exclude.length > 0) {
          setExcludePaths(response.default_exclude.join("\n"));
        }

        // 未加载配置且当前未选择时，默认选中推荐入口
        if (!config) {
          setSelectedEntryPoints((prev) => {
            if (prev.length === 0) {
              const recommended = response.candidates
                .filter((c) => c.recommended)
                .map((c) => `${c.file}:${c.function}`);
              return recommended;
            }
            return prev;
          });
        }
      } catch (error: any) {
        console.error("Failed to scan entry points:", error);
        const errorMessage = error?.message || error?.toString() || "";
        if (
          errorMessage.includes("go.mod") ||
          errorMessage.includes("Go module") ||
          errorMessage.includes("invalid Go module")
        ) {
          setModuleError(errorMessage);
        } else {
          showToast(t("codeAnalysis.error.scan"), "error");
        }
      } finally {
        setScanning(false);
      }
    },
    [t, showToast],
  );

  // 切换入口函数选择
  const toggleEntryPoint = (candidate: EntryPointCandidate) => {
    const key = `${candidate.file}:${candidate.function}`;
    setSelectedEntryPoints((prev) =>
      prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key],
    );
  };

  // 加载已保存的项目配置
  const loadProjectConfig = useCallback(
    async (path: string): Promise<ProjectConfig | null> => {
      try {
        const config = (await apiService.getProjectConfig(
          path,
        )) as ProjectConfig;
        if (config && !(config as any).error) {
          setSelectedEntryPoints(config.entry_points || []);
          setExcludePaths((config.exclude || []).join("\n"));
          setAlgorithm(config.algorithm || "rta");
          if (config.integration_test_dir) {
            setIntegrationTestDir(config.integration_test_dir);
          }
          if (config.integration_test_tag) {
            setIntegrationTestTag(config.integration_test_tag);
          }
          return config;
        }
      } catch (error) {
        // 未注册项目时不提示错误
      }
      return null;
    },
    [],
  );

  // 初始化项目配置和入口函数
  const initializeProject = useCallback(
    async (path: string) => {
      const valid = await checkStatus(path);
      if (!valid) {
        setLoading(false);
        return;
      }
      const config = await loadProjectConfig(path);
      await scanEntryPoints(path, config);
      setLoading(false);
    },
    [checkStatus, loadProjectConfig, scanEntryPoints],
  );

  // 初始化加载
  useEffect(() => {
    if (initializedProjectPathRef.current) {
      return;
    }
    const workspacePath = (window as any).__WORKSPACE_PATH__;
    if (workspacePath) {
      setProjectPath(workspacePath);
      initializedProjectPathRef.current = workspacePath;
      initializeProject(workspacePath);
    } else {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 项目路径变化时自动重新扫描
  useEffect(() => {
    if (!projectPath || projectPath === initializedProjectPathRef.current) {
      return;
    }

    const timer = setTimeout(() => {
      initializedProjectPathRef.current = projectPath;
      initializeProject(projectPath);
    }, 500);

    return () => clearTimeout(timer);
  }, [projectPath, initializeProject]);

  // 停止轮询
  const stopPolling = useCallback(() => {
    if (pollingIntervalRef.current) {
      clearTimeout(pollingIntervalRef.current);
      pollingIntervalRef.current = null;
    }
  }, []);

  // 组件卸载时清理
  useEffect(() => {
    return () => {
      if (pollingIntervalRef.current) {
        clearTimeout(pollingIntervalRef.current);
      }
    };
  }, []);

  // 轮询任务进度
  const pollProgress = useCallback(
    async (taskId: string) => {
      if (taskCompletedRef.current) {
        return;
      }

      // 超时保护：超过 10 分钟自动放弃
      const POLL_TIMEOUT_MS = 10 * 60 * 1000;
      const elapsed = Date.now() - pollStartTimeRef.current;
      if (elapsed > POLL_TIMEOUT_MS) {
        taskCompletedRef.current = true;
        stopPolling();
        setGenerating(false);
        setLastError({
          message: t("codeAnalysis.error.timeout"),
          code: "TIMEOUT",
        });
        showToast(t("codeAnalysis.error.timeout"), "error");
        setCurrentTask(null);
        return;
      }

      try {
        const task = (await apiService.getGenerationProgress(
          taskId,
        )) as GenerationTask;
        setCurrentTask(task);

        // 进度停滞检测：如果进度 2 分钟内没有变化，标记为可能异常
        const now = Date.now();
        if (task.status === "running") {
          if (lastProgressRef.current.progress !== task.progress) {
            lastProgressRef.current = { progress: task.progress, time: now };
          } else if (now - lastProgressRef.current.time > 2 * 60 * 1000) {
            // 进度停滞超过 2 分钟，自动视为失败
            taskCompletedRef.current = true;
            stopPolling();
            setGenerating(false);
            setLastError({
              message: t("codeAnalysis.error.stalled"),
              code: "STALLED",
            });
            showToast(t("codeAnalysis.error.stalled"), "error");
            setCurrentTask(null);
            return;
          }
        }

        if (task.status === "completed") {
          if (taskCompletedRef.current) {
            return;
          }
          taskCompletedRef.current = true;

          stopPolling();
          setGenerating(false);
          if (task.result) {
            setGenerateResult({
              commit: task.result.commit,
              func_count: task.result.func_count,
              edge_count: task.result.edge_count,
              generation_time_ms: task.result.generation_time_ms,
              db_path: task.result.db_path,
              actual_algorithm: task.result.actual_algorithm,
              fallback: task.result.fallback,
              fallback_reason: task.result.fallback_reason,
            });
            setLastError(null);

            showToast(
              t("codeAnalysis.success.generate", {
                funcCount: task.result.func_count,
                edgeCount: task.result.edge_count,
                time: (task.result.generation_time_ms / 1000).toFixed(1),
              }),
              "success",
            );
          }
          await checkStatus(projectPath);
          setCurrentTask(null);
        } else if (task.status === "failed") {
          if (taskCompletedRef.current) {
            return;
          }
          taskCompletedRef.current = true;

          stopPolling();
          setGenerating(false);
          setLastError({
            message: task.error || t("codeAnalysis.error.generate"),
            code: task.error_code,
            suggestion: task.suggestion,
            details: task.details,
          });
          if (task.error_code === "ALGORITHM_FAILED") {
            showToast(t("codeAnalysis.error.algorithmFailed"), "error");
          } else {
            showToast(task.error || t("codeAnalysis.error.generate"), "error");
          }
          setCurrentTask(null);
        }
      } catch (error: any) {
        console.error("Failed to poll progress:", error);
        const errorMessage = error?.message || error?.toString() || "";
        if (
          errorMessage.includes("404") ||
          errorMessage.includes("not found") ||
          errorMessage.includes("Task not found")
        ) {
          taskCompletedRef.current = true;
          stopPolling();
          setGenerating(false);
          setLastError({
            message: t("codeAnalysis.error.taskNotFound"),
            code: "TASK_NOT_FOUND",
          });
          showToast(t("codeAnalysis.error.taskNotFound"), "error");
          setCurrentTask(null);
        }
      }
    },
    [projectPath, stopPolling, showToast, t, checkStatus],
  );

  // 生成调用图
  const handleGenerate = async () => {
    try {
      if (selectedEntryPoints.length === 0) {
        showToast(t("codeAnalysis.error.noEntryPoints"), "error");
        return;
      }

      taskCompletedRef.current = false;
      pollStartTimeRef.current = Date.now();
      lastProgressRef.current = { progress: 0, time: Date.now() };

      setGenerating(true);
      setCurrentTask(null);
      setLastError(null);
      showToast(t("codeAnalysis.generating"), "success");

      const result = await apiService.generateCallGraphWithConfig({
        project_path: projectPath,
        entry_points: selectedEntryPoints,
        exclude: excludePaths.split("\n").filter((p) => p.trim()),
        algorithm,
        commit: baseCommit || undefined,
        integration_test_dir: integrationTestDir || undefined,
        integration_test_tag: integrationTestTag || undefined,
      });
      const taskId = result.task_id;

      setCurrentTask({
        task_id: taskId,
        project_id: "",
        project_path: projectPath,
        commit: "",
        status: "pending",
        progress: 0,
        message: t("codeAnalysis.taskStarting"),
      });

      // 渐进式轮询：2s → 3s → 5s，避免频繁请求
      let pollCount = 0;
      const getPollInterval = () => {
        pollCount++;
        if (pollCount <= 2) return 2000;
        if (pollCount <= 5) return 3000;
        return 5000;
      };
      const schedulePoll = () => {
        if (taskCompletedRef.current) return;
        pollingIntervalRef.current = setTimeout(() => {
          pollProgress(taskId).then(() => schedulePoll());
        }, getPollInterval());
      };
      schedulePoll();

      await pollProgress(taskId);
    } catch (error) {
      console.error("Failed to start call graph generation:", error);
      showToast(t("codeAnalysis.error.generate"), "error");
      setGenerating(false);
      setCurrentTask(null);
    }
  };

  // 影响面分析
  const handleAnalyzeImpact = async () => {
    if (!callGraphStatus?.exists) {
      showToast(t("codeAnalysis.impact.needCallGraph"), "error");
      return;
    }

    try {
      setAnalyzingImpact(true);
      setDiffResult(null);
      setImpactResult(null);
      setExpandedImpacts(new Set());

      // 1. 分析 diff
      const commitRange = impactCommitRange.trim() || "HEAD~1..HEAD";
      const diff = await apiService.analyzeDiff(projectPath, commitRange);
      setDiffResult(diff);

      if (!diff.changed_functions || diff.changed_functions.length === 0) {
        setAnalyzingImpact(false);
        return;
      }

      // 2. 查询影响面
      const functions = diff.changed_functions.map((fn) => fn.full_name);
      const impact = await apiService.queryImpact({
        projectPath,
        functions,
        depth: impactDepth,
      });
      setImpactResult(impact);

      // 默认展开前 3 个
      const defaultExpanded = new Set<number>();
      for (let i = 0; i < Math.min(3, impact.impacts.length); i++) {
        defaultExpanded.add(i);
      }
      setExpandedImpacts(defaultExpanded);
    } catch (error) {
      console.error("Impact analysis failed:", error);
      showToast(
        error instanceof Error
          ? error.message
          : t("codeAnalysis.impact.error"),
        "error",
      );
    } finally {
      setAnalyzingImpact(false);
    }
  };

  // 切换展开/折叠
  const toggleImpactExpand = (index: number) => {
    setExpandedImpacts((prev) => {
      const next = new Set(prev);
      if (next.has(index)) {
        next.delete(index);
      } else {
        next.add(index);
      }
      return next;
    });
  };

  // 渲染状态指示器
  const renderStatusIndicator = () => {
    if (!callGraphStatus) return null;

    if (callGraphStatus.exists) {
      if (callGraphStatus.up_to_date) {
        return (
          <span className="ca-status ca-status-success">
            <span className="ca-status-dot" />
            {t("codeAnalysis.upToDate")}
          </span>
        );
      }
      return (
        <span className="ca-status ca-status-warning">
          <span className="ca-status-dot" />
          {t("codeAnalysis.outdated", {
            count: callGraphStatus.commits_behind || 0,
          })}
        </span>
      );
    }
    return (
      <span className="ca-status ca-status-idle">
        <span className="ca-status-dot" />
        {t("codeAnalysis.notGenerated")}
      </span>
    );
  };

  // 加载中状态
  if (loading) {
    return (
      <div className="ca-container">
        <Loading message={t("common.loading")} />
      </div>
    );
  }

  // 没有工作区路径
  if (!projectPath) {
    return (
      <div className="ca-container">
        <div className="ca-empty">
          <div className="ca-empty-icon">📂</div>
          <h3>{t("codeAnalysis.noWorkspace")}</h3>
          <p>{t("codeAnalysis.noWorkspaceDesc")}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="ca-container">
      <ToastContainer toasts={toasts} />

      {/* 头部 */}
      <header className="ca-header">
        <div className="ca-header-content">
          <h1 className="ca-title">
            <span className="ca-title-icon">⚡</span>
            {t("codeAnalysis.title")}
          </h1>
          <p className="ca-subtitle">{t("codeAnalysis.subtitle")}</p>
        </div>
      </header>

      {/* Go 模块错误 */}
      {moduleError && (
        <div className="ca-error-banner">
          <div className="ca-error-banner-icon">⚠️</div>
          <div className="ca-error-banner-content">
            <strong>{t("codeAnalysis.error.invalidGoModule")}</strong>
            <p>{moduleError}</p>
          </div>
          <Button
            onClick={() => {
              setModuleError(null);
              initializeProject(projectPath);
            }}
            variant="secondary"
            className="ca-error-banner-action"
          >
            {t("common.retry")}
          </Button>
        </div>
      )}

      {/* 主面板 */}
      <div className="ca-panel">
        {/* 项目信息行 */}
        <div className="ca-section ca-section-project">
          <div className="ca-project-row">
            <div className="ca-project-info">
              <span className="ca-project-name">
                {projectName || projectPath.split("/").pop()}
              </span>
              {renderStatusIndicator()}
            </div>
            {callGraphStatus?.exists && (
              <div className="ca-project-meta">
                <code className="ca-commit">
                  {callGraphStatus.current_commit?.substring(0, 7)}
                </code>
                <span className="ca-func-count">
                  {callGraphStatus.func_count?.toLocaleString()} 函数
                </span>
              </div>
            )}
          </div>
          <div className="ca-project-path-row">
            <input
              value={projectPath}
              onChange={(e) => setProjectPath(e.target.value)}
              className="ca-input ca-input-path"
              placeholder={t("codeAnalysis.projectPathPlaceholder")}
            />
          </div>
        </div>

        {/* 进度条 */}
        {generating && currentTask && (
          <div className="ca-section ca-section-progress">
            <div className="ca-progress">
              <div className="ca-progress-info">
                <span className="ca-progress-message">
                  {currentTask.status === "pending" && "⏳ "}
                  {currentTask.status === "running" && "🔄 "}
                  {currentTask.message || t("codeAnalysis.taskRunning")}
                </span>
                <span className="ca-progress-percent">
                  {currentTask.progress}%
                </span>
              </div>
              <div className="ca-progress-bar">
                <div
                  className="ca-progress-fill"
                  style={{ width: `${currentTask.progress}%` }}
                />
              </div>
            </div>
          </div>
        )}

        {/* 错误提示 */}
        {lastError && (
          <div className="ca-section ca-section-error">
            <div className="ca-error-box">
              <span className="ca-error-icon">⚠️</span>
              <div className="ca-error-content">
                <div className="ca-error-message">{lastError.message}</div>
                {lastError.details && (
                  <div className="ca-error-details">{lastError.details}</div>
                )}
                {lastError.suggestion && (
                  <div className="ca-error-suggestion">
                    💡 {lastError.suggestion}
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* 入口函数选择 */}
        <div className="ca-section ca-section-entries">
          <div className="ca-section-header">
            <h2 className="ca-section-title">
              {t("codeAnalysis.entryPoints")}
            </h2>
            <button
              className="ca-link-button"
              onClick={() => scanEntryPoints(projectPath)}
              disabled={scanning}
            >
              {scanning ? "⏳" : "↻"} {t("codeAnalysis.rescan")}
            </button>
          </div>

          {scanning ? (
            <div className="ca-scanning">
              <Loading message={t("codeAnalysis.scanning")} />
            </div>
          ) : (
            <div className="ca-entry-grid">
              {candidates.map((candidate, index) => {
                const key = `${candidate.file}:${candidate.function}`;
                const isSelected = selectedEntryPoints.includes(key);
                return (
                  <div
                    key={index}
                    className={`ca-entry-item ${isSelected ? "selected" : ""}`}
                    onClick={() => toggleEntryPoint(candidate)}
                  >
                    <div className="ca-entry-check">
                      {isSelected ? "✓" : ""}
                    </div>
                    <div className="ca-entry-content">
                      <span className="ca-entry-func">
                        {candidate.function}()
                      </span>
                      <span className="ca-entry-file">{candidate.file}</span>
                    </div>
                    {candidate.recommended && (
                      <span className="ca-entry-badge">★</span>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* 算法选择 */}
        <div className="ca-section ca-section-algorithm">
          <div className="ca-section-header">
            <h2 className="ca-section-title">{t("codeAnalysis.algorithm")}</h2>
            <button
              className="ca-link-button"
              onClick={() => setShowAdvanced(!showAdvanced)}
            >
              {showAdvanced ? "收起选项" : "更多选项"}
            </button>
          </div>
          <div className="ca-algorithm-grid">
            {ALGORITHM_OPTIONS.map((opt) => (
              <div
                key={opt.value}
                className={`ca-algorithm-item ${algorithm === opt.value ? "selected" : ""}`}
                onClick={() => setAlgorithm(opt.value)}
              >
                <div className="ca-algorithm-radio">
                  {algorithm === opt.value && "●"}
                </div>
                <div className="ca-algorithm-info">
                  <span className="ca-algorithm-label">
                    {opt.label}
                    {opt.recommended && (
                      <span className="ca-algorithm-rec">{t("codeAnalysis.recommended")}</span>
                    )}
                  </span>
                  <span className="ca-algorithm-desc">{t(opt.descKey)}</span>
                </div>
              </div>
            ))}
          </div>
          {algorithm === "vta" && (
            <div className="ca-algorithm-warning">
              ⚠ {t("codeAnalysis.vtaWarning")}
            </div>
          )}
        </div>

        {/* 高级配置 */}
        {showAdvanced && (
          <div className="ca-section ca-section-advanced">
            {/* Base Commit */}
            <div className="ca-field">
              <div className="ca-section-header">
                <h2 className="ca-section-title">
                  {t("codeAnalysis.baseCommit")}
                </h2>
              </div>
              <input
                value={baseCommit}
                onChange={(e) => setBaseCommit(e.target.value)}
                className="ca-input"
                placeholder="HEAD"
              />
              <span className="ca-hint">
                {t("codeAnalysis.baseCommitHint")}
              </span>
            </div>

            {/* 集成测试配置 */}
            <div className="ca-field">
              <div className="ca-section-header">
                <h2 className="ca-section-title">
                  {t("codeAnalysis.integrationTest")}
                </h2>
              </div>
              <div className="ca-field-row">
                <div className="ca-field-col">
                  <label className="ca-label">{t("codeAnalysis.testDir")}</label>
                  <input
                    value={integrationTestDir}
                    onChange={(e) => setIntegrationTestDir(e.target.value)}
                    className="ca-input"
                    placeholder="test/integration/"
                  />
                </div>
                <div className="ca-field-col">
                  <label className="ca-label">{t("codeAnalysis.testTag")}</label>
                  <input
                    value={integrationTestTag}
                    onChange={(e) => setIntegrationTestTag(e.target.value)}
                    className="ca-input"
                    placeholder="integration"
                  />
                </div>
              </div>
              <span className="ca-hint">
                {t("codeAnalysis.integrationTestHint")}
              </span>
            </div>

            {/* 排除路径 */}
            <div className="ca-field">
              <div className="ca-section-header">
                <h2 className="ca-section-title">
                  {t("codeAnalysis.excludePaths")}
                </h2>
              </div>
              <textarea
                value={excludePaths}
                onChange={(e) => setExcludePaths(e.target.value)}
                className="ca-textarea"
                rows={3}
                placeholder="vendor/&#10;*_test.go&#10;*.pb.go"
              />
              <span className="ca-hint">
                {t("codeAnalysis.excludePathsHint")}
              </span>
            </div>
          </div>
        )}

        {/* 操作按钮 */}
        <div className="ca-section ca-section-actions">
          <Button
            onClick={handleGenerate}
            loading={generating}
            variant="primary"
            disabled={
              generating || selectedEntryPoints.length === 0 || !!moduleError
            }
            className="ca-generate-button"
          >
            {generating
              ? t("codeAnalysis.generating")
              : callGraphStatus?.exists
                ? t("codeAnalysis.regenerate")
                : t("codeAnalysis.generate")}
          </Button>
        </div>
      </div>

      {/* 生成结果 */}
      {generateResult && !generating && (
        <div className="ca-result">
          <div className="ca-result-header">
            <span className="ca-result-title">
              ✓ {t("codeAnalysis.generateResult")}
            </span>
            {generateResult.actual_algorithm && (
              <span className="ca-result-algorithm">
                {generateResult.actual_algorithm.toUpperCase()}
              </span>
            )}
          </div>
          <div className="ca-result-stats">
            <div className="ca-result-stat">
              <span className="ca-result-stat-value">
                {generateResult.func_count.toLocaleString()}
              </span>
              <span className="ca-result-stat-label">
                {t("codeAnalysis.functions")}
              </span>
            </div>
            <div className="ca-result-stat">
              <span className="ca-result-stat-value">
                {generateResult.edge_count.toLocaleString()}
              </span>
              <span className="ca-result-stat-label">
                {t("codeAnalysis.edges")}
              </span>
            </div>
            <div className="ca-result-stat">
              <span className="ca-result-stat-value">
                {(generateResult.generation_time_ms / 1000).toFixed(1)}s
              </span>
              <span className="ca-result-stat-label">
                {t("codeAnalysis.generationTime")}
              </span>
            </div>
          </div>
        </div>
      )}

      {/* 影响面分析 */}
      {callGraphStatus?.exists && (
        <div className="ca-section">
          <h3 className="ca-section-title">
            {t("codeAnalysis.impact.title")}
          </h3>
          <p className="ca-section-subtitle">
            {t("codeAnalysis.impact.subtitle")}
          </p>

          {/* 分析配置 */}
          <div className="ca-impact-config">
            <div className="ca-form-group">
              <label className="ca-label">
                {t("codeAnalysis.impact.commitRange")}
              </label>
              <input
                type="text"
                className="ca-input"
                value={impactCommitRange}
                onChange={(e) => setImpactCommitRange(e.target.value)}
                placeholder={t("codeAnalysis.impact.commitRangePlaceholder")}
              />
              <span className="ca-hint">
                {t("codeAnalysis.impact.commitRangeHint")}
              </span>
            </div>
            <div className="ca-form-group ca-form-group-inline">
              <label className="ca-label">
                {t("codeAnalysis.impact.depth")}
              </label>
              <select
                className="ca-select"
                value={impactDepth}
                onChange={(e) => setImpactDepth(Number(e.target.value))}
              >
                {[1, 2, 3, 5, 10].map((d) => (
                  <option key={d} value={d}>
                    {d}
                  </option>
                ))}
              </select>
            </div>
            <Button
              onClick={handleAnalyzeImpact}
              disabled={analyzingImpact || generating}
              className="ca-btn-primary"
            >
              {analyzingImpact
                ? t("codeAnalysis.impact.analyzing")
                : t("codeAnalysis.impact.analyze")}
            </Button>
          </div>

          {/* 分析中状态 */}
          {analyzingImpact && (
            <div className="ca-loading-inline">
              <Loading />
              <span>{t("codeAnalysis.impact.analyzing")}</span>
            </div>
          )}

          {/* Diff 结果 */}
          {diffResult && !analyzingImpact && (
            <div className="ca-impact-results">
              {diffResult.changed_functions.length === 0 ? (
                <div className="ca-empty-hint">
                  {t("codeAnalysis.impact.noChanges")}
                </div>
              ) : (
                <>
                  <div className="ca-impact-diff">
                    <h4>
                      {t("codeAnalysis.impact.changedFunctions")} (
                      {diffResult.changed_functions.length})
                    </h4>
                    <div className="ca-impact-func-list">
                      {diffResult.changed_functions.map((fn, i) => (
                        <div key={i} className="ca-impact-func-item">
                          <span
                            className={`ca-change-badge ca-change-${fn.change_type}`}
                          >
                            {fn.change_type === "added"
                              ? "+"
                              : fn.change_type === "deleted"
                                ? "-"
                                : "~"}
                          </span>
                          <span className="ca-func-name">{fn.name}</span>
                          <span className="ca-func-file">
                            {fn.file}:{fn.line_start}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* 影响面结果 */}
                  {impactResult && (
                    <div className="ca-impact-detail">
                      <h4>{t("codeAnalysis.impact.impactResult")}</h4>

                      {/* 汇总统计 */}
                      <div className="ca-impact-summary">
                        <div className="ca-impact-stat">
                          <span className="ca-impact-stat-value">
                            {impactResult.summary.functions_analyzed}
                          </span>
                          <span className="ca-impact-stat-label">
                            {t("codeAnalysis.impact.changedFunctions")}
                          </span>
                        </div>
                        <div className="ca-impact-stat">
                          <span className="ca-impact-stat-value">
                            {impactResult.summary.total_affected}
                          </span>
                          <span className="ca-impact-stat-label">
                            {t("codeAnalysis.impact.affectedFunctions")}
                          </span>
                        </div>
                        <div className="ca-impact-stat">
                          <span className="ca-impact-stat-value">
                            {impactResult.summary.affected_files.length}
                          </span>
                          <span className="ca-impact-stat-label">
                            {t("codeAnalysis.impact.affectedFiles")}
                          </span>
                        </div>
                      </div>

                      {/* 每个函数的影响面 */}
                      {impactResult.impacts.map((impact, idx) => (
                        <div key={idx} className="ca-impact-item">
                          <div
                            className="ca-impact-item-header"
                            onClick={() => toggleImpactExpand(idx)}
                          >
                            <span
                              className={`ca-expand-icon ${expandedImpacts.has(idx) ? "expanded" : ""}`}
                            >
                              ▶
                            </span>
                            <span className="ca-impact-func-name">
                              {impact.display_name}
                            </span>
                            <span className="ca-impact-callers-count">
                              {impact.total_callers}{" "}
                              {t("codeAnalysis.impact.callers")}
                            </span>
                          </div>
                          {expandedImpacts.has(idx) && (
                            <div className="ca-impact-item-body">
                              {impact.file && (
                                <div className="ca-impact-file">
                                  {impact.file}
                                </div>
                              )}
                              {impact.callers.length === 0 ? (
                                <div className="ca-empty-hint">
                                  {t("codeAnalysis.impact.noCallers")}
                                </div>
                              ) : (
                                <div className="ca-caller-list">
                                  {impact.callers.map((caller, ci) => (
                                    <div
                                      key={ci}
                                      className="ca-caller-item"
                                      style={{
                                        paddingLeft: `${(caller.depth - 1) * 16 + 8}px`,
                                      }}
                                    >
                                      <span className="ca-depth-badge">
                                        {caller.depth}
                                      </span>
                                      <span className="ca-caller-name">
                                        {caller.display_name}
                                      </span>
                                      {caller.file && (
                                        <span className="ca-caller-file">
                                          {caller.file}:{caller.line}
                                        </span>
                                      )}
                                    </div>
                                  ))}
                                </div>
                              )}
                            </div>
                          )}
                        </div>
                      ))}

                      {/* 分析版本 */}
                      {impactResult.analysis_commit && (
                        <div className="ca-impact-version">
                          {t("codeAnalysis.impact.analysisCommit")}:{" "}
                          {impactResult.analysis_commit}
                        </div>
                      )}
                    </div>
                  )}
                </>
              )}
            </div>
          )}
        </div>
      )}

      {/* MCP 工具使用指引 */}
      {callGraphStatus?.exists && (
        <div className="ca-section ca-mcp-guide">
          <h3 className="ca-section-title">{t("codeAnalysis.mcpGuide.title")}</h3>
          <p className="ca-section-subtitle">{t("codeAnalysis.mcpGuide.subtitle")}</p>
          <div className="ca-mcp-tools">
            <div className="ca-mcp-tool-item">
              <div className="ca-mcp-tool-name">
                <code>analyze_diff_impact</code>
              </div>
              <div className="ca-mcp-tool-desc">{t("codeAnalysis.mcpGuide.analyzeDiff")}</div>
              <div className="ca-mcp-tool-example">{t("codeAnalysis.mcpGuide.analyzeDiffExample")}</div>
            </div>
            <div className="ca-mcp-tool-item">
              <div className="ca-mcp-tool-name">
                <code>query_impact</code>
              </div>
              <div className="ca-mcp-tool-desc">{t("codeAnalysis.mcpGuide.queryImpact")}</div>
              <div className="ca-mcp-tool-example">{t("codeAnalysis.mcpGuide.queryImpactExample")}</div>
            </div>
            <div className="ca-mcp-tool-item">
              <div className="ca-mcp-tool-name">
                <code>search_function</code>
              </div>
              <div className="ca-mcp-tool-desc">{t("codeAnalysis.mcpGuide.searchFunction")}</div>
              <div className="ca-mcp-tool-example">{t("codeAnalysis.mcpGuide.searchFunctionExample")}</div>
            </div>
          </div>
          <div className="ca-mcp-tip">
            {t("codeAnalysis.mcpGuide.tip")}
          </div>
        </div>
      )}
    </div>
  );
};

export default CodeAnalysisConfig;
