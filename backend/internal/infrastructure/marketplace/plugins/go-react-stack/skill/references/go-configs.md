# Go 配置参考

## go.mod 模板

```go
module github.com/user/project-name

go 1.24.0

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.6.0
	github.com/stretchr/testify v1.11.1
	github.com/swaggo/files v1.0.1
	github.com/swaggo/gin-swagger v1.6.0
	github.com/swaggo/swag v1.8.12
)
```

## .golangci.yml 模板

```yaml
# golangci-lint v2.x 配置
version: "2"

run:
  timeout: 10m
  concurrency: 0

linters:
  enable:
    - nestif
    - gocognit
    - gocyclo
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused

  settings:
    nestif:
      min-complexity: 4
    gocognit:
      min-complexity: 15
    gocyclo:
      min-complexity: 10

  exclusions:
    generated: lax
    paths:
      - vendor
      - bin
      - ".*\\.pb\\.go$"
      - ".*\\.gen\\.go$"
    rules:
      - path: "_test\\.go"
        linters:
          - gocyclo
          - errcheck
          - gocognit
          - nestif

issues:
  max-issues-per-linter: 50
  max-same-issues: 3
```

## Makefile 模板

```makefile
.PHONY: all build test clean run wire swagger lint fmt

# 版本信息
VERSION ?= 0.1.0
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建参数
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# 输出目录
BIN_DIR := bin

all: test build

# 运行测试
test:
	go test -v -race -cover ./...

# 运行测试并生成覆盖率报告
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 生成 Wire 代码
wire:
	cd internal/wire && go run github.com/google/wire/cmd/wire

# 生成 Swagger 文档
swagger:
	@echo "生成 Swagger 文档..."
	@go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# 本地构建
build: wire
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/server ./cmd/server

# 运行
run: wire
	go run ./cmd/server

# 清理构建产物
clean:
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	find . -name "wire_gen.go" -delete

# 代码检查
lint:
	golangci-lint run --config .golangci.yml ./...

# 格式化
fmt:
	go fmt ./...
	goimports -w -local github.com/user/project-name .
```

## main.go 模板

```go
// @title Project API
// @version 1.0
// @description 项目 API 服务
// @host localhost:8080
// @BasePath /api/v1
// @schemes http
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/user/project-name/internal/infrastructure/config"
	"github.com/user/project-name/internal/wire"
)

func main() {
	// 加载配置
	cfg := config.NewConfig()
	port := cfg.Server.HTTPPort

	// Wire 自动生成的初始化函数
	app, err := wire.InitializeAll()
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}

	// 启动所有服务
	if err := app.Start(); err != nil {
		log.Fatalf("启动应用失败: %v", err)
	}

	// 优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("正在关闭应用...")
	if err := app.Stop(); err != nil {
		log.Printf("关闭应用时出错: %v", err)
	}
	log.Println("应用已关闭")
}
```

## HTTP Server 模板

```go
package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/user/project-name/internal/interfaces/http/handler"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/user/project-name/docs" // Swagger docs
)

// HTTPServer HTTP 服务器
type HTTPServer struct {
	router   *gin.Engine
	httpPort string
	server   *http.Server
}

// NewServer 创建 HTTP 服务器
func NewServer(
	exampleHandler *handler.ExampleHandler,
) *HTTPServer {
	router := gin.Default()

	// 注册路由
	api := router.Group("/api/v1")
	{
		api.GET("/examples", exampleHandler.List)
		api.GET("/examples/:id", exampleHandler.Get)
		api.POST("/examples", exampleHandler.Create)
		api.PUT("/examples/:id", exampleHandler.Update)
		api.DELETE("/examples/:id", exampleHandler.Delete)
	}

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return &HTTPServer{
		router:   router,
		httpPort: ":8080",
	}
}

// Start 启动服务器
func (s *HTTPServer) Start() error {
	s.server = &http.Server{
		Addr:    s.httpPort,
		Handler: s.router,
	}

	fmt.Printf("HTTP 服务器启动在端口 %s\n", s.httpPort)
	return s.server.ListenAndServe()
}

// Stop 停止服务器
func (s *HTTPServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}
```
