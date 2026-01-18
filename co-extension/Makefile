.PHONY: all build compile-debug watch clean install lint fmt test

# 构建工具
ESBUILD := npx esbuild
TSC := npx tsc

# 源文件
SRC_DIR := src
DIST_DIR := dist
EXTENSION_ENTRY := $(SRC_DIR)/extension.ts
WEBVIEW_ENTRY := $(SRC_DIR)/webview/index.tsx
WEBVIEW_CSS := $(SRC_DIR)/webview/index.css

all: compile-debug

# 开发构建（带 sourcemap）
compile-debug: build-extension build-webview
	@echo "构建完成"
	@echo "✓ Extension: $(DIST_DIR)/extension.js"
	@echo "✓ Webview: $(DIST_DIR)/webview/index.js"

# 构建 Extension
build-extension:
	@echo "构建 Extension（开发模式）..."
	@-powershell -NoProfile -Command "if (-not (Test-Path '$(DIST_DIR)')) { New-Item -ItemType Directory -Path '$(DIST_DIR)' -Force | Out-Null }" 2>nul || -mkdir -p "$(DIST_DIR)" 2>/dev/null || true
	$(ESBUILD) $(EXTENSION_ENTRY) \
		--bundle \
		--platform=node \
		--target=node18 \
		--format=cjs \
		--outfile=$(DIST_DIR)/extension.js \
		--external:vscode \
		--sourcemap \
		--minify=false
	@echo "Extension 构建完成: $(DIST_DIR)/extension.js"

# 构建 Webview
build-webview:
	@echo "构建 Webview（开发模式）..."
	@-powershell -NoProfile -Command "if (-not (Test-Path '$(DIST_DIR)/webview')) { New-Item -ItemType Directory -Path '$(DIST_DIR)/webview' -Force | Out-Null }" 2>nul || -mkdir -p "$(DIST_DIR)/webview" 2>/dev/null || true
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
	@-powershell -NoProfile -Command "Copy-Item '$(WEBVIEW_CSS)' '$(DIST_DIR)/webview/index.css' -Force -ErrorAction SilentlyContinue" 2>nul || -cp "$(WEBVIEW_CSS)" "$(DIST_DIR)/webview/index.css" 2>/dev/null || true
	@echo "Webview 构建完成: $(DIST_DIR)/webview/index.js"

# 生产构建
build: build-extension-prod build-webview-prod
	@echo "构建完成"

# 构建 Extension（生产模式）
build-extension-prod:
	@echo "构建 Extension（生产模式）..."
	@-powershell -NoProfile -Command "if (-not (Test-Path '$(DIST_DIR)')) { New-Item -ItemType Directory -Path '$(DIST_DIR)' -Force | Out-Null }" 2>nul || -mkdir -p "$(DIST_DIR)" 2>/dev/null || true
	$(ESBUILD) $(EXTENSION_ENTRY) \
		--bundle \
		--platform=node \
		--target=node18 \
		--format=cjs \
		--outfile=$(DIST_DIR)/extension.js \
		--external:vscode \
		--sourcemap \
		--minify
	@echo "Extension 构建完成: $(DIST_DIR)/extension.js"

# 构建 Webview（生产模式）
build-webview-prod:
	@echo "构建 Webview（生产模式）..."
	@-powershell -NoProfile -Command "if (-not (Test-Path '$(DIST_DIR)/webview')) { New-Item -ItemType Directory -Path '$(DIST_DIR)/webview' -Force | Out-Null }" 2>nul || -mkdir -p "$(DIST_DIR)/webview" 2>/dev/null || true
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
	@-powershell -NoProfile -Command "Copy-Item '$(WEBVIEW_CSS)' '$(DIST_DIR)/webview/index.css' -Force -ErrorAction SilentlyContinue" 2>nul || -cp "$(WEBVIEW_CSS)" "$(DIST_DIR)/webview/index.css" 2>/dev/null || true
	@echo "Webview 构建完成: $(DIST_DIR)/webview/index.js"

# 监听模式
watch:
	@echo "启动监听模式（Extension + Webview）..."
	@-powershell -NoProfile -Command "if (-not (Test-Path '$(DIST_DIR)/webview')) { New-Item -ItemType Directory -Path '$(DIST_DIR)/webview' -Force | Out-Null }" 2>nul || -mkdir -p "$(DIST_DIR)/webview" 2>/dev/null || true
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

# 清理
clean:
	@echo "清理构建产物..."
	@-powershell -NoProfile -Command "if (Test-Path '$(DIST_DIR)') { Remove-Item -Recurse -Force '$(DIST_DIR)' -ErrorAction SilentlyContinue }" 2>nul || -rm -rf "$(DIST_DIR)" 2>/dev/null || true
	@echo "清理完成"

# 安装依赖
install:
	npm install

# 代码检查
lint:
	npm run lint

# 格式化
fmt:
	npx prettier --write "src/**/*.ts"

# 运行测试
test:
	npm test
