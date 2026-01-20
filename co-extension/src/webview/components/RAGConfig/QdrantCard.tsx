/**
 * Qdrant 状态卡片
 * 整合下载/安装/启动/停止功能到一个简洁的卡片中
 */

import React, { useState, useEffect, useCallback, useRef } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { useToast } from "../../hooks";

// Qdrant 状态类型
type QdrantStatus = "not-installed" | "installed" | "running" | "downloading";

// 下载进度信息
interface DownloadProgress {
  percent: number;
  downloaded: number;
  totalSize: number;
}

interface QdrantCardProps {
  // 状态变化回调，通知父组件
  onStatusChange?: (status: QdrantStatus) => void;
}

export const QdrantCard: React.FC<QdrantCardProps> = ({ onStatusChange }) => {
  const { t } = useTranslation();
  const { showToast } = useToast();

  // 状态
  const [status, setStatus] = useState<QdrantStatus>("not-installed");
  const [version, setVersion] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [downloadProgress, setDownloadProgress] = useState<DownloadProgress | null>(null);
  const [downloadError, setDownloadError] = useState<string | null>(null);
  
  // 上传相关状态
  const [showUpload, setShowUpload] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  
  // 轮询定时器
  const pollTimerRef = useRef<number | null>(null);

  // 更新状态并通知父组件
  const updateStatus = useCallback((newStatus: QdrantStatus) => {
    setStatus(newStatus);
    onStatusChange?.(newStatus);
  }, [onStatusChange]);

  // 获取 Qdrant 状态
  const fetchStatus = useCallback(async () => {
    try {
      const response = await apiService.getQdrantStatus();
      // 后端直接返回 JSON 对象，不是 { data: ... } 包装
      if (response && typeof response === "object") {
        const data = response as any;
        
        // 更新版本
        if (data.version) {
          setVersion(data.version);
        }

        // 检查下载状态
        if (data.download_status === "downloading") {
          updateStatus("downloading");
          if (data.download_info) {
            setDownloadProgress({
              percent: data.download_info.percent || 0,
              downloaded: data.download_info.downloaded || 0,
              totalSize: data.download_info.total_size || 0,
            });
          }
          return true; // 返回 true 表示需要继续轮询
        }

        // 更新状态
        if (data.is_running) {
          updateStatus("running");
        } else if (data.binary_exists) {
          updateStatus("installed");
        } else {
          updateStatus("not-installed");
        }
        
        setDownloadProgress(null);
        return false; // 返回 false 表示不需要继续轮询
      }
    } catch (error) {
      console.error("Failed to fetch Qdrant status:", error);
    }
    return false;
  }, [updateStatus]);

  // 轮询下载状态
  const startPolling = useCallback(() => {
    const poll = async () => {
      const shouldContinue = await fetchStatus();
      if (shouldContinue) {
        pollTimerRef.current = window.setTimeout(poll, 1000);
      }
    };
    poll();
  }, [fetchStatus]);

  // 停止轮询
  const stopPolling = useCallback(() => {
    if (pollTimerRef.current) {
      clearTimeout(pollTimerRef.current);
      pollTimerRef.current = null;
    }
  }, []);

  // 初始化获取状态
  useEffect(() => {
    fetchStatus().then((shouldPoll) => {
      if (shouldPoll) {
        startPolling();
      }
    });
    return () => stopPolling();
  }, [fetchStatus, startPolling, stopPolling]);

  // 下载 Qdrant
  const handleDownload = async () => {
    setLoading(true);
    setDownloadError(null);
    try {
      const response = await apiService.downloadQdrant() as { success: boolean; error?: string };
      if (response.success) {
        updateStatus("downloading");
        showToast(t("rag.qdrant.downloadStarted"), "success");
        startPolling();
      } else {
        const errorMsg = response.error || t("rag.qdrant.downloadFailed");
        setDownloadError(errorMsg);
        setShowUpload(true); // 下载失败时显示上传选项
        showToast(errorMsg, "error");
      }
    } catch (error) {
      const errorMsg = t("rag.qdrant.downloadFailed");
      setDownloadError(errorMsg);
      setShowUpload(true);
      showToast(errorMsg, "error");
    } finally {
      setLoading(false);
    }
  };

  // 启动 Qdrant
  const handleStart = async () => {
    setLoading(true);
    try {
      await apiService.startQdrant();
      showToast(t("rag.qdrant.startSuccess"), "success");
      updateStatus("running");
    } catch (error) {
      showToast(t("rag.qdrant.startFailed"), "error");
    } finally {
      setLoading(false);
    }
  };

  // 停止 Qdrant
  const handleStop = async () => {
    setLoading(true);
    try {
      await apiService.stopQdrant();
      showToast(t("rag.qdrant.stopSuccess"), "success");
      updateStatus("installed");
    } catch (error) {
      showToast(t("rag.qdrant.stopFailed"), "error");
    } finally {
      setLoading(false);
    }
  };

  // 处理文件上传
  const handleFileUpload = useCallback(async (file: File) => {
    // 验证文件类型
    const validExtensions = [".tar.gz", ".tgz", ".zip"];
    const isValidFile = validExtensions.some((ext) =>
      file.name.toLowerCase().endsWith(ext)
    );

    if (!isValidFile) {
      showToast(t("rag.qdrant.uploadInvalidFormat"), "error");
      return;
    }

    setUploading(true);

    try {
      // 读取文件为 base64
      const fileBase64 = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => {
          const result = reader.result as string;
          // 移除 data URL 前缀
          const base64 = result.split(",")[1];
          resolve(base64);
        };
        reader.onerror = () => reject(new Error("Failed to read file"));
        reader.readAsDataURL(file);
      });

      // 调用上传 API
      const response = (await apiService.uploadQdrantPackage(
        file.name,
        fileBase64
      )) as { success: boolean; error?: string; install_path?: string };

      if (response.success) {
        showToast(t("rag.qdrant.uploadSuccess"), "success");
        setShowUpload(false);
        setDownloadError(null);
        updateStatus("installed");
        // 刷新状态获取版本信息
        fetchStatus();
      } else {
        showToast(response.error || t("rag.qdrant.uploadFailed"), "error");
      }
    } catch (error) {
      showToast(t("rag.qdrant.uploadFailed"), "error");
    } finally {
      setUploading(false);
    }
  }, [t, showToast, updateStatus, fetchStatus]);

  // 处理文件选择
  const handleFileSelect = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (file) {
        handleFileUpload(file);
      }
      // 重置 input 以便可以再次选择同一文件
      e.target.value = "";
    },
    [handleFileUpload]
  );

  // 处理拖拽
  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragOver(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragOver(false);
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setDragOver(false);

      const file = e.dataTransfer.files?.[0];
      if (file) {
        handleFileUpload(file);
      }
    },
    [handleFileUpload]
  );

  // 格式化字节数
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
  };

  // 获取状态颜色
  const getStatusColor = () => {
    switch (status) {
      case "running":
        return "var(--vscode-terminal-ansiGreen)";
      case "installed":
        return "var(--vscode-charts-yellow, #cca700)";
      case "downloading":
        return "var(--vscode-charts-blue, #3794ff)";
      default:
        return "var(--vscode-descriptionForeground)";
    }
  };

  // 获取状态文本
  const getStatusText = () => {
    switch (status) {
      case "running":
        return t("rag.qdrant.statusRunning");
      case "installed":
        return t("rag.qdrant.statusStopped");
      case "downloading":
        return t("rag.qdrant.statusDownloading");
      default:
        return t("rag.qdrant.statusNotInstalled");
    }
  };

  return (
    <div className="rag-card rag-qdrant-card">
      <div className="rag-card-header">
        <div className="rag-card-title">
          <span
            className="rag-status-dot"
            style={{ backgroundColor: getStatusColor() }}
          />
          <span>Qdrant</span>
          {version && <span className="rag-card-version">{version}</span>}
        </div>
        <span className="rag-card-status">{getStatusText()}</span>
      </div>

      <div className="rag-card-content">
        {/* 下载进度条 */}
        {status === "downloading" && downloadProgress && (
          <div className="rag-download-progress">
            <div className="rag-progress-bar">
              <div
                className="rag-progress-fill"
                style={{ width: `${downloadProgress.percent}%` }}
              />
            </div>
            <div className="rag-progress-text">
              <span>{downloadProgress.percent.toFixed(1)}%</span>
              <span>
                {formatBytes(downloadProgress.downloaded)} /{" "}
                {formatBytes(downloadProgress.totalSize)}
              </span>
            </div>
          </div>
        )}

        {/* 下载错误提示 */}
        {downloadError && status === "not-installed" && (
          <div className="rag-error-hint">
            <span>❌ {downloadError}</span>
          </div>
        )}

        {/* 操作按钮 */}
        <div className="rag-card-actions">
          {status === "not-installed" && (
            <>
              <button
                className="rag-btn rag-btn-primary"
                onClick={handleDownload}
                disabled={loading || uploading}
              >
                {loading ? t("common.loading") : t("rag.qdrant.download")}
              </button>
              <button
                className="rag-btn rag-btn-secondary"
                onClick={() => setShowUpload(!showUpload)}
                disabled={loading || uploading}
              >
                {t("rag.qdrant.upload")}
              </button>
            </>
          )}

          {status === "installed" && (
            <button
              className="rag-btn rag-btn-primary"
              onClick={handleStart}
              disabled={loading}
            >
              {loading ? t("common.loading") : t("rag.qdrant.start")}
            </button>
          )}

          {status === "running" && (
            <button
              className="rag-btn rag-btn-secondary"
              onClick={handleStop}
              disabled={loading}
            >
              {loading ? t("common.loading") : t("rag.qdrant.stop")}
            </button>
          )}

          {status === "downloading" && (
            <button className="rag-btn" disabled>
              {t("rag.qdrant.downloading")}
            </button>
          )}
        </div>

        {/* 上传区域 */}
        {showUpload && status === "not-installed" && (
          <div className="rag-upload-section">
            <input
              ref={fileInputRef}
              type="file"
              accept=".tar.gz,.tgz,.zip"
              onChange={handleFileSelect}
              style={{ display: "none" }}
            />

            <div
              className={`rag-upload-dropzone ${dragOver ? "drag-over" : ""} ${
                uploading ? "uploading" : ""
              }`}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onDrop={handleDrop}
              onClick={() => !uploading && fileInputRef.current?.click()}
            >
              {uploading ? (
                <>
                  <span className="rag-spinner" />
                  <span>{t("rag.qdrant.uploading")}</span>
                </>
              ) : (
                <>
                  <span className="rag-upload-icon">📦</span>
                  <span>{t("rag.qdrant.uploadHint")}</span>
                  <small>{t("rag.qdrant.uploadFormats")}</small>
                </>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
