/**
 * 插件工具函数
 */

import { Plugin, UsageInstruction } from "../types";

/**
 * 生成插件的使用说明
 */
export function generateUsageInstructions(plugin: Plugin): UsageInstruction[] {
  const instructions: UsageInstruction[] = [];

  if (plugin.skill) {
    instructions.push({
      type: "Skill",
      title: "Skill 组件",
      description: `此插件包含 Skill: ${plugin.skill.skill_name}。安装后，该 Skill 将自动添加到项目的 AGENTS.md 文件中，可在对话中使用。`,
    });
  }

  if (plugin.mcp) {
    instructions.push({
      type: "MCP",
      title: "MCP 组件",
      description: `此插件包含 MCP 服务器: ${plugin.mcp.server_name}。安装后，MCP 配置将添加到 ~/.cursor/mcp.json 中，需要重启 Cursor 才能生效。`,
    });
  }

  if (plugin.command && plugin.command.commands && plugin.command.commands.length > 0) {
    const commandNames = plugin.command.commands
      .map(cmd => `/${cmd.command_id}`)
      .join("、");
    instructions.push({
      type: "Command",
      title: "Command 组件",
      description: `此插件包含命令: ${commandNames}。安装后，可在 Cursor 中使用此命令。`,
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
