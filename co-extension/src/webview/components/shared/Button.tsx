/**
 * 可复用按钮组件
 */

import React from "react";

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary" | "danger";
  size?: "small" | "medium" | "large";
  loading?: boolean;
  icon?: React.ReactNode;
}

export const Button: React.FC<ButtonProps> = ({
  variant = "primary",
  size = "medium",
  loading = false,
  icon,
  children,
  className = "",
  disabled,
  ...props
}) => {
  const baseStyle: React.CSSProperties = {
    backgroundColor: variant === "primary" 
      ? "var(--vscode-button-background)" 
      : "var(--vscode-button-secondaryBackground)",
    color: variant === "primary"
      ? "var(--vscode-button-foreground)"
      : "var(--vscode-button-secondaryForeground)",
    border: variant === "primary" ? "none" : "1px solid var(--vscode-button-border)",
    padding: size === "small" ? "4px 8px" : size === "large" ? "12px 24px" : "8px 16px",
    borderRadius: "4px",
    cursor: disabled || loading ? "not-allowed" : "pointer",
    fontSize: size === "small" ? "12px" : "13px",
    fontWeight: 500,
    display: "flex",
    alignItems: "center",
    gap: "6px",
    opacity: disabled || loading ? 0.5 : 1,
    transition: "all 0.2s ease",
  };

  return (
    <button
      style={baseStyle}
      className={className}
      disabled={disabled || loading}
      onMouseEnter={(e) => {
        if (!disabled && !loading) {
          e.currentTarget.style.backgroundColor = "var(--vscode-button-hoverBackground)";
        }
      }}
      onMouseLeave={(e) => {
        if (!disabled && !loading) {
          e.currentTarget.style.backgroundColor = variant === "primary"
            ? "var(--vscode-button-background)"
            : "var(--vscode-button-secondaryBackground)";
        }
      }}
      {...props}
    >
      {loading && <span style={{ 
        display: "inline-block",
        width: "12px",
        height: "12px",
        border: "2px solid currentColor",
        borderTopColor: "transparent",
        borderRadius: "50%",
        animation: "spin 0.8s linear infinite"
      }} />}
      {icon && <span>{icon}</span>}
      {children}
    </button>
  );
};
