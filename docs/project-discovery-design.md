# 项目发现和查询系统设计方案

**版本**: 1.0  
**日期**: 2026-01-18  
**目标**: 以项目名为单位查询 Cursor 数据，支持多工作区合并

---

## 1. 核心需求

### 1.1 功能需求

1. **用户可以使用项目名查询数据**
   - 不需要输入完整路径
   - 示例：`cocursor query --project cocursor`

2. **自动识别同一项目**
   - 同一项目可能有多个工作区（如重名、备份）
   - 查询同一项目时，返回所有工作区的数据

3. **支持跨平台**
   - 前端：VSCode 插件（Windows/Linux/Mac）
   - 后端：Go 服务（Windows/Linux/Mac）
   - 通过 HTTP API 通信

### 1.2 数据处理原则

| 数据类型 | 是否合并 | 说明 |
|---------|----------|------|
| AI Prompts | ❌ 不合并 | 按时间排序，保留 `source` 字段 |
| AI Generations | ❌ 不合并 | 按时间排序，保留 `source` 字段 |
| Composer Sessions | ❌ 不合并 | 按时间排序，保留 `source` 字段 |
| 接受率统计 | ✅ 需要合并 | 累加所有工作区的数据，重新计算接受率 |

---

## 2. 同一项目判断规则

### 2.1 判断优先级

```
P0: Git 远程 URL 相同
   → 99.9% 准确率
   
P1: 物理路径完全相同（解析符号链接）
   → 100% 准确率
   
P2: 项目名相同 + 路径相似度 > 90%
   → 85-95% 准确率
```

### 2.2 详细规则

#### 规则 P0：Git 远程 URL

```go
// 判断条件
if ws1.GitRemoteURL != "" && ws2.GitRemoteURL != "" {
    return normalizeGitURL(ws1.GitRemoteURL) == normalizeGitURL(ws2.GitRemoteURL)
}

// Git URL 规范化
func normalizeGitURL(url string) string {
    // 1. 移除 .git 后缀
    url = strings.TrimSuffix(url, ".git")
    
    // 2. 统一协议
    url = strings.Replace(url, "git@github.com:", "https://github.com/", 1)
    url = strings.Replace(url, "ssh://git@github.com/", "https://github.com/", 1)
    
    // 3. 统一大小写
    return strings.ToLower(url)
}
```

#### 规则 P1：物理路径相同

```go
// 判断条件
realPath1, _ := filepath.EvalSymlinks(ws1.Path)
realPath2, _ := filepath.EvalSymlinks(ws2.Path)
return realPath1 == realPath2
```

#### 规则 P2：项目名相同 + 路径相似度

```go
// 判断条件
if ws1.ProjectName == ws2.ProjectName {
    similarity := calculatePathSimilarity(ws1.Path, ws2.Path)
    return similarity > 0.9  // 阈值 90%
}

// 路径相似度计算（最长公共子序列）
func calculatePathSimilarity(path1, path2 string) float64 {
    // 简化路径：统一分隔符、移除尾部斜杠
    path1 = simplifyPath(path1)
    path2 = simplifyPath(path2)
    
    // 计算最长公共子序列
    lcs := longestCommonSubsequence(path1, path2)
    maxLength := max(len(path1), len(path2))
    similarity := float64(len(lcs)) / float64(maxLength)
    
    return similarity
}

func simplifyPath(path string) string {
    path = strings.ReplaceAll(path, "\\", "/")
    path = strings.TrimRight(path, "/")
    return strings.ToLower(path)
}
```

### 2.3 特殊场景处理

#### 场景 A：Monorepo（不应合并）

```
/workspace/monorepo/service-a  vs /workspace/monorepo/service-b
→ 判断规则：父目录相同，但子目录不同
→ 处理方式：返回 false（不合并），记录到配置文件供用户确认
```

#### 场景 B：Fork 项目（不应合并）

```
git@github.com:user/repo.git vs git@github.com:original/repo.git
→ 判断规则：Git URL 不同
→ 处理方式：返回 false（不合并）
```

---

## 3. 后端设计

### 3.1 数据结构

```go
// 项目信息（包含多个工作区）
type ProjectInfo struct {
    ProjectName   string             `json:"project_name"`            // 项目名称（唯一）
    ProjectID    string             `json:"project_id"`             // 项目唯一 ID
    Workspaces   []*WorkspaceInfo    `json:"workspaces"`            // 包含的所有工作区
    GitRemoteURL string             `json:"git_remote_url,omitempty"` // Git 远程仓库 URL（如果有）
    CreatedAt    time.Time          `json:"created_at"`             // 项目首次发现时间
    LastUpdated time.Time          `json:"last_updated_at"`         // 最后更新时间
}

// 单个工作区信息
type WorkspaceInfo struct {
    WorkspaceID   string `json:"workspace_id"`   // Cursor 工作区 ID
    Path          string `json:"path"`           // 项目路径
    ProjectName   string `json:"project_name"`   // 所属项目名
    GitRemoteURL string `json:"git_remote_url,omitempty"` // Git 远程 URL
    GitBranch     string `json:"git_branch,omitempty"`     // Git 分支
    IsActive     bool   `json:"is_active"`      // 是否为当前活跃的工作区
    IsPrimary    bool   `json:"is_primary"`     // 是否为主工作区（最新的）
}

// 项目管理器（内存缓存）
type ProjectManager struct {
    mu           sync.RWMutex
    projects     map[string]*ProjectInfo  // project_name -> *ProjectInfo
    pathMap      map[string]string        // normalized path -> project_name
    discovery    *ProjectDiscovery
}
```

### 3.2 后端启动流程

```
1. 后端服务启动
   ↓
2. 初始化 ProjectManager
   ↓
3. 扫描所有 Cursor 工作区
   - 读取 workspaceStorage 目录
   - 解析每个工作区的 workspace.json
   - 读取每个工作区的 Git 信息（如果 .git 存在）
   ↓
4. 按"同一项目"规则分组
   - 使用 P0 > P1 > P2 优先级
   - 生成 ProjectInfo（包含多个 WorkspaceInfo）
   ↓
5. 保存到内存
   - 保存到 ProjectManager.projects
   - 建立路径映射 ProjectManager.pathMap
   ↓
6. 完成，准备接受查询
```

### 3.3 分组算法

```go
func (pm *ProjectManager) groupBySameProject(workspaces []*WorkspaceInfo) map[string]*ProjectInfo {
    groups := make(map[string]*ProjectInfo)
    processed := make(map[string]bool)
    
    for _, ws := range workspaces {
        if processed[ws.WorkspaceID] {
            continue
        }
        
        // 查找所有属于同一项目的工作区
        sameProject := pm.findSameProject(ws, workspaces)
        
        // 生成项目唯一标识符
        projectKey := pm.generateProjectKey(sameProject)
        
        // 创建或更新 ProjectInfo
        if existing, exists := groups[projectKey]; exists {
            // 已存在，添加新的工作区
            existing.Workspaces = append(existing.Workspaces, sameProject...)
            existing.LastUpdated = time.Now()
            
            // 重新判断哪个是主工作区（最新的）
            pm.updatePrimaryWorkspace(existing)
        } else {
            // 新项目，创建 ProjectInfo
            groups[projectKey] = &ProjectInfo{
                ProjectName:   projectKey,
                ProjectID:    projectKey,
                Workspaces:   sameProject,
                GitRemoteURL: sameProject[0].GitRemoteURL,
                CreatedAt:    time.Now(),
                LastUpdated:  time.Now(),
            }
        }
        
        // 标记已处理
        for _, s := range sameProject {
            processed[s.WorkspaceID] = true
        }
    }
    
    return groups
}
```

---

## 4. 前端设计

### 4.1 插件激活和上报

```typescript
// co-extension/src/extension.ts

import { checkAndReportProject } from './utils/projectReporter';
import { watchWorkspaceChanges } from './utils/workspaceDetector';

export function activate(context: vscode.ExtensionContext) {
    // 1. 立即检测并上报当前项目
    checkAndReportProject();
    
    // 2. 监听工作区变化
    const watcher = watchWorkspaceChanges((newPath) => {
        console.log('检测到工作区变化:', newPath);
        checkAndReportProject();
    });
    
    // 3. 清理
    context.subscriptions.push(watcher);
}
```

### 4.2 上报内容

```typescript
// co-extension/src/services/api.ts

interface ProjectReportRequest {
    path: string;      // 当前工作区路径
    timestamp: number;  // 时间戳
}

interface ProjectReportResponse {
    success: boolean;
    project_name: string;     // 后端确认的项目名
    project_id: string;        // 项目唯一 ID
    is_active: boolean;        // 是否更新成功
    message?: string;
}

// 上报当前项目
export async function reportCurrentProject(): Promise<void> {
    const path = getCurrentWorkspacePath();
    
    if (!path) {
        console.warn('无法获取当前工作区路径');
        return;
    }
    
    const response = await fetch('http://localhost:8080/api/v1/project/activate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            path: path,
            timestamp: Date.now(),
        } as ProjectReportRequest),
    });
    
    if (response.ok) {
        const result = await response.json();
        console.log('项目上报成功:', result);
    } else {
        console.error('项目上报失败:', response.status);
    }
}
```

---

## 5. API 设计

### 5.1 后端 API

#### POST /api/v1/project/activate
**功能**：接收前端上报的当前项目，更新活跃状态

**请求**：
```json
{
  "path": "d:/code/cocursor",
  "timestamp": 1737166400000
}
```

**响应**：
```json
{
  "success": true,
  "project_name": "cocursor",
  "project_id": "cocursor",
  "is_active": true,
  "message": "活跃状态已更新"
}
```

#### GET /api/v1/project/list
**功能**：列出所有已发现的项目

**响应**：
```json
{
  "success": true,
  "projects": [
    {
      "project_name": "cocursor",
      "project_id": "cocursor",
      "workspaces": [
        {
          "workspace_id": "d4b798d47e9a14d74eb7965f996e8739",
          "path": "d:/code/cocursor",
          "git_remote_url": "git@github.com:user/cocursor.git",
          "is_active": true,
          "is_primary": true
        }
      ],
      "git_remote_url": "git@github.com:user/cocursor.git",
      "created_at": "2026-01-18T10:00:00Z",
      "last_updated_at": "2026-01-18T10:05:30Z"
    }
  ],
  "total": 6
}
```

#### GET /api/v1/project/{project_name}/prompts
**功能**：查询项目的 AI 对话历史（不合并）

**参数**：
- `project_name`: 项目名称

**响应**：
```json
{
  "success": true,
  "project_name": "cocursor",
  "workspaces": [...],
  "prompts": [
    {
      "text": "安装make命令",
      "commandType": 4,
      "timestamp": 1768643511672,
      "source": "d4b798d47e9a14d74eb7965f996e8739"  // 来源工作区
    },
    ...
  ],
  "total": 3620
}
```

#### GET /api/v1/project/{project_name}/stats/acceptance
**功能**：查询项目的接受率统计（合并）

**响应**：
```json
{
  "success": true,
  "project_name": "cocursor",
  "workspaces": [
    {
      "workspace_id": "d4b798d47e9a14d74eb7965f996e8739",
      "raw_stats": {
        "tab_suggested_lines": 0,
        "tab_accepted_lines": 0,
        "composer_suggested_lines": 5,
        "composer_accepted_lines": 45
      }
    },
    {
      "workspace_id": "other-workspace-id",
      "raw_stats": {
        "tab_suggested_lines": 40,
        "tab_accepted_lines": 11,
        "composer_suggested_lines": 3363,
        "composer_accepted_lines": 9063
      }
    }
  ],
  "merged_stats": {
    "tab_suggested_lines": 40,
    "tab_accepted_lines": 11,
    "tab_acceptance_rate": 27.5,
    "composer_suggested_lines": 3368,
    "composer_accepted_lines": 9108,
    "composer_acceptance_rate": 270.2,  // 异常，已标记
    "data_quality": "warning",
    "warning_message": "Composer 接受率异常：建议 3368 行，接受 9108 行（270.2%）"
  }
}
```

---

## 6. 前端 UI 设计

### 6.1 项目列表视图

```
┌─────────────────────────────────────────┐
│  Projects                         │
├─────────────────────────────────────────┤
│  📂 cocursor (Active)            │
│     path: d:/code/cocursor         │
│     workspaces: 1                  │
│  📂 wecode                        │
│     path: d:/code/wecode           │
│     workspaces: 2                  │
├─────────────────────────────────────────┤
│  [Refresh]  [Settings]            │
└─────────────────────────────────────────┘
```

### 6.2 项目详情视图

```
┌──────────────────────────────────────────────────┐
│  cocursor Project Details                   │
├──────────────────────────────────────────────────┤
│  📊 Statistics                           │
│  ├─ AI Conversations: 3,620            │
│  ├─ AI Generations: 120                 │
│  ├─ Composer Sessions: 35                │
│  └─ Acceptance Rate: 0% (warning)       │
├──────────────────────────────────────────────────┤
│  📝 Recent Activity                       │
│  ├─ [Prompts] [Generations] [Sessions]    │
│  └─ Timeline View                        │
└──────────────────────────────────────────────────┘
```

---

## 7. 配置文件（可选）

### 7.1 位置和格式

```
位置：C:\Users\TANG\.cocursor\projects.json
格式：JSON
```

### 7.2 配置示例

```json
{
  "projects": {
    "cocursor": {
      "name": "cocursor",
      "project_id": "cocursor",
      "workspaces": {
        "d4b798d47e9a14d74eb7965f996e8739": {
          "path": "d:/code/cocursor",
          "git_remote_url": "git@github.com:user/cocursor.git",
          "is_primary": true
        }
      },
      "git_remote_url": "git@github.com:user/cocursor.git",
      "created_at": "2026-01-18T10:00:00Z"
    }
  },
  "settings": {
    "auto_discovery": true,
    "path_similarity_threshold": 0.9,
    "merge_strategy": "strict"
  }
}
```

### 7.3 配置说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `auto_discovery` | 是否自动发现项目 | `true` |
| `path_similarity_threshold` | 路径相似度阈值（0.0-1.0） | `0.9` |
| `merge_strategy` | 合并策略：`strict`/`relaxed` | `strict` |

---

## 8. 实施优先级

### P0（必须）- 第一周

1. ✅ 后端启动时扫描所有工作区
2. ✅ 实现同一项目判断规则（P0、P1、P2）
3. ✅ 实现项目分组算法
4. ✅ 实现基础查询 API（project/list、prompts、stats）
5. ✅ 前端插件激活时上报当前项目
6. ✅ 前端监听工作区变化

### P1（重要）- 第二周

1. ✅ 实现前端 UI（项目列表、详情视图）
2. ✅ 实现高级查询 API（generations、sessions）
3. ✅ 添加配置文件支持（可选）
4. ✅ 优化路径相似度算法

### P2（优化）- 后续迭代

1. ⏳ Monorepo 检测和提示
2. ⏳ Fork 项目检测和区分
3. ⏳ 项目使用历史记录
4. ⏳ 数据可视化（图表、趋势图）

---

## 9. 关键技术点

### 9.1 Git 信息读取

```go
// 读取 .git/config 获取远程 URL
func readGitRemoteURL(projectPath string) string {
    gitConfigPath := filepath.Join(projectPath, ".git", "config")
    
    if !fileExists(gitConfigPath) {
        return ""
    }
    
    content, err := os.ReadFile(gitConfigPath)
    if err != nil {
        return ""
    }
    
    // 解析配置文件
    lines := strings.Split(string(content), "\n")
    for i, line := range lines {
        if strings.Contains(line, "[remote \"") {
            if i+1 < len(lines) {
                nextLine := lines[i+1]
                if strings.Contains(nextLine, "url =") {
                    url := strings.TrimSpace(strings.TrimPrefix(nextLine, "url = "))
                    return url
                }
            }
        }
    }
    
    return ""
}
```

### 9.2 符号链接解析

```go
// 解析符号链接获取真实路径
func resolveSymlinks(path string) string {
    realPath, err := filepath.EvalSymlinks(path)
    if err != nil {
        return path
    }
    return realPath
}
```

### 9.3 路径规范化

```go
// 规范化路径（跨平台）
func normalizePath(path string) string {
    // 1. 统一分隔符为 /
    path = strings.ReplaceAll(path, "\\", "/")
    
    // 2. 移除尾部斜杠
    path = strings.TrimRight(path, "/")
    
    // 3. 转小写（Windows 大小写不敏感）
    return strings.ToLower(path)
}
```

---

## 10. 风险和缓解

### 10.1 识别风险

| 风险 | 影响 | 概率 | 缓解措施 |
|--------|------|------|----------|
| 路径相似度阈值设置不当 | 误判不同项目为同一项目 | 中 | 可配置阈值，提供用户确认 |
| Git 远程 URL 不标准 | 无法匹配同一项目 | 低 | 尝试多种 URL 格式 |
| Monorepo 子项目被误合并 | 数据混乱 | 中 | 检测父目录和子目录关系 |
| 符号链接循环 | 性能问题 | 低 | 限制解析深度 |
| 工作区数据损坏 | 扫描失败 | 低 | 异常处理，记录错误日志 |

### 10.2 缓存失效

**场景**：用户打开新项目，但后端未重启

**解决方案**：
- 前端上报触发缓存更新
- 提供手动刷新接口

---

## 11. 总结

### 11.1 核心特性

✅ **后端启动时自动扫描**：无需依赖前端上报  
✅ **智能项目判断**：Git URL > 物理路径 > 路径相似度  
✅ **数据合并策略**：统计数据合并，原始数据排序  
✅ **跨平台支持**：Windows/Linux/Mac 统一处理  
✅ **最小化前端上报**：只用于活跃状态更新  

### 11.2 技术栈

- **后端**：Go + Gin
- **前端**：TypeScript + VSCode API
- **通信**：HTTP REST API
- **存储**：内存缓存（启动时加载）

### 11.3 预期效果

- ✅ 用户可以用项目名查询，无需记忆路径
- ✅ 同一项目的多个工作区自动合并统计
- ✅ 前端实时显示活跃状态
- ✅ 配置灵活，支持特殊场景

---

**文档版本**: 1.0  
**最后更新**: 2026-01-18
