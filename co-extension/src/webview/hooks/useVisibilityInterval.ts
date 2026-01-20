/**
 * 页面可见性感知的定时器 Hook
 * 当页面不可见时自动暂停轮询，可见时恢复
 */

import { useEffect, useRef, useCallback } from "react";

/**
 * 页面可见性感知的 Interval Hook
 * @param callback 定时执行的回调函数
 * @param delay 间隔时间（毫秒），传 null 禁用
 * @param options 配置项
 * @param options.runOnVisible 页面变为可见时是否立即执行一次，默认 true
 */
export function useVisibilityInterval(
  callback: () => void,
  delay: number | null,
  options: { runOnVisible?: boolean } = {}
): void {
  const { runOnVisible = true } = options;
  const savedCallback = useRef(callback);
  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const isVisibleRef = useRef(!document.hidden);

  // 保存最新的 callback
  useEffect(() => {
    savedCallback.current = callback;
  }, [callback]);

  // 启动定时器
  const startInterval = useCallback(() => {
    if (delay === null || intervalRef.current) return;
    
    intervalRef.current = setInterval(() => {
      savedCallback.current();
    }, delay);
  }, [delay]);

  // 停止定时器
  const stopInterval = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  // 监听页面可见性变化
  useEffect(() => {
    if (delay === null) return;

    const handleVisibilityChange = () => {
      const isNowVisible = !document.hidden;
      const wasVisible = isVisibleRef.current;
      isVisibleRef.current = isNowVisible;

      if (isNowVisible && !wasVisible) {
        // 页面从隐藏变为可见
        if (runOnVisible) {
          // 立即执行一次
          savedCallback.current();
        }
        // 恢复定时器
        startInterval();
      } else if (!isNowVisible && wasVisible) {
        // 页面从可见变为隐藏
        stopInterval();
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);

    // 初始状态：如果页面可见，启动定时器
    if (!document.hidden) {
      startInterval();
    }

    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      stopInterval();
    };
  }, [delay, runOnVisible, startInterval, stopInterval]);
}
