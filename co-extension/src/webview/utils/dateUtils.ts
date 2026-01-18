/**
 * 日期工具函数
 */

export interface WeekRange {
  start: string;
  end: string;
}

export type WeekOption = "thisWeek" | "lastWeek" | "twoWeeksAgo";

/**
 * 获取周的起止日期
 */
export function getWeekRange(weeksAgo: number): WeekRange {
  const today = new Date();
  const dayOfWeek = today.getDay(); // 0 = 周日, 1 = 周一, ...
  const mondayOffset = dayOfWeek === 0 ? -6 : 1 - dayOfWeek; // 调整到周一

  const targetDate = new Date(today);
  targetDate.setDate(today.getDate() + mondayOffset - weeksAgo * 7);

  const weekStart = new Date(targetDate);
  const weekEnd = new Date(targetDate);
  weekEnd.setDate(weekStart.getDate() + 6);

  return {
    start: weekStart.toISOString().split("T")[0],
    end: weekEnd.toISOString().split("T")[0],
  };
}

/**
 * 获取周选项对应的日期范围
 */
export function getWeekRangeByOption(option: WeekOption): WeekRange {
  switch (option) {
    case "thisWeek":
      return getWeekRange(0);
    case "lastWeek":
      return getWeekRange(1);
    case "twoWeeksAgo":
      return getWeekRange(2);
  }
}
