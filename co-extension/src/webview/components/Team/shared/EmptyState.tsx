/**
 * 统一空状态组件
 */

import React from "react";

interface EmptyStateProps {
  /** 图标（codicon class） */
  icon?: string;
  /** 主标题 */
  title: string;
  /** 描述 */
  description?: string;
  /** 操作按钮 */
  action?: React.ReactNode;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  icon = "inbox",
  title,
  description,
  action,
}) => {
  return (
    <div className="ct-empty-state">
      <span className={`codicon codicon-${icon} ct-empty-state-icon`} />
      <p className="ct-empty-state-title">{title}</p>
      {description && (
        <span className="ct-empty-state-desc">{description}</span>
      )}
      {action && <div className="ct-empty-state-action">{action}</div>}
    </div>
  );
};
