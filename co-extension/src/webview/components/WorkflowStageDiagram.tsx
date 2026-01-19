import React from "react";
import { useTranslation } from "react-i18next";

/**
 * 可视化工作流阶段流程图组件
 * 显示四个阶段：init → proposal → apply → archive
 */
export const WorkflowStageDiagram: React.FC = () => {
  const { t } = useTranslation();

  const stages = [
    {
      key: "init",
      name: t("workflows.stage.init"),
      label: "init"
    },
    {
      key: "proposal",
      name: t("workflows.stage.proposal"),
      label: "proposal"
    },
    {
      key: "apply",
      name: t("workflows.stage.apply"),
      label: "apply"
    },
    {
      key: "archive",
      name: t("workflows.stage.archive"),
      label: "archive"
    }
  ];

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        gap: "8px",
        padding: "16px",
        flexWrap: "wrap"
      }}
    >
      {stages.map((stage, index) => (
        <React.Fragment key={stage.key}>
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              minWidth: "80px"
            }}
          >
            <div
              style={{
                padding: "8px 12px",
                backgroundColor: "var(--vscode-button-background)",
                color: "var(--vscode-button-foreground)",
                borderRadius: "4px",
                fontSize: "12px",
                fontWeight: 600,
                textAlign: "center",
                border: "1px solid var(--vscode-button-border)",
                minWidth: "70px"
              }}
            >
              {stage.label}
            </div>
            <div
              style={{
                marginTop: "4px",
                fontSize: "11px",
                color: "var(--vscode-descriptionForeground)",
                textAlign: "center"
              }}
            >
              {stage.name}
            </div>
          </div>
          {index < stages.length - 1 && (
            <div
              style={{
                fontSize: "16px",
                color: "var(--vscode-foreground)",
                margin: "0 4px"
              }}
            >
              →
            </div>
          )}
        </React.Fragment>
      ))}
    </div>
  );
};
