import React from "react";

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary" | "danger";
  size?: "small" | "medium" | "large";
  children: React.ReactNode;
}

export const Button: React.FC<ButtonProps> = ({
  variant = "primary",
  size = "medium",
  className = "",
  children,
  ...props
}) => {
  const baseClasses = "cocursor-button";
  const variantClasses = {
    primary: "cocursor-button-primary",
    secondary: "cocursor-button-secondary",
    danger: "cocursor-button-danger"
  };
  const sizeClasses = {
    small: "cocursor-button-small",
    medium: "cocursor-button-medium",
    large: "cocursor-button-large"
  };

  return (
    <button
      className={`${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]} ${className}`}
      {...props}
    >
      {children}
    </button>
  );
};
