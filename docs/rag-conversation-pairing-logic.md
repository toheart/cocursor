# 对话配对逻辑分析

## Transcript 文件格式分析

### 实际格式

从 `parseTranscript` 函数分析，transcript 文件格式：

```
user:
用户消息内容

assistant:
AI 回复内容
[Tool call] tool_name
参数1: 值1
参数2: 值2

[Tool result]
工具调用结果（被跳过，不包含在消息中）

继续 AI 回复内容

assistant:
AI 继续回复（如果 AI 有多条回复）

user:
下一个用户消息
```

### 关键发现

1. **角色标记**：`user:` 和 `assistant:` 是消息边界
2. **Tool Call**：在 `assistant:` 消息内部，使用 `[Tool call]` 标记
3. **Tool Result**：使用 `[Tool result]` 标记，被跳过（不包含在消息内容中）
4. **连续消息**：
   - 可能出现连续的 `assistant:` 消息（AI 多次回复）
   - 可能出现连续的 `user:` 消息（用户多次输入）

### 解析后的消息结构

```go
Message {
    Type: "user" | "ai",
    Text: "消息文本内容（已过滤 tool call/result）",
    Timestamp: 时间戳,
    Tools: []ToolCall,  // 仅 AI 消息有，包含该消息中的所有 tool call
}
```

**注意**：
- Tool call 信息保存在 `Message.Tools` 中，但文本内容中已过滤
- 一个 `assistant:` 消息可能包含多个 tool call
- Tool result 不包含在消息中（被跳过）

## 配对策略设计

### 场景分析

#### 场景 1：标准对话（1对1）
```
user: 问题1
assistant: 回答1
user: 问题2
assistant: 回答2
```
**配对**：Turn[0] = (问题1, 回答1), Turn[1] = (问题2, 回答2)

#### 场景 2：用户连续提问（1对多）
```
user: 问题1
user: 补充问题1
assistant: 回答1和补充问题1
```
**配对**：Turn[0] = (问题1+补充问题1, 回答1和补充问题1)

#### 场景 3：AI 多次回复（1对多）
```
user: 问题1
assistant: 回答1部分1
assistant: 回答1部分2
user: 问题2
```
**配对**：Turn[0] = (问题1, 回答1部分1+回答1部分2)

#### 场景 4：带 Tool Call
```
user: 帮我读取文件
assistant: 我来帮你读取文件
[Tool call] read_file
path: /path/to/file

[Tool result]
文件内容...

文件内容已读取，包含...
```
**配对**：Turn[0] = (帮我读取文件, 我来帮你读取文件\n文件内容已读取，包含...)
**注意**：Tool call 和 result 不包含在文本中，但 Tool 信息保存在 Message.Tools

#### 场景 5：复杂场景（混合）
```
user: 问题1
assistant: 回答1
[Tool call] tool1
assistant: 继续回答1
user: 问题2补充
assistant: 回答2
```
**配对**：
- Turn[0] = (问题1, 回答1+继续回答1)
- Turn[1] = (问题2补充, 回答2)

## 配对算法实现

### 核心逻辑

```go
// pairMessages 将消息配对为对话对
func pairMessages(messages []*Message) []*ConversationTurn {
    var turns []*ConversationTurn
    var currentUserMsgs []*Message  // 累积的用户消息
    var currentAIMsgs []*Message    // 累积的 AI 消息
    
    for _, msg := range messages {
        if msg.Type == MessageTypeUser {
            // 如果之前有未配对的 AI 消息，先创建对话对
            if len(currentAIMsgs) > 0 {
                turn := createTurn(currentUserMsgs, currentAIMsgs)
                turns = append(turns, turn)
                currentUserMsgs = nil
                currentAIMsgs = nil
            }
            // 累积用户消息（可能连续多条）
            currentUserMsgs = append(currentUserMsgs, msg)
            
        } else if msg.Type == MessageTypeAI {
            // 累积 AI 消息（可能连续多条）
            // 注意：即使有 tool call，也属于同一个对话对
            currentAIMsgs = append(currentAIMsgs, msg)
        }
    }
    
    // 处理最后未配对的消息
    if len(currentUserMsgs) > 0 || len(currentAIMsgs) > 0 {
        turn := createTurn(currentUserMsgs, currentAIMsgs)
        turns = append(turns, turn)
    }
    
    return turns
}

// createTurn 创建对话对（合并多条连续消息）
func createTurn(userMsgs, aiMsgs []*Message) *ConversationTurn {
    // 合并用户消息文本
    userText := combineMessageTexts(userMsgs)
    
    // 合并 AI 消息文本
    aiText := combineMessageTexts(aiMsgs)
    
    // 收集所有 tool call（从所有 AI 消息中）
    var allTools []*ToolCall
    for _, aiMsg := range aiMsgs {
        allTools = append(allTools, aiMsg.Tools...)
    }
    
    // 生成对话对文本（用于向量化）
    combinedText := fmt.Sprintf("用户: %s\n\nAI: %s", userText, aiText)
    
    // 生成消息 ID 列表
    userMessageIDs := make([]string, len(userMsgs))
    for i, msg := range userMsgs {
        userMessageIDs[i] = msg.MessageID
    }
    
    aiMessageIDs := make([]string, len(aiMsgs))
    for i, msg := range aiMsgs {
        aiMessageIDs[i] = msg.MessageID
    }
    
    return &ConversationTurn{
        UserMessages:  userMsgs,
        AIMessages:    aiMsgs,
        UserText:      userText,
        AIText:        aiText,
        CombinedText:  combinedText,
        UserMessageIDs: userMessageIDs,
        AIMessageIDs:  aiMessageIDs,
        Tools:         allTools,  // 所有 tool call
        TurnIndex:     len(turns),
        Timestamp:     getFirstTimestamp(userMsgs, aiMsgs),
    }
}

// combineMessageTexts 合并多条消息的文本
func combineMessageTexts(messages []*Message) string {
    if len(messages) == 0 {
        return ""
    }
    
    var parts []string
    for _, msg := range messages {
        text := strings.TrimSpace(msg.Text)
        if text != "" {
            parts = append(parts, text)
        }
    }
    
    return strings.Join(parts, "\n\n")
}
```

### 配对示例

#### 示例 1：标准对话
```
输入:
  [0] user: "如何设计 RAG？"
  [1] ai: "需要向量库和嵌入模型..."
  [2] user: "推荐什么向量库？"
  [3] ai: "推荐 Qdrant..."

配对:
  Turn[0]:
    UserMessages: [0]
    AIMessages: [1]
    CombinedText: "用户: 如何设计 RAG？\n\nAI: 需要向量库和嵌入模型..."
  
  Turn[1]:
    UserMessages: [2]
    AIMessages: [3]
    CombinedText: "用户: 推荐什么向量库？\n\nAI: 推荐 Qdrant..."
```

#### 示例 2：用户连续提问
```
输入:
  [0] user: "如何设计 RAG？"
  [1] user: "需要考虑哪些因素？"
  [2] ai: "需要向量库和嵌入模型，还要考虑..."

配对:
  Turn[0]:
    UserMessages: [0, 1]  // 合并两条用户消息
    AIMessages: [2]
    CombinedText: "用户: 如何设计 RAG？\n\n需要考虑哪些因素？\n\nAI: 需要向量库和嵌入模型，还要考虑..."
```

#### 示例 3：AI 多次回复
```
输入:
  [0] user: "帮我实现 RAG"
  [1] ai: "我来帮你实现"
  [2] ai: "首先需要配置向量库"
  [3] ai: "然后集成嵌入模型"

配对:
  Turn[0]:
    UserMessages: [0]
    AIMessages: [1, 2, 3]  // 合并三条 AI 消息
    CombinedText: "用户: 帮我实现 RAG\n\nAI: 我来帮你实现\n\n首先需要配置向量库\n\n然后集成嵌入模型"
```

#### 示例 4：带 Tool Call
```
输入:
  [0] user: "读取文件内容"
  [1] ai: "我来读取文件"  // Tools: [read_file]
  [2] ai: "文件内容已读取，包含..."

配对:
  Turn[0]:
    UserMessages: [0]
    AIMessages: [1, 2]
    Tools: [read_file]  // 从消息 [1] 中提取
    CombinedText: "用户: 读取文件内容\n\nAI: 我来读取文件\n\n文件内容已读取，包含..."
```

## 索引策略（混合策略）

### 消息级别索引

**索引内容**：每条消息的文本（已过滤 tool call/result）

**用途**：精确匹配、关键词搜索

**示例**：
```json
{
  "id": "composer-xxx:msg-0",
  "vector": [...],
  "payload": {
    "content": "如何设计 RAG？",  // 纯文本，无 tool call
    "message_type": "user",
    "has_tools": false
  }
}
```

### 对话对级别索引

**索引内容**：用户消息 + AI 回复的完整对话对

**用途**：上下文理解、架构讨论

**示例**：
```json
{
  "id": "composer-xxx:turn-0",
  "vector": [...],
  "payload": {
    "user_text": "如何设计 RAG？\n\n需要考虑哪些因素？",
    "ai_text": "需要向量库和嵌入模型，还要考虑跨平台兼容...",
    "combined_text": "用户: 如何设计 RAG？\n\n需要考虑哪些因素？\n\nAI: 需要向量库和嵌入模型，还要考虑跨平台兼容...",
    "user_message_ids": ["msg-0", "msg-1"],
    "ai_message_ids": ["msg-2", "msg-3", "msg-4"],
    "tools": ["read_file", "write_file"],  // 所有 tool call
    "message_count": 5
  }
}
```

## 搜索策略

### 混合搜索

```go
func (s *RAGService) Search(req SearchRequest) (*SearchResponse, error) {
    // 1. 对话对级别搜索（优先，获取上下文）
    turnResults, err := s.searchTurns(req.Query, req.Limit*2)
    
    // 2. 消息级别搜索（精确匹配）
    messageResults, err := s.searchMessages(req.Query, req.Limit*2)
    
    // 3. 合并结果（对话对优先）
    merged := mergeResults(turnResults, messageResults, req.Limit)
    
    return &SearchResponse{
        Results: merged,
        Total:   len(merged),
    }, nil
}

// mergeResults 合并搜索结果，对话对优先
func mergeResults(turnResults, messageResults []SearchResult, limit int) []SearchResult {
    // 1. 对话对结果加权（提高优先级）
    for i := range turnResults {
        turnResults[i].Score *= 1.2  // 对话对加权 20%
    }
    
    // 2. 合并结果
    allResults := append(turnResults, messageResults...)
    
    // 3. 去重（相同会话的相同内容）
    deduped := deduplicateResults(allResults)
    
    // 4. 按分数排序
    sort.Slice(deduped, func(i, j int) bool {
        return deduped[i].Score > deduped[j].Score
    })
    
    // 5. 返回 Top-K
    if len(deduped) > limit {
        return deduped[:limit]
    }
    return deduped
}
```

## UI 展示设计

### 对话对结果展示

```
┌─────────────────────────────────────────┐
│  会话: RAG 功能设计讨论                   │
│  项目: cocursor                          │
│  相似度: 0.85                            │
├─────────────────────────────────────────┤
│  👤 用户:                                │
│  如何设计 RAG 功能？                      │
│  需要考虑哪些因素？                        │
│                                          │
│  🤖 AI:                                  │
│  需要向量库和嵌入模型，推荐使用 Qdrant   │
│  嵌入式模式，支持跨平台自动下载...        │
│                                          │
│  [展开查看单条消息] [查看完整会话]        │
└─────────────────────────────────────────┘
```

### 展开单条消息

```
┌─────────────────────────────────────────┐
│  会话: RAG 功能设计讨论                   │
│  项目: cocursor                          │
│  相似度: 0.85                            │
├─────────────────────────────────────────┤
│  👤 用户消息 (2条):                      │
│  [0] 如何设计 RAG 功能？                 │
│  [1] 需要考虑哪些因素？                  │
│                                          │
│  🤖 AI 回复 (3条):                      │
│  [0] 需要向量库和嵌入模型...             │
│  [1] 推荐使用 Qdrant...                 │
│  [2] 还需要考虑跨平台兼容...             │
│                                          │
│  🔧 工具调用:                            │
│  • read_file (path: ...)                │
│                                          │
│  [收起] [查看完整会话]                   │
└─────────────────────────────────────────┘
```
