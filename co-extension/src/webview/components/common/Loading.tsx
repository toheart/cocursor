import React from "react";

interface LoadingProps {
  message?: string;
  className?: string;
}

export const Loading: React.FC<LoadingProps> = ({
  message = "加载中...",
  className = ""
}) => {
  return (
    <div className={`cocursor-loading ${className}`}>
      {message}
    </div>
  );
};
