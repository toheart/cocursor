import React from "react";
import { useLocation, useNavigate } from "react-router-dom";

interface BreadcrumbItem {
  label: string;
  path: string;
}

export const NavigationBar: React.FC = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const viewType = (window as any).__VIEW_TYPE__ as "workAnalysis" | "recentSessions" | "marketplace" | undefined;

  // 根据路径生成面包屑
  const getBreadcrumbs = (): BreadcrumbItem[] => {
    const path = location.pathname;
    const crumbs: BreadcrumbItem[] = [];

    // 首页不显示面包屑
    if (path === "/" || path === "") {
      return crumbs;
    }

    // 非首页才添加首页面包屑
    crumbs.push({ label: "首页", path: "/" });

    if (path.startsWith("/sessions/")) {
      crumbs.push({ label: "会话详情", path: path });
    } else if (path === "/workflows" || path.startsWith("/workflows/")) {
      crumbs.push({ label: "工作流", path: "/workflows" });
      if (path.startsWith("/workflows/") && path !== "/workflows") {
        crumbs.push({ label: "详情", path: path });
      }
    }

    return crumbs;
  };

  const breadcrumbs = getBreadcrumbs();
  // 只有在有面包屑且不是首页时才显示返回按钮
  const canGoBack = breadcrumbs.length > 0 && location.pathname !== "/" && location.pathname !== "";

  const handleBack = () => {
    if (breadcrumbs.length > 1) {
      // 返回到上一级
      const previousPath = breadcrumbs[breadcrumbs.length - 2].path;
      navigate(previousPath);
    } else {
      navigate("/");
    }
  };

  const handleBreadcrumbClick = (path: string) => {
    navigate(path);
  };

  const getPageTitle = (): string => {
    const path = location.pathname;
    if (viewType === "recentSessions") {
      if (path.startsWith("/sessions/")) return "会话详情";
      return "最近对话";
    }
    if (viewType === "marketplace") {
      return "插件市场";
    }
    if (path === "/work-analysis") return "工作分析";
    if (path === "/workflows" || path.startsWith("/workflows/")) {
      if (path.startsWith("/workflows/") && path !== "/workflows") {
        return "工作流详情";
      }
      return "工作流";
    }
    if (path.startsWith("/sessions/")) return "会话详情";
    return "工作分析";
  };

  return (
    <div className="cocursor-navbar">
      <div className="cocursor-navbar-left">
        {canGoBack && (
          <button
            className="cocursor-navbar-back"
            onClick={handleBack}
            title="返回"
          >
            ← 返回
          </button>
        )}
        <h2 className="cocursor-navbar-title">{getPageTitle()}</h2>
      </div>
      {breadcrumbs.length > 0 && (
        <nav className="cocursor-navbar-breadcrumbs">
          {breadcrumbs.map((crumb, index) => (
            <React.Fragment key={crumb.path}>
              {index > 0 && <span className="cocursor-navbar-separator">/</span>}
              <button
                className={`cocursor-navbar-crumb ${
                  index === breadcrumbs.length - 1 ? "active" : ""
                }`}
                onClick={() => handleBreadcrumbClick(crumb.path)}
                disabled={index === breadcrumbs.length - 1}
              >
                {crumb.label}
              </button>
            </React.Fragment>
          ))}
        </nav>
      )}
    </div>
  );
};
