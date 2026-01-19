# TypeScript/React 配置参考

## package.json 模板

```json
{
  "name": "project-frontend",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "lint": "eslint src --ext ts,tsx"
  },
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.20.0",
    "axios": "^1.6.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.0",
    "@types/react-dom": "^18.2.0",
    "@typescript-eslint/eslint-plugin": "^6.0.0",
    "@typescript-eslint/parser": "^6.0.0",
    "@vitejs/plugin-react": "^4.2.0",
    "eslint": "^8.0.0",
    "eslint-plugin-react": "^7.33.0",
    "eslint-plugin-react-hooks": "^4.6.0",
    "typescript": "^5.0.0",
    "vite": "^5.0.0"
  }
}
```

## tsconfig.json 模板

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "esModuleInterop": true,
    "forceConsistentCasingInFileNames": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  },
  "include": ["src"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

## vite.config.ts 模板

```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});
```

## .eslintrc.json 模板

```json
{
  "root": true,
  "parser": "@typescript-eslint/parser",
  "parserOptions": {
    "ecmaVersion": 2020,
    "sourceType": "module",
    "ecmaFeatures": {
      "jsx": true
    }
  },
  "plugins": ["@typescript-eslint", "react", "react-hooks"],
  "extends": [
    "eslint:recommended",
    "plugin:@typescript-eslint/recommended",
    "plugin:react/recommended",
    "plugin:react-hooks/recommended"
  ],
  "settings": {
    "react": {
      "version": "detect"
    }
  },
  "rules": {
    "@typescript-eslint/naming-convention": [
      "warn",
      {
        "selector": "variable",
        "format": ["camelCase", "UPPER_CASE"],
        "leadingUnderscore": "allow"
      },
      {
        "selector": "function",
        "format": ["camelCase"]
      },
      {
        "selector": "typeLike",
        "format": ["PascalCase"]
      }
    ],
    "@typescript-eslint/explicit-function-return-type": "warn",
    "@typescript-eslint/no-explicit-any": "warn",
    "@typescript-eslint/no-unused-vars": [
      "warn",
      {
        "argsIgnorePattern": "^_"
      }
    ],
    "react/react-in-jsx-scope": "off",
    "react/prop-types": "off"
  },
  "env": {
    "node": true,
    "es6": true,
    "browser": true
  }
}
```

## API 服务模板 (src/services/api.ts)

```typescript
import axios, { AxiosInstance, AxiosResponse } from 'axios';

// API 响应格式
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
  page?: PageInfo;
}

export interface PageInfo {
  page: number;
  pageSize: number;
  total: number;
  pages: number;
}

// API 错误
export class ApiError extends Error {
  constructor(
    message: string,
    public code: number,
    public detail?: string
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

// API 服务类
class ApiService {
  private client: AxiosInstance;

  constructor(baseURL: string = '/api/v1') {
    this.client = axios.create({
      baseURL,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // 响应拦截器
    this.client.interceptors.response.use(
      (response: AxiosResponse<ApiResponse<unknown>>) => {
        const data = response.data;
        if (data.code !== 0) {
          throw new ApiError(data.message, data.code, data.detail);
        }
        return response;
      },
      (error) => {
        if (error.response) {
          const data = error.response.data;
          throw new ApiError(
            data.message || '请求失败',
            data.code || error.response.status,
            data.detail
          );
        }
        throw new ApiError(error.message || '网络错误', 0);
      }
    );
  }

  async get<T>(path: string): Promise<T> {
    const response = await this.client.get<ApiResponse<T>>(path);
    return response.data.data;
  }

  async post<T>(path: string, data?: unknown): Promise<T> {
    const response = await this.client.post<ApiResponse<T>>(path, data);
    return response.data.data;
  }

  async put<T>(path: string, data?: unknown): Promise<T> {
    const response = await this.client.put<ApiResponse<T>>(path, data);
    return response.data.data;
  }

  async delete<T>(path: string): Promise<T> {
    const response = await this.client.delete<ApiResponse<T>>(path);
    return response.data.data;
  }
}

export const api = new ApiService();
```

## useApi Hook 模板 (src/hooks/useApi.ts)

```typescript
import { useState, useEffect } from 'react';
import { api, ApiError } from '../services/api';

export function useApi<T>(
  apiCall: () => Promise<T>,
  deps: unknown[] = []
): {
  data: T | null;
  loading: boolean;
  error: Error | null;
  refetch: () => void;
} {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await apiCall();
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return { data, loading, error, refetch: fetchData };
}
```

## App.tsx 模板

```typescript
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Home } from './components/Home';

function App(): JSX.Element {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
```
