import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { WorkflowStageDiagram } from "./WorkflowStageDiagram";

/**
 * 可折叠的工作流帮助区域组件
 * 包含详细的使用说明、流程图、阶段说明等
 */
export const WorkflowHelpSection: React.FC = () => {
  const { t } = useTranslation();
  const [isExpanded, setIsExpanded] = useState(false);

  const toggleExpand = () => {
    setIsExpanded(!isExpanded);
  };

  return (
    <div
      style={{
        marginBottom: "16px",
        border: "1px solid var(--vscode-panel-border)",
        borderRadius: "4px",
        backgroundColor: "var(--vscode-editor-background)"
      }}
    >
      {/* 标题栏（可点击展开/折叠） */}
      <div
        onClick={toggleExpand}
        style={{
          padding: "12px 16px",
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          cursor: "pointer",
          userSelect: "none",
          borderBottom: isExpanded ? "1px solid var(--vscode-panel-border)" : "none",
          transition: "background-color 0.2s"
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.backgroundColor = "var(--vscode-list-hoverBackground)";
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.backgroundColor = "transparent";
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
          <span style={{ fontSize: "14px" }}>ℹ️</span>
          <span style={{ fontSize: "14px", fontWeight: 600 }}>
            {t("workflows.onboarding.helpSection.title")}
          </span>
        </div>
        <span style={{ fontSize: "12px", color: "var(--vscode-descriptionForeground)" }}>
          {isExpanded
            ? t("workflows.onboarding.helpSection.collapse")
            : t("workflows.onboarding.helpSection.expand")}
        </span>
      </div>

      {/* 展开的内容 */}
      {isExpanded && (
        <div
          style={{
            padding: "16px",
            maxHeight: "none",
            overflow: "visible",
            transition: "max-height 0.3s ease"
          }}
        >
          {/* 重要提示 */}
          <div
            style={{
              padding: "12px",
              marginBottom: "16px",
              backgroundColor: "var(--vscode-textBlockQuote-background)",
              borderLeft: "3px solid var(--vscode-testing-iconQueued)",
              borderRadius: "4px"
            }}
          >
            <div
              style={{
                fontSize: "13px",
                fontWeight: 600,
                marginBottom: "4px",
                color: "var(--vscode-foreground)"
              }}
            >
              ⚠️ {t("workflows.onboarding.skillInstallWarning.title")}
            </div>
            <div
              style={{
                fontSize: "12px",
                color: "var(--vscode-descriptionForeground)",
                lineHeight: "1.5"
              }}
            >
              {t("workflows.onboarding.skillInstallWarning.message")}
            </div>
          </div>

          {/* 流程图 */}
          <div style={{ marginBottom: "20px" }}>
            <div
              style={{
                fontSize: "13px",
                fontWeight: 600,
                marginBottom: "12px",
                color: "var(--vscode-foreground)"
              }}
            >
              📊 {t("workflows.onboarding.workflowDiagram.title")}
            </div>
            <WorkflowStageDiagram />
          </div>

          {/* 四个阶段详细说明 */}
          <div style={{ marginBottom: "20px" }}>
            <div
              style={{
                fontSize: "13px",
                fontWeight: 600,
                marginBottom: "12px",
                color: "var(--vscode-foreground)"
              }}
            >
              {t("workflows.onboarding.stages.title")}
            </div>

            {/* Init 阶段 */}
            <div style={{ marginBottom: "16px", paddingLeft: "8px" }}>
              <div style={{ fontSize: "12px", fontWeight: 600, marginBottom: "4px" }}>
                1️⃣ {t("workflows.onboarding.stages.init.name")}
              </div>
              <div style={{ fontSize: "11px", color: "var(--vscode-descriptionForeground)", marginLeft: "16px", lineHeight: "1.6" }}>
                <div>{t("workflows.onboarding.stages.init.function")}</div>
                <div>{t("workflows.onboarding.stages.init.command")}</div>
                <div>{t("workflows.onboarding.stages.init.output")}</div>
              </div>
            </div>

            {/* Proposal 阶段 */}
            <div style={{ marginBottom: "16px", paddingLeft: "8px" }}>
              <div style={{ fontSize: "12px", fontWeight: 600, marginBottom: "4px" }}>
                2️⃣ {t("workflows.onboarding.stages.proposal.name")}
              </div>
              <div style={{ fontSize: "11px", color: "var(--vscode-descriptionForeground)", marginLeft: "16px", lineHeight: "1.6" }}>
                <div>{t("workflows.onboarding.stages.proposal.function")}</div>
                <div>{t("workflows.onboarding.stages.proposal.command")}</div>
                <div>{t("workflows.onboarding.stages.proposal.output")}</div>
                <div style={{ marginTop: "4px" }}>
                  <div>{t("workflows.onboarding.stages.proposal.outputItems.proposal")}</div>
                  <div>{t("workflows.onboarding.stages.proposal.outputItems.tasks")}</div>
                  <div>{t("workflows.onboarding.stages.proposal.outputItems.design")}</div>
                  <div>{t("workflows.onboarding.stages.proposal.outputItems.specDelta")}</div>
                </div>
                <div style={{ marginTop: "4px", fontWeight: 600 }}>
                  {t("workflows.onboarding.stages.proposal.mcpTools")}
                </div>
                <div style={{ marginLeft: "8px" }}>
                  <div>{t("workflows.onboarding.stages.proposal.mcpToolsList.openspec_list")}</div>
                  <div>{t("workflows.onboarding.stages.proposal.mcpToolsList.openspec_validate")}</div>
                  <div>{t("workflows.onboarding.stages.proposal.mcpToolsList.record_openspec_workflow")}</div>
                </div>
              </div>
            </div>

            {/* Apply 阶段 */}
            <div style={{ marginBottom: "16px", paddingLeft: "8px" }}>
              <div style={{ fontSize: "12px", fontWeight: 600, marginBottom: "4px" }}>
                3️⃣ {t("workflows.onboarding.stages.apply.name")}
              </div>
              <div style={{ fontSize: "11px", color: "var(--vscode-descriptionForeground)", marginLeft: "16px", lineHeight: "1.6" }}>
                <div>{t("workflows.onboarding.stages.apply.function")}</div>
                <div>{t("workflows.onboarding.stages.apply.command")}</div>
                <div style={{ fontWeight: 600, marginTop: "4px" }}>
                  {t("workflows.onboarding.stages.apply.process")}
                </div>
                <div style={{ marginLeft: "8px" }}>
                  <div>{t("workflows.onboarding.stages.apply.processItems.taskOrder")}</div>
                  <div>{t("workflows.onboarding.stages.apply.processItems.autoUpdate")}</div>
                  <div>{t("workflows.onboarding.stages.apply.processItems.autoSummary")}</div>
                </div>
                <div style={{ marginTop: "4px", fontWeight: 600 }}>
                  {t("workflows.onboarding.stages.apply.mcpTools")}
                </div>
                <div style={{ marginLeft: "8px" }}>
                  <div>{t("workflows.onboarding.stages.apply.mcpToolsList.openspec_list")}</div>
                  <div>{t("workflows.onboarding.stages.apply.mcpToolsList.record_openspec_workflow")}</div>
                  <div>{t("workflows.onboarding.stages.apply.mcpToolsList.generate_openspec_workflow_summary")}</div>
                </div>
              </div>
            </div>

            {/* Archive 阶段 */}
            <div style={{ marginBottom: "16px", paddingLeft: "8px" }}>
              <div style={{ fontSize: "12px", fontWeight: 600, marginBottom: "4px" }}>
                4️⃣ {t("workflows.onboarding.stages.archive.name")}
              </div>
              <div style={{ fontSize: "11px", color: "var(--vscode-descriptionForeground)", marginLeft: "16px", lineHeight: "1.6" }}>
                <div>{t("workflows.onboarding.stages.archive.function")}</div>
                <div>{t("workflows.onboarding.stages.archive.command")}</div>
                <div style={{ fontWeight: 600, marginTop: "4px" }}>
                  {t("workflows.onboarding.stages.archive.process")}
                </div>
                <div style={{ marginLeft: "8px" }}>
                  <div>{t("workflows.onboarding.stages.archive.processItems.moveDirectory")}</div>
                  <div>{t("workflows.onboarding.stages.archive.processItems.mergeSpec")}</div>
                  <div>{t("workflows.onboarding.stages.archive.processItems.recordState")}</div>
                </div>
                <div style={{ marginTop: "4px", fontWeight: 600 }}>
                  {t("workflows.onboarding.stages.archive.mcpTools")}
                </div>
                <div style={{ marginLeft: "8px" }}>
                  <div>{t("workflows.onboarding.stages.archive.mcpToolsList.record_openspec_workflow")}</div>
                </div>
              </div>
            </div>
          </div>

          {/* 工作流状态跟踪 */}
          <div style={{ marginBottom: "20px" }}>
            <div
              style={{
                fontSize: "13px",
                fontWeight: 600,
                marginBottom: "8px",
                color: "var(--vscode-foreground)"
              }}
            >
              📋 {t("workflows.onboarding.stateTracking.title")}
            </div>
            <div style={{ fontSize: "11px", color: "var(--vscode-descriptionForeground)", marginLeft: "8px", lineHeight: "1.6" }}>
              <div>{t("workflows.onboarding.stateTracking.description")}</div>
              <div style={{ marginTop: "4px", fontWeight: 600 }}>
                {t("workflows.onboarding.stateTracking.includes")}
              </div>
              <div style={{ marginLeft: "8px" }}>
                <div>{t("workflows.onboarding.stateTracking.includesItems.stage")}</div>
                <div>{t("workflows.onboarding.stateTracking.includesItems.status")}</div>
                <div>{t("workflows.onboarding.stateTracking.includesItems.progress")}</div>
                <div>{t("workflows.onboarding.stateTracking.includesItems.summary")}</div>
              </div>
            </div>
          </div>

          {/* 状态说明 */}
          <div style={{ marginBottom: "20px" }}>
            <div
              style={{
                fontSize: "13px",
                fontWeight: 600,
                marginBottom: "8px",
                color: "var(--vscode-foreground)"
              }}
            >
              📋 {t("workflows.onboarding.statusExplanation.title")}
            </div>
            <div style={{ fontSize: "11px", color: "var(--vscode-descriptionForeground)", marginLeft: "8px", lineHeight: "1.6" }}>
              <div>{t("workflows.onboarding.statusExplanation.inProgress")}</div>
              <div>{t("workflows.onboarding.statusExplanation.completed")}</div>
              <div>{t("workflows.onboarding.statusExplanation.paused")}</div>
            </div>
          </div>

          {/* 使用提示 */}
          <div>
            <div
              style={{
                fontSize: "13px",
                fontWeight: 600,
                marginBottom: "8px",
                color: "var(--vscode-foreground)"
              }}
            >
              💡 {t("workflows.onboarding.usageTips.title")}
            </div>
            <div style={{ fontSize: "11px", color: "var(--vscode-descriptionForeground)", marginLeft: "8px", lineHeight: "1.6" }}>
              <div>{t("workflows.onboarding.usageTips.commandPalette")}</div>
              <div>{t("workflows.onboarding.usageTips.mcpTools")}</div>
              <div>{t("workflows.onboarding.usageTips.viewDetail")}</div>
              <div>{t("workflows.onboarding.usageTips.progressBar")}</div>
              <div>{t("workflows.onboarding.usageTips.viewInUI")}</div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
