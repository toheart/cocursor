package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, httpCode int, errCode int, message string) {
	c.JSON(httpCode, ErrorResponse{
		Code:    errCode,
		Message: message,
	})
}

// ErrorWithDetail 带详情的错误响应
func ErrorWithDetail(c *gin.Context, httpCode int, errCode int, message, detail string) {
	c.JSON(httpCode, ErrorResponse{
		Code:    errCode,
		Message: message,
		Detail:  detail,
	})
}

// PageInfo 分页信息
type PageInfo struct {
	Page     int `json:"page"`      // 当前页码（从 1 开始）
	PageSize int `json:"pageSize"`  // 每页条数
	Total    int `json:"total"`     // 总条数
	Pages    int `json:"pages"`     // 总页数
}

// ResponseWithPage 带分页的响应结构
type ResponseWithPage struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Page    *PageInfo   `json:"page,omitempty"`
}

// SuccessWithPage 成功响应（带分页）
func SuccessWithPage(c *gin.Context, data interface{}, page, pageSize, total int) {
	pages := (total + pageSize - 1) / pageSize // 向上取整
	c.JSON(http.StatusOK, ResponseWithPage{
		Code:    0,
		Message: "success",
		Data:    data,
		Page: &PageInfo{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
			Pages:    pages,
		},
	})
}
