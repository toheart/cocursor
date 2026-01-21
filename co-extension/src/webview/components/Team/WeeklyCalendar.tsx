/**
 * 周报日历组件
 * 显示 7 天 × N 成员的网格布局
 */

import React from "react";
import { useTranslation } from "react-i18next";
import { TeamDayColumn, MemberDayCell } from "../../types";

interface WeeklyCalendarProps {
  calendar: TeamDayColumn[];
  weekStart: string;
  onCellClick: (memberId: string, date: string) => void;
}

// 星期几名称
const WEEKDAY_NAMES = ["", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

// 格式化日期显示（月/日）
function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return `${d.getMonth() + 1}/${d.getDate()}`;
}

// 判断日期是否为今天
function isToday(dateStr: string): boolean {
  const today = new Date().toISOString().split("T")[0];
  return dateStr === today;
}

// 获取活跃度颜色类
function getActivityClass(level: number): string {
  switch (level) {
    case 0: return "activity-none";
    case 1: return "activity-low";
    case 2: return "activity-medium";
    case 3: return "activity-high";
    case 4: return "activity-very-high";
    default: return "activity-none";
  }
}

export const WeeklyCalendar: React.FC<WeeklyCalendarProps> = ({
  calendar,
  weekStart,
  onCellClick,
}) => {
  const { t } = useTranslation();

  // 获取所有成员列表（从第一天的数据中提取）
  const members = React.useMemo(() => {
    if (!calendar || calendar.length === 0) return [];
    const firstDay = calendar[0];
    return firstDay.members.map(m => ({
      id: m.member_id,
      name: m.member_name,
    }));
  }, [calendar]);

  // 构建成员ID到成员数据的映射
  const getMemberCell = React.useCallback((day: TeamDayColumn, memberId: string): MemberDayCell | undefined => {
    return day.members.find(m => m.member_id === memberId);
  }, []);

  if (!calendar || calendar.length === 0 || members.length === 0) {
    return (
      <div className="cocursor-weekly-calendar-empty">
        <span className="cocursor-empty-icon">📊</span>
        <span>{t("weeklyReport.noData")}</span>
      </div>
    );
  }

  return (
    <div className="cocursor-weekly-calendar">
      {/* 表头：日期行 */}
      <div className="cocursor-weekly-calendar-header">
        <div className="cocursor-weekly-calendar-header-cell corner">
          {t("weeklyReport.member")}
        </div>
        {calendar.map((day) => (
          <div
            key={day.date}
            className={`cocursor-weekly-calendar-header-cell ${isToday(day.date) ? "today" : ""}`}
          >
            <span className="cocursor-weekly-calendar-weekday">
              {t(`weeklyReport.weekday.${WEEKDAY_NAMES[day.day_of_week].toLowerCase()}`)}
            </span>
            <span className="cocursor-weekly-calendar-date">{formatDate(day.date)}</span>
          </div>
        ))}
      </div>

      {/* 表体：成员行 */}
      <div className="cocursor-weekly-calendar-body">
        {members.map((member) => (
          <div key={member.id} className="cocursor-weekly-calendar-row">
            {/* 成员名列 */}
            <div className="cocursor-weekly-calendar-member-cell">
              <div className="cocursor-weekly-calendar-member-avatar">
                {member.name.charAt(0).toUpperCase()}
              </div>
              <span className="cocursor-weekly-calendar-member-name" title={member.name}>
                {member.name}
              </span>
            </div>

            {/* 每日数据格子 */}
            {calendar.map((day) => {
              const cell = getMemberCell(day, member.id);
              if (!cell) {
                return (
                  <div
                    key={day.date}
                    className="cocursor-weekly-calendar-cell activity-none"
                  />
                );
              }

              return (
                <div
                  key={day.date}
                  className={`cocursor-weekly-calendar-cell ${getActivityClass(cell.activity_level)} ${
                    isToday(day.date) ? "today" : ""
                  }`}
                  onClick={() => onCellClick(member.id, day.date)}
                  title={`${member.name} - ${day.date}`}
                >
                  {/* 提交数 */}
                  {cell.commits > 0 && (
                    <span className="cocursor-weekly-calendar-commits">
                      {cell.commits}
                    </span>
                  )}
                  
                  {/* 状态图标 */}
                  <div className="cocursor-weekly-calendar-icons">
                    {cell.has_report && (
                      <span className="cocursor-weekly-calendar-icon report" title={t("weeklyReport.hasReport")}>
                        📝
                      </span>
                    )}
                    {!cell.is_online && (
                      <span className="cocursor-weekly-calendar-icon offline" title={t("weeklyReport.offline")}>
                        ⚫
                      </span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        ))}
      </div>

      {/* 图例 */}
      <div className="cocursor-weekly-calendar-legend">
        <span className="cocursor-weekly-calendar-legend-label">{t("weeklyReport.less")}</span>
        <div className="cocursor-weekly-calendar-legend-item activity-none" />
        <div className="cocursor-weekly-calendar-legend-item activity-low" />
        <div className="cocursor-weekly-calendar-legend-item activity-medium" />
        <div className="cocursor-weekly-calendar-legend-item activity-high" />
        <div className="cocursor-weekly-calendar-legend-item activity-very-high" />
        <span className="cocursor-weekly-calendar-legend-label">{t("weeklyReport.more")}</span>
      </div>
    </div>
  );
};

export default WeeklyCalendar;
