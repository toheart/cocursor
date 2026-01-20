.PHONY: all build build-all build-backend build-frontend test test-backend test-frontend \
        lint lint-backend lint-frontend clean clean-backend clean-frontend \
        install wire swagger run help ci-test ci-build

# 默认目标：显示帮助信息
.DEFAULT_GOAL := help

# 检测操作系统
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
else
    DETECTED_OS := $(shell uname -s)
endif

# ============================================================================
# 帮助信息
# ============================================================================

help:
	@echo "Cocursor 项目构建系统"
	@echo ""
	@echo "使用方法: make <target>"
	@echo ""
	@echo "构建目标:"
	@echo "  build          - 构建后端和前端（开发模式）"
	@echo "  build-all      - 构建所有平台的二进制文件"
	@echo "  build-backend  - 仅构建后端"
	@echo "  build-frontend - 仅构建前端"
	@echo ""
	@echo "测试目标:"
	@echo "  test           - 运行所有测试"
	@echo "  test-backend   - 运行后端测试"
	@echo "  test-frontend  - 运行前端 lint"
	@echo ""
	@echo "代码质量:"
	@echo "  lint           - 运行所有代码检查"
	@echo "  lint-backend   - 运行后端 lint"
	@echo "  lint-frontend  - 运行前端 lint"
	@echo ""
	@echo "开发工具:"
	@echo "  wire           - 生成后端 Wire 依赖注入代码"
	@echo "  swagger        - 生成 Swagger API 文档"
	@echo "  run            - 运行后端服务"
	@echo "  install        - 安装所有依赖"
	@echo ""
	@echo "清理目标:"
	@echo "  clean          - 清理所有构建产物"
	@echo "  clean-backend  - 清理后端构建产物"
	@echo "  clean-frontend - 清理前端构建产物"
	@echo ""
	@echo "CI 目标:"
	@echo "  ci-test        - CI 测试（后端测试 + 前端 lint）"
	@echo "  ci-build       - CI 构建（生成 wire + 构建前端）"

# ============================================================================
# 完整构建
# ============================================================================

# 开发构建
build: build-backend build-frontend
	@echo "✓ 构建完成"

# 多平台构建
build-all:
	$(MAKE) -C backend build-all
	@echo "✓ 多平台构建完成"

# ============================================================================
# 后端目标
# ============================================================================

build-backend:
	$(MAKE) -C backend build

test-backend:
	$(MAKE) -C backend test

lint-backend:
	$(MAKE) -C backend lint

clean-backend:
	$(MAKE) -C backend clean

wire:
	$(MAKE) -C backend wire

swagger:
	$(MAKE) -C backend swagger

run:
	$(MAKE) -C backend run

# ============================================================================
# 前端目标
# ============================================================================

build-frontend:
	$(MAKE) -C co-extension build

test-frontend: lint-frontend

lint-frontend:
	$(MAKE) -C co-extension lint

clean-frontend:
	$(MAKE) -C co-extension clean

# ============================================================================
# 组合目标
# ============================================================================

# 运行所有测试
test: test-backend test-frontend
	@echo "✓ 所有测试完成"

# 运行所有 lint
lint: lint-backend lint-frontend
	@echo "✓ 所有代码检查完成"

# 清理所有构建产物
clean: clean-backend clean-frontend
	@echo "✓ 清理完成"

# 安装所有依赖
install:
	cd backend && go mod download
	cd co-extension && npm ci
	@echo "✓ 依赖安装完成"

# ============================================================================
# CI 专用目标
# ============================================================================

# CI 测试阶段：运行测试和 lint
ci-test:
	@echo "=== CI 测试阶段 ==="
	$(MAKE) -C backend test
	$(MAKE) -C co-extension lint
	@echo "✓ CI 测试完成"

# CI 构建阶段：生成代码并构建
ci-build: wire
	@echo "=== CI 构建阶段 ==="
	$(MAKE) -C co-extension build
	@echo "✓ CI 构建完成"
