import React from "react";
import { useTranslation } from "react-i18next";
import { TimeDistributionSummary } from "../../services/api";

interface TimeDistributionChartProps {
  distribution: TimeDistributionSummary;
}

// 时段配置：图标、翻译key
const TIME_SLOTS = [
  { key: "morning", icon: "🌅", timeRange: "9-12" },
  { key: "afternoon", icon: "🌤️", timeRange: "14-18" },
  { key: "evening", icon: "🌙", timeRange: "19-22" },
  { key: "night", icon: "🌚", timeRange: "22-2" },
] as const;

/**
 * 时间分布图表组件
 * 展示各时段的会话数和时长
 */
export const TimeDistributionChart: React.FC<TimeDistributionChartProps> = ({ distribution }) => {
  const { t } = useTranslation();

  // 检查是否有任何数据
  const hasData = TIME_SLOTS.some(
    (slot) => distribution[slot.key]?.sessions > 0 || distribution[slot.key]?.hours > 0
  );
  if (!hasData) return null;

  // 格式化时长
  const formatHours = (hours: number): string => {
    if (!hours || hours === 0) return "-";
    if (hours < 1) {
      return `${Math.round(hours * 60)}${t("dailyReport.minutesShort")}`;
    }
    return `${hours.toFixed(1)}${t("dailyReport.hoursShort")}`;
  };

  return (
    <div className="cocursor-daily-report-section">
      <h4 className="cocursor-daily-report-section-title">
        <span className="section-icon">⏰</span>
        {t("dailyReport.timeDistribution")}
      </h4>
      <div className="cocursor-time-distribution-grid">
        {TIME_SLOTS.map((slot) => {
          const data = distribution[slot.key];
          const sessions = data?.sessions || 0;
          const hours = data?.hours || 0;
          return (
            <div key={slot.key} className="cocursor-time-slot-card">
              <div className="time-slot-icon">{slot.icon}</div>
              <div className="time-slot-name">{t(`dailyReport.timeSlot.${slot.key}`)}</div>
              <div className="time-slot-range">{slot.timeRange}</div>
              <div className="time-slot-stats">
                <span className="time-slot-sessions">
                  {sessions > 0 ? `${sessions}${t("dailyReport.sessionsShort")}` : "-"}
                </span>
                <span className="time-slot-hours">{formatHours(hours)}</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
