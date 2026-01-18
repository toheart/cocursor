/**
 * 消息工具函数
 */

import { Message, CodeBlock, ToolCall } from "../types";

const AI_MERGE_TIME_THRESHOLD = 30000; // 30秒内的AI消息合并

/**
 * 合并连续的 AI 消息
 */
export function mergeAIMessages(messages: Message[]): Message[] {
  if (!messages || messages.length === 0) return messages;

  const merged: Message[] = [];
  let currentAIGroup: Message[] = [];

  for (let i = 0; i < messages.length; i++) {
    const msg = messages[i];

    if (msg.type === "ai") {
      // 检查是否应该与上一个 AI 消息合并
      if (currentAIGroup.length > 0) {
        const lastMsg = currentAIGroup[currentAIGroup.length - 1];
        const timeDiff = msg.timestamp - lastMsg.timestamp;

        if (timeDiff <= AI_MERGE_TIME_THRESHOLD) {
          // 合并到当前组
          currentAIGroup.push(msg);
          continue;
        } else {
          // 时间间隔太长，先保存当前组
          merged.push(mergeAIGroup(currentAIGroup));
          currentAIGroup = [msg];
        }
      } else {
        // 开始新的 AI 消息组
        currentAIGroup = [msg];
      }
    } else {
      // 用户消息，先保存当前的 AI 组
      if (currentAIGroup.length > 0) {
        merged.push(mergeAIGroup(currentAIGroup));
        currentAIGroup = [];
      }
      merged.push(msg);
    }
  }

  // 保存最后的 AI 组
  if (currentAIGroup.length > 0) {
    merged.push(mergeAIGroup(currentAIGroup));
  }

  return merged;
}

/**
 * 合并 AI 消息组
 */
function mergeAIGroup(group: Message[]): Message {
  if (group.length === 1) return group[0];

  // 合并文本
  const texts = group.map(m => m.text).filter(t => t.trim());
  const mergedText = texts.join("\n\n");

  // 合并代码块
  const allCodeBlocks: CodeBlock[] = [];
  group.forEach(m => {
    if (m.code_blocks) {
      allCodeBlocks.push(...m.code_blocks);
    }
  });

  // 合并工具调用
  const allTools: ToolCall[] = [];
  group.forEach(m => {
    if (m.tools) {
      allTools.push(...m.tools);
    }
  });

  // 合并文件引用
  const allFiles: string[] = [];
  group.forEach(m => {
    if (m.files) {
      allFiles.push(...m.files);
    }
  });

  // 使用第一条消息的时间戳
  return {
    type: "ai",
    text: mergedText,
    timestamp: group[0].timestamp,
    code_blocks: allCodeBlocks.length > 0 ? allCodeBlocks : undefined,
    tools: allTools.length > 0 ? allTools : undefined,
    files: allFiles.length > 0 ? Array.from(new Set(allFiles)) : undefined, // 去重
  };
}
