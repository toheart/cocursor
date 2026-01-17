import React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./components/App";
import "./index.css";

// 初始化 React 应用
console.log("Webview: 开始初始化 React 应用");

const container = document.getElementById("root");
if (!container) {
  console.error("Webview: 找不到 root 元素");
  throw new Error("Root element not found");
}

console.log("Webview: 找到 root 元素，创建 React root");
const root = createRoot(container);

try {
  root.render(
    <React.StrictMode>
      <App />
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
