/**
 * 插件工具函数
 */

import { TFunction } from "i18next";
import { Plugin, UsageInstruction } from "../types";

/**
 * 生成插件的使用说明（国际化版本）
 * @param plugin 插件对象
 * @param t 国际化翻译函数
 */
export function generateUsageInstructions(plugin: Plugin, t: TFunction): UsageInstruction[] {
  const instructions: UsageInstruction[] = [];

  if (plugin.skill) {
    instructions.push({
      type: "Skill",
      title: t("marketplace.usage.skill.title"),
      description: t("marketplace.usage.skill.description", { name: plugin.skill.skill_name }),
    });
  }

  if (plugin.mcp) {
    instructions.push({
      type: "MCP",
      title: t("marketplace.usage.mcp.title"),
      description: t("marketplace.usage.mcp.description", { name: plugin.mcp.server_name }),
    });
  }

  if (plugin.command && plugin.command.commands && plugin.command.commands.length > 0) {
    const commandNames = plugin.command.commands
      .map(cmd => `/${cmd.command_id}`)
      .join(", ");
    instructions.push({
      type: "Command",
      title: t("marketplace.usage.command.title"),
      description: t("marketplace.usage.command.description", { names: commandNames }),
    });
  }

  return instructions;
}

/**
 * 获取组件图标
 */
export function getComponentIcon(type: UsageInstruction["type"]): string {
  switch (type) {
    case "Skill":
      return "🎯";
    case "MCP":
      return "🔌";
    case "Command":
      return "⚡";
  }
}
