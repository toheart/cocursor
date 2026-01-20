import React from "react";
import { createRoot } from "react-dom/client";
import { Router } from "./Router";

// 导入所有样式模块
// 注意：加载顺序很重要，后面的样式会覆盖前面的
import "./styles/base.css";
import "./styles/futuristic.css"; // 加载未来科技感样式（需要优先级控制）
import "./styles/animations.css";
import "./styles/navbar.css";
import "./styles/components.css";
import "./styles/sessions.css";
import "./styles/work-analysis.css";
import "./styles/markdown.css";
import "./styles/marketplace.css";
import "./styles/workflow.css";
import "./styles/team.css";
import "./styles/rag.css"; // RAG 样式最后加载，确保覆盖其他样式

// 初始化 i18n
import "./i18n/config";

// 初始化 React 应用
console.log("Webview: 开始初始化 React 应用");

const container = document.getElementById("root");
if (!container) {
  console.error("Webview: 找不到 root 元素");
  throw new Error("Root element not found");
}

console.log("Webview: 找到 root 元素，创建 React root");
const root = createRoot(container);

// 路由导航消息在 Router 组件中处理

try {
  root.render(
    <React.StrictMode>
      <Router />
    </React.StrictMode>
  );
  console.log("Webview: React 应用已渲染");
} catch (error) {
  console.error("Webview: React 渲染失败", error);
  container.innerHTML = `<div style="padding: 20px; color: red;">
    <h2>加载错误</h2>
    <p>${error instanceof Error ? error.message : String(error)}</p>
  </div>`;
}
