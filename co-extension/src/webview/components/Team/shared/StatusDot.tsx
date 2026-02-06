/**
 * 连接/在线状态指示器
 */

import React from "react";

interface StatusDotProps {
  /** 是否在线/连接 */
  online: boolean;
  /** 标签文本 */
  label?: string;
  /** 大小 */
  size?: "small" | "medium";
}

export const StatusDot: React.FC<StatusDotProps> = ({
  online,
  label,
  size = "small",
}) => {
  return (
    <span className={`ct-status-dot-wrapper ${size}`}>
      <span className={`ct-status-dot ${online ? "online" : "offline"}`} />
      {label && <span className="ct-status-dot-label">{label}</span>}
    </span>
  );
};
