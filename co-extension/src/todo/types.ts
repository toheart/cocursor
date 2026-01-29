// 待办事项类型定义

/**
 * 待办事项
 */
export interface TodoItem {
  id: string;
  content: string;
  completed: boolean;
  createdAt: number; // Unix 毫秒时间戳
  completedAt?: number; // Unix 毫秒时间戳，可选
}

/**
 * 创建待办请求
 */
export interface CreateTodoRequest {
  content: string;
}

/**
 * 更新待办请求
 */
export interface UpdateTodoRequest {
  completed?: boolean;
  content?: string;
}

/**
 * API 响应结构
 */
export interface ApiResponse<T> {
  code: number;
  message: string;
  data?: T;
}
