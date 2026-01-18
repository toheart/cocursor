import React from "react";

interface ErrorProps {
  message: string;
  className?: string;
  onRetry?: () => void;
}

export const Error: React.FC<ErrorProps> = ({
  message,
  className = "",
  onRetry
}) => {
  return (
    <div className={`cocursor-error ${className}`}>
      <span>错误: {message}</span>
      {onRetry && (
        <button
          onClick={onRetry}
          className="cocursor-error-retry"
          type="button"
        >
          重试
        </button>
      )}
    </div>
  );
};
