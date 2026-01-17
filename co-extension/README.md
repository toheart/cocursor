# CoCursor Extension

VSCode/Cursor 插件前端，用于分析 Cursor 后台数据并进行团队协作。

## 技术栈

- **语言**: TypeScript
- **框架**: VS Code Extension API + React
- **构建工具**: esbuild
- **UI**: VS Code TreeView + React Webview
- **HTTP 客户端**: Axios（通过 Extension 代理）

## 目录结构

```
co-extension/
├── src/
│   ├── extension.ts              # 插件入口文件
│   ├── webview/
│   │   ├── webviewPanel.ts       # Webview 面板管理
│   │   ├── index.tsx             # React 应用入口
│   │   ├── index.css             # 样式文件
│   │   ├── components/
│   │   │   └── App.tsx            # 主应用组件
│   │   ├── services/
│   │   │   └── api.ts             # API 服务（通过 Extension 代理）
│   │   └── types/
│   │       └── vscode.d.ts        # VSCode Webview API 类型
│   └── types/
│       └── message.ts             # 消息类型定义
├── dist/                         # 构建输出目录
│   ├── extension.js              # Extension 构建产物
│   └── webview/
│       ├── index.js              # Webview 构建产物
│       └── index.css             # 样式文件
├── package.json                  # 插件清单和依赖
├── tsconfig.json                 # TypeScript 配置
├── Makefile                      # 构建脚本
└── .eslintrc.json               # ESLint 配置
```

## 开发

### 安装依赖

```bash
cd co-extension
npm install
```

### 构建

```bash
# 开发构建（Extension + Webview）
make compile-debug

# 生产构建
make build

# 监听模式（自动重新构建）
make watch
```

### 运行

在 VSCode 中按 F5 启动调试。

### 命令

- `cocursor.openDashboard` - 打开仪表板（React Webview）
- `cocursor.refreshTasks` - 刷新任务列表
- `cocursor.addTask` - 添加任务

## 架构说明

### Extension 层（Node.js 环境）

- 运行在 VSCode Extension Host 中
- 负责与 VSCode API 交互
- 管理 Webview 面板生命周期
- 代理后端 API 调用

### Webview 层（浏览器环境）

- 运行在隔离的浏览器环境中
- 使用 React 构建 UI
- 通过 `postMessage` 与 Extension 通信
- 无法直接访问 Node.js API 或后端 API

### 消息通信

Webview 和 Extension 通过 `postMessage` 进行双向通信：

```typescript
// Webview -> Extension
vscode.postMessage({ command: "fetchChats" });

// Extension -> Webview
webview.postMessage({ type: "fetchChats-response", data: {...} });
```

## 代码规范

遵循项目 TypeScript 代码规范，参见 `openspec/specs/typescript-style/spec.md`。
