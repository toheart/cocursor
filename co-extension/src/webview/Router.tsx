import React, { useEffect } from "react";
import {
  HashRouter,
  Routes,
  Route,
  Navigate,
  useNavigate,
  useLocation,
} from "react-router-dom";
import { useTranslation } from "react-i18next";
import { WorkAnalysis } from "./components/WorkAnalysis";
import { SessionList } from "./components/SessionList";
import { SessionDetail } from "./components/SessionDetail";
import { Marketplace } from "./components/Marketplace";
import { RAGSearch } from "./components/RAGSearch";
import { RAGConfig } from "./components/RAGConfig";
import { NavigationBar } from "./components/NavigationBar";
import {
  TeamHomePage,
  TeamCreatePage,
  TeamJoinPage,
  IdentityPage,
  NetworkPage,
  TeamDetailPage,
  MembersPage,
  WeeklyPage,
  SessionsPage,
  SkillsPage,
} from "./components/Team/pages";
import { CodeAnalysisConfig } from "./components/CodeAnalysis";
import { getVscodeApi } from "./services/api";

// 内部组件：处理初始路由导航
const RouterContent: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const viewType = (window as any).__VIEW_TYPE__ as
    | "workAnalysis"
    | "recentSessions"
    | "marketplace"
    | "ragSearch"
    | "team"
    | "codeAnalysis"
    | undefined;

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
        viewType === "recentSessions"
          ? `${t("navigation.recentSessions")} - CoCursor`
          : viewType === "marketplace"
            ? `${t("navigation.marketplace")} - CoCursor`
            : viewType === "codeAnalysis"
              ? `${t("navigation.codeAnalysis")} - CoCursor`
              : `${t("navigation.workAnalysis")} - CoCursor`,
      "/work-analysis": `${t("navigation.workAnalysis")} - CoCursor`,
      "/marketplace": `${t("navigation.marketplace")} - CoCursor`,
    };

    const title =
      titles[location.pathname] ||
      (location.pathname.startsWith("/sessions/")
        ? `${t("navigation.sessionDetail")} - CoCursor`
        : viewType === "recentSessions"
          ? `${t("navigation.recentSessions")} - CoCursor`
          : viewType === "marketplace"
            ? `${t("navigation.marketplace")} - CoCursor`
            : viewType === "ragSearch"
              ? `${t("navigation.ragSearch")} - CoCursor`
              : viewType === "team"
                ? `${t("navigation.team")} - CoCursor`
                : viewType === "codeAnalysis"
                  ? `${t("navigation.codeAnalysis")} - CoCursor`
                  : `${t("navigation.workAnalysis")} - CoCursor`);

    document.title = title;

    // 通知 Extension 更新 WebView 标题
    // 使用共享的 vscode API 实例，避免重复获取
    const vscode = getVscodeApi();
    vscode.postMessage({
      command: "updateTitle",
      payload: { title },
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

  if (viewType === "ragSearch") {
    return (
      <div className="cocursor-router-container">
        <NavigationBar />
        <div className="cocursor-router-content">
          <Routes>
            <Route path="/" element={<RAGSearch />} />
            <Route path="/config" element={<RAGConfig />} />
            <Route path="/sessions/:sessionId" element={<SessionDetail />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </div>
      </div>
    );
  }

  if (viewType === "team") {
    return (
      <div className="cocursor-router-container">
        <NavigationBar />
        <div className="cocursor-router-content">
          <Routes>
            <Route path="/" element={<TeamHomePage />} />
            <Route path="/create" element={<TeamCreatePage />} />
            <Route path="/join" element={<TeamJoinPage />} />
            <Route path="/identity" element={<IdentityPage />} />
            <Route path="/network" element={<NetworkPage />} />
            <Route path="/team/:teamId" element={<TeamDetailPage />}>
              <Route path="members" element={<MembersPage />} />
              <Route path="weekly" element={<WeeklyPage />} />
              <Route path="sessions" element={<SessionsPage />} />
              <Route path="skills" element={<SkillsPage />} />
              <Route index element={<Navigate to="members" replace />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </div>
      </div>
    );
  }

  if (viewType === "codeAnalysis") {
    return (
      <div className="cocursor-router-container">
        <NavigationBar />
        <div className="cocursor-router-content">
          <Routes>
            <Route path="/" element={<CodeAnalysisConfig />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </div>
      </div>
    );
  }

  // 默认是工作分析
  return (
    <div className="cocursor-router-container">
      <NavigationBar />
      <div className="cocursor-router-content">
        <Routes>
          <Route path="/" element={<WorkAnalysis />} />
          <Route path="/work-analysis" element={<WorkAnalysis />} />
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
