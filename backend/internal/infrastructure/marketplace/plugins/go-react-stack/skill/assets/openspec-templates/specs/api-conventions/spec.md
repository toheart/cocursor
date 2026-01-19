# API Conventions

This specification defines HTTP API design standards for the project.

## Version Control

- Use URL path versioning: `/api/v1/`, `/api/v2/`
- Current version: `v1`
- Version upgrade strategy: Increment major version for breaking changes

## Unified Response Format

All API responses use a unified JSON structure.

### Success Response

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

### Paginated Response

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

### Error Response

```json
{
  "code": 100001,
  "message": "Invalid parameter",
  "detail": "conversation id is required"
}
```

## Pagination Parameters

- `page`: Page number, starts from **1**, default 1
- `pageSize`: Items per page, default 20, max 100

## Business Error Codes

Error code format: `XXYYYY`

- `XX` = Module code (10-99)
- `YYYY` = Error sequence number (0001-9999)

| Range   | Module     | Examples                                                               |
| ------ | -------- | ------------------------------------------------------------------ |
| 0      | Success     | 0                                                                  |
| 10XXXX | Common Errors | 100001 Invalid parameter, 100002 Unauthorized, 100003 Forbidden, 100004 Resource not found |
| 20XXXX | Example Module | 200001 Example not found, 200002 No access to example                             |

## API Documentation (Swagger)

### Overview

- Use [swaggo/swag](https://github.com/swaggo/swag) v1.8.12 to generate OpenAPI 3.0 documentation
- Documentation endpoint: `GET /swagger/*any` (Swagger UI)
- All Handler methods **must** include Swagger annotations

### Generation Command

```bash
# Install swag tool (first time)
go install github.com/swaggo/swag/cmd/swag@v1.8.12

# Generate documentation
cd backend
swag init -g cmd/server/main.go -o docs

# Rebuild after regeneration
go build ./...
```

### main.go Annotation Template

Add API metadata before `package main` in `main.go`:

```go
// @title Project API
// @version 1.0
// @description Project API service
// @host localhost:8080
// @BasePath /api/v1
// @schemes http
```

### Handler Method Annotation Rules

Each Handler method must include the following annotations:

| Annotation             | Required | Description                                       |
| ---------------- | ---- | ------------------------------------------ |
| `@Summary`       | ✅    | Brief description (one sentence)                         |
| `@Description`   | ❌    | Detailed description (optional)                           |
| `@Tags`          | ✅    | Category tags (example, user, etc.) |
| `@Accept`        | ✅    | Request Content-Type, usually `json`           |
| `@Produce`       | ✅    | Response Content-Type, usually `json`           |
| `@Param`         | ❌    | Parameter definition (path/query/body/header)         |
| `@Success`       | ✅    | Success response (HTTP status code + type)             |
| `@Failure`       | ✅    | Error response (at least include 400, 500)              |
| `@Router`        | ✅    | Route path and method                             |
| `@Security`      | ❌    | Authentication method (if needed)                           |

### Annotation Example

```go
// ListExamples Get example list
// @Summary Get example list
// @Description Get all example items
// @Tags Example
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(20)
// @Success 200 {object} Response "Success"
// @Failure 400 {object} ErrorResponse "Invalid parameter"
// @Failure 500 {object} ErrorResponse "Internal error"
// @Router /examples [get]
func (h *ExampleHandler) ListExamples(c *gin.Context) {}
```

### Route Registration

Add Swagger UI route in `server.go`:

```go
import (
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
    _ "github.com/user/project/docs" // Swagger docs
)

// In setupRoutes() add
router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

### Dependency Versions

```
github.com/swaggo/swag v1.8.12
github.com/swaggo/gin-swagger v1.6.0
github.com/swaggo/files v1.0.1
```

### File Structure

```
backend/
├── cmd/server/
│   └── main.go          # API metadata annotations
├── docs/                 # Auto-generated, don't edit manually
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
└── internal/interfaces/http/handler/
    └── *.go             # Handler method Swagger annotations
```

### Notes

1. **Don't manually edit `docs/` directory**: Each `swag init` run will overwrite
2. **Type references**: Use package names (e.g., `example.ExampleDTO`), avoid aliases
3. **Route paths**: Paths in `@Router` are relative to `@BasePath`, don't include `/api/v1` prefix
4. **Version matching**: swag CLI version must match swaggo/swag version in go.mod

## Go Response Structures

```go
// Response Unified response structure
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// PagedResponse Paginated response structure
type PagedResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
    Page    *PageInfo   `json:"page,omitempty"`
}

// PageInfo Pagination information
type PageInfo struct {
    Page     int `json:"page"`
    PageSize int `json:"pageSize"`
    Total    int `json:"total"`
    Pages    int `json:"pages"`
}

// ErrorResponse Error response
type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Detail  string `json:"detail,omitempty"`
}
```

## Helper Functions

```go
// Success Success response
func Success(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{
        Code:    0,
        Message: "success",
        Data:    data,
    })
}

// SuccessWithPage Paginated success response
func SuccessWithPage(c *gin.Context, data interface{}, page *PageInfo) {
    c.JSON(http.StatusOK, PagedResponse{
        Code:    0,
        Message: "success",
        Data:    data,
        Page:    page,
    })
}

// Error Error response
func Error(c *gin.Context, httpCode int, errCode int, message string) {
    c.JSON(httpCode, ErrorResponse{
        Code:    errCode,
        Message: message,
    })
}

// ErrorWithDetail Error response with detail
func ErrorWithDetail(c *gin.Context, httpCode int, errCode int, message, detail string) {
    c.JSON(httpCode, ErrorResponse{
        Code:    errCode,
        Message: message,
        Detail:  detail,
    })
}
```
