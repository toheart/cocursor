package handler

import (
	"net/http"
	"time"

	"github.com/cocursor/backend/internal/domain/todo"
	"github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TodoHandler 待办事项处理器
type TodoHandler struct {
	repo storage.TodoRepository
}

// NewTodoHandler 创建待办事项处理器
func NewTodoHandler(repo storage.TodoRepository) *TodoHandler {
	return &TodoHandler{repo: repo}
}

// TodoDTO 待办事项 DTO
type TodoDTO struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	Completed   bool   `json:"completed"`
	CreatedAt   int64  `json:"createdAt"`   // Unix 毫秒时间戳
	CompletedAt *int64 `json:"completedAt"` // Unix 毫秒时间戳，可选
}

// CreateTodoRequest 创建待办请求
type CreateTodoRequest struct {
	Content string `json:"content" binding:"required"`
}

// UpdateTodoRequest 更新待办请求
type UpdateTodoRequest struct {
	Completed *bool   `json:"completed"`
	Content   *string `json:"content"`
}

// toDTO 将领域模型转换为 DTO
func toDTO(item *todo.TodoItem) *TodoDTO {
	dto := &TodoDTO{
		ID:        item.ID,
		Content:   item.Content,
		Completed: item.Completed,
		CreatedAt: item.CreatedAt.UnixMilli(),
	}
	if item.CompletedAt != nil {
		ts := item.CompletedAt.UnixMilli()
		dto.CompletedAt = &ts
	}
	return dto
}

// List 获取待办列表
// @Summary 获取待办列表
// @Tags 待办
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /todos [get]
func (h *TodoHandler) List(c *gin.Context) {
	items, err := h.repo.FindAll()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 800001, "获取待办列表失败")
		return
	}

	dtos := make([]*TodoDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, toDTO(item))
	}

	response.Success(c, dtos)
}

// Create 创建待办
// @Summary 创建待办
// @Tags 待办
// @Accept json
// @Produce json
// @Param body body CreateTodoRequest true "待办内容"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.ErrorResponse
// @Router /todos [post]
func (h *TodoHandler) Create(c *gin.Context) {
	var req CreateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 100001, "参数错误")
		return
	}

	item := &todo.TodoItem{
		ID:        uuid.New().String(),
		Content:   req.Content,
		Completed: false,
		CreatedAt: time.Now(),
	}

	if err := h.repo.Save(item); err != nil {
		response.Error(c, http.StatusInternalServerError, 800002, "创建待办失败")
		return
	}

	response.Success(c, toDTO(item))
}

// Update 更新待办
// @Summary 更新待办
// @Tags 待办
// @Accept json
// @Produce json
// @Param id path string true "待办ID"
// @Param body body UpdateTodoRequest true "更新内容"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /todos/{id} [patch]
func (h *TodoHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, 100001, "缺少待办ID")
		return
	}

	var req UpdateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 100001, "参数错误")
		return
	}

	// 查找现有待办
	item, err := h.repo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 800003, "查询待办失败")
		return
	}
	if item == nil {
		response.Error(c, http.StatusNotFound, 800004, "待办不存在")
		return
	}

	// 更新字段
	if req.Content != nil {
		item.Content = *req.Content
	}
	if req.Completed != nil {
		if *req.Completed {
			item.MarkComplete()
		} else {
			item.MarkIncomplete()
		}
	}

	if err := h.repo.Save(item); err != nil {
		response.Error(c, http.StatusInternalServerError, 800005, "更新待办失败")
		return
	}

	response.Success(c, toDTO(item))
}

// Delete 删除待办
// @Summary 删除待办
// @Tags 待办
// @Accept json
// @Produce json
// @Param id path string true "待办ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.ErrorResponse
// @Router /todos/{id} [delete]
func (h *TodoHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, 100001, "缺少待办ID")
		return
	}

	if err := h.repo.Delete(id); err != nil {
		response.Error(c, http.StatusInternalServerError, 800006, "删除待办失败")
		return
	}

	response.Success(c, nil)
}

// DeleteCompleted 清除所有已完成待办
// @Summary 清除已完成待办
// @Tags 待办
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /todos/completed [delete]
func (h *TodoHandler) DeleteCompleted(c *gin.Context) {
	count, err := h.repo.DeleteCompleted()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 800007, "清除已完成待办失败")
		return
	}

	response.Success(c, gin.H{"deleted": count})
}
