/**
 * 会话健康状态卡片组件
 */

import React, { useMemo } from "react";
import { SessionHealth } from "../../types";
import { getEntropyColor, getEntropyStatusText } from "../../utils/healthUtils";

interface SessionHealthCardProps {
  health: SessionHealth;
  className?: string;
}

export const SessionHealthCard: React.FC<SessionHealthCardProps> = ({
  health,
  className = "",
}) => {
  const entropyColor = useMemo(() => getEntropyColor(health.entropy), [health.entropy]);
  const statusText = useMemo(() => getEntropyStatusText(health.status), [health.status]);
  const progressWidth = useMemo(() => Math.min((health.entropy / 100) * 100, 100), [health.entropy]);
  const isDangerous = useMemo(() => health.entropy >= 70, [health.entropy]);

  return (
    <div className={`cocursor-session-health ${className}`}>
      <div className="cocursor-session-health-header">
        <h2>会话健康状态</h2>
      </div>
      <div className="cocursor-entropy-display">
        <div className="cocursor-entropy-value">
          <span className="cocursor-entropy-label">熵值:</span>
          <span
            className="cocursor-entropy-number"
            style={{ color: entropyColor, fontWeight: "bold" }}
          >
            {health.entropy.toFixed(2)}
          </span>
        </div>
        <div className="cocursor-entropy-status">
          <span className="cocursor-entropy-label">状态:</span>
          <span
            className="cocursor-entropy-status-text"
            style={{ color: entropyColor }}
          >
            {statusText}
          </span>
        </div>
        <div className="cocursor-entropy-progress">
          <div
            className="cocursor-entropy-progress-bar"
            style={{
              width: `${progressWidth}%`,
              backgroundColor: entropyColor,
              animation: isDangerous ? "pulse 1s ease-in-out infinite" : "none",
            }}
          />
        </div>
        {health.warning && (
          <div
            className="cocursor-entropy-warning"
            style={{
              color: "var(--vscode-errorForeground)",
              marginTop: "8px",
              padding: "8px",
              backgroundColor: "var(--vscode-inputValidation-errorBackground)",
              borderRadius: "4px",
            }}
          >
            ⚠️ {health.warning}
          </div>
        )}
      </div>
    </div>
  );
};
