import React, { useState, useEffect, useCallback } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import i18n from "../i18n/config";
import { getVscodeApi } from "../services/api";

interface BreadcrumbItem {
  label: string;
  path: string;
}

export const NavigationBar: React.FC = () => {
  const { t, i18n: i18nInstance } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const viewType = (window as any).__VIEW_TYPE__ as "workAnalysis" | "recentSessions" | "marketplace" | "ragSearch" | "team" | undefined;
  const [currentLanguage, setCurrentLanguage] = useState<string>(i18nInstance.language);

  // 根据路径生成面包屑
  const getBreadcrumbs = useCallback((): BreadcrumbItem[] => {
    const path = location.pathname;
    const crumbs: BreadcrumbItem[] = [];

    // 首页不显示面包屑
    if (path === "/" || path === "") {
      return crumbs;
    }

    // 非首页才添加首页面包屑
    crumbs.push({ label: t("navigation.home"), path: "/" });

    if (path.startsWith("/sessions/")) {
      crumbs.push({ label: t("navigation.sessionDetail"), path: path });
    } else if (viewType === "ragSearch") {
      if (path === "/config") {
        crumbs.push({ label: t("navigation.ragConfig"), path: path });
      } else {
        crumbs.push({ label: t("navigation.ragSearch"), path: "/" });
      }
    }

    return crumbs;
  }, [location.pathname, viewType, t]);

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

  // 监听语言变化
  useEffect(() => {
    const handleLanguageChanged = (lng: string) => {
      setCurrentLanguage(lng);
    };
    
    i18nInstance.on('languageChanged', handleLanguageChanged);
    return () => {
      i18nInstance.off('languageChanged', handleLanguageChanged);
    };
  }, [i18nInstance]);

  // 切换语言：发送消息到 extension，由 extension 统一管理并广播
  const handleLanguageChange = (newLang: 'zh-CN' | 'en') => {
    try {
      const vscode = getVscodeApi();
      vscode.postMessage({
        command: 'changeLanguage',
        payload: { language: newLang }
      });
    } catch (e) {
      // 降级方案：如果 vscode API 不可用，直接修改
      console.warn('Failed to get vscode API, using fallback language change', e);
      i18nInstance.changeLanguage(newLang);
      try {
        localStorage.setItem('cocursor-language', newLang);
      } catch (err) {
        // localStorage 可能不可用，忽略错误
      }
    }
  };

  const getPageTitle = useCallback((): string => {
    const path = location.pathname;
    if (viewType === "recentSessions") {
      if (path.startsWith("/sessions/")) return t("navigation.sessionDetail");
      return t("navigation.recentSessions");
    }
    if (viewType === "marketplace") {
      return t("navigation.marketplace");
    }
    if (viewType === "ragSearch") {
      if (path === "/config") return t("navigation.ragConfig");
      return t("navigation.ragSearch");
    }
    if (viewType === "team") {
      return t("navigation.team");
    }
    if (path === "/work-analysis") return t("navigation.workAnalysis");
    if (path.startsWith("/sessions/")) return t("navigation.sessionDetail");
    return t("navigation.workAnalysis");
  }, [location.pathname, viewType, t]);

  return (
    <div className="cocursor-navbar">
      <div className="cocursor-navbar-left">
        {canGoBack && (
          <button
            className="cocursor-navbar-back"
            onClick={handleBack}
            title={t("common.back")}
          >
            ← {t("common.back")}
          </button>
        )}
        <h2 className="cocursor-navbar-title">{getPageTitle()}</h2>
      </div>
      <div className="cocursor-navbar-right" style={{ display: "flex", alignItems: "center", gap: "8px" }}>
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
        {/* 语言切换按钮 */}
        <div
          style={{
            display: "flex",
            gap: "4px",
            padding: "4px",
            backgroundColor: "var(--vscode-button-secondaryBackground)",
            borderRadius: "4px",
            border: "1px solid var(--vscode-button-border)"
          }}
        >
          <button
            onClick={() => handleLanguageChange('zh-CN')}
            style={{
              padding: "4px 8px",
              fontSize: "12px",
              backgroundColor: currentLanguage === 'zh-CN' 
                ? "var(--vscode-button-background)" 
                : "transparent",
              color: currentLanguage === 'zh-CN'
                ? "var(--vscode-button-foreground)"
                : "var(--vscode-foreground)",
              border: "none",
              borderRadius: "2px",
              cursor: "pointer",
              transition: "all 0.2s"
            }}
            title={t("navigation.switchToChinese")}
          >
            {t("navigation.chinese")}
          </button>
          <button
            onClick={() => handleLanguageChange('en')}
            style={{
              padding: "4px 8px",
              fontSize: "12px",
              backgroundColor: currentLanguage === 'en' 
                ? "var(--vscode-button-background)" 
                : "transparent",
              color: currentLanguage === 'en'
                ? "var(--vscode-button-foreground)"
                : "var(--vscode-foreground)",
              border: "none",
              borderRadius: "2px",
              cursor: "pointer",
              transition: "all 0.2s"
            }}
            title={t("navigation.switchToEnglish")}
          >
            {t("navigation.english")}
          </button>
        </div>
      </div>
    </div>
  );
};
