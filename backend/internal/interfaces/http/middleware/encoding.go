package middleware

import (
	"bytes"
	"io"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// EnsureUTF8Body 确保请求体是 UTF-8 编码的中间件
// 在 Windows 下使用 curl 时，可能会以 GBK 编码发送中文内容
// 此中间件检测并转换非 UTF-8 编码的请求体
func EnsureUTF8Body() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只处理有请求体的请求
		if c.Request.Body == nil || c.Request.ContentLength == 0 {
			c.Next()
			return
		}

		// 读取原始请求体
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}
		c.Request.Body.Close()

		// 如果是空内容，直接恢复
		if len(bodyBytes) == 0 {
			c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			c.Next()
			return
		}

		// 检查是否是有效的 UTF-8
		if utf8.Valid(bodyBytes) {
			// 已经是有效的 UTF-8，直接恢复请求体
			c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			c.Next()
			return
		}

		// 尝试从 GBK 转换为 UTF-8
		// Windows 中文系统默认使用 GBK (代码页 936)
		utf8Bytes, err := convertGBKToUTF8(bodyBytes)
		if err != nil {
			// 转换失败，使用原始数据
			c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			c.Next()
			return
		}

		// 验证转换后的数据是否是有效的 UTF-8
		if utf8.Valid(utf8Bytes) {
			c.Request.Body = io.NopCloser(bytes.NewReader(utf8Bytes))
			// 更新 Content-Length
			c.Request.ContentLength = int64(len(utf8Bytes))
		} else {
			// 转换后仍然无效，使用原始数据
			c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		c.Next()
	}
}

// convertGBKToUTF8 将 GBK 编码的字节转换为 UTF-8
func convertGBKToUTF8(gbkBytes []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(gbkBytes), simplifiedchinese.GBK.NewDecoder())
	return io.ReadAll(reader)
}
