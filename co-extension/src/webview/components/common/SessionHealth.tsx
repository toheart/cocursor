import React from "react";
import { SessionHealth as SessionHealthType } from "../../services/api";
import { getEntropyColor, getEntropyStatusText } from "../../utils/helpers";

interface SessionHealthProps {
  health: SessionHealthType;
  className?: string;
}

export const SessionHealth: React.FC<SessionHealthProps> = ({
  health,
  className = ""
}) => {
  const entropyColor = getEntropyColor(health.entropy);
  const statusText = getEntropyStatusText(health.status);

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
            style={{
              color: entropyColor,
              fontWeight: "bold"
            }}
          >
            {health.entropy.toFixed(2)}
          </span>
        </div>
        <div className="cocursor-entropy-status">
          <span className="cocursor-entropy-label">状态:</span>
          <span
            className="cocursor-entropy-status-text"
            style={{
              color: entropyColor
            }}
          >
            {statusText}
          </span>
        </div>
        <div className="cocursor-entropy-progress">
          <div
            className="cocursor-entropy-progress-bar"
            style={{
              width: `${Math.min((health.entropy / 100) * 100, 100)}%`,
              backgroundColor: entropyColor,
              animation:
                health.entropy >= 70
                  ? "pulse 1s ease-in-out infinite"
                  : "none"
            }}
          />
        </div>
        {health.warning && (
          <div className="cocursor-entropy-warning">
            ⚠️ {health.warning}
          </div>
        )}
      </div>
    </div>
  );
};
