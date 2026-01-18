/**
 * 可复用卡片组件
 */

import React from "react";

interface CardProps {
  children: React.ReactNode;
  className?: string;
  hoverable?: boolean;
  onClick?: () => void;
}

export const Card: React.FC<CardProps> = ({
  children,
  className = "",
  hoverable = false,
  onClick,
}) => {
  return (
    <div
      className={`cocursor-card ${className}`}
      onClick={onClick}
      style={{
        cursor: onClick ? "pointer" : "default",
        transition: hoverable ? "all 0.3s ease" : "none",
      }}
      onMouseEnter={hoverable ? (e) => {
        e.currentTarget.style.transform = "translateY(-2px)";
        e.currentTarget.style.boxShadow = "0 8px 20px rgba(0, 0, 0, 0.2)";
      } : undefined}
      onMouseLeave={hoverable ? (e) => {
        e.currentTarget.style.transform = "translateY(0)";
        e.currentTarget.style.boxShadow = "0 4px 12px rgba(0, 0, 0, 0.15)";
      } : undefined}
    >
      {children}
    </div>
  );
};
