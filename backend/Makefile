.PHONY: all build build-all test clean clean-data clean-all run lint lint-fix fmt wire swagger \
        build-windows build-windows-arm64 build-darwin build-linux install help \
        build-dev build-darwin-dev build-linux-dev build-windows-dev test-coverage mock

# ============================================================================
# 跨平台兼容配置
# ============================================================================

# 检测操作系统
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    # Windows 下使用 PowerShell 获取日期
    BUILD_TIME := $(shell powershell -NoProfile -Command "Get-Date -AsUTC -Format 'yyyy-MM-ddTHH:mm:ssZ'")
    GIT_COMMIT := $(shell git rev-parse --short HEAD 2>nul || echo unknown)
    MKDIR = powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path"
    RMDIR = powershell -NoProfile -Command "Remove-Item -Recurse -Force -ErrorAction SilentlyContinue"
    define ensure_dir
		@powershell -NoProfile -Command "if (-not (Test-Path '$(1)')) { New-Item -ItemType Directory -Force -Path '$(1)' | Out-Null }"
    endef
else
    DETECTED_OS := $(shell uname -s)
    BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
    GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    MKDIR = mkdir -p
    RMDIR = rm -rf
    define ensure_dir
		@mkdir -p $(1)
    endef
endif

# ============================================================================
# 版本信息
# ============================================================================

VERSION ?= 0.1.0
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# ============================================================================
# 输出目录
# ============================================================================

BIN_DIR := bin
EXT_BIN_DIR := ../co-extension/bin

# ============================================================================
# 默认目标
# ============================================================================

.DEFAULT_GOAL := help

all: test build

# ============================================================================
# 帮助信息
# ============================================================================

help:
	@echo "Cocursor Backend 构建系统"
	@echo ""
	@echo "使用方法: make <target>"
	@echo ""
	@echo "构建目标:"
	@echo "  build          - 本地构建"
	@echo "  build-all      - 多平台构建（输出到 extension/bin）"
	@echo "  build-dev      - 构建并复制到 extension/bin（根据当前系统）"
	@echo "  install        - 构建并安装到 extension/bin"
	@echo ""
	@echo "平台构建:"
	@echo "  build-windows       - Windows x64"
	@echo "  build-windows-arm64 - Windows ARM64"
	@echo "  build-darwin        - macOS x64 + ARM64"
	@echo "  build-linux         - Linux x64 + ARM64"
	@echo ""
	@echo "测试目标:"
	@echo "  test           - 运行测试"
	@echo "  test-coverage  - 运行测试并生成覆盖率报告"
	@echo ""
	@echo "代码生成:"
	@echo "  wire           - 生成 Wire 依赖注入代码"
	@echo "  swagger        - 生成 Swagger API 文档"
	@echo "  mock           - 生成 mock 文件"
	@echo ""
	@echo "代码质量:"
	@echo "  lint           - 运行代码检查"
	@echo "  fmt            - 格式化代码"
	@echo ""
	@echo "清理目标:"
	@echo "  clean          - 清理构建产物"
	@echo "  clean-data     - 清理本地数据"
	@echo "  clean-all      - 完全清理"
	@echo ""
	@echo "其他:"
	@echo "  run            - 运行后端服务"
	@echo ""
	@echo "当前系统: $(DETECTED_OS)"
	@echo "版本: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"

# ============================================================================
# 测试
# ============================================================================

test:
	go test -v -race -cover ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# ============================================================================
# 代码生成
# ============================================================================

wire:
	cd internal/wire && go run github.com/google/wire/cmd/wire

# 生成 Swagger 文档
# 注意：如果生成的 docs.go 包含 LeftDelim/RightDelim 字段导致编译错误，
# 会自动移除这些字段以兼容旧版本的 swag 库
swagger:
	@echo "生成 Swagger 文档..."
ifeq ($(OS),Windows_NT)
	@go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o docs --parseDependency --parseInternal
	@powershell -NoProfile -Command "if (Test-Path 'docs/docs.go') { (Get-Content 'docs/docs.go') -replace '.*LeftDelim:.*', '' -replace '.*RightDelim:.*', '' | Set-Content 'docs/docs.go' }"
else
	@go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o docs --parseDependency --parseInternal
	@if [ -f docs/docs.go ]; then \
		sed -i.bak '/LeftDelim:/d; /RightDelim:/d' docs/docs.go 2>/dev/null || \
		(if command -v sed >/dev/null 2>&1; then sed -i '' '/LeftDelim:/d; /RightDelim:/d' docs/docs.go; fi); \
		rm -f docs/docs.go.bak 2>/dev/null || true; \
	fi
endif
	@echo "✓ Swagger 文档已生成到 docs/ 目录"

mock:
	go generate ./...

# ============================================================================
# 本地构建
# ============================================================================

build: wire
	$(call ensure_dir,$(BIN_DIR))
	go build $(LDFLAGS) -o $(BIN_DIR)/cocursor ./cmd/server

install: build
	$(call ensure_dir,$(EXT_BIN_DIR))
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "Copy-Item '$(BIN_DIR)/cocursor' '$(EXT_BIN_DIR)/cocursor-windows-amd64.exe' -Force"
	@echo "✓ 已安装 daemon 到 extension/bin/cocursor-windows-amd64.exe"
else
    ifeq ($(DETECTED_OS),Darwin)
		@cp $(BIN_DIR)/cocursor $(EXT_BIN_DIR)/cocursor-darwin-amd64
		@echo "✓ 已安装 daemon 到 extension/bin/cocursor-darwin-amd64"
    else
		@cp $(BIN_DIR)/cocursor $(EXT_BIN_DIR)/cocursor-linux-amd64
		@echo "✓ 已安装 daemon 到 extension/bin/cocursor-linux-amd64"
    endif
endif

# ============================================================================
# 开发构建（根据当前系统自动选择）
# ============================================================================

build-dev: wire
ifeq ($(OS),Windows_NT)
	$(MAKE) build-windows-dev
else
    ifeq ($(DETECTED_OS),Darwin)
		$(MAKE) build-darwin-dev
    else
		$(MAKE) build-linux-dev
    endif
endif

build-darwin-dev:
	$(call ensure_dir,$(EXT_BIN_DIR))
ifeq ($(shell uname -m),arm64)
	go build -a $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursordaemon-darwin-arm64 ./cmd/server
	@echo "✓ 已构建 daemon 到 extension/bin/cocursordaemon-darwin-arm64"
else
	go build -a $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursordaemon-darwin-amd64 ./cmd/server
	@echo "✓ 已构建 daemon 到 extension/bin/cocursordaemon-darwin-amd64"
endif

build-linux-dev:
	$(call ensure_dir,$(EXT_BIN_DIR))
	go build -a $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursordaemon-linux-amd64 ./cmd/server
	@echo "✓ 已构建 daemon 到 extension/bin/cocursordaemon-linux-amd64"

build-windows-dev:
	$(call ensure_dir,$(EXT_BIN_DIR))
ifeq ($(OS),Windows_NT)
	go build -a $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursordaemon-windows-amd64.exe ./cmd/server
else
	GOOS=windows GOARCH=amd64 go build -a $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursordaemon-windows-amd64.exe ./cmd/server
endif
	@echo "✓ 已构建 daemon 到 extension/bin/cocursordaemon-windows-amd64.exe"

# ============================================================================
# 多平台构建（输出到 extension/bin）
# 使用 CGO_ENABLED=0 确保静态链接，避免 glibc 版本依赖问题
# ============================================================================

build-all: wire build-windows build-windows-arm64 build-darwin build-linux
	@echo "✓ 多平台构建完成"

build-windows:
	$(call ensure_dir,$(EXT_BIN_DIR))
ifeq ($(OS),Windows_NT)
	set CGO_ENABLED=0&& set GOOS=windows&& set GOARCH=amd64&& go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-windows-amd64.exe ./cmd/server
else
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-windows-amd64.exe ./cmd/server
endif

build-windows-arm64:
	$(call ensure_dir,$(EXT_BIN_DIR))
ifeq ($(OS),Windows_NT)
	set CGO_ENABLED=0&& set GOOS=windows&& set GOARCH=arm64&& go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-windows-arm64.exe ./cmd/server
else
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-windows-arm64.exe ./cmd/server
endif

build-darwin:
	$(call ensure_dir,$(EXT_BIN_DIR))
ifeq ($(OS),Windows_NT)
	set CGO_ENABLED=0&& set GOOS=darwin&& set GOARCH=amd64&& go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-darwin-amd64 ./cmd/server
	set CGO_ENABLED=0&& set GOOS=darwin&& set GOARCH=arm64&& go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-darwin-arm64 ./cmd/server
else
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-darwin-amd64 ./cmd/server
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-darwin-arm64 ./cmd/server
endif

build-linux:
	$(call ensure_dir,$(EXT_BIN_DIR))
ifeq ($(OS),Windows_NT)
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-linux-amd64 ./cmd/server
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=arm64&& go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-linux-arm64 ./cmd/server
else
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-linux-amd64 ./cmd/server
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-linux-arm64 ./cmd/server
endif

# ============================================================================
# 运行
# ============================================================================

run: wire
	go run ./cmd/server

# ============================================================================
# 清理
# ============================================================================

clean:
	@echo "清理构建产物..."
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "if (Test-Path '$(BIN_DIR)') { Remove-Item -Recurse -Force '$(BIN_DIR)' }"
	@powershell -NoProfile -Command "Remove-Item -Force 'coverage.out', 'coverage.html' -ErrorAction SilentlyContinue"
	@powershell -NoProfile -Command "Get-ChildItem -Recurse -Filter 'wire_gen.go' | Remove-Item -Force"
else
	@rm -rf $(BIN_DIR)
	@rm -f coverage.out coverage.html
	@find . -name "wire_gen.go" -delete
endif
	@echo "✓ 清理完成"

# 清理本地数据（包括数据库、配置等持久化文件）
# 警告：此操作会删除所有本地数据，包括团队配置、评论、归档记录等
clean-data:
	@echo "清理本地数据目录 ~/.cocursor ..."
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "Remove-Item -Force '$$HOME/.cocursor/*.db', '$$HOME/.cocursor/*.json' -ErrorAction SilentlyContinue"
else
	@rm -rf ~/.cocursor/*.db
	@rm -rf ~/.cocursor/*.json
endif
	@echo "✓ 已清理数据库和配置文件（保留 node_id）"

clean-all: clean clean-data
	@echo "✓ 已完成完全清理"

# ============================================================================
# 代码质量
# ============================================================================

# 代码检查（使用 .golangci.yml 配置）
# 需要先安装: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
lint:
	golangci-lint run --config .golangci.yml ./...

fmt:
	go fmt ./...
ifeq ($(OS),Windows_NT)
	goimports -w -local github.com/cocursor .
else
	goimports -w -local github.com/cocursor .
endif
