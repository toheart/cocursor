/**
 * 代码分析配置组件
 * 一站式面板设计：配置 + 生成合并为紧凑的操作界面
 */

import React, { useState, useEffect, useCallback, useRef } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
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
  { value: "static", label: "Static", desc: "最快，精度低" },
  { value: "cha", label: "CHA", desc: "保守，快速" },
  { value: "rta", label: "RTA", desc: "推荐，平衡", recommended: true },
  { value: "vta", label: "VTA", desc: "最精确，较慢" },
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

  // 展开/折叠状态
  const [showAdvanced, setShowAdvanced] = useState(false);

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
      clearInterval(pollingIntervalRef.current);
      pollingIntervalRef.current = null;
    }
  }, []);

  // 组件卸载时清理
  useEffect(() => {
    return () => {
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
    };
  }, []);

  // 轮询任务进度
  const pollProgress = useCallback(
    async (taskId: string) => {
      if (taskCompletedRef.current) {
        return;
      }

      try {
        const task = (await apiService.getGenerationProgress(
          taskId,
        )) as GenerationTask;
        setCurrentTask(task);

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

      setGenerating(true);
      setCurrentTask(null);
      setLastError(null);
      showToast(t("codeAnalysis.generating"), "success");

      const result = await apiService.generateCallGraphWithConfig({
        project_path: projectPath,
        entry_points: selectedEntryPoints,
        exclude: excludePaths.split("\n").filter((p) => p.trim()),
        algorithm,
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

      const interval = setInterval(() => {
        pollProgress(taskId);
      }, 1000);
      pollingIntervalRef.current = interval;

      await pollProgress(taskId);
    } catch (error) {
      console.error("Failed to start call graph generation:", error);
      showToast(t("codeAnalysis.error.generate"), "error");
      setGenerating(false);
      setCurrentTask(null);
    }
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
                      <span className="ca-algorithm-rec">推荐</span>
                    )}
                  </span>
                  <span className="ca-algorithm-desc">{opt.desc}</span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* 高级配置 */}
        {showAdvanced && (
          <div className="ca-section ca-section-advanced">
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

      {/* 使用说明 */}
      <div className="ca-tips">
        <h3>{t("codeAnalysis.howToUse")}</h3>
        <ol>
          <li>{t("codeAnalysis.step1")}</li>
          <li>{t("codeAnalysis.step2")}</li>
          <li>{t("codeAnalysis.step3")}</li>
        </ol>
      </div>
    </div>
  );
};

export default CodeAnalysisConfig;
