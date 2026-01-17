.PHONY: all build build-all test clean clean-data clean-all run lint lint-fix fmt

# 版本信息
VERSION ?= 0.1.0
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建参数
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# 输出目录
BIN_DIR := bin
EXT_BIN_DIR := ../co-extension/bin

all: test build

# 运行测试
test:
	go test -v -race -cover ./...

# 运行测试并生成覆盖率报告
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 本地构建
build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/cocursor .

# 构建并安装到 extension/bin（用于开发调试）
install: build
	@mkdir -p $(EXT_BIN_DIR)
	@cp $(BIN_DIR)/cocursor $(EXT_BIN_DIR)/cocursor-darwin-amd64
	@echo "已安装 daemon 到 extension/bin/cocursor-darwin-amd64"

# 构建并复制到 extension/bin（用于开发）
build-dev: build-windows-dev

build-windows-dev:
	@mkdir -p $(EXT_BIN_DIR)
	go build -a $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursordaemon-windows-amd64.exe .

# 多平台构建（输出到 extension/bin）
build-all: build-windows build-windows-arm64 build-darwin build-linux

build-windows:
	@mkdir -p $(EXT_BIN_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-windows-amd64.exe .

build-darwin:
	@mkdir -p $(EXT_BIN_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-darwin-arm64 .

build-linux:
	@mkdir -p $(EXT_BIN_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-linux-arm64 .

build-windows-arm64:
	@mkdir -p $(EXT_BIN_DIR)
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(EXT_BIN_DIR)/cocursor-windows-arm64.exe .

# 运行
run:
	go run .

# 清理构建产物
clean:
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html

# 清理本地数据（包括数据库、配置等持久化文件）
# 警告：此操作会删除所有本地数据，包括团队配置、评论、归档记录等
clean-data:
	@echo "清理本地数据目录 ~/.cocursor ..."
	rm -rf ~/.cocursor/*.db
	rm -rf ~/.cocursor/*.json
	@echo "已清理数据库和配置文件（保留 node_id）"

# 完全清理（构建产物 + 本地数据）
clean-all: clean clean-data
	@echo "已完成完全清理"

# 生成 mock
mock:
	go generate ./...

# 代码检查（使用 .golangci.yml 配置）
# 需要先安装: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
lint:
	golangci-lint run --config .golangci.yml ./...

# 格式化
fmt:
	go fmt ./...
	goimports -w -local github.com/cocursor .
