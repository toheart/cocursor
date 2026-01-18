# 插件市场实现计划

## 阶段划分说明

每个阶段专注于一个功能模块，修改文件数控制在 3-5 个，便于理解和审查。

---

## 阶段 1：基础数据结构与模型

**目标**：定义插件相关的数据结构

**修改文件**（3 个）：
1. `backend/internal/domain/marketplace/models.go` - 新建，定义 Plugin、SkillComponent、MCPComponent、CommandComponent 等结构体
2. `backend/internal/domain/marketplace/state.go` - 新建，定义插件状态管理结构（InstalledPlugin、PluginState）
3. `backend/internal/domain/marketplace/wire.go` - 新建，Wire 依赖注入配置（暂时为空，后续使用）

**关键内容**：
- Plugin 结构体（包含基础信息和组件）
- SkillComponent、MCPComponent、CommandComponent 结构体
- PluginState 结构体（用于状态文件）
- JSON 标签和验证逻辑

**验收标准**：
- 结构体定义完整
- JSON 序列化/反序列化正常
- 代码通过编译

---

## 阶段 2：插件加载与扫描

**目标**：实现从 embed 文件系统扫描和加载插件

**修改文件**（3 个）：
1. `backend/internal/infrastructure/marketplace/plugin_loader.go` - 新建，实现插件扫描和加载
2. `backend/internal/infrastructure/marketplace/state_manager.go` - 新建，实现状态文件读写
3. `backend/internal/infrastructure/marketplace/wire.go` - 新建，Wire 配置

**关键内容**：
- 使用 `embed` 打包插件文件
- 扫描 `internal/marketplace/plugins/` 目录
- 读取 `plugin.json` 文件
- 读取/写入 `~/.cocursor/plugins-state.json`
- 合并已安装状态到插件列表

**验收标准**：
- 能够扫描并加载所有内置插件
- 能够读取和写入状态文件
- 能够正确标记已安装插件

---

## 阶段 3：Skill 安装功能

**目标**：实现 Skill 文件的安装和卸载

**修改文件**（3 个）：
1. `backend/internal/infrastructure/marketplace/skill_installer.go` - 新建，实现 Skill 安装/卸载
2. `backend/internal/infrastructure/marketplace/plugin_loader.go` - 修改，添加 Skill 文件读取方法
3. `backend/internal/infrastructure/marketplace/state_manager.go` - 修改，添加状态更新方法

**关键内容**：
- 从 embed 读取 Skill 文件（SKILL.md、scripts/、references/、assets/）
- 创建 `~/.claude/skills/<skill_name>/` 目录
- 复制所有文件到目标目录
- 检查 Skill 名称冲突
- 卸载时删除整个目录

**验收标准**：
- Skill 文件能够正确安装到目标目录
- Skill 名称冲突检查有效
- 卸载能够完全删除文件

---

## 阶段 4：MCP 配置功能

**目标**：实现 MCP 配置的读取、写入和环境变量检测

**修改文件**（3 个）：
1. `backend/internal/infrastructure/marketplace/mcp_config.go` - 新建，实现 MCP 配置管理
2. `backend/internal/infrastructure/marketplace/skill_installer.go` - 修改，集成 MCP 安装
3. `backend/internal/infrastructure/marketplace/plugin_loader.go` - 修改，添加环境变量提取方法

**关键内容**：
- 读取/写入 `~/.cursor/mcp.json`
- 支持 JSONC 格式（移除注释）
- 构建 MCP 配置对象
- 检测 headers 中的环境变量（`${env:VAR}` 格式）
- 提取环境变量名称列表

**验收标准**：
- MCP 配置能够正确写入 mcp.json
- 环境变量检测准确
- 配置格式符合 Cursor 要求

---

## 阶段 5：Command 安装功能

**目标**：实现 Command 文件的安装和卸载

**修改文件**（2 个）：
1. `backend/internal/infrastructure/marketplace/command_installer.go` - 新建，实现 Command 安装/卸载
2. `backend/internal/infrastructure/marketplace/plugin_loader.go` - 修改，添加 Command 文件读取方法

**关键内容**：
- 从 embed 读取 command.md 文件
- 写入 `~/.cursor/commands/<command_id>.md`
- 卸载时删除文件

**验收标准**：
- Command 文件能够正确安装
- 卸载能够删除文件

---

## 阶段 6：服务层与 Handler

**目标**：实现业务逻辑层和 HTTP 接口

**修改文件**（4 个）：
1. `backend/internal/application/marketplace/service.go` - 新建，实现 PluginService
2. `backend/internal/application/marketplace/wire.go` - 新建，Wire 配置
3. `backend/internal/interfaces/http/handler/marketplace.go` - 新建，实现 MarketplaceHandler
4. `backend/internal/interfaces/http/handler/wire.go` - 修改，添加 MarketplaceHandler

**关键内容**：
- PluginService：整合所有安装/卸载逻辑
- MarketplaceHandler：实现所有 API 端点
- 错误处理和响应格式
- 路由注册

**验收标准**：
- 所有 API 端点正常工作
- 错误处理完善
- 响应格式符合规范

---

## 阶段 7：路由注册与集成

**目标**：注册路由并集成到主应用

**修改文件**（3 个）：
1. `backend/internal/interfaces/http/server.go` - 修改，注册 marketplace 路由
2. `backend/internal/wire/app.go` - 修改，添加 PluginService 依赖
3. `backend/internal/application/wire.go` - 修改，添加 marketplace ProviderSet

**关键内容**：
- 注册 `/api/v1/marketplace/*` 路由
- Wire 依赖注入配置
- 确保服务正确初始化

**验收标准**：
- 路由能够正常访问
- 服务能够正常启动
- 依赖注入正确

---

## 阶段 8：前端集成

**目标**：更新前端 Marketplace 组件，连接后端 API

**修改文件**（3 个）：
1. `co-extension/src/webview/components/Marketplace.tsx` - 修改，连接后端 API
2. `co-extension/src/webview/services/api.ts` - 修改，添加 marketplace API 方法
3. `co-extension/src/webview/components/Marketplace.tsx` - 修改，添加环境变量提示

**关键内容**：
- 调用后端 API 获取插件列表
- 实现安装/卸载功能
- 显示环境变量配置提示
- 错误处理和用户反馈

**验收标准**：
- 前端能够显示插件列表
- 安装/卸载功能正常
- 环境变量提示正确显示

---

## 阶段 9：测试与文档

**目标**：添加测试和更新文档

**修改文件**（4-5 个）：
1. `backend/internal/infrastructure/marketplace/plugin_loader_test.go` - 新建，测试插件加载
2. `backend/internal/infrastructure/marketplace/skill_installer_test.go` - 新建，测试 Skill 安装
3. `backend/internal/application/marketplace/service_test.go` - 新建，测试服务层
4. `backend/internal/interfaces/http/handler/marketplace_test.go` - 新建，测试 Handler
5. `docs/marketplace-usage.md` - 新建，使用文档（可选）

**关键内容**：
- 单元测试覆盖主要功能
- 集成测试验证完整流程
- 使用临时目录进行文件系统测试
- 文档说明如何使用插件市场

**验收标准**：
- 测试覆盖率 > 70%
- 所有测试通过
- 文档清晰易懂

---

## 实施顺序建议

1. **阶段 1-2**：基础准备，建立数据结构和加载机制
2. **阶段 3-5**：核心功能，逐个实现安装功能
3. **阶段 6-7**：接口层，暴露 API 并集成
4. **阶段 8**：前端集成，完成用户体验
5. **阶段 9**：测试和文档，确保质量

## 注意事项

- 每个阶段完成后进行代码审查
- 确保每个阶段都能独立编译和运行
- 使用临时目录进行测试，避免污染系统
- 注意跨平台兼容性（Windows/macOS/Linux）
- 环境变量提示要清晰明确
