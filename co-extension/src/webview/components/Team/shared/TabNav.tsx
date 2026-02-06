/**
 * Tab 导航组件
 * 基于路由的 Tab 切换
 */

import React from "react";
import { useNavigate, useLocation } from "react-router-dom";

export interface TabItem {
  /** Tab 标识 */
  id: string;
  /** 显示文本 */
  label: string;
  /** 图标（codicon class） */
  icon?: string;
  /** 计数 badge */
  count?: number;
  /** 对应的路由路径 */
  path: string;
}

interface TabNavProps {
  tabs: TabItem[];
}

export const TabNav: React.FC<TabNavProps> = ({ tabs }) => {
  const navigate = useNavigate();
  const location = useLocation();

  // 判断当前激活的 tab
  const activeTab = tabs.find((tab) => location.pathname.endsWith(tab.path)) || tabs[0];

  return (
    <div className="ct-tab-nav">
      {tabs.map((tab) => (
        <button
          key={tab.id}
          className={`ct-tab-nav-item ${activeTab?.id === tab.id ? "active" : ""}`}
          onClick={() => navigate(tab.path, { replace: true })}
        >
          {tab.icon && <span className={`codicon codicon-${tab.icon}`} />}
          <span className="ct-tab-nav-label">{tab.label}</span>
          {tab.count !== undefined && tab.count > 0 && (
            <span className="ct-tab-nav-count">{tab.count}</span>
          )}
        </button>
      ))}
    </div>
  );
};
