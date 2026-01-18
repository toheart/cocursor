import * as vscode from "vscode";
import axios from "axios";
import { getCurrentWorkspacePath, normalizePath } from "./workspaceDetector";

interface ProjectReportRequest {
  path: string;
  timestamp: number;
}

interface ProjectReportResponse {
  success: boolean;
  project_name: string;
  project_id: string;
  is_active: boolean;
  message?: string;
}

let lastReportedPath: string | null = null;

/**
 * 检测并上报当前项目
 */
export async function checkAndReportProject(): Promise<void> {
  const currentPath = getCurrentWorkspacePath();

  if (!currentPath) {
    console.warn("CoCursor: 无法获取当前工作区路径");
    return;
  }

  // 规范化路径
  const normalizedPath = normalizePath(currentPath);

  // 首次加载
  if (lastReportedPath === null) {
    await reportCurrentProject(normalizedPath);
    lastReportedPath = normalizedPath;
    return;
  }

  // 路径变化 = 新项目
  if (lastReportedPath !== normalizedPath) {
    console.log(`CoCursor: 检测到工作区变化: ${lastReportedPath} -> ${normalizedPath}`);
    await reportCurrentProject(normalizedPath);
    lastReportedPath = normalizedPath;
    return;
  }

  // 路径相同，无需重复上报
}

/**
 * 上报当前项目到后端
 * @param path 项目路径
 */
async function reportCurrentProject(path: string): Promise<void> {
  try {
    const response = await axios.post<{
      code: number;
      message: string;
      data: ProjectReportResponse;
    }>(
      "http://localhost:19960/api/v1/project/activate",
      {
        path: path,
        timestamp: Date.now(),
      } as ProjectReportRequest,
      {
        timeout: 5000,
        headers: {
          "Content-Type": "application/json",
        },
      }
    );

    // 后端返回格式: { code: 0, message: "success", data: { success: true, ... } }
    if (response.data.code === 0 && response.data.data && response.data.data.success) {
      console.log(
        `CoCursor: 项目上报成功: ${response.data.data.project_name} (${response.data.data.project_id})`
      );
    } else {
      console.warn(`CoCursor: 项目上报失败: ${response.data.data?.message || response.data.message || "未知错误"}`);
    }
  } catch (error) {
    // 静默失败，不阻塞扩展
    const message = error instanceof Error ? error.message : String(error);
    console.log(`CoCursor: 项目上报失败（可能后端未启动）: ${message}`);
  }
}
