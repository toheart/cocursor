/**
 * 确认弹窗组件
 * 替代浏览器原生 confirm()，风格与 VS Code 一致
 */

import React from "react";
import { useTranslation } from "react-i18next";

interface ConfirmDialogProps {
  /** 标题 */
  title: string;
  /** 描述信息 */
  message: string;
  /** 确认按钮文本 */
  confirmText?: string;
  /** 取消按钮文本 */
  cancelText?: string;
  /** 是否为危险操作（红色确认按钮） */
  danger?: boolean;
  /** 加载中 */
  loading?: boolean;
  /** 确认回调 */
  onConfirm: () => void;
  /** 取消回调 */
  onCancel: () => void;
}

export const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  title,
  message,
  confirmText,
  cancelText,
  danger = false,
  loading = false,
  onConfirm,
  onCancel,
}) => {
  const { t } = useTranslation();

  return (
    <div className="ct-dialog-overlay" onClick={onCancel}>
      <div className="ct-dialog" onClick={(e) => e.stopPropagation()}>
        <div className="ct-dialog-header">
          <span
            className={`codicon codicon-${danger ? "warning" : "info"} ct-dialog-icon ${danger ? "danger" : ""}`}
          />
          <h3 className="ct-dialog-title">{title}</h3>
        </div>
        <p className="ct-dialog-message">{message}</p>
        <div className="ct-dialog-actions">
          <button
            className="ct-btn secondary"
            onClick={onCancel}
            disabled={loading}
          >
            {cancelText || t("common.cancel")}
          </button>
          <button
            className={`ct-btn ${danger ? "danger" : "primary"}`}
            onClick={onConfirm}
            disabled={loading}
          >
            {loading && <span className="ct-btn-spinner" />}
            {confirmText || t("common.confirm")}
          </button>
        </div>
      </div>
    </div>
  );
};
