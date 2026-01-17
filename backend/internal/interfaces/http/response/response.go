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
