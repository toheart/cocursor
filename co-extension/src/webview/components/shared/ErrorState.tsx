/**
 * 可复用错误状态组件
 */

import React from "react";
import { useTranslation } from "react-i18next";

interface ErrorStateProps {
  error: string;
  onRetry?: () => void;
  retryText?: string;
}

export const ErrorState: React.FC<ErrorStateProps> = ({
  error,
  onRetry,
  retryText,
}) => {
  const { t } = useTranslation();
  const defaultRetryText = retryText || t("common.retry");
  return (
    <div className="cocursor-error">
      <div style={{ marginBottom: "8px" }}>⚠️ {error}</div>
      {onRetry && (
        <button 
          onClick={onRetry}
          style={{
            padding: "6px 12px",
            backgroundColor: "var(--vscode-button-background)",
            color: "var(--vscode-button-foreground)",
            border: "none",
            borderRadius: "4px",
            cursor: "pointer",
            fontSize: "12px"
          }}
        >
          {defaultRetryText}
        </button>
      )}
    </div>
  );
};
