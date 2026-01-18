import React from "react";

interface EmptyProps {
  message: string;
  description?: string;
  className?: string;
}

export const Empty: React.FC<EmptyProps> = ({
  message,
  description,
  className = ""
}) => {
  return (
    <div className={`cocursor-empty ${className}`}>
      <p>{message}</p>
      {description && <span>{description}</span>}
    </div>
  );
};
