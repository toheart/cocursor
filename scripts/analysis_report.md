# Cursor 数据库分析报告

**分析时间**: 2026-01-18

## 1. 全局存储分析

### 1.1 基本信息
- **数据库路径**: `C:\Users\TANG\AppData\Roaming\Cursor\User\globalStorage\state.vscdb`
- **总记录数**: 197 条

### 1.2 AI 代码追踪数据

发现 4 天的统计数据：

| 日期 | Tab 建议 | Tab 接受 | Tab 接受率 | Composer 建议 | Composer 接受 | Composer 接受率 |
|------|----------|----------|------------|--------------|--------------|-----------------|
| 2026-01-14 | 0 | 0 | - | 5 | 45 | **900%** ❌ |
| 2026-01-16 | 40 | 11 | 27.5% | 3363 | 9063 | **269.5%** ❌ |
| 2026-01-17 | 150 | 54 | 36% | 0 | 24955 | **异常** ❌ |
| 2026-01-18 | 1 | 0 | 0% | 0 | 43 | **异常** ❌ |

**问题分析**：
- ✅ Tab 数据相对正常（建议和接受行数合理）
- ❌ Composer 数据严重异常：
  - 01-14: 接受率 900%（建议 5 行，接受 45 行）
  - 01-16: 接受率 269.5%（建议 3363 行，接受 9063 行）
  - 01-17, 01-18: 建议行为 0，但有大额接受行数

**可能原因**：
1. Cursor 的 `composerSuggestedLines` 统计逻辑有问题
2. 可能统计的是 Composer 会话中的代码修改，而非 AI 建议的代码
3. 数据聚合时未正确区分"建议"和"实际修改"

## 2. 工作区分析

### 2.1 发现的工作区列表

| 项目名 | 工作区 ID | 路径 | 记录数 | AI Prompts | AI Generations | Composer Data |
|--------|-----------|-------|---------|------------|----------------|---------------|
| cocursor | d4b798d47e9a14d74eb7965f996e8739 | d:/code/cocursor | 176 | 100KB | 30KB | 20KB |
| wecode-1 | bf0871dff1e9e2d800e8ff4f97c9444b | d:/code/wecode | 77 | 502B | 594B | 1.5KB |
| wecode-2 | ca45e3de82bddb73a4fc1291a910f3e9 | d:/code/wecode | 72 | 42B | 267B | 1KB |
| nsq-1 | 8c9de0af69d4b33fe3e10c7cc030549f | d:/2025/github/nsq | 68 | 2B | 2B | 469B |
| nsq-2 | f448e6e71945186fe5a505fbaf753753 | d:/2025/github/nsq | 51 | 2B | 2B | 470B |
| github | c9bf9293c2c47eef24da0f69e3884363 | d:/2025/github | 54 | 2B | 2B | 470B |
| goanalysis | 861ca156f6e5c2aad73afd2854c92261 | c:/Users/TANG/Videos/goanalysis | 89 | 126B | 620B | 1KB |

### 2.2 重名问题分析

发现 2 个重名项目：

1. **wecode** → wecode-1, wecode-2
   - 相同路径：`d:/code/wecode`
   - 可能原因：多次打开、不同的打开方式、或工作区迁移

2. **nsq** → nsq-1, nsq-2
   - 相同路径：`d:/2025/github/nsq`
   - 可能原因：同上

### 2.3 活跃度分析

**最活跃项目**: `cocursor` (d4b798d47e9a14d74eb7965f996e8739)
- AI Prompts: 100,852 bytes (~3,000+ 条对话)
- AI Generations: 30,316 bytes (~100+ 条生成)
- Composer Data: 20,697 bytes (~35+ 个会话)

**说明**: cocursor 项目是当前主要使用的工作区，数据量远超其他项目。

## 3. 最近打开的项目

从全局存储提取的最近打开记录（Top 10）：

1. **file:///d%3A/code/cocursor** - cocursor 项目 ✅
2. **file:///d%3A/code/wecode** - wecode 项目 ✅
3. **file:///d%3A/2025/github/nsq** - nsq 项目 ✅
4. **file:///d%3A/2025/github** - github 目录
5. **file:///d%3A/code/wecode/docs/cursor-database-design.md** - 单个文件
6. **file:///d%3A/code/wecode/.gitignore** - 单个文件
7. **file:///d%3A/code/README.md** - 单个文件
8. **file:///c%3A/Windows/System32/drivers/etc/hosts** - 系统文件
9. **file:///c%3A/Users/TANG/Documents/...** - 微信文件
10. **file:///c%3A/Users/TANG/go/pkg/...** - Go 模块文件

**发现**：
- 前几条是真正的项目路径
- 后面混合了单个文件、系统文件、临时文件
- 需要过滤非项目目录的记录

## 4. 项目名映射建议

### 4.1 建议的项目配置文件

```json
{
  "projects": {
    "cocursor": {
      "name": "cocursor",
      "workspace_id": "d4b798d47e9a14d74eb7965f996e8739",
      "path": "d:/code/cocursor",
      "aliases": ["coc"],
      "active": true
    },
    "wecode": {
      "name": "wecode",
      "workspace_id": "bf0871dff1e9e2d800e8ff4f97c9444b",
      "path": "d:/code/wecode",
      "aliases": ["we"],
      "active": false
    },
    "wecode-old": {
      "name": "wecode-old",
      "workspace_id": "ca45e3de82bddb73a4fc1291a910f3e9",
      "path": "d:/code/wecode",
      "aliases": ["we-old"],
      "active": false,
      "note": "可能是旧的工作区"
    },
    "nsq": {
      "name": "nsq",
      "workspace_id": "8c9de0af69d4b33fe3e10c7cc030549f",
      "path": "d:/2025/github/nsq",
      "aliases": [],
      "active": false
    },
    "nsq-old": {
      "name": "nsq-old",
      "workspace_id": "f448e6e71945186fe5a505fbaf753753",
      "path": "d:/2025/github/nsq",
      "aliases": [],
      "active": false,
      "note": "可能是旧的工作区"
    },
    "github": {
      "name": "github",
      "workspace_id": "c9bf9293c2c47eef24da0f69e3884363",
      "path": "d:/2025/github",
      "aliases": ["gh"],
      "active": false
    },
    "goanalysis": {
      "name": "goanalysis",
      "workspace_id": "861ca156f6e5c2aad73afd2854c92261",
      "path": "c:/Users/TANG/Videos/goanalysis",
      "aliases": [],
      "active": false
    }
  },
  "last_scan": "2026-01-18T10:00:00Z"
}
```

### 4.2 项目命名策略建议

**规则 1**: 使用路径的最后一个目录名作为默认项目名
- `d:/code/cocursor` → `cocursor`
- `d:/2025/github/nsq` → `nsq`

**规则 2**: 重名处理
- 如果发现重名，添加序号或描述性后缀
- `nsq` → `nsq-1`, `nsq-2` 或 `nsq`, `nsq-old`
- 建议让用户手动指定更有意义的名称

**规则 3**: 支持别名
- 为常用项目添加短别名
- `cocursor` → 别名 `coc`
- `github` → 别名 `gh`

## 5. 数据查询接口设计

基于以上分析，建议的查询接口：

### 5.1 按项目名查询

```bash
# 查询 cocursor 项目
cocursor query --project cocursor

# 使用别名查询
cocursor query --project coc

# 查询接受率
cocursor query --project cocursor --stats acceptance

# 查询 AI 对话历史
cocursor query --project cocursor --stats conversations
```

### 5.2 按工作区 ID 查询（高级）

```bash
# 直接查询工作区 ID
cocursor query --workspace d4b798d47e9a14d74eb7965f996e8739
```

### 5.3 自动匹配

```bash
# 根据当前目录自动匹配项目
cd d:/code/cocursor
cocursor query --auto

# 列出所有项目
cocursor list
```

## 6. 实现建议

### 短期（1-2 周）

1. **实现项目配置文件**
   - 位置: `~/.cocursor/projects.json`
   - 支持手动编辑和命令行管理

2. **改进路径匹配**
   - 规范化路径（统一分隔符、解析符号链接）
   - 支持模糊匹配（部分路径、项目名）

3. **实现基本查询命令**
   - `cocursor list` - 列出所有项目
   - `cocursor query --project <name>` - 查询项目数据
   - `cocursor alias --add <project> <alias>` - 添加别名

### 中期（1 个月）

1. **自动发现机制**
   - 启动时扫描所有 Cursor 工作区
   - 智能生成项目名
   - 检测重名并提示用户

2. **数据清洗和修复**
   - 修复 Composer 接受率计算
   - 提供数据质量报告
   - 支持手动修正统计数据

### 长期（2-3 个月）

1. **多维度匹配**
   - Git 远程仓库 URL
   - 项目指纹
   - 路径相似度
   - 用户历史

2. **智能推荐**
   - 根据使用频率推荐项目
   - 根据项目类型推荐查询方式
   - 提供数据可视化

## 7. 数据质量改进

### 7.1 Composer 接受率问题

**问题**: `composerAcceptedLines` > `composerSuggestedLines` 导致接受率超过 100%

**临时解决方案**:
1. 在计算时限制接受率最大值为 100%
2. 添加 `data_quality` 标记异常数据
3. 提供 `warning_message` 说明问题

**长期解决方案**:
1. 从 `composer.composerData` 提取真实的代码变更统计
2. 使用 `totalLinesAdded` 作为"实际修改"的参考
3. 重新定义"接受率"的语义

### 7.2 建议的语义修正

| 当前语义 | 问题 | 建议语义 |
|---------|------|----------|
| Composer 建议行数 | 统计不准确 | Composer 会话中的代码修改行数 |
| Composer 接受行数 | 可能重复统计 | 从 Composer 数据的实际统计 |
| 接受率 | 超过 100% 无意义 | 代码采纳率或修改效率 |

## 8. 总结

### 关键发现

1. **项目数量**: 7 个工作区，6 个唯一项目路径
2. **重名问题**: 2 个项目有重名（nsq, wecode）
3. **数据异常**: Composer 接受率数据严重异常，需要修正
4. **活跃项目**: cocursor 是最活跃的项目，数据量远超其他

### 推荐方案

采用**混合方案**（自动发现 + 用户自定义）：

1. 首次启动自动扫描所有工作区
2. 使用路径生成默认项目名
3. 允许用户重命名、添加别名
4. 支持多维度匹配（路径、项目名、工作区 ID）
5. 提供数据质量报告和修正工具

### 优先级

**P0（必须）**:
- 实现项目配置文件
- 修复 Composer 接受率计算
- 实现基本的查询命令

**P1（重要）**:
- 自动发现机制
- 路径规范化
- 别名支持

**P2（优化）**:
- 多维度匹配
- 智能推荐
- 数据可视化
