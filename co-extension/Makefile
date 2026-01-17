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

# 构建 Extension
build-extension:
	@echo "构建 Extension（开发模式）..."
	@mkdir -p $(DIST_DIR)
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
	@mkdir -p $(DIST_DIR)/webview
	$(ESBUILD) $(WEBVIEW_ENTRY) \
		--bundle \
		--platform=browser \
		--target=es2020 \
		--format=iife \
		--outfile=$(DIST_DIR)/webview/index.js \
		--jsx=automatic \
		--sourcemap \
		--minify=false
	@cp $(WEBVIEW_CSS) $(DIST_DIR)/webview/index.css
	@echo "Webview 构建完成: $(DIST_DIR)/webview/index.js"

# 生产构建
build: build-extension-prod build-webview-prod
	@echo "构建完成"

# 构建 Extension（生产模式）
build-extension-prod:
	@echo "构建 Extension（生产模式）..."
	@mkdir -p $(DIST_DIR)
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
	@mkdir -p $(DIST_DIR)/webview
	$(ESBUILD) $(WEBVIEW_ENTRY) \
		--bundle \
		--platform=browser \
		--target=es2020 \
		--format=iife \
		--outfile=$(DIST_DIR)/webview/index.js \
		--jsx=automatic \
		--sourcemap \
		--minify
	@cp $(WEBVIEW_CSS) $(DIST_DIR)/webview/index.css
	@echo "Webview 构建完成: $(DIST_DIR)/webview/index.js"

# 监听模式
watch:
	@echo "启动监听模式（Extension + Webview）..."
	@mkdir -p $(DIST_DIR)/webview
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
		--watch &
	@echo "监听模式已启动，按 Ctrl+C 停止"

# 清理
clean:
	@echo "清理构建产物..."
	rm -rf $(DIST_DIR)
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
