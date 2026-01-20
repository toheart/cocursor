/**
 * 索引操作卡片
 * 显示索引统计，提供全量索引和清除数据操作
 */

import React, { useState, useEffect, useCallback, useRef } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { useToast } from "../../hooks";

interface IndexStats {
  totalIndexed: number;
  lastFullScan?: number;
}

interface IndexProgress {
  status: string;
  totalFiles: number;
  processedFiles: number;
  percentage: number;
}

interface IndexCardProps {
  // Qdrant 是否运行中
  qdrantRunning: boolean;
  // Embedding 是否已配置
  embeddingConfigured: boolean;
}

export const IndexCard: React.FC<IndexCardProps> = ({
  qdrantRunning,
  embeddingConfigured,
}) => {
  const { t } = useTranslation();
  const { showToast } = useToast();

  // 统计数据
  const [stats, setStats] = useState<IndexStats>({ totalIndexed: 0 });
  
  // 索引进度
  const [indexing, setIndexing] = useState(false);
  const [progress, setProgress] = useState<IndexProgress | null>(null);
  
  // 轮询定时器
  const pollTimerRef = useRef<number | null>(null);

  // 是否可以操作
  const canOperate = qdrantRunning && embeddingConfigured;

  // 获取统计信息
  const fetchStats = useCallback(async () => {
    try {
      const response = (await apiService.getRAGStats()) as {
        total_indexed?: number;
        last_full_scan?: number;
      };
      if (response) {
        setStats({
          totalIndexed: response.total_indexed || 0,
          lastFullScan: response.last_full_scan,
        });
      }
    } catch (error) {
      console.error("Failed to fetch RAG stats:", error);
    }
  }, []);

  // 获取索引进度
  const fetchProgress = useCallback(async () => {
    try {
      const response = (await apiService.getIndexProgress()) as {
        running: boolean;
        progress?: {
          status: string;
          total_files: number;
          processed_files: number;
          percentage: number;
        };
      };
      
      if (response.running && response.progress) {
        setIndexing(true);
        setProgress({
          status: response.progress.status,
          totalFiles: response.progress.total_files,
          processedFiles: response.progress.processed_files,
          percentage: response.progress.percentage,
        });
        return true; // 需要继续轮询
      } else {
        setIndexing(false);
        setProgress(null);
        // 索引完成后刷新统计
        fetchStats();
        return false;
      }
    } catch (error) {
      console.error("Failed to fetch index progress:", error);
      return false;
    }
  }, [fetchStats]);

  // 轮询进度
  const startPolling = useCallback(() => {
    const poll = async () => {
      const shouldContinue = await fetchProgress();
      if (shouldContinue) {
        pollTimerRef.current = window.setTimeout(poll, 1000);
      }
    };
    poll();
  }, [fetchProgress]);

  // 停止轮询
  const stopPolling = useCallback(() => {
    if (pollTimerRef.current) {
      clearTimeout(pollTimerRef.current);
      pollTimerRef.current = null;
    }
  }, []);

  // 初始化
  useEffect(() => {
    fetchStats();
    // 检查是否正在索引
    fetchProgress().then((isRunning) => {
      if (isRunning) {
        startPolling();
      }
    });
    return () => stopPolling();
  }, [fetchStats, fetchProgress, startPolling, stopPolling]);

  // 触发全量索引
  const handleFullIndex = async () => {
    if (!canOperate) {
      showToast(t("rag.index.requirementsNotMet"), "error");
      return;
    }

    try {
      await apiService.triggerFullIndex();
      showToast(t("rag.index.started"), "success");
      setIndexing(true);
      startPolling();
    } catch (error) {
      showToast(t("rag.index.failed"), "error");
    }
  };

  // 清除数据
  const handleClearData = async () => {
    if (!window.confirm(t("rag.index.clearConfirm"))) {
      return;
    }

    try {
      await apiService.clearAllData();
      showToast(t("rag.index.cleared"), "success");
      setStats({ totalIndexed: 0 });
    } catch (error) {
      showToast(t("rag.index.clearFailed"), "error");
    }
  };

  // 格式化时间
  const formatTime = (timestamp: number) => {
    if (!timestamp) return "-";
    return new Date(timestamp * 1000).toLocaleString();
  };

  // 刷新加载状态
  const [refreshing, setRefreshing] = useState(false);

  // 手动刷新统计
  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await fetchStats();
    } finally {
      setRefreshing(false);
    }
  };

  return (
    <div className="rag-card rag-index-card">
      <div className="rag-card-header">
        <div className="rag-card-title">
          <span>📊</span>
          <span>{t("rag.index.title")}</span>
        </div>
        <button
          className="rag-refresh-btn"
          onClick={handleRefresh}
          disabled={refreshing || indexing}
          title={t("common.refresh")}
        >
          <span className={refreshing ? "rag-spin" : ""}>↻</span>
        </button>
      </div>

      <div className="rag-card-content">
        {/* 统计数字 */}
        <div className="rag-index-stats">
          <div className="rag-stat-number">{stats.totalIndexed.toLocaleString()}</div>
          <div className="rag-stat-label">{t("rag.index.totalIndexed")}</div>
          {stats.lastFullScan && stats.lastFullScan > 0 && (
            <div className="rag-stat-time">
              {t("rag.index.lastScan")}: {formatTime(stats.lastFullScan)}
            </div>
          )}
        </div>

        {/* 索引进度 */}
        {indexing && progress && (
          <div className="rag-index-progress">
            <div className="rag-progress-bar">
              <div
                className="rag-progress-fill"
                style={{ width: `${progress.percentage}%` }}
              />
            </div>
            <div className="rag-progress-text">
              <span>{progress.percentage}%</span>
              <span>
                {progress.processedFiles} / {progress.totalFiles}
              </span>
            </div>
          </div>
        )}

        {/* 提示信息 */}
        {!canOperate && (
          <div className="rag-index-hint">
            {!qdrantRunning && <span>⚠️ {t("rag.index.qdrantRequired")}</span>}
            {!embeddingConfigured && <span>⚠️ {t("rag.index.embeddingRequired")}</span>}
          </div>
        )}

        {/* 操作按钮 */}
        <div className="rag-card-actions">
          <button
            className="rag-btn rag-btn-primary"
            onClick={handleFullIndex}
            disabled={!canOperate || indexing}
          >
            {indexing ? t("rag.index.indexing") : t("rag.index.fullIndex")}
          </button>
          <button
            className="rag-btn rag-btn-danger"
            onClick={handleClearData}
            disabled={indexing}
          >
            {t("rag.index.clear")}
          </button>
        </div>
      </div>
    </div>
  );
};
