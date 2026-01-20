import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { apiService } from "../services/api";
import { WorkflowHelpSection } from "./WorkflowHelpSection";

interface WorkflowSummary {
  tasks_completed: number;
  tasks_total: number;
  files_changed: string[];
  time_spent: string;
  summary: string;
}

interface WorkflowItem {
  id: number;
  workspace_id: string;
  project_path: string;
  change_id: string;
  stage: string; // init|proposal|apply|archive
  status: string; // in_progress|completed|paused
  started_at: number; // Unix 毫秒时间戳
  updated_at: number; // Unix 毫秒时间戳
  metadata?: Record<string, any>;
  summary?: WorkflowSummary;
}

export const WorkflowList: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [workflows, setWorkflows] = useState<WorkflowItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("all");

  useEffect(() => {
    loadWorkflows();
  }, [statusFilter]);

  const getWorkspacePath = (): string => {
    const workspacePath = (window as any).__WORKSPACE_PATH__;
    if (!workspacePath) {
      console.warn("Workspace path not found");
      return "";
    }
    return workspacePath;
  };

  const loadWorkflows = async () => {
    setLoading(true);
    setError(null);
    try {
      const workspacePath = getWorkspacePath();
      const status = statusFilter !== "all" ? statusFilter : undefined;
      console.log("[WorkflowList] Loading workflows:", { workspacePath, status });
      
      const response = await apiService.getWorkflows(workspacePath, status);
      console.log("[WorkflowList] Received response:", response);

      // 处理响应数据：webviewPanel 返回的是数组或包含 error 的对象
      if (Array.isArray(response)) {
        console.log("[WorkflowList] Response is array, length:", response.length);
        setWorkflows(response as WorkflowItem[]);
      } else if (response && typeof response === "object") {
        if ("error" in response) {
          console.error("[WorkflowList] Response contains error:", (response as { error: string }).error);
          setError((response as { error: string }).error);
          setWorkflows([]);
        } else if ("data" in response && Array.isArray((response as { data: WorkflowItem[] }).data)) {
          console.log("[WorkflowList] Response has data array, length:", (response as { data: WorkflowItem[] }).data.length);
          setWorkflows((response as { data: WorkflowItem[] }).data);
        } else {
          console.warn("[WorkflowList] Unexpected response format:", response);
          setWorkflows([]);
        }
      } else {
        console.warn("[WorkflowList] Unexpected response type:", typeof response, response);
        setWorkflows([]);
      }
    } catch (err) {
      console.error("[WorkflowList] Failed to load workflows:", err);
      setError(err instanceof Error ? err.message : t("common.error"));
      setWorkflows([]);
    } finally {
      setLoading(false);
    }
  };

  const handleWorkflowClick = (workflow: WorkflowItem) => {
    navigate(`/workflows/${workflow.change_id}`);
  };

  const formatDate = (timestamp: number): string => {
    const date = new Date(timestamp);
    return date.toLocaleString(undefined, {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit"
    });
  };

  const getStageLabel = (stage: string): string => {
    return t(`workflows.stage.${stage}`) || stage;
  };

  const getStatusLabel = (status: string): string => {
    return t(`workflows.status.${status}`) || status;
  };

  const getStatusColor = (status: string): string => {
    const colors: Record<string, string> = {
      in_progress: "var(--vscode-testing-iconQueued)", // 黄色
      completed: "var(--vscode-testing-iconPassed)", // 绿色
      paused: "var(--vscode-descriptionForeground)" // 灰色
    };
    return colors[status] || "var(--vscode-foreground)";
  };

  const getProgress = (workflow: WorkflowItem): number => {
    if (workflow.summary) {
      const { tasks_completed, tasks_total } = workflow.summary;
      if (tasks_total > 0) {
        return Math.round((tasks_completed / tasks_total) * 100);
      }
    }
    // 从 metadata 中获取进度
    if (workflow.metadata) {
      const completed = workflow.metadata.tasks_completed as number;
      const total = workflow.metadata.tasks_total as number;
      if (total && total > 0) {
        return Math.round((completed / total) * 100);
      }
    }
    return 0;
  };

  return (
    <div className="cocursor-workflow-list">
      <div className="cocursor-workflow-list-header">
        <h2>{t("workflows.title")}</h2>
        <div className="cocursor-workflow-list-header-controls">
          <select
            className="cocursor-select"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
          >
            <option value="all">{t("workflows.status.all")}</option>
            <option value="in_progress">{t("workflows.status.inProgress")}</option>
            <option value="completed">{t("workflows.status.completed")}</option>
            <option value="paused">{t("workflows.status.paused")}</option>
          </select>
          <button
            className="cocursor-btn cocursor-btn-small"
            onClick={loadWorkflows}
            disabled={loading}
          >
            {loading ? t("workflows.loading") : t("workflows.refresh")}
          </button>
        </div>
      </div>

      <main className="cocursor-main">
        {/* 可折叠帮助区域 */}
        <WorkflowHelpSection />

        {error && (
          <div className="cocursor-error">
            {t("workflows.error")}: {error}
          </div>
        )}

        {loading ? (
          <div className="cocursor-loading">
            {t("workflows.loading")}
          </div>
        ) : workflows.length === 0 ? (
          <div className="cocursor-empty">
            {t("workflows.noWorkflows")}
          </div>
        ) : (
          <div className="cocursor-workflows">
            {workflows.map((workflow) => {
              const progress = getProgress(workflow);
              return (
                <div
                  key={workflow.id}
                  className="cocursor-workflow-item"
                  onClick={() => handleWorkflowClick(workflow)}
                >
                  <div className="cocursor-workflow-item-header">
                    <div className="cocursor-workflow-item-left">
                      <div className="cocursor-workflow-item-title-row">
                        <h3 className="cocursor-workflow-item-title">
                          {workflow.change_id}
                        </h3>
                        <span className="cocursor-badge cocursor-badge-default">
                          {getStageLabel(workflow.stage)}
                        </span>
                        <span
                          className="cocursor-badge cocursor-badge-outline"
                          style={{
                            color: getStatusColor(workflow.status),
                            borderColor: getStatusColor(workflow.status)
                          }}
                        >
                          {getStatusLabel(workflow.status)}
                        </span>
                      </div>
                      <div className="cocursor-workflow-item-path">
                        {workflow.project_path}
                      </div>
                    </div>
                    <div className="cocursor-workflow-item-meta">
                      <div>{t("workflows.startTime")}: {formatDate(workflow.started_at)}</div>
                      <div>{t("workflows.updateTime")}: {formatDate(workflow.updated_at)}</div>
                    </div>
                  </div>

                  {/* 进度条 */}
                  {progress > 0 && (
                    <div className="cocursor-workflow-item-progress">
                      <div className="cocursor-workflow-item-progress-header">
                        <span className="cocursor-workflow-item-progress-text">
                          {t("workflows.progress")}: {progress}%
                        </span>
                        {workflow.summary && (
                          <span className="cocursor-workflow-item-progress-text">
                            {workflow.summary.tasks_completed} / {workflow.summary.tasks_total} {t("workflows.tasks")}
                          </span>
                        )}
                      </div>
                      <div className="cocursor-progress">
                        <div
                          className="cocursor-progress-bar"
                          style={{
                            width: `${progress}%`,
                            backgroundColor: getStatusColor(workflow.status)
                          }}
                        />
                      </div>
                    </div>
                  )}

                  {/* 文件变更统计 */}
                  {workflow.summary && workflow.summary.files_changed && workflow.summary.files_changed.length > 0 && (
                    <div className="cocursor-workflow-item-files">
                      {t("workflows.changedFiles")}: {workflow.summary.files_changed.length}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </main>
    </div>
  );
};
