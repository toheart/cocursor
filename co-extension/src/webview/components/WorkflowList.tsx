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
    return date.toLocaleString({
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
      <div style={{ padding: "12px 16px", borderBottom: "1px solid var(--vscode-panel-border)", display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <h2 style={{ margin: 0, fontSize: "14px", fontWeight: 600 }}>{t("workflows.title")}</h2>
        <div style={{ display: "flex", gap: "8px", alignItems: "center" }}>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            style={{
              padding: "4px 8px",
              fontSize: "12px",
              backgroundColor: "var(--vscode-dropdown-background)",
              color: "var(--vscode-dropdown-foreground)",
              border: "1px solid var(--vscode-dropdown-border)",
              borderRadius: "2px"
            }}
          >
            <option value="all">{t("workflows.status.all")}</option>
            <option value="in_progress">{t("workflows.status.inProgress")}</option>
            <option value="completed">{t("workflows.status.completed")}</option>
            <option value="paused">{t("workflows.status.paused")}</option>
          </select>
          <button
            onClick={loadWorkflows}
            disabled={loading}
            style={{ padding: "4px 8px", fontSize: "12px" }}
          >
            {loading ? t("workflows.loading") : t("workflows.refresh")}
          </button>
        </div>
      </div>

      <main style={{ padding: "16px" }}>
        {/* 可折叠帮助区域 */}
        <WorkflowHelpSection />

        {error && (
          <div className="cocursor-error" style={{ padding: "12px", marginBottom: "16px", backgroundColor: "var(--vscode-inputValidation-errorBackground)", color: "var(--vscode-errorForeground)", borderRadius: "4px" }}>
            {t("workflows.error")}: {error}
          </div>
        )}

        {loading ? (
          <div className="cocursor-loading" style={{ padding: "16px", textAlign: "center", color: "var(--vscode-descriptionForeground)" }}>
            {t("workflows.loading")}
          </div>
        ) : workflows.length === 0 ? (
          <div className="cocursor-empty" style={{ padding: "16px", textAlign: "center", color: "var(--vscode-descriptionForeground)" }}>
            {t("workflows.noWorkflows")}
          </div>
        ) : (
          <div className="cocursor-workflows">
            {workflows.map((workflow) => {
              const progress = getProgress(workflow);
              return (
                <div
                  key={workflow.id}
                  onClick={() => handleWorkflowClick(workflow)}
                  style={{
                    padding: "16px",
                    borderBottom: "1px solid var(--vscode-panel-border)",
                    cursor: "pointer",
                    transition: "background-color 0.2s"
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.backgroundColor = "var(--vscode-list-hoverBackground)";
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.backgroundColor = "transparent";
                  }}
                >
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: "8px" }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ display: "flex", alignItems: "center", gap: "8px", marginBottom: "4px" }}>
                        <h3 style={{ margin: 0, fontSize: "14px", fontWeight: 600 }}>
                          {workflow.change_id}
                        </h3>
                        <span
                          style={{
                            padding: "2px 6px",
                            fontSize: "11px",
                            borderRadius: "2px",
                            backgroundColor: "var(--vscode-badge-background)",
                            color: "var(--vscode-badge-foreground)"
                          }}
                        >
                          {getStageLabel(workflow.stage)}
                        </span>
                        <span
                          style={{
                            padding: "2px 6px",
                            fontSize: "11px",
                            borderRadius: "2px",
                            color: getStatusColor(workflow.status),
                            border: `1px solid ${getStatusColor(workflow.status)}`
                          }}
                        >
                          {getStatusLabel(workflow.status)}
                        </span>
                      </div>
                      <div style={{ fontSize: "12px", color: "var(--vscode-descriptionForeground)", marginBottom: "8px" }}>
                        {workflow.project_path}
                      </div>
                    </div>
                    <div style={{ fontSize: "12px", color: "var(--vscode-descriptionForeground)", textAlign: "right" }}>
                      <div>{t("workflows.startTime")}: {formatDate(workflow.started_at)}</div>
                      <div>{t("workflows.updateTime")}: {formatDate(workflow.updated_at)}</div>
                    </div>
                  </div>

                  {/* 进度条 */}
                  {progress > 0 && (
                    <div style={{ marginTop: "8px" }}>
                      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "4px" }}>
                        <span style={{ fontSize: "12px", color: "var(--vscode-descriptionForeground)" }}>
                          {t("workflows.progress")}: {progress}%
                        </span>
                        {workflow.summary && (
                          <span style={{ fontSize: "12px", color: "var(--vscode-descriptionForeground)" }}>
                            {workflow.summary.tasks_completed} / {workflow.summary.tasks_total} {t("workflows.tasks")}
                          </span>
                        )}
                      </div>
                      <div
                        style={{
                          width: "100%",
                          height: "4px",
                          backgroundColor: "var(--vscode-progressBar-background)",
                          borderRadius: "2px",
                          overflow: "hidden"
                        }}
                      >
                        <div
                          style={{
                            width: `${progress}%`,
                            height: "100%",
                            backgroundColor: getStatusColor(workflow.status),
                            transition: "width 0.3s ease"
                          }}
                        />
                      </div>
                    </div>
                  )}

                  {/* 文件变更统计 */}
                  {workflow.summary && workflow.summary.files_changed && workflow.summary.files_changed.length > 0 && (
                    <div style={{ marginTop: "8px", fontSize: "12px", color: "var(--vscode-descriptionForeground)" }}>
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
