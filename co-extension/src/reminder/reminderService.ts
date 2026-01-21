import * as vscode from "vscode";
import axios from "axios";
import { Logger } from "../utils/logger";

/**
 * 每日总结提醒服务
 * 在配置的时间点提醒用户进行每日总结
 */
export class ReminderService {
  private context: vscode.ExtensionContext;
  private checkInterval: NodeJS.Timeout | null = null;
  private readonly CHECK_INTERVAL_MS = 60000; // 每分钟检查一次
  private readonly SNOOZE_DELAY_MS = 5 * 60 * 1000; // 5分钟后再提醒

  constructor(context: vscode.ExtensionContext) {
    this.context = context;
  }

  /**
   * 启动提醒服务
   */
  start(): void {
    Logger.debug("ReminderService: 启动提醒服务");
    // 启动时立即检查一次
    this.check();
    // 每分钟检查一次
    this.checkInterval = setInterval(() => this.check(), this.CHECK_INTERVAL_MS);
  }

  /**
   * 停止提醒服务
   */
  stop(): void {
    Logger.debug("ReminderService: 停止提醒服务");
    if (this.checkInterval) {
      clearInterval(this.checkInterval);
      this.checkInterval = null;
    }
  }

  /**
   * 检查是否需要触发提醒
   */
  private async check(): Promise<void> {
    const config = vscode.workspace.getConfiguration("cocursor.reminder");
    const enabled = config.get<boolean>("enabled", false);

    if (!enabled) {
      return;
    }

    const now = new Date();
    const dayOfWeek = now.getDay();

    // 周末跳过（周日=0，周六=6）
    if (dayOfWeek === 0 || dayOfWeek === 6) {
      return;
    }

    const currentTime = this.formatTime(now);
    const today = this.formatDate(now);

    // 检查下班前提醒
    const eveningTime = config.get<string>("eveningTime", "17:50");
    if (currentTime === eveningTime) {
      if (!this.hasRemindedToday("evening") && !this.hasSkippedToday("evening")) {
        await this.showEveningReminder(today);
      }
    }

    // 检查次日补充提醒
    const morningTime = config.get<string>("morningTime", "09:00");
    if (currentTime === morningTime) {
      if (!this.hasRemindedToday("morning") && !this.hasSkippedToday("morning")) {
        await this.checkAndShowMorningReminder();
      }
    }
  }

  /**
   * 显示下班前提醒
   */
  private async showEveningReminder(today: string): Promise<void> {
    this.markReminded("evening", today);
    Logger.info("ReminderService: displaying evening reminder");

    const selection = await vscode.window.showInformationMessage(
      "⏰ 今天的工作快结束了，记得在 Cursor Chat 中执行 /daily-summary 进行每日总结！",
      "稍后提醒",
      "今日不再提醒"
    );

    if (selection === "稍后提醒") {
      this.scheduleSnooze("evening", today);
    } else if (selection === "今日不再提醒") {
      this.markSkipped("evening", today);
    }
  }

  /**
   * 检查昨日总结状态并显示提醒
   */
  private async checkAndShowMorningReminder(): Promise<void> {
    const today = this.formatDate(new Date());
    const previousWorkday = this.getPreviousWorkday();

    // 如果没有需要检查的工作日（比如今天是周一，上周五之前没有工作日）
    if (!previousWorkday) {
      return;
    }

    // 检查前一工作日是否已完成总结
    const hasSummary = await this.checkDailySummary(previousWorkday);
    if (hasSummary) {
      Logger.debug(`ReminderService: ${previousWorkday} summary exists, skipping reminder`);
      return;
    }

    this.markReminded("morning", today);
    Logger.info(`ReminderService: displaying morning reminder for ${previousWorkday}`);

    const selection = await vscode.window.showWarningMessage(
      `📝 ${previousWorkday} 的工作总结还未完成，可以在 Cursor Chat 中执行 /daily-summary ${previousWorkday} 补充`,
      "知道了",
      "不再提醒"
    );

    if (selection === "不再提醒") {
      this.markSkipped("morning", today);
    }
  }

  /**
   * 获取前一个工作日的日期
   * 周一返回上周五，其他工作日返回前一天
   */
  private getPreviousWorkday(): string | null {
    const today = new Date();
    const dayOfWeek = today.getDay();

    let daysToSubtract = 1;
    if (dayOfWeek === 1) {
      // 周一，返回上周五
      daysToSubtract = 3;
    } else if (dayOfWeek === 0) {
      // 周日（理论上不会到这里，因为周末不触发）
      daysToSubtract = 2;
    } else if (dayOfWeek === 6) {
      // 周六（理论上不会到这里）
      daysToSubtract = 1;
    }

    const previousDay = new Date(today);
    previousDay.setDate(previousDay.getDate() - daysToSubtract);

    return this.formatDate(previousDay);
  }

  /**
   * 调用后端 API 检查指定日期是否已有总结
   */
  private async checkDailySummary(date: string): Promise<boolean> {
    try {
      const config = vscode.workspace.getConfiguration("cocursor.daemon");
      const port = config.get<number>("port", 19960);
      const response = await axios.get(
        `http://localhost:${port}/api/daily-summary`,
        {
          params: { date },
          timeout: 5000,
        }
      );

      // 检查返回的数据是否有有效的总结内容
      const data = response.data;
      if (data && data.summary && typeof data.summary === "string" && data.summary.trim() !== "") {
        return true;
      }
      return false;
    } catch (error) {
      // API 调用失败时，假设未完成总结
      Logger.debug(`ReminderService: failed to check summary for ${date}: ${error}`);
      return false;
    }
  }

  /**
   * 安排延迟提醒（稍后提醒）
   */
  private scheduleSnooze(type: "evening" | "morning", date: string): void {
    Logger.debug(`ReminderService: scheduling snooze for ${type} reminder in 5 minutes`);
    setTimeout(() => {
      // 清除已提醒标记，允许再次提醒
      this.clearReminded(type, date);
      Logger.debug(`ReminderService: snooze expired for ${type}, reminder can trigger again`);
    }, this.SNOOZE_DELAY_MS);
  }

  // ==================== 状态管理方法 ====================

  /**
   * 检查今天是否已提醒过
   */
  private hasRemindedToday(type: "evening" | "morning"): boolean {
    const today = this.formatDate(new Date());
    const key = `cocursor.reminded_${type}_${today}`;
    return this.context.globalState.get<boolean>(key, false);
  }

  /**
   * 标记今天已提醒
   */
  private markReminded(type: "evening" | "morning", date: string): void {
    const key = `cocursor.reminded_${type}_${date}`;
    this.context.globalState.update(key, true);
  }

  /**
   * 清除已提醒标记
   */
  private clearReminded(type: "evening" | "morning", date: string): void {
    const key = `cocursor.reminded_${type}_${date}`;
    this.context.globalState.update(key, false);
  }

  /**
   * 检查今天是否已跳过
   */
  private hasSkippedToday(type: "evening" | "morning"): boolean {
    const today = this.formatDate(new Date());
    const key = `cocursor.skip_${type}_${today}`;
    return this.context.globalState.get<boolean>(key, false);
  }

  /**
   * 标记今天跳过提醒
   */
  private markSkipped(type: "evening" | "morning", date: string): void {
    const key = `cocursor.skip_${type}_${date}`;
    this.context.globalState.update(key, true);
  }

  // ==================== 工具方法 ====================

  /**
   * 格式化时间为 HH:mm
   */
  private formatTime(date: Date): string {
    const hours = date.getHours().toString().padStart(2, "0");
    const minutes = date.getMinutes().toString().padStart(2, "0");
    return `${hours}:${minutes}`;
  }

  /**
   * 格式化日期为 YYYY-MM-DD
   */
  private formatDate(date: Date): string {
    return date.toISOString().split("T")[0];
  }
}
