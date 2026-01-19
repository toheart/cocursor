/**
 * 密码输入框组件
 * 支持密文/明文切换、复制、清除
 */

import React, { useState, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";

interface PasswordInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  error?: string;
  label?: string;
  showCopy?: boolean;
  showClear?: boolean;
}

export const PasswordInput: React.FC<PasswordInputProps> = ({
  value,
  onChange,
  placeholder,
  disabled = false,
  error,
  label,
  showCopy = true,
  showClear = true,
}) => {
  const { t } = useTranslation();
  const [isVisible, setIsVisible] = useState(false);
  const [showCopyFeedback, setShowCopyFeedback] = useState(false);
  const autoHideTimer = useRef<NodeJS.Timeout | null>(null);

  // 自动隐藏定时器（10秒后切回密文）
  useEffect(() => {
    if (isVisible) {
      autoHideTimer.current = setTimeout(() => {
        setIsVisible(false);
      }, 10000);
    }

    return () => {
      if (autoHideTimer.current) {
        clearTimeout(autoHideTimer.current);
      }
    };
  }, [isVisible]);

  // 切换显示/隐藏
  const toggleVisibility = () => {
    setIsVisible(!isVisible);
  };

  // 复制到剪贴板
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setShowCopyFeedback(true);
      setTimeout(() => setShowCopyFeedback(false), 2000);
    } catch (error) {
      console.error("Failed to copy:", error);
    }
  };

  // 清空输入
  const handleClear = () => {
    onChange("");
  };

  return (
    <div className="cocursor-rag-password-input">
      {label && (
        <label className="cocursor-rag-password-label">{label}</label>
      )}
      <div className="cocursor-rag-password-wrapper">
        <input
          type={isVisible ? "text" : "password"}
          className={`cocursor-rag-password-field ${error ? "error" : ""}`}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          disabled={disabled}
        />
        <div className="cocursor-rag-password-actions">
          {value && showCopy && (
            <button
              type="button"
              className="cocursor-rag-password-action"
              onClick={handleCopy}
              title={showCopyFeedback ? t("common.copied") : t("rag.config.copy")}
            >
              {showCopyFeedback ? "✓" : "📋"}
            </button>
          )}
          {value && showClear && (
            <button
              type="button"
              className="cocursor-rag-password-action"
              onClick={handleClear}
              title={t("rag.config.clear")}
            >
              🗑️
            </button>
          )}
          <button
            type="button"
            className="cocursor-rag-password-action"
            onClick={toggleVisibility}
            title={isVisible ? t("rag.config.hide") : t("rag.config.show")}
          >
            {isVisible ? "🙈" : "👁️"}
          </button>
        </div>
      </div>
      {error && (
        <div className="cocursor-rag-password-error">{error}</div>
      )}
    </div>
  );
};
