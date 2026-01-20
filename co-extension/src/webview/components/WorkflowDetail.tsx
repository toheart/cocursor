import React, { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { apiService } from "../services/api";

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
  stage: string;
  status: string;
  started_at: number;
  updated_at: number;
  metadata?: Record<string, any>;
  summary?: WorkflowSummary;
}

export const WorkflowDetail: React.FC = () => {
  const { t } = useTranslation();
  const { changeId } = useParams<{ changeId: string }>();
  const navigate = useNavigate();
  const [workflow, setWorkflow] = useState<WorkflowItem | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (changeId) {
      loadWorkflowDetail();
    }
  }, [changeId]);

  const getWorkspacePath = (): string => {
    const workspacePath = (window as any).__WORKSPACE_PATH__;
    if (!workspacePath) {
      console.warn("Workspace path not found");
      return "";
    }
    return workspacePath;
  };

  const loadWorkflowDetail = async () => {
    if (!changeId) return;

    setLoading(true);
    setError(null);
    try {
      const workspacePath = getWorkspacePath();
      const response = await apiService.getWorkflowDetail(changeId, workspacePath) as {
        data?: WorkflowItem;
        error?: string;
      };

      if (response.error) {
        setError(response.error);
        setWorkflow(null);
      } else {
        setWorkflow(response.data || null);
      }
    } catch (err) {
      console.error("Failed to load workflow detail:", err);
      setError(err instanceof Error ? err.message : t("common.error"));
      setWorkflow(null);
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (timestamp: number): string => {
    const date = new Date(timestamp);
    return date.toLocaleString(undefined, {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit"
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
      in_progress: "var(--vscode-testing-iconQueued)",
      completed: "var(--vscode-testing-iconPassed)",
      paused: "var(--vscode-descriptionForeground)"
    };
    return colors[status] || "var(--vscode-foreground)";
  };

  if (loading) {
    return (
      <div className="cocursor-loading">
        {t("workflows.loading")}
      </div>
    );
  }

  if (error) {
    return (
      <div className="cocursor-container">
        <div className="cocursor-error">
          {t("workflows.error")}: {error}
        </div>
        <button
          className="cocursor-btn cocursor-btn-primary"
          onClick={() => navigate("/workflows")}
          style={{ marginTop: "16px" }}
        >
          {t("workflows.backToList")}
        </button>
      </div>
    );
  }

  if (!workflow) {
    return (
      <div className="cocursor-empty">
        {t("workflows.notFound")}
        <button
          className="cocursor-btn cocursor-btn-primary"
          onClick={() => navigate("/workflows")}
          style={{ marginTop: "16px", display: "block", margin: "16px auto 0" }}
        >
          {t("workflows.backToList")}
        </button>
      </div>
    );
  }

  const progress = workflow.summary
    ? Math.round((workflow.summary.tasks_completed / workflow.summary.tasks_total) * 100)
    : 0;

  return (
    <div className="cocursor-workflow-detail">
      {/* 头部信息 */}
      <div className="cocursor-workflow-detail-header">
        <div className="cocursor-workflow-detail-title-row">
          <h2 className="cocursor-workflow-detail-title">
            {workflow.change_id}
          </h2>
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
        <div className="cocursor-workflow-detail-path">
          {workflow.project_path}
        </div>
      </div>

      {/* 基本信息 */}
      <div className="cocursor-workflow-info-card">
        <h3 className="cocursor-workflow-info-card-title">{t("workflows.basicInfo")}</h3>
        <div className="cocursor-workflow-info-row">
          <div className="cocursor-workflow-info-label">{t("workflows.workspaceId")}</div>
          <div className="cocursor-workflow-info-value">{workflow.workspace_id}</div>
        </div>
        <div className="cocursor-workflow-info-row">
          <div className="cocursor-workflow-info-label">{t("workflows.changeId")}</div>
          <div className="cocursor-workflow-info-value">{workflow.change_id}</div>
        </div>
        <div className="cocursor-workflow-info-row">
          <div className="cocursor-workflow-info-label">{t("workflows.startTime")}</div>
          <div className="cocursor-workflow-info-value">{formatDate(workflow.started_at)}</div>
        </div>
        <div className="cocursor-workflow-info-row">
          <div className="cocursor-workflow-info-label">{t("workflows.updateTime")}</div>
          <div className="cocursor-workflow-info-value">{formatDate(workflow.updated_at)}</div>
        </div>
      </div>

      {/* 进度信息 */}
      {workflow.summary && (
        <div className="cocursor-workflow-info-card">
          <h3 className="cocursor-workflow-info-card-title">{t("workflows.progressInfo")}</h3>
          <div style={{ marginBottom: "12px" }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "8px" }}>
              <span className="cocursor-workflow-info-label">
                {t("workflows.taskProgress")}
              </span>
              <span className="cocursor-workflow-info-value">
                {workflow.summary.tasks_completed} / {workflow.summary.tasks_total} ({progress}%)
              </span>
            </div>
            <div className="cocursor-progress" style={{ height: "8px" }}>
              <div
                className="cocursor-progress-bar"
                style={{
                  width: `${progress}%`,
                  backgroundColor: getStatusColor(workflow.status)
                }}
              />
            </div>
          </div>
          {workflow.summary.time_spent && (
            <div className="cocursor-workflow-info-label">
              {t("workflows.timeSpent")}: {workflow.summary.time_spent}
            </div>
          )}
        </div>
      )}

      {/* 文件变更 */}
      {workflow.summary && workflow.summary.files_changed && workflow.summary.files_changed.length > 0 && (
        <div className="cocursor-workflow-info-card">
          <h3 className="cocursor-workflow-info-card-title">
            {t("workflows.changedFiles")} ({workflow.summary.files_changed.length})
          </h3>
          <div style={{ maxHeight: "200px", overflowY: "auto" }}>
            {workflow.summary.files_changed.map((file, index) => (
              <div
                key={index}
                style={{
                  padding: "4px 8px",
                  fontSize: "12px",
                  fontFamily: "var(--vscode-editor-font-family)",
                  color: "var(--vscode-foreground)",
                  borderBottom: index < workflow.summary!.files_changed.length - 1 ? "1px solid var(--vscode-panel-border)" : "none"
                }}
              >
                {file}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 工作总结 */}
      {workflow.summary && workflow.summary.summary && (
        <div className="cocursor-workflow-info-card">
          <h3 className="cocursor-workflow-info-card-title">{t("workflows.summary")}</h3>
          <div
            style={{
              fontSize: "12px",
              color: "var(--vscode-foreground)",
              lineHeight: "1.6",
              whiteSpace: "pre-wrap"
            }}
          >
            {workflow.summary.summary}
          </div>
        </div>
      )}

      {/* 元数据 */}
      {workflow.metadata && Object.keys(workflow.metadata).length > 0 && (
        <div className="cocursor-workflow-info-card">
          <h3 className="cocursor-workflow-info-card-title">{t("workflows.metadata")}</h3>
          <pre
            style={{
              fontSize: "11px",
              fontFamily: "var(--vscode-editor-font-family)",
              color: "var(--vscode-foreground)",
              margin: 0,
              padding: "8px",
              backgroundColor: "var(--vscode-textCodeBlock-background)",
              borderRadius: "4px",
              overflow: "auto",
              maxHeight: "300px"
            }}
          >
            {JSON.stringify(workflow.metadata, null, 2)}
          </pre>
        </div>
      )}

      {/* 返回按钮 */}
      <div style={{ marginTop: "24px" }}>
        <button
          className="cocursor-btn cocursor-btn-primary"
          onClick={() => navigate("/workflows")}
        >
          {t("workflows.backToList")}
        </button>
      </div>
    </div>
  );
};
