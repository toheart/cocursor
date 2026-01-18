/**
 * 自定义 Hooks - SessionStorage 相关
 */

import { useState, useEffect, useCallback } from "react";

interface SessionStorageOptions<T> {
  key: string;
  defaultValue: T;
}

/**
 * SessionStorage Hook
 */
export function useSessionStorage<T>(
  options: SessionStorageOptions<T>
): [T, (value: T | ((prev: T) => T)) => void] {
  const { key, defaultValue } = options;
  const [storedValue, setStoredValue] = useState<T>(() => {
    try {
      const item = sessionStorage.getItem(key);
      return item ? JSON.parse(item) : defaultValue;
    } catch (error) {
      console.error(`读取 sessionStorage 失败 (${key}):`, error);
      return defaultValue;
    }
  });

  const setValue = useCallback(
    (value: T | ((prev: T) => T)) => {
      try {
        const valueToStore = value instanceof Function ? value(storedValue) : value;
        setStoredValue(valueToStore);
        sessionStorage.setItem(key, JSON.stringify(valueToStore));
      } catch (error) {
        console.error(`保存 sessionStorage 失败 (${key}):`, error);
      }
    },
    [key, storedValue]
  );

  return [storedValue, setValue];
}

/**
 * SessionStorage Hook - 分离的状态
 */
export function useSessionStorageState<T>(
  key: string,
  defaultValue: T
): [T, (value: T) => void] {
  return useSessionStorage({ key, defaultValue });
}
