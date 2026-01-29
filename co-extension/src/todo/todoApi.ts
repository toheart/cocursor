// 待办事项 API 服务

import axios from "axios";
import { TodoItem, CreateTodoRequest, UpdateTodoRequest, ApiResponse } from "./types";

const API_BASE = "http://localhost:19960/api/v1";

/**
 * 待办事项 API 服务
 */
export class TodoApi {
  /**
   * 获取所有待办事项
   */
  async list(): Promise<TodoItem[]> {
    try {
      const response = await axios.get<ApiResponse<TodoItem[]>>(`${API_BASE}/todos`);
      return response.data.data || [];
    } catch (error) {
      console.error("Failed to fetch todos:", error);
      return [];
    }
  }

  /**
   * 创建待办事项
   */
  async create(content: string): Promise<TodoItem | null> {
    try {
      const request: CreateTodoRequest = { content };
      const response = await axios.post<ApiResponse<TodoItem>>(`${API_BASE}/todos`, request);
      return response.data.data || null;
    } catch (error) {
      console.error("Failed to create todo:", error);
      return null;
    }
  }

  /**
   * 更新待办事项
   */
  async update(id: string, updates: UpdateTodoRequest): Promise<TodoItem | null> {
    try {
      const response = await axios.patch<ApiResponse<TodoItem>>(`${API_BASE}/todos/${id}`, updates);
      return response.data.data || null;
    } catch (error) {
      console.error("Failed to update todo:", error);
      return null;
    }
  }

  /**
   * 切换待办完成状态
   */
  async toggle(item: TodoItem): Promise<TodoItem | null> {
    return this.update(item.id, { completed: !item.completed });
  }

  /**
   * 删除待办事项
   */
  async delete(id: string): Promise<boolean> {
    try {
      await axios.delete(`${API_BASE}/todos/${id}`);
      return true;
    } catch (error) {
      console.error("Failed to delete todo:", error);
      return false;
    }
  }

  /**
   * 清除所有已完成的待办事项
   */
  async deleteCompleted(): Promise<number> {
    try {
      const response = await axios.delete<ApiResponse<{ deleted: number }>>(`${API_BASE}/todos/completed`);
      return response.data.data?.deleted || 0;
    } catch (error) {
      console.error("Failed to delete completed todos:", error);
      return 0;
    }
  }
}

// 导出单例
export const todoApi = new TodoApi();
