/**
 * 统一页面头部组件
 * 提供返回按钮 + 标题 + 可选操作区
 */

import React from "react";
import { useNavigate } from "react-router-dom";

interface PageHeaderProps {
  /** 标题 */
  title: string;
  /** 副标题 */
  subtitle?: string;
  /** 返回路径（不传则不显示返回按钮） */
  backTo?: string;
  /** 右侧操作区 */
  actions?: React.ReactNode;
  /** 标题右侧的 badge */
  badge?: React.ReactNode;
}

export const PageHeader: React.FC<PageHeaderProps> = ({
  title,
  subtitle,
  backTo,
  actions,
  badge,
}) => {
  const navigate = useNavigate();

  return (
    <div className="ct-page-header">
      <div className="ct-page-header-left">
        {backTo && (
          <button
            className="ct-page-header-back"
            onClick={() => navigate(backTo)}
            aria-label="Back"
          >
            <span className="codicon codicon-chevron-left" />
          </button>
        )}
        <div className="ct-page-header-title-group">
          <div className="ct-page-header-title-row">
            <h2 className="ct-page-header-title">{title}</h2>
            {badge}
          </div>
          {subtitle && (
            <span className="ct-page-header-subtitle">{subtitle}</span>
          )}
        </div>
      </div>
      {actions && <div className="ct-page-header-actions">{actions}</div>}
    </div>
  );
};
