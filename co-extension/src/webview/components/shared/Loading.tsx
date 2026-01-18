/**
 * 可复用加载状态组件
 */

import React from "react";
import { useTranslation } from "react-i18next";

interface LoadingProps {
  message?: string;
  size?: "small" | "medium" | "large";
}

export const Loading: React.FC<LoadingProps> = ({
  message,
  size = "medium",
}) => {
  const { t } = useTranslation();
  const defaultMessage = message || t("common.loading");
  return (
    <div className="cocursor-loading">
      {defaultMessage}
    </div>
  );
};
