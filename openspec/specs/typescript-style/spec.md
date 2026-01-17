# TypeScript 编码规范

本规范定义 cocursor 项目的 TypeScript/VS Code 插件代码编写标准。

## 基础规范

- 使用 ESLint 进行代码检查
- 注释使用中文
- 使用 async/await 处理异步操作
- 类型定义放在同文件或单独的 `.d.ts` 文件

## 命名规范

### 变量和函数

- 使用 camelCase：`getUserName`, `isActive`
- 布尔值使用 `is`, `has`, `can`, `should` 前缀：`isLoading`, `hasError`
- 常量使用 UPPER_SNAKE_CASE：`MAX_RETRY_COUNT`, `API_BASE_URL`

### 类和接口

- 使用 PascalCase：`UserService`, `ChatMessage`
- 接口不使用 `I` 前缀：`User` 而非 `IUser`
- 类型别名使用 PascalCase：`type UserId = string`

### 文件命名

- 使用 camelCase：`chatService.ts`, `workspaceView.ts`
- React/Webview 组件使用 PascalCase：`ChatPanel.tsx`
- 测试文件使用 `.test.ts` 后缀：`api.test.ts`

## 类型规范

### 优先使用类型推断

```typescript
// 好
const name = "cocursor";
const count = 42;

// 避免（冗余类型注解）
const name: string = "cocursor";
const count: number = 42;
```

### 明确函数返回类型

```typescript
// 好 - 明确返回类型
async function fetchUser(id: string): Promise<User> {
  return await api.get(`/users/${id}`);
}

// 避免 - 隐式返回类型
async function fetchUser(id: string) {
  return await api.get(`/users/${id}`);
}
```

### 使用联合类型替代枚举

```typescript
// 推荐
type Status = "pending" | "active" | "completed";

// 可用但不推荐
enum Status {
  Pending = "pending",
  Active = "active",
  Completed = "completed",
}
```

### 避免 any

```typescript
// 好
function parseData(data: unknown): User {
  if (isUser(data)) {
    return data;
  }
  throw new Error("Invalid data");
}

// 避免
function parseData(data: any): User {
  return data;
}
```

## 异步处理

### 使用 async/await

```typescript
// 好
async function loadData() {
  try {
    const user = await fetchUser();
    const posts = await fetchPosts(user.id);
    return { user, posts };
  } catch (error) {
    console.error("Failed to load data:", error);
    throw error;
  }
}

// 避免 Promise 链
function loadData() {
  return fetchUser()
    .then((user) => fetchPosts(user.id).then((posts) => ({ user, posts })))
    .catch((error) => {
      console.error("Failed to load data:", error);
      throw error;
    });
}
```

### 并行请求使用 Promise.all

```typescript
// 好 - 并行执行
const [user, settings] = await Promise.all([fetchUser(), fetchSettings()]);

// 避免 - 串行执行
const user = await fetchUser();
const settings = await fetchSettings();
```

## 错误处理

### 使用自定义错误类

```typescript
class ApiError extends Error {
  constructor(message: string, public code: number, public detail?: string) {
    super(message);
    this.name = "ApiError";
  }
}
```

### 统一错误处理

```typescript
async function safeCall<T>(
  fn: () => Promise<T>
): Promise<[T, null] | [null, Error]> {
  try {
    const result = await fn();
    return [result, null];
  } catch (error) {
    return [null, error instanceof Error ? error : new Error(String(error))];
  }
}

// 使用
const [data, error] = await safeCall(() => api.fetchData());
if (error) {
  vscode.window.showErrorMessage(error.message);
  return;
}
```

## VS Code 插件规范

### 命令注册

```typescript
// 统一在 commands/index.ts 中注册
export function registerCommands(context: vscode.ExtensionContext) {
  context.subscriptions.push(
    vscode.commands.registerCommand("cocursor.refresh", handleRefresh),
    vscode.commands.registerCommand("cocursor.openChat", handleOpenChat)
  );
}
```

### Disposable 管理

```typescript
// 好 - 使用 subscriptions 管理
context.subscriptions.push(watcher);

// 好 - 组合多个 disposable
const disposables: vscode.Disposable[] = [];
disposables.push(watcher1, watcher2);
context.subscriptions.push(...disposables);
```

### 配置访问

```typescript
// 封装配置访问
function getConfig<T>(key: string, defaultValue: T): T {
  return vscode.workspace.getConfiguration("cocursor").get<T>(key, defaultValue);
}

// 使用
const port = getConfig("daemon.port", 19960);
```

## API 调用规范

### 使用统一的 API 服务

```typescript
// services/api.ts
class ApiService {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  async get<T>(path: string): Promise<ApiResponse<T>> {
    const response = await axios.get(`${this.baseUrl}${path}`);
    return this.handleResponse<T>(response);
  }

  private handleResponse<T>(response: AxiosResponse): ApiResponse<T> {
    const data = response.data;
    if (data.code !== 0) {
      throw new ApiError(data.message, data.code, data.detail);
    }
    return data;
  }
}
```

### 响应类型定义

```typescript
// 与后端 API 规范对应
interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
  page?: PageInfo;
}

interface PageInfo {
  page: number;
  pageSize: number;
  total: number;
  pages: number;
}

interface ApiError {
  code: number;
  message: string;
  detail?: string;
}
```

## Webview 规范

### 消息通信

```typescript
// Webview -> Extension
interface WebviewMessage {
  command: string;
  payload?: unknown;
}

// Extension -> Webview
interface ExtensionMessage {
  type: string;
  data?: unknown;
}
```

### 状态管理

```typescript
// 使用 getState/setState 持久化
const vscode = acquireVsCodeApi();

// 保存状态
vscode.setState({ scrollPosition: 100 });

// 恢复状态
const state = vscode.getState() ?? { scrollPosition: 0 };
```

## 导入顺序

按以下顺序组织导入，组间用空行分隔：

1. Node.js 内置模块
2. 第三方库
3. VS Code API
4. 项目内部模块

```typescript
import * as path from "path";
import * as fs from "fs";

import axios from "axios";

import * as vscode from "vscode";

import { ApiService } from "./services/api";
import { ChatPanel } from "./webview/chatPanel";
```
