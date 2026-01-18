# 分析仪表板和会话查看功能

## 概述

分析仪表板功能提供了三个核心能力：
1. **Token 消耗统计** - 在侧边栏实时显示今日 Token 使用情况
2. **工作分析** - 通过 WebView 页面展示代码变更趋势、文件分析、时间分布等
3. **会话查看** - 通过 WebView 查看和重新渲染历史对话

## Token 消耗统计

### 数据来源

Token 消耗通过分析 `aiService.prompts` 和 `aiService.generations` 的文本内容进行估算：
- **估算方法**：中文字符按 1.5 字符/token，其他字符按 4 字符/token
- **分类统计**：根据 `generations.type` 字段分类（tab/composer/chat）
- **趋势对比**：与昨日 Token 使用量对比，计算百分比变化

### 侧边栏展示

- **位置**：CoCursor 侧边栏顶部
- **显示内容**：
  - 今日总 Token 数（格式化：K/M）
  - 趋势指示器（↑/↓ + 百分比）
  - 可展开显示分类统计（Tab/Composer/Chat）
- **更新频率**：每 5 分钟自动刷新

### API

```
GET /api/v1/stats/token-usage?date=YYYY-MM-DD&project_name=xxx
```

响应格式：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "date": "2026-01-17",
    "total_tokens": 12500,
    "by_type": {
      "tab": 2100,
      "composer": 8300,
      "chat": 2100
    },
    "trend": "+15%"
  }
}
```

## 工作分析

### 功能模块

1. **概览卡片**
   - 总代码变更（添加/删除行数、文件数）
   - 平均接受率
   - 活跃会话数

2. **代码变更趋势**
   - 时间序列图表（按日期）
   - 显示添加/删除行数、变更文件数

3. **文件分析**
   - Top N 最常编辑文件
   - 文件引用次数统计

4. **时间分布**
   - 24 小时 × 7 天的热力图数据
   - 显示活跃时段分布

5. **效率指标**
   - 平均会话熵值
   - 平均上下文使用率
   - 熵值趋势（按日期）

### 跨项目聚合

- **不提供 `project_name`**：聚合所有项目的数据
- **提供 `project_name`**：只分析指定项目
- **数据合并策略**：累加所有工作区的统计数据

### API

```
GET /api/v1/stats/work-analysis?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD&project_name=xxx
```

## 会话查看

### 会话列表

- **虚拟滚动**：使用 `react-window` 支持大量会话
- **分页加载**：默认每页 20 条，最大 100 条
- **搜索过滤**：支持按会话名称搜索
- **排序**：按最后更新时间倒序

### 会话详情

- **对话重建**：组合 `prompts` + `generations` 按时间排序
- **消息渲染**：
  - 用户消息（右侧气泡）
  - AI 消息（左侧气泡）
  - 代码块高亮（使用 highlight.js）
  - 文件引用显示
- **消息限制**：默认加载 100 条，最大 1000 条

### 数据结构关联

由于 `prompts` 没有时间戳字段，采用以下策略：
- **Generations**：使用 `unixMs` 精确时间戳
- **Prompts**：根据数组索引估算时间（假设每条消息间隔 1 分钟）
- **时间范围过滤**：只包含会话创建时间到更新时间范围内的消息

### API

```
GET /api/v1/sessions/list?project_name=xxx&limit=20&offset=0&search=keyword
GET /api/v1/sessions/:sessionId/detail?limit=100
```

## WebView 路由系统

### 路由结构

- `/` - 默认页面（原有 App 组件）
- `/work-analysis` - 工作分析页面
- `/sessions` - 会话列表页面
- `/sessions/:sessionId` - 会话详情页面

### 技术实现

- **路由库**：react-router-dom（使用 HashRouter，适配 WebView）
- **导航方式**：
  - 侧边栏点击 → 打开 WebView 并导航到指定路由
  - 页面内导航 → 使用 React Router 的 `useNavigate`

## Token 计算机制

### 当前实现

基于文本字符数的粗略估算：
- 中文字符：1.5 字符/token
- 其他字符：4 字符/token

### 优化方向

1. **使用 tiktoken 库**：更精确的 Token 计算
2. **SQLite 探索**：查找 Cursor 是否直接存储 Token 数据
3. **缓存机制**：避免重复计算

## 工作分析算法

### 代码变更趋势

- 按日期聚合所有会话的代码变更
- 统计每日的添加行数、删除行数、变更文件数

### 文件引用分析

- 从 `composer.subtitle` 字段提取文件列表
- 统计每个文件被引用的次数
- 按引用次数排序，返回 Top N

### 时间分布

- 从会话创建时间提取小时和星期
- 统计每个时段（小时 × 星期）的活跃次数
- 生成热力图数据（24 × 7 矩阵）

### 效率指标

- **平均熵值**：所有会话熵值的平均值
- **平均上下文使用率**：所有会话上下文使用率的平均值
- **熵值趋势**：按日期计算平均熵值

## 会话数据结构说明

### 数据来源

1. **`composer.composerData`** - 会话元数据
   - 会话 ID、名称、时间戳
   - 代码变更统计
   - 文件引用列表

2. **`aiService.prompts`** - 用户输入
   - 格式：`[{"text": "...", "commandType": 4}, ...]`
   - 无时间戳字段

3. **`aiService.generations`** - AI 回复
   - 格式：`[{"unixMs": ..., "type": "...", "textDescription": "..."}, ...]`
   - 有时间戳字段

### 消息组合策略

由于 prompts 和 generations 没有直接关联字段，采用时间排序策略：
1. 收集所有 prompts（估算时间戳）
2. 收集所有 generations（使用实际时间戳）
3. 按时间戳排序
4. 限制数量（默认 100 条）

### 代码块识别

- 支持 Markdown 格式：`` ```language\ncode\n``` ``
- 自动提取语言和代码内容
- 使用 highlight.js 进行语法高亮

## 性能优化

1. **虚拟滚动**：会话列表使用 react-window，只渲染可见项
2. **消息限制**：单次加载最多 100 条消息
3. **分页加载**：会话列表支持分页，避免一次性加载大量数据
4. **缓存机制**：Token 数据缓存 5 分钟，减少 API 调用

## 待完善功能

1. **图表集成**：工作分析页面的图表需要集成 recharts
2. **项目列表加载**：工作分析页面的项目选择器需要加载项目列表
3. **Token 精确计算**：探索 SQLite 数据库，查找更精确的 Token 数据
4. **消息关联优化**：如果找到 prompts 和 generations 的关联字段，优化消息组合逻辑
