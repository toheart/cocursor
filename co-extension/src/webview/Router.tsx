import React, { useEffect } from "react";
import { HashRouter, Routes, Route, Navigate, useNavigate, useLocation } from "react-router-dom";
import { WorkAnalysis } from "./components/WorkAnalysis";
import { SessionList } from "./components/SessionList";
import { SessionDetail } from "./components/SessionDetail";
import { Marketplace } from "./components/Marketplace";
import { WorkflowList } from "./components/WorkflowList";
import { WorkflowDetail } from "./components/WorkflowDetail";
import { NavigationBar } from "./components/NavigationBar";
import { getVscodeApi } from "./services/api";

// 内部组件：处理初始路由导航
const RouterContent: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const viewType = (window as any).__VIEW_TYPE__ as "workAnalysis" | "recentSessions" | "marketplace" | undefined;

  useEffect(() => {
    // 获取初始路由
    const initialRoute = (window as any).__INITIAL_ROUTE__;
    if (initialRoute && initialRoute !== "/" && location.pathname === "/") {
      navigate(initialRoute, { replace: true });
    }

    // 监听来自 Extension 的路由导航消息
    const handleMessage = (event: MessageEvent) => {
      if (event.data.type === "navigate" && event.data.route) {
        // 移除 # 前缀（HashRouter 会自动添加）
        const route = event.data.route.startsWith("#") 
          ? event.data.route.substring(1) 
          : event.data.route;
        navigate(route, { replace: true });
      }
    };

    window.addEventListener("message", handleMessage);
    return () => {
      window.removeEventListener("message", handleMessage);
    };
  }, [navigate, location.pathname]);

  // 更新 WebView 标题
  useEffect(() => {
    const titles: Record<string, string> = {
      "/": 
        viewType === "recentSessions" ? "最近对话 - CoCursor" :
        viewType === "marketplace" ? "插件市场 - CoCursor" :
        "工作分析 - CoCursor",
      "/work-analysis": "工作分析 - CoCursor",
      "/marketplace": "插件市场 - CoCursor",
      "/workflows": "工作流 - CoCursor",
    };
    
    const title = titles[location.pathname] || 
      (location.pathname.startsWith("/sessions/") 
        ? "会话详情 - CoCursor"
        : location.pathname.startsWith("/workflows/")
        ? "工作流详情 - CoCursor"
        : viewType === "recentSessions" ? "最近对话 - CoCursor" 
        : viewType === "marketplace" ? "插件市场 - CoCursor"
        : "工作分析 - CoCursor");
    
    document.title = title;
    
    // 通知 Extension 更新 WebView 标题
    // 使用共享的 vscode API 实例，避免重复获取
    const vscode = getVscodeApi();
    vscode.postMessage({
      command: "updateTitle",
      payload: { title }
    });
  }, [location.pathname, viewType]);

  // 根据 viewType 渲染不同的路由
  if (viewType === "recentSessions") {
    return (
      <div className="cocursor-router-container">
        <NavigationBar />
        <div className="cocursor-router-content">
          <Routes>
            <Route path="/" element={<SessionList />} />
            <Route path="/sessions/:sessionId" element={<SessionDetail />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </div>
      </div>
    );
  }

  if (viewType === "marketplace") {
    return (
      <div className="cocursor-router-container">
        <NavigationBar />
        <div className="cocursor-router-content">
          <Routes>
            <Route path="/" element={<Marketplace />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </div>
      </div>
    );
  }

  // 默认是工作分析，但支持工作流路由
  return (
    <div className="cocursor-router-container">
      <NavigationBar />
      <div className="cocursor-router-content">
        <Routes>
          <Route path="/" element={<WorkAnalysis />} />
          <Route path="/work-analysis" element={<WorkAnalysis />} />
          <Route path="/workflows" element={<WorkflowList />} />
          <Route path="/workflows/:changeId" element={<WorkflowDetail />} />
          <Route path="/sessions/:sessionId" element={<SessionDetail />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </div>
    </div>
  );
};

export const Router: React.FC = () => {
  return (
    <HashRouter>
      <RouterContent />
    </HashRouter>
  );
};
