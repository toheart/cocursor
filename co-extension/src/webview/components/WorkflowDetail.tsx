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
    return date.toLocaleString({
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
      <div style={{ padding: "16px", textAlign: "center", color: "var(--vscode-descriptionForeground)" }}>
        {t("workflows.loading")}
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: "16px" }}>
        <div className="cocursor-error" style={{ padding: "12px", backgroundColor: "var(--vscode-inputValidation-errorBackground)", color: "var(--vscode-errorForeground)", borderRadius: "4px" }}>
          {t("workflows.error")}: {error}
        </div>
        <button
          onClick={() => navigate("/workflows")}
          style={{ marginTop: "16px", padding: "8px 16px" }}
        >
          {t("workflows.backToList")}
        </button>
      </div>
    );
  }

  if (!workflow) {
    return (
      <div style={{ padding: "16px", textAlign: "center", color: "var(--vscode-descriptionForeground)" }}>
        {t("workflows.notFound")}
        <button
          onClick={() => navigate("/workflows")}
          style={{ marginTop: "16px", padding: "8px 16px", display: "block", margin: "16px auto 0" }}
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
    <div className="cocursor-workflow-detail" style={{ padding: "16px" }}>
      {/* 头部信息 */}
      <div style={{ marginBottom: "24px" }}>
        <div style={{ display: "flex", alignItems: "center", gap: "12px", marginBottom: "12px" }}>
          <h2 style={{ margin: 0, fontSize: "18px", fontWeight: 600 }}>
            {workflow.change_id}
          </h2>
          <span
            style={{
              padding: "4px 8px",
              fontSize: "12px",
              borderRadius: "4px",
              backgroundColor: "var(--vscode-badge-background)",
              color: "var(--vscode-badge-foreground)"
            }}
          >
            {getStageLabel(workflow.stage)}
          </span>
          <span
            style={{
              padding: "4px 8px",
              fontSize: "12px",
              borderRadius: "4px",
              color: getStatusColor(workflow.status),
              border: `1px solid ${getStatusColor(workflow.status)}`
            }}
          >
            {getStatusLabel(workflow.status)}
          </span>
        </div>
        <div style={{ fontSize: "12px", color: "var(--vscode-descriptionForeground)" }}>
          {workflow.project_path}
        </div>
      </div>

      {/* 基本信息 */}
      <div style={{ marginBottom: "24px", padding: "16px", backgroundColor: "var(--vscode-editor-background)", borderRadius: "4px", border: "1px solid var(--vscode-panel-border)" }}>
        <h3 style={{ margin: "0 0 12px 0", fontSize: "14px", fontWeight: 600 }}>{t("workflows.basicInfo")}</h3>
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "12px", fontSize: "12px" }}>
          <div>
            <div style={{ color: "var(--vscode-descriptionForeground)", marginBottom: "4px" }}>{t("workflows.workspaceId")}</div>
            <div style={{ color: "var(--vscode-foreground)" }}>{workflow.workspace_id}</div>
          </div>
          <div>
            <div style={{ color: "var(--vscode-descriptionForeground)", marginBottom: "4px" }}>{t("workflows.changeId")}</div>
            <div style={{ color: "var(--vscode-foreground)" }}>{workflow.change_id}</div>
          </div>
          <div>
            <div style={{ color: "var(--vscode-descriptionForeground)", marginBottom: "4px" }}>{t("workflows.startTime")}</div>
            <div style={{ color: "var(--vscode-foreground)" }}>{formatDate(workflow.started_at)}</div>
          </div>
          <div>
            <div style={{ color: "var(--vscode-descriptionForeground)", marginBottom: "4px" }}>{t("workflows.updateTime")}</div>
            <div style={{ color: "var(--vscode-foreground)" }}>{formatDate(workflow.updated_at)}</div>
          </div>
        </div>
      </div>

      {/* 进度信息 */}
      {workflow.summary && (
        <div style={{ marginBottom: "24px", padding: "16px", backgroundColor: "var(--vscode-editor-background)", borderRadius: "4px", border: "1px solid var(--vscode-panel-border)" }}>
          <h3 style={{ margin: "0 0 12px 0", fontSize: "14px", fontWeight: 600 }}>{t("workflows.progressInfo")}</h3>
          <div style={{ marginBottom: "12px" }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "8px" }}>
              <span style={{ fontSize: "12px", color: "var(--vscode-descriptionForeground)" }}>
                {t("workflows.taskProgress")}
              </span>
              <span style={{ fontSize: "12px", color: "var(--vscode-foreground)", fontWeight: 600 }}>
                {workflow.summary.tasks_completed} / {workflow.summary.tasks_total} ({progress}%)
              </span>
            </div>
            <div
              style={{
                width: "100%",
                height: "8px",
                backgroundColor: "var(--vscode-progressBar-background)",
                borderRadius: "4px",
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
          {workflow.summary.time_spent && (
            <div style={{ fontSize: "12px", color: "var(--vscode-descriptionForeground)" }}>
              {t("workflows.timeSpent")}: {workflow.summary.time_spent}
            </div>
          )}
        </div>
      )}

      {/* 文件变更 */}
      {workflow.summary && workflow.summary.files_changed && workflow.summary.files_changed.length > 0 && (
        <div style={{ marginBottom: "24px", padding: "16px", backgroundColor: "var(--vscode-editor-background)", borderRadius: "4px", border: "1px solid var(--vscode-panel-border)" }}>
          <h3 style={{ margin: "0 0 12px 0", fontSize: "14px", fontWeight: 600 }}>
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
        <div style={{ marginBottom: "24px", padding: "16px", backgroundColor: "var(--vscode-editor-background)", borderRadius: "4px", border: "1px solid var(--vscode-panel-border)" }}>
          <h3 style={{ margin: "0 0 12px 0", fontSize: "14px", fontWeight: 600 }}>{t("workflows.summary")}</h3>
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
        <div style={{ marginBottom: "24px", padding: "16px", backgroundColor: "var(--vscode-editor-background)", borderRadius: "4px", border: "1px solid var(--vscode-panel-border)" }}>
          <h3 style={{ margin: "0 0 12px 0", fontSize: "14px", fontWeight: 600 }}>{t("workflows.metadata")}</h3>
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
          onClick={() => navigate("/workflows")}
          style={{ padding: "8px 16px", fontSize: "12px" }}
        >
          {t("workflows.backToList")}
        </button>
      </div>
    </div>
  );
};
