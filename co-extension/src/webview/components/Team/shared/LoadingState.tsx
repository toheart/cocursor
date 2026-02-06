/**
 * 统一加载状态组件
 */

import React from "react";
import { useTranslation } from "react-i18next";

interface LoadingStateProps {
  /** 提示文本 */
  text?: string;
  /** 是否小尺寸（内联使用） */
  compact?: boolean;
}

export const LoadingState: React.FC<LoadingStateProps> = ({
  text,
  compact = false,
}) => {
  const { t } = useTranslation();

  return (
    <div className={`ct-loading-state ${compact ? "compact" : ""}`}>
      <div className="ct-loading-spinner" />
      {!compact && (
        <span className="ct-loading-text">{text || t("common.loading")}</span>
      )}
    </div>
  );
};
