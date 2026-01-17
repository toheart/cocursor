# API 规范

本规范定义 cocursor 项目的 HTTP API 设计标准。

## 版本控制

- 使用 URL 路径版本：`/api/v1/`, `/api/v2/`
- 当前版本：`v1`
- 版本升级策略：破坏性变更时递增主版本号

## 统一响应格式

所有 API 响应使用统一的 JSON 结构。

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

### 分页响应

```json
{
  "code": 0,
  "message": "success",
  "data": [ ... ],
  "page": {
    "page": 1,
    "pageSize": 20,
    "total": 150,
    "pages": 8
  }
}
```

### 错误响应

```json
{
  "code": 100001,
  "message": "参数错误",
  "detail": "conversation id is required"
}
```

## 分页参数

- `page`: 页码，从 **1** 开始，默认 1
- `pageSize`: 每页条数，默认 20，最大 100

## 业务错误码

错误码格式：`XXYYYY`

- `XX` = 模块代码 (10-99)
- `YYYY` = 错误序号 (0001-9999)

| 范围   | 模块     | 示例                                                               |
| ------ | -------- | ------------------------------------------------------------------ |
| 0      | 成功     | 0                                                                  |
| 10XXXX | 通用错误 | 100001 参数错误, 100002 未授权, 100003 禁止访问, 100004 资源不存在 |
| 20XXXX | 对话模块 | 200001 对话不存在, 200002 无权访问对话                             |
| 30XXXX | 团队模块 | 300001 团队不存在, 300002 团队码无效, 300003 已在团队中            |
| 40XXXX | 节点模块 | 400001 节点不存在, 400002 节点离线                                 |
| 50XXXX | 评论模块 | 500001 评论不存在, 500002 无权删除评论                             |
| 60XXXX | 归档模块 | 600001 会话未归档, 600002 归档失败                                 |
| 70XXXX | 统计模块 | 700001 统计数据不可用                                              |

## API 文档（Swagger）

### 概述

- 使用 [swaggo/swag](https://github.com/swaggo/swag) v1.8.12 生成 OpenAPI 3.0 文档
- 文档端点：`GET /swagger/*any`（Swagger UI）
- 所有 Handler 方法**必须**添加 Swagger 注释

### 生成命令

```bash
# 安装 swag 工具（首次）
go install github.com/swaggo/swag/cmd/swag@v1.8.12

# 生成文档
cd daemon
swag init -g cmd/cocursordaemon/main.go -o docs

# 重新生成后需重新编译
go build ./...
```

### main.go 注释模板

在 `main.go` 的 `package main` 之前添加 API 元信息：

```go
// @title cocursor Daemon API
// @version 1.0
// @description cocursor 守护进程 API 服务
// @host localhost:19960
// @BasePath /api/v1
// @schemes http
```

### Handler 方法注释规范

每个 Handler 方法必须包含以下注释：

| 注释             | 必需 | 说明                                       |
| ---------------- | ---- | ------------------------------------------ |
| `@Summary`       | ✅    | 简短描述（一句话）                         |
| `@Description`   | ❌    | 详细描述（可选）                           |
| `@Tags`          | ✅    | 分类标签（对话、团队、节点、评论、统计等） |
| `@Accept`        | ✅    | 请求 Content-Type，通常为 `json`           |
| `@Produce`       | ✅    | 响应 Content-Type，通常为 `json`           |
| `@Param`         | ❌    | 参数定义（path/query/body/header）         |
| `@Success`       | ✅    | 成功响应（HTTP 状态码 + 类型）             |
| `@Failure`       | ✅    | 错误响应（至少包含 400、500）              |
| `@Router`        | ✅    | 路由路径和方法                             |
| `@Security`      | ❌    | 认证方式（如需）                           |

### 注释示例

```go
// ListConversations 获取对话列表
// @Summary 获取对话列表
// @Description 获取本地所有 AI 对话列表
// @Tags 对话
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(20)
// @Success 200 {object} Response "成功"
// @Failure 400 {object} ErrorResponse "参数错误"
// @Failure 500 {object} ErrorResponse "内部错误"
// @Router /chats [get]
func (h *ChatHandler) ListConversations(c *gin.Context) {}
```

### 路由注册

在 `routes.go` 中添加 Swagger UI 路由：

```go
import (
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
    _ "github.com/cocursor/daemon/docs" // Swagger docs
)

// 在 setupRoutes() 中添加
s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

### 依赖版本

```
github.com/swaggo/swag v1.8.12
github.com/swaggo/gin-swagger v1.6.0
github.com/swaggo/files v1.0.1
```

### 文件结构

```
daemon/
├── cmd/cocursordaemon/
│   └── main.go          # API 元信息注释
├── docs/                 # 自动生成，不要手动编辑
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
└── internal/interfaces/http/handler/
    └── *.go             # Handler 方法的 Swagger 注释
```

### 注意事项

1. **不要手动编辑 `docs/` 目录**：每次运行 `swag init` 会覆盖
2. **类型引用**：使用包名（如 `stats.UsageStatsDTO`），避免使用别名（如 `appStats`）
3. **路由路径**：`@Router` 中的路径相对于 `@BasePath`，不包含 `/api/v1` 前缀
4. **版本匹配**：swag CLI 版本必须与 go.mod 中的 swaggo/swag 版本一致

## Go 响应结构体

```go
// Response 统一响应结构
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// PagedResponse 分页响应结构
type PagedResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
    Page    *PageInfo   `json:"page,omitempty"`
}

// PageInfo 分页信息
type PageInfo struct {
    Page     int `json:"page"`
    PageSize int `json:"pageSize"`
    Total    int `json:"total"`
    Pages    int `json:"pages"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Detail  string `json:"detail,omitempty"`
}
```

## 辅助函数

```go
// Success 成功响应
func Success(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{
        Code:    0,
        Message: "success",
        Data:    data,
    })
}

// SuccessWithPage 分页成功响应
func SuccessWithPage(c *gin.Context, data interface{}, page *PageInfo) {
    c.JSON(http.StatusOK, PagedResponse{
        Code:    0,
        Message: "success",
        Data:    data,
        Page:    page,
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
```
