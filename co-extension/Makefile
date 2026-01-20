.PHONY: all build compile-debug watch clean install lint fmt test \
        build-extension build-webview build-extension-prod build-webview-prod

# ============================================================================
# 跨平台兼容配置
# ============================================================================

# 检测操作系统
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    MKDIR = powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path"
    RMDIR = powershell -NoProfile -Command "Remove-Item -Recurse -Force -ErrorAction SilentlyContinue"
    COPY = powershell -NoProfile -Command "Copy-Item -Force"
    # Windows 下检测目录存在
    define ensure_dir
		@powershell -NoProfile -Command "if (-not (Test-Path '$(1)')) { New-Item -ItemType Directory -Force -Path '$(1)' | Out-Null }"
    endef
else
    DETECTED_OS := $(shell uname -s)
    MKDIR = mkdir -p
    RMDIR = rm -rf
    COPY = cp
    define ensure_dir
		@mkdir -p $(1)
    endef
endif

# ============================================================================
# 构建工具和路径
# ============================================================================

ESBUILD := npx esbuild
TSC := npx tsc

# 源文件
SRC_DIR := src
DIST_DIR := dist
EXTENSION_ENTRY := $(SRC_DIR)/extension.ts
WEBVIEW_ENTRY := $(SRC_DIR)/webview/index.tsx
WEBVIEW_CSS := $(SRC_DIR)/webview/index.css

# ============================================================================
# 默认目标
# ============================================================================

all: compile-debug

# ============================================================================
# 开发构建（带 sourcemap，不压缩）
# ============================================================================

compile-debug: build-extension build-webview
	@echo "构建完成（开发模式）"
	@echo "✓ Extension: $(DIST_DIR)/extension.js"
	@echo "✓ Webview: $(DIST_DIR)/webview/index.js"

build-extension:
	@echo "构建 Extension（开发模式）..."
	$(call ensure_dir,$(DIST_DIR))
	$(ESBUILD) $(EXTENSION_ENTRY) \
		--bundle \
		--platform=node \
		--target=node18 \
		--format=cjs \
		--outfile=$(DIST_DIR)/extension.js \
		--external:vscode \
		--sourcemap \
		--minify=false
	@echo "✓ Extension 构建完成"

build-webview:
	@echo "构建 Webview（开发模式）..."
	$(call ensure_dir,$(DIST_DIR)/webview)
ifeq ($(OS),Windows_NT)
	$(ESBUILD) $(WEBVIEW_ENTRY) \
		--bundle \
		--platform=browser \
		--target=es2020 \
		--format=iife \
		--outfile=$(DIST_DIR)/webview/index.js \
		--jsx=automatic \
		--sourcemap \
		--minify=false \
		--define:process.env.NODE_ENV='"development"' \
		--global-name=cocursor
	@powershell -NoProfile -Command "if (Test-Path '$(WEBVIEW_CSS)') { Copy-Item '$(WEBVIEW_CSS)' '$(DIST_DIR)/webview/index.css' -Force }"
else
	$(ESBUILD) $(WEBVIEW_ENTRY) \
		--bundle \
		--platform=browser \
		--target=es2020 \
		--format=iife \
		--outfile=$(DIST_DIR)/webview/index.js \
		--jsx=automatic \
		--sourcemap \
		--minify=false \
		--define:process.env.NODE_ENV='"development"' \
		--global-name=cocursor
	@cp "$(WEBVIEW_CSS)" "$(DIST_DIR)/webview/index.css" 2>/dev/null || true
endif
	@echo "✓ Webview 构建完成"

# ============================================================================
# 生产构建（压缩优化）
# ============================================================================

build: build-extension-prod build-webview-prod
	@echo "构建完成（生产模式）"
	@echo "✓ Extension: $(DIST_DIR)/extension.js"
	@echo "✓ Webview: $(DIST_DIR)/webview/index.js"

build-extension-prod:
	@echo "构建 Extension（生产模式）..."
	$(call ensure_dir,$(DIST_DIR))
	$(ESBUILD) $(EXTENSION_ENTRY) \
		--bundle \
		--platform=node \
		--target=node18 \
		--format=cjs \
		--outfile=$(DIST_DIR)/extension.js \
		--external:vscode \
		--sourcemap \
		--minify
	@echo "✓ Extension 构建完成"

build-webview-prod:
	@echo "构建 Webview（生产模式）..."
	$(call ensure_dir,$(DIST_DIR)/webview)
ifeq ($(OS),Windows_NT)
	$(ESBUILD) $(WEBVIEW_ENTRY) \
		--bundle \
		--platform=browser \
		--target=es2020 \
		--format=iife \
		--outfile=$(DIST_DIR)/webview/index.js \
		--jsx=automatic \
		--sourcemap \
		--minify \
		--define:process.env.NODE_ENV='"production"' \
		--global-name=cocursor
	@powershell -NoProfile -Command "if (Test-Path '$(WEBVIEW_CSS)') { Copy-Item '$(WEBVIEW_CSS)' '$(DIST_DIR)/webview/index.css' -Force }"
else
	$(ESBUILD) $(WEBVIEW_ENTRY) \
		--bundle \
		--platform=browser \
		--target=es2020 \
		--format=iife \
		--outfile=$(DIST_DIR)/webview/index.js \
		--jsx=automatic \
		--sourcemap \
		--minify \
		--define:process.env.NODE_ENV='"production"' \
		--global-name=cocursor
	@cp "$(WEBVIEW_CSS)" "$(DIST_DIR)/webview/index.css" 2>/dev/null || true
endif
	@echo "✓ Webview 构建完成"

# ============================================================================
# 监听模式
# ============================================================================

watch:
	@echo "启动监听模式（Extension + Webview）..."
	$(call ensure_dir,$(DIST_DIR)/webview)
ifeq ($(OS),Windows_NT)
	@echo "Windows 下请使用 'npm run watch' 代替"
else
	@$(ESBUILD) $(EXTENSION_ENTRY) \
		--bundle \
		--platform=node \
		--target=node18 \
		--format=cjs \
		--outfile=$(DIST_DIR)/extension.js \
		--external:vscode \
		--sourcemap \
		--watch &
	@$(ESBUILD) $(WEBVIEW_ENTRY) \
		--bundle \
		--platform=browser \
		--target=es2020 \
		--format=iife \
		--outfile=$(DIST_DIR)/webview/index.js \
		--jsx=automatic \
		--sourcemap \
		--watch \
		--define:process.env.NODE_ENV='"development"' \
		--global-name=cocursor &
	@echo "监听模式已启动，按 Ctrl+C 停止"
endif

# ============================================================================
# 工具目标
# ============================================================================

clean:
	@echo "清理构建产物..."
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "if (Test-Path '$(DIST_DIR)') { Remove-Item -Recurse -Force '$(DIST_DIR)' }"
else
	@rm -rf "$(DIST_DIR)"
endif
	@echo "✓ 清理完成"

install:
	npm install

lint:
	npm run lint

fmt:
	npx prettier --write "src/**/*.ts" "src/**/*.tsx"

test:
	npm test

# ============================================================================
# 帮助信息
# ============================================================================

help:
	@echo "Cocursor Extension 构建系统"
	@echo ""
	@echo "使用方法: make <target>"
	@echo ""
	@echo "构建目标:"
	@echo "  all / compile-debug  - 开发构建（默认）"
	@echo "  build                - 生产构建（压缩优化）"
	@echo "  build-extension      - 仅构建 Extension（开发）"
	@echo "  build-webview        - 仅构建 Webview（开发）"
	@echo ""
	@echo "开发工具:"
	@echo "  watch    - 监听模式"
	@echo "  install  - 安装依赖"
	@echo "  lint     - 代码检查"
	@echo "  fmt      - 格式化代码"
	@echo "  test     - 运行测试"
	@echo "  clean    - 清理构建产物"
	@echo ""
	@echo "当前系统: $(DETECTED_OS)"
