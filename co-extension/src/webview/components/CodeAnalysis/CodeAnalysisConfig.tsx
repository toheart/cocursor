/**
 * 代码分析配置组件
 * 用于配置 Go 代码影响面分析功能
 */

import React, { useState, useEffect, useCallback } from "react";
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
  started_at?: string;
  completed_at?: string;
}

// 算法选项
const ALGORITHM_OPTIONS = [
  { value: "static", label: "Static (最快，精度低)" },
  { value: "cha", label: "CHA (保守，快速)" },
  { value: "rta", label: "RTA (推荐，平衡)" },
  { value: "vta", label: "VTA (最精确，较慢)" },
];

export const CodeAnalysisConfig: React.FC = () => {
  const { t } = useTranslation();
  const { showToast, toasts } = useToast();

  // 状态
  const [loading, setLoading] = useState(true);
  const [scanning, setScanning] = useState(false);
  const [registering, setRegistering] = useState(false);
  const [generating, setGenerating] = useState(false);

  // 项目信息
  const [projectPath, setProjectPath] = useState<string>("");
  const [projectName, setProjectName] = useState<string>("");
  const [remoteUrl, setRemoteUrl] = useState<string>("");

  // 配置
  const [candidates, setCandidates] = useState<EntryPointCandidate[]>([]);
  const [selectedEntryPoints, setSelectedEntryPoints] = useState<string[]>([]);
  const [excludePaths, setExcludePaths] =
    useState<string>("vendor/\n*_test.go");
  const [algorithm, setAlgorithm] = useState<string>("rta");

  // 调用图状态
  const [callGraphStatus, setCallGraphStatus] =
    useState<CallGraphStatus | null>(null);
  const [generateResult, setGenerateResult] = useState<GenerateResponse | null>(
    null,
  );

  // Go 模块验证错误
  const [moduleError, setModuleError] = useState<string | null>(null);

  // 异步生成任务状态
  const [currentTask, setCurrentTask] = useState<GenerationTask | null>(null);
  const [pollingInterval, setPollingInterval] = useState<ReturnType<
    typeof setInterval
  > | null>(null);

  // 初始化加载
  useEffect(() => {
    const workspacePath = (window as any).__WORKSPACE_PATH__;
    if (workspacePath) {
      setProjectPath(workspacePath);
      checkStatus(workspacePath);
    } else {
      setLoading(false);
    }
  }, []);

  // 检查调用图状态
  const checkStatus = async (path: string) => {
    try {
      setLoading(true);
      setModuleError(null);
      const status = await apiService.checkCallGraphStatus(path);
      const callGraphStatus = status as CallGraphStatus;
      setCallGraphStatus(callGraphStatus);

      // 检查是否为有效的 Go 模块
      if (!callGraphStatus.valid_go_module) {
        // 不是有效的 Go 模块，设置错误状态
        setModuleError(
          callGraphStatus.go_module_error || t("codeAnalysis.error.noGoMod"),
        );
        return;
      }

      if (callGraphStatus.project_registered) {
        // 项目已注册，显示状态
      } else {
        // 项目未注册，扫描入口函数
        await scanEntryPoints(path);
      }
    } catch (error) {
      console.error("Failed to check status:", error);
      showToast(t("codeAnalysis.error.checkStatus"), "error");
    } finally {
      setLoading(false);
    }
  };

  // 扫描入口函数
  const scanEntryPoints = async (path: string) => {
    try {
      setScanning(true);
      setModuleError(null);
      const result = await apiService.scanEntryPoints(path);
      const response = result as ScanEntryPointsResponse;

      setProjectName(response.project_name);
      setRemoteUrl(response.remote_url);
      setCandidates(response.candidates);
      setExcludePaths(response.default_exclude.join("\n"));

      // 自动选择推荐的入口函数
      const recommended = response.candidates
        .filter((c) => c.recommended)
        .map((c) => `${c.file}:${c.function}`);
      setSelectedEntryPoints(recommended);
    } catch (error: any) {
      console.error("Failed to scan entry points:", error);
      // 检查是否是 Go 模块验证错误
      const errorMessage = error?.message || error?.toString() || "";
      if (
        errorMessage.includes("go.mod") ||
        errorMessage.includes("Go module") ||
        errorMessage.includes("invalid Go module")
      ) {
        // 设置模块错误状态，显示专门的错误界面
        setModuleError(errorMessage);
      } else {
        showToast(t("codeAnalysis.error.scan"), "error");
      }
    } finally {
      setScanning(false);
    }
  };

  // 切换入口函数选择
  const toggleEntryPoint = (candidate: EntryPointCandidate) => {
    const key = `${candidate.file}:${candidate.function}`;
    setSelectedEntryPoints((prev) =>
      prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key],
    );
  };

  // 注册项目
  const handleRegister = async () => {
    if (selectedEntryPoints.length === 0) {
      showToast(t("codeAnalysis.error.noEntryPoints"), "error");
      return;
    }

    try {
      setRegistering(true);
      await apiService.registerProject({
        project_path: projectPath,
        entry_points: selectedEntryPoints,
        exclude: excludePaths.split("\n").filter((p) => p.trim()),
        algorithm,
      });

      showToast(t("codeAnalysis.success.register"), "success");

      // 刷新状态
      await checkStatus(projectPath);
    } catch (error) {
      console.error("Failed to register project:", error);
      showToast(t("codeAnalysis.error.register"), "error");
    } finally {
      setRegistering(false);
    }
  };

  // 停止轮询
  const stopPolling = useCallback(() => {
    if (pollingInterval) {
      clearInterval(pollingInterval);
      setPollingInterval(null);
    }
  }, [pollingInterval]);

  // 组件卸载时清理
  useEffect(() => {
    return () => {
      if (pollingInterval) {
        clearInterval(pollingInterval);
      }
    };
  }, [pollingInterval]);

  // 轮询任务进度
  const pollProgress = useCallback(
    async (taskId: string) => {
      try {
        const task = (await apiService.getGenerationProgress(
          taskId,
        )) as GenerationTask;
        setCurrentTask(task);

        if (task.status === "completed") {
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

            // 显示成功消息
            showToast(
              t("codeAnalysis.success.generate", {
                funcCount: task.result.func_count,
                edgeCount: task.result.edge_count,
                time: (task.result.generation_time_ms / 1000).toFixed(1),
              }),
              "success",
            );

            // 如果发生了算法降级，额外显示警告
            if (task.result.fallback) {
              setTimeout(() => {
                showToast(
                  t("codeAnalysis.fallbackWarning", {
                    algorithm: task.result?.actual_algorithm?.toUpperCase(),
                  }),
                  "error",
                );
              }, 500);
            }
          }
          // 刷新状态
          await checkStatus(projectPath);
          setCurrentTask(null);
        } else if (task.status === "failed") {
          stopPolling();
          setGenerating(false);
          showToast(task.error || t("codeAnalysis.error.generate"), "error");
          setCurrentTask(null);
        }
      } catch (error) {
        console.error("Failed to poll progress:", error);
        // 轮询失败不停止，继续重试
      }
    },
    [projectPath, stopPolling, showToast, t],
  );

  // 生成调用图（使用异步 API）
  const handleGenerate = async () => {
    try {
      setGenerating(true);
      setCurrentTask(null);
      showToast(t("codeAnalysis.generating"), "success");

      // 使用异步 API 启动任务
      const result = await apiService.generateCallGraphAsync(projectPath);
      const taskId = result.task_id;

      // 初始化任务状态
      setCurrentTask({
        task_id: taskId,
        project_id: "",
        project_path: projectPath,
        commit: "",
        status: "pending",
        progress: 0,
        message: t("codeAnalysis.taskStarting"),
      });

      // 开始轮询进度（每 1 秒）
      const interval = setInterval(() => {
        pollProgress(taskId);
      }, 1000);
      setPollingInterval(interval);

      // 立即执行一次
      await pollProgress(taskId);
    } catch (error) {
      console.error("Failed to start call graph generation:", error);
      showToast(t("codeAnalysis.error.generate"), "error");
      setGenerating(false);
      setCurrentTask(null);
    }
  };

  // 加载中状态
  if (loading) {
    return (
      <div className="cocursor-code-analysis">
        <Loading message={t("common.loading")} />
      </div>
    );
  }

  // 没有工作区路径
  if (!projectPath) {
    return (
      <div className="cocursor-code-analysis">
        <div className="cocursor-code-analysis-empty">
          <div className="cocursor-code-analysis-empty-icon">📂</div>
          <h3>{t("codeAnalysis.noWorkspace")}</h3>
          <p>{t("codeAnalysis.noWorkspaceDesc")}</p>
        </div>
      </div>
    );
  }

  // Go 模块验证失败
  if (moduleError) {
    return (
      <div className="cocursor-code-analysis">
        <ToastContainer toasts={toasts} />

        {/* 页面标题 */}
        <div className="cocursor-code-analysis-header">
          <div className="cocursor-code-analysis-title-row">
            <span className="cocursor-code-analysis-icon">🔍</span>
            <h1 className="cocursor-code-analysis-title">
              {t("codeAnalysis.title")}
            </h1>
          </div>
          <p className="cocursor-code-analysis-subtitle">
            {t("codeAnalysis.subtitle")}
          </p>
        </div>

        {/* 项目信息卡片 */}
        <div className="cocursor-code-analysis-card">
          <div className="cocursor-code-analysis-card-header">
            <h2>{t("codeAnalysis.projectInfo")}</h2>
          </div>
          <div className="cocursor-code-analysis-card-body">
            <div className="cocursor-code-analysis-info-row">
              <span className="cocursor-code-analysis-info-label">
                {t("codeAnalysis.projectName")}
              </span>
              <span className="cocursor-code-analysis-info-value">
                {projectPath.split("/").pop()}
              </span>
            </div>
            <div className="cocursor-code-analysis-info-row">
              <span className="cocursor-code-analysis-info-label">
                {t("codeAnalysis.projectPath")}
              </span>
              <span className="cocursor-code-analysis-info-value cocursor-code-analysis-path">
                {projectPath}
              </span>
            </div>
          </div>
        </div>

        {/* 错误提示卡片 */}
        <div className="cocursor-code-analysis-card cocursor-code-analysis-error-card">
          <div className="cocursor-code-analysis-card-header">
            <h2>{t("codeAnalysis.error.invalidGoModule")}</h2>
          </div>
          <div className="cocursor-code-analysis-card-body">
            <div className="cocursor-code-analysis-error-content">
              <div className="cocursor-code-analysis-error-icon">⚠️</div>
              <div className="cocursor-code-analysis-error-message">
                <p>{t("codeAnalysis.error.noGoMod")}</p>
                <p className="cocursor-code-analysis-error-detail">
                  {moduleError}
                </p>
              </div>
            </div>
            <div className="cocursor-code-analysis-actions">
              <Button
                onClick={() => {
                  setModuleError(null);
                  checkStatus(projectPath);
                }}
                variant="secondary"
              >
                {t("common.retry")}
              </Button>
            </div>
          </div>
        </div>

        {/* 使用说明 */}
        <div className="cocursor-code-analysis-card cocursor-code-analysis-tips">
          <div className="cocursor-code-analysis-card-header">
            <h2>{t("codeAnalysis.howToUse")}</h2>
          </div>
          <div className="cocursor-code-analysis-card-body">
            <ol className="cocursor-code-analysis-steps">
              <li>{t("codeAnalysis.step1")}</li>
              <li>{t("codeAnalysis.step2")}</li>
              <li>{t("codeAnalysis.step3")}</li>
            </ol>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="cocursor-code-analysis">
      <ToastContainer toasts={toasts} />

      {/* 页面标题 */}
      <div className="cocursor-code-analysis-header">
        <div className="cocursor-code-analysis-title-row">
          <span className="cocursor-code-analysis-icon">🔍</span>
          <h1 className="cocursor-code-analysis-title">
            {t("codeAnalysis.title")}
          </h1>
        </div>
        <p className="cocursor-code-analysis-subtitle">
          {t("codeAnalysis.subtitle")}
        </p>
      </div>

      {/* 项目信息卡片 */}
      <div className="cocursor-code-analysis-card">
        <div className="cocursor-code-analysis-card-header">
          <h2>{t("codeAnalysis.projectInfo")}</h2>
        </div>
        <div className="cocursor-code-analysis-card-body">
          <div className="cocursor-code-analysis-info-row">
            <span className="cocursor-code-analysis-info-label">
              {t("codeAnalysis.projectName")}
            </span>
            <span className="cocursor-code-analysis-info-value">
              {projectName || projectPath.split("/").pop()}
            </span>
          </div>
          <div className="cocursor-code-analysis-info-row">
            <span className="cocursor-code-analysis-info-label">
              {t("codeAnalysis.projectPath")}
            </span>
            <span className="cocursor-code-analysis-info-value cocursor-code-analysis-path">
              {projectPath}
            </span>
          </div>
          {remoteUrl && (
            <div className="cocursor-code-analysis-info-row">
              <span className="cocursor-code-analysis-info-label">
                {t("codeAnalysis.remoteUrl")}
              </span>
              <span className="cocursor-code-analysis-info-value cocursor-code-analysis-path">
                {remoteUrl}
              </span>
            </div>
          )}
        </div>
      </div>

      {/* 调用图状态卡片 */}
      {callGraphStatus?.project_registered && (
        <div className="cocursor-code-analysis-card">
          <div className="cocursor-code-analysis-card-header">
            <h2>{t("codeAnalysis.callGraphStatus")}</h2>
            <div className="cocursor-code-analysis-status-badge">
              {callGraphStatus.exists ? (
                callGraphStatus.up_to_date ? (
                  <span className="cocursor-code-analysis-badge-success">
                    ✓ {t("codeAnalysis.upToDate")}
                  </span>
                ) : (
                  <span className="cocursor-code-analysis-badge-warning">
                    ⚠{" "}
                    {t("codeAnalysis.outdated", {
                      count: callGraphStatus.commits_behind || 0,
                    })}
                  </span>
                )
              ) : (
                <span className="cocursor-code-analysis-badge-info">
                  {t("codeAnalysis.notGenerated")}
                </span>
              )}
            </div>
          </div>
          <div className="cocursor-code-analysis-card-body">
            {callGraphStatus.exists && (
              <>
                <div className="cocursor-code-analysis-info-row">
                  <span className="cocursor-code-analysis-info-label">
                    {t("codeAnalysis.currentCommit")}
                  </span>
                  <code className="cocursor-code-analysis-commit">
                    {callGraphStatus.current_commit?.substring(0, 7)}
                  </code>
                </div>
                <div className="cocursor-code-analysis-info-row">
                  <span className="cocursor-code-analysis-info-label">
                    {t("codeAnalysis.funcCount")}
                  </span>
                  <span className="cocursor-code-analysis-info-value">
                    {callGraphStatus.func_count?.toLocaleString()}
                  </span>
                </div>
              </>
            )}
            {/* 进度条 */}
            {generating && currentTask && (
              <div className="cocursor-code-analysis-progress">
                <div className="cocursor-code-analysis-progress-header">
                  <span className="cocursor-code-analysis-progress-status">
                    {currentTask.status === "pending" && "⏳"}
                    {currentTask.status === "running" && "🔄"}
                    {currentTask.message || t("codeAnalysis.taskRunning")}
                  </span>
                  <span className="cocursor-code-analysis-progress-percent">
                    {currentTask.progress}%
                  </span>
                </div>
                <div className="cocursor-code-analysis-progress-bar">
                  <div
                    className="cocursor-code-analysis-progress-fill"
                    style={{ width: `${currentTask.progress}%` }}
                  />
                </div>
              </div>
            )}

            <div className="cocursor-code-analysis-actions">
              <Button
                onClick={handleGenerate}
                loading={generating}
                variant="primary"
              >
                {callGraphStatus.exists
                  ? t("codeAnalysis.regenerate")
                  : t("codeAnalysis.generate")}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* 入口函数配置卡片（未注册时显示） */}
      {!callGraphStatus?.project_registered && (
        <div className="cocursor-code-analysis-card">
          <div className="cocursor-code-analysis-card-header">
            <h2>{t("codeAnalysis.entryPoints")}</h2>
          </div>
          <div className="cocursor-code-analysis-card-body">
            {scanning ? (
              <Loading message={t("codeAnalysis.scanning")} />
            ) : (
              <>
                <p className="cocursor-code-analysis-hint">
                  {t("codeAnalysis.entryPointsHint")}
                </p>
                <div className="cocursor-code-analysis-entry-list">
                  {candidates.map((candidate, index) => {
                    const key = `${candidate.file}:${candidate.function}`;
                    const isSelected = selectedEntryPoints.includes(key);
                    return (
                      <div
                        key={index}
                        className={`cocursor-code-analysis-entry-item ${
                          isSelected ? "selected" : ""
                        }`}
                        onClick={() => toggleEntryPoint(candidate)}
                      >
                        <div className="cocursor-code-analysis-entry-checkbox">
                          {isSelected ? "☑" : "☐"}
                        </div>
                        <div className="cocursor-code-analysis-entry-info">
                          <div className="cocursor-code-analysis-entry-file">
                            {candidate.file}
                          </div>
                          <div className="cocursor-code-analysis-entry-meta">
                            <span className="cocursor-code-analysis-entry-func">
                              {candidate.function}()
                            </span>
                            <span
                              className={`cocursor-code-analysis-entry-type ${candidate.type}`}
                            >
                              {candidate.type}
                            </span>
                            {candidate.recommended && (
                              <span className="cocursor-code-analysis-entry-recommended">
                                ★ {t("codeAnalysis.recommended")}
                              </span>
                            )}
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* 高级配置卡片（未注册时显示） */}
      {!callGraphStatus?.project_registered && (
        <div className="cocursor-code-analysis-card">
          <div className="cocursor-code-analysis-card-header">
            <h2>{t("codeAnalysis.advancedConfig")}</h2>
          </div>
          <div className="cocursor-code-analysis-card-body">
            {/* 算法选择 */}
            <div className="cocursor-code-analysis-form-group">
              <label>{t("codeAnalysis.algorithm")}</label>
              <select
                value={algorithm}
                onChange={(e) => setAlgorithm(e.target.value)}
                className="cocursor-code-analysis-select"
              >
                {ALGORITHM_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            {/* 排除路径 */}
            <div className="cocursor-code-analysis-form-group">
              <label>{t("codeAnalysis.excludePaths")}</label>
              <textarea
                value={excludePaths}
                onChange={(e) => setExcludePaths(e.target.value)}
                className="cocursor-code-analysis-textarea"
                rows={4}
                placeholder="vendor/&#10;*_test.go&#10;*.pb.go"
              />
              <span className="cocursor-code-analysis-form-hint">
                {t("codeAnalysis.excludePathsHint")}
              </span>
            </div>

            {/* 注册按钮 */}
            <div className="cocursor-code-analysis-actions">
              <Button
                onClick={handleRegister}
                loading={registering}
                variant="primary"
                disabled={selectedEntryPoints.length === 0}
              >
                {t("codeAnalysis.registerAndGenerate")}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* 生成结果卡片 */}
      {generateResult && (
        <div
          className={`cocursor-code-analysis-card cocursor-code-analysis-result ${generateResult.fallback ? "cocursor-code-analysis-result-warning" : ""}`}
        >
          <div className="cocursor-code-analysis-card-header">
            <h2>{t("codeAnalysis.generateResult")}</h2>
            {generateResult.actual_algorithm && (
              <span className="cocursor-code-analysis-algorithm-badge">
                {generateResult.actual_algorithm.toUpperCase()}
              </span>
            )}
          </div>
          <div className="cocursor-code-analysis-card-body">
            {/* 降级警告 */}
            {generateResult.fallback && generateResult.fallback_reason && (
              <div className="cocursor-code-analysis-fallback-warning">
                <div className="cocursor-code-analysis-fallback-icon">⚠️</div>
                <div className="cocursor-code-analysis-fallback-content">
                  <div className="cocursor-code-analysis-fallback-title">
                    {t("codeAnalysis.algorithmFallback")}
                  </div>
                  <div className="cocursor-code-analysis-fallback-reason">
                    {generateResult.fallback_reason}
                  </div>
                </div>
              </div>
            )}

            <div className="cocursor-code-analysis-stats">
              <div className="cocursor-code-analysis-stat">
                <div className="cocursor-code-analysis-stat-value">
                  {generateResult.func_count.toLocaleString()}
                </div>
                <div className="cocursor-code-analysis-stat-label">
                  {t("codeAnalysis.functions")}
                </div>
              </div>
              <div className="cocursor-code-analysis-stat">
                <div className="cocursor-code-analysis-stat-value">
                  {generateResult.edge_count.toLocaleString()}
                </div>
                <div className="cocursor-code-analysis-stat-label">
                  {t("codeAnalysis.edges")}
                </div>
              </div>
              <div className="cocursor-code-analysis-stat">
                <div className="cocursor-code-analysis-stat-value">
                  {(generateResult.generation_time_ms / 1000).toFixed(1)}s
                </div>
                <div className="cocursor-code-analysis-stat-label">
                  {t("codeAnalysis.generationTime")}
                </div>
              </div>
            </div>
            <div className="cocursor-code-analysis-info-row">
              <span className="cocursor-code-analysis-info-label">Commit</span>
              <code className="cocursor-code-analysis-commit">
                {generateResult.commit.substring(0, 7)}
              </code>
            </div>
          </div>
        </div>
      )}

      {/* 使用说明 */}
      <div className="cocursor-code-analysis-card cocursor-code-analysis-tips">
        <div className="cocursor-code-analysis-card-header">
          <h2>{t("codeAnalysis.howToUse")}</h2>
        </div>
        <div className="cocursor-code-analysis-card-body">
          <ol className="cocursor-code-analysis-steps">
            <li>{t("codeAnalysis.step1")}</li>
            <li>{t("codeAnalysis.step2")}</li>
            <li>{t("codeAnalysis.step3")}</li>
          </ol>
        </div>
      </div>
    </div>
  );
};

export default CodeAnalysisConfig;
