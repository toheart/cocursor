/**
 * 自定义 Hooks - API 相关
 */

import { useState, useCallback, useRef, useEffect } from "react";

interface ApiState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

interface UseApiOptions<T> {
  initialData?: T | null;
  onSuccess?: (data: T) => void;
  onError?: (error: Error) => void;
}

/**
 * 通用的 API 请求 Hook
 */
export function useApi<T>(
  fetchFn: () => Promise<T>,
  options: UseApiOptions<T> = {}
): ApiState<T> & { refetch: () => Promise<void> } {
  const [state, setState] = useState<ApiState<T>>({
    data: options.initialData || null,
    loading: true,
    error: null,
  });

  const isMountedRef = useRef(true);
  const fetchFnRef = useRef(fetchFn);
  const optionsRef = useRef(options);

  // 更新 refs
  useEffect(() => {
    fetchFnRef.current = fetchFn;
    optionsRef.current = options;
  }, [fetchFn, options]);

  const execute = useCallback(async () => {
    setState(prev => ({ ...prev, loading: true, error: null }));
    isMountedRef.current = true;

    try {
      const data = await fetchFnRef.current();
      
      if (!isMountedRef.current) return;
      
      setState({ data, loading: false, error: null });
      optionsRef.current.onSuccess?.(data);
    } catch (error) {
      if (!isMountedRef.current) return;
      
      const errorMessage = error instanceof Error ? error.message : "未知错误";
      setState(prev => ({ ...prev, loading: false, error: errorMessage }));
      optionsRef.current.onError?.(error instanceof Error ? error : new Error(errorMessage));
    }
  }, []);

  // 自动执行一次（仅在首次挂载时）
  useEffect(() => {
    execute();
    return () => {
      isMountedRef.current = false;
    };
  }, []); // 空依赖数组，只在挂载时执行一次

  return {
    ...state,
    refetch: execute,
  };
}

/**
 * 防抖 Hook
 */
export function useDebounce<T>(value: T, delay: number = 300): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);

  return debouncedValue;
}
