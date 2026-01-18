/**
 * Toast 通知容器组件
 */

import React from "react";
import { Toast } from "../../types";

interface ToastContainerProps {
  toasts: Toast[];
  onRemove?: (id: string) => void;
}

export const ToastContainer: React.FC<ToastContainerProps> = ({
  toasts,
  onRemove,
}) => {
  return (
    <div className="cocursor-marketplace-toasts">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={`cocursor-marketplace-toast cocursor-marketplace-toast-${toast.type}`}
          onClick={() => onRemove?.(toast.id)}
        >
          {toast.type === "success" ? "✓" : "✗"} {toast.message}
        </div>
      ))}
    </div>
  );
};
