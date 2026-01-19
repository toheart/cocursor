# GitHub Actions 工作流说明

本目录包含 CoCursor 项目的 GitHub Actions 工作流配置，用于自动化构建、测试和发布 VS Code 扩展。

## 工作流文件

### 1. ci.yml - 持续集成
**触发条件**：
- 推送到 `main` 或 `develop` 分支
- 针对 `main` 或 `develop` 分支的 Pull Request

**执行内容**：
- 安装 Go 1.24 和 Node.js 18
- 运行前端代码检查（ESLint）
- 运行后端单元测试
- 编译 TypeScript
- 构建前端扩展
- 构建 Linux 平台的后端 daemon 二进制
- 验证二进制文件
- 上传构建产物

### 2. build-and-release.yml - 构建和发布
**触发条件**：
- 推送标签（如 `v1.0.0`）
- 手动触发（workflow_dispatch）

**执行内容**：
- 在 4 个平台上并行构建：
  - Linux x64
  - Windows x64
  - macOS x64
  - macOS ARM64
- 每个平台都会：
  - 构建前端扩展（TypeScript + React）
  - 构建对应平台的后端 Go daemon 二进制
  - 打包成 VSIX 文件
  - 上传构建产物
- 创建 GitHub Release（发布所有平台的 VSIX 文件）
- 发布到 VS Code Marketplace（需要配置 VSCE_PAT）

## 使用说明

### 1. 发布新版本

#### 自动发布（推送到 GitHub）
```bash
# 创建并推送版本标签
git tag v1.0.0
git push origin v1.0.0
```

这将自动：
- 构建所有平台的 VSIX 文件
- 创建 GitHub Release
- 发布到 VS Code Marketplace（如果配置了 VSCE_PAT）

#### 手动触发发布
1. 访问 GitHub 仓库的 Actions 页面
2. 选择 "Build and Release VSCode Extension" 工作流
3. 点击 "Run workflow" 按钮
4. 勾选 "Publish to VS Code Marketplace" 选项（如果需要发布到市场）
5. 点击 "Run workflow"

### 2. 配置 Secrets

在 GitHub 仓库设置中添加以下 Secrets：

#### VSCE_PAT（必需，用于发布到 Marketplace）
1. 访问 [VS Code Marketplace](https://marketplace.visualstudio.com/manage/publishers)
2. 登录你的发布者账号
3. 获取或创建 Personal Access Token（PAT）
4. 在 GitHub 仓库设置中：Settings → Secrets and variables → Actions → New repository secret
5. Name: `VSCE_PAT`
6. Value: 粘贴你的 VS Code Marketplace PAT

**注意**：只有配置了 `VSCE_PAT`，工作流才能成功发布到 VS Code Marketplace。

### 3. 打包产物

每个平台生成的 VSIX 文件命名：
- `cocursor-linux-x64.vsix` - Linux x64 平台
- `cocursor-win32-x64.vsix` - Windows x64 平台
- `cocursor-darwin-x64.vsix` - macOS Intel 平台
- `cocursor-darwin-arm64.vsix` - macOS Apple Silicon 平台

### 4. 本地构建

如果需要在本地构建和测试：

```bash
# 进入扩展目录
cd co-extension

# 安装依赖
npm install

# 构建前端
make build

# 构建后端（在项目根目录）
cd ../backend
make build-all

# 或者在 co-extension 目录使用 vsce 打包
cd ../co-extension
npx @vscode/vsce package
```

## 文件结构

```
.github/
├── workflows/
│   ├── ci.yml                 # CI 工作流
│   └── build-and-release.yml  # 构建和发布工作流
└── README.md                  # 本文件

co-extension/
├── .vscodeignore             # 打包时的忽略规则
├── package.json              # 扩展配置（包含 files 字段）
├── dist/                     # 编译后的前端代码
│   ├── extension.js          # 扩展主入口
│   └── webview/             # Webview 界面
└── bin/                     # 平台特定的后端二进制
    ├── cocursor-linux-amd64
    ├── cocursor-windows-amd64.exe
    ├── cocursor-darwin-amd64
    └── cocursor-darwin-arm64
```

## 注意事项

1. **Go 版本**：项目要求 Go 1.24 或更高版本
2. **Node.js 版本**：项目使用 Node.js 18
3. **二进制文件**：每个平台的 VSIX 只包含对应平台的 daemon 二进制
4. **非开源项目**：不会包含源代码（.ts, .tsx 文件），只包含编译后的代码
5. **市场发布**：需要配置 VSCE_PAT 才能发布到 VS Code Marketplace

## 故障排查

### 发布失败：找不到 VSCE_PAT
**错误信息**：`Error: Input required and not supplied: vsce-pat`

**解决方法**：在 GitHub 仓库的 Secrets 中添加 VSCE_PAT

### 构建失败：Go 版本不匹配
**错误信息**：`go.mod requires go >= 1.24`

**解决方法**：确保 workflow 中使用的 Go 版本与 go.mod 一致（当前为 1.24）

### 打包失败：缺少 bin 目录
**错误信息**：找不到后端二进制文件

**解决方法**：确保 workflow 在打包前成功构建了后端二进制文件

## 相关资源

- [VS Code Extension API](https://code.visualstudio.com/api)
- [vsce (Visual Studio Code Extension Manager)](https://github.com/microsoft/vscode-vsce)
- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [Go 交叉编译](https://go.dev/doc/install/source#environment)
