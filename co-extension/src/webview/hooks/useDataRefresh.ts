/**
 * 数据刷新 Hook
 * 监听来自 Extension 的数据更新通知，自动触发刷新
 */

import { useEffect, useRef, useCallback } from "react";
import { ExtensionMessage } from "../../types/message";

interface UseDataRefreshOptions {
  /**
   * 监听的数据类型，如 "dailySummary"、"workAnalysis" 等
   * 如果不指定，则监听所有类型的刷新通知
   */
  dataType?: string | string[];
  /**
   * 是否在面板变为可见时也触发刷新
   * @default true
   */
  refreshOnVisible?: boolean;
}

/**
 * 监听数据刷新事件的 Hook
 * @param onRefresh 刷新回调函数
 * @param options 配置选项
 */
export function useDataRefresh(
  onRefresh: () => void,
  options: UseDataRefreshOptions = {}
): void {
  const { dataType, refreshOnVisible = true } = options;
  const onRefreshRef = useRef(onRefresh);

  // 保存最新的回调
  useEffect(() => {
    onRefreshRef.current = onRefresh;
  }, [onRefresh]);

  // 检查数据类型是否匹配
  const isDataTypeMatch = useCallback((eventDataType: string): boolean => {
    // 如果事件类型是 "all"，总是匹配
    if (eventDataType === "all") {
      return true;
    }
    // 如果没有指定要监听的类型，总是匹配
    if (!dataType) {
      return true;
    }
    // 检查是否匹配
    if (Array.isArray(dataType)) {
      return dataType.includes(eventDataType);
    }
    return dataType === eventDataType;
  }, [dataType]);

  useEffect(() => {
    const handleMessage = (event: MessageEvent<ExtensionMessage>) => {
      const { type, data } = event.data || {};

      // 处理数据更新通知
      if (type === "dataUpdated") {
        const eventDataType = (data as { dataType?: string })?.dataType || "all";
        if (isDataTypeMatch(eventDataType)) {
          console.log("[useDataRefresh] received dataUpdated event, triggering refresh");
          onRefreshRef.current();
        }
      }

      // 处理面板变为可见的通知
      if (type === "panelBecameVisible" && refreshOnVisible) {
        console.log("[useDataRefresh] panel became visible, triggering refresh");
        onRefreshRef.current();
      }
    };

    window.addEventListener("message", handleMessage);

    return () => {
      window.removeEventListener("message", handleMessage);
    };
  }, [isDataTypeMatch, refreshOnVisible]);
}
