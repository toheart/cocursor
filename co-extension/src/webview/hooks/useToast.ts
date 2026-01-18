/**
 * 自定义 Hooks - Toast 相关
 */

import { useState, useCallback } from "react";
import { Toast } from "../types";

export interface ToastOptions {
  duration?: number;
}

/**
 * Toast Hook
 */
export function useToast() {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const showToast = useCallback(
    (message: string, type: "success" | "error", options: ToastOptions = {}) => {
      const id = Date.now().toString();
      const newToast: Toast = { id, message, type };
      
      setToasts(prev => [...prev, newToast]);

      // 自动移除
      setTimeout(() => {
        setToasts(prev => prev.filter(t => t.id !== id));
      }, options.duration || 3000);
    },
    []
  );

  const removeToast = useCallback((id: string) => {
    setToasts(prev => prev.filter(t => t.id !== id));
  }, []);

  return {
    toasts,
    showToast,
    removeToast,
  };
}
