/**
 * 可复用空状态组件
 */

import React from "react";

interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: React.ReactNode;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  icon,
  title,
  description,
  action,
}) => {
  return (
    <div className="cocursor-empty" style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "12px" }}>
      {icon && <div style={{ fontSize: "48px", opacity: 0.5 }}>{icon}</div>}
      <h3 style={{ margin: 0, fontSize: "16px", fontWeight: 600 }}>{title}</h3>
      {description && (
        <p style={{ margin: 0, fontSize: "13px", color: "var(--vscode-descriptionForeground)" }}>{description}</p>
      )}
      {action && <div style={{ marginTop: "8px" }}>{action}</div>}
    </div>
  );
};
