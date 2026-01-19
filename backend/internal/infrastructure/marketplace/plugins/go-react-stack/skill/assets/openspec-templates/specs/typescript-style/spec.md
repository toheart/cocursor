# TypeScript Code Style Guide

This specification defines TypeScript/React code writing standards for the project.

## Basic Rules

- Use ESLint for code checking
- Comments in English
- Use async/await for asynchronous operations
- Type definitions in the same file or separate `.d.ts` files

## Naming Conventions

### Variables and Functions

- Use camelCase: `getUserName`, `isActive`
- Boolean values use `is`, `has`, `can`, `should` prefix: `isLoading`, `hasError`
- Constants use UPPER_SNAKE_CASE: `MAX_RETRY_COUNT`, `API_BASE_URL`

### Classes and Interfaces

- Use PascalCase: `UserService`, `ChatMessage`
- Interfaces don't use `I` prefix: `User` not `IUser`
- Type aliases use PascalCase: `type UserId = string`

### File Naming

- Use camelCase: `chatService.ts`, `workspaceView.ts`
- React components use PascalCase: `ChatPanel.tsx`
- Test files use `.test.ts` suffix: `api.test.ts`

## Type Rules

### Prefer Type Inference

```typescript
// Good
const name = "project";
const count = 42;

// Avoid (redundant type annotations)
const name: string = "project";
const count: number = 42;
```

### Explicit Function Return Types

```typescript
// Good - explicit return type
async function fetchUser(id: string): Promise<User> {
  return await api.get(`/users/${id}`);
}

// Avoid - implicit return type
async function fetchUser(id: string) {
  return await api.get(`/users/${id}`);
}
```

### Use Union Types Instead of Enums

```typescript
// Recommended
type Status = "pending" | "active" | "completed";

// Acceptable but not recommended
enum Status {
  Pending = "pending",
  Active = "active",
  Completed = "completed",
}
```

### Avoid any

```typescript
// Good
function parseData(data: unknown): User {
  if (isUser(data)) {
    return data;
  }
  throw new Error("Invalid data");
}

// Avoid
function parseData(data: any): User {
  return data;
}
```

## Async Handling

### Use async/await

```typescript
// Good
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

// Avoid Promise chains
function loadData() {
  return fetchUser()
    .then((user) => fetchPosts(user.id).then((posts) => ({ user, posts })))
    .catch((error) => {
      console.error("Failed to load data:", error);
      throw error;
    });
}
```

### Use Promise.all for Parallel Requests

```typescript
// Good - parallel execution
const [user, settings] = await Promise.all([fetchUser(), fetchSettings()]);

// Avoid - serial execution
const user = await fetchUser();
const settings = await fetchSettings();
```

## Error Handling

### Use Custom Error Classes

```typescript
class ApiError extends Error {
  constructor(message: string, public code: number, public detail?: string) {
    super(message);
    this.name = "ApiError";
  }
}
```

### Unified Error Handling

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

// Usage
const [data, error] = await safeCall(() => api.fetchData());
if (error) {
  console.error(error.message);
  return;
}
```

## API Call Conventions

### Use Unified API Service

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

### Response Type Definitions

```typescript
// Corresponds to backend API specification
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

## Import Order

Organize imports in the following order, with blank lines between groups:

1. Node.js built-in modules
2. Third-party libraries
3. Project internal modules

```typescript
import * as path from "path";
import * as fs from "fs";

import axios from "axios";

import { ApiService } from "./services/api";
import { ChatPanel } from "./components/ChatPanel";
```
