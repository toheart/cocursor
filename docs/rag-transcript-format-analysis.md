# Transcript 文件格式分析

## 实际对话格式

### 基本结构

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
工具结果内容（被跳过，不包含在消息中）

继续的 AI 文本内容

assistant:
另一条 AI 消息（可能连续多条）

user:
下一条用户消息
```

### 关键发现

1. **角色标记**：
   - `user:` - 用户消息开始
   - `assistant:` - AI 消息开始

2. **工具调用**：
   - `[Tool call] tool_name` - 工具调用标记
   - 工具参数在下一行开始，直到空行或下一个标记
   - 工具调用**附加到 AI 消息**（存储在 `msg.Tools` 字段）
   - **工具结果被跳过**，不包含在消息文本中

3. **消息边界**：
   - 遇到新的 `user:` 或 `assistant:` 时，保存上一条消息
   - 一个 `assistant:` 标记 = 一条 AI 消息（可能包含多个工具调用）

### 实际对话示例

```
user:
如何设计 RAG 功能？

assistant:
需要向量库和嵌入模型。让我先查看一下相关代码。
[Tool call] read_file
path: backend/internal/application/cursor/session_service.go

[Tool result]
文件内容...

根据代码分析，我建议使用 Qdrant。

assistant:
另外，还需要考虑跨平台兼容性。
[Tool call] codebase_search
query: How to handle cross-platform dependencies?

[Tool result]
搜索结果...

建议使用纯 Go 实现。

user:
Qdrant 如何集成？

assistant:
可以使用嵌入式模式，通过 Go SDK 集成。
```

**解析后的消息结构**：
```
Message[0]: Type=user, Text="如何设计 RAG 功能？"
Message[1]: Type=ai, Text="需要向量库和嵌入模型。让我先查看一下相关代码。\n\n根据代码分析，我建议使用 Qdrant。", Tools=[read_file]
Message[2]: Type=ai, Text="另外，还需要考虑跨平台兼容性。\n\n建议使用纯 Go 实现。", Tools=[codebase_search]
Message[3]: Type=user, Text="Qdrant 如何集成？"
Message[4]: Type=ai, Text="可以使用嵌入式模式，通过 Go SDK 集成。"
```

## 配对逻辑分析

### 观察

1. **用户消息**：通常一条 `user:` 对应一条用户消息
2. **AI 消息**：可能连续多条 `assistant:` 标记
   - 原因：AI 可能分多次回复
   - 或者：工具调用后继续回复

### 配对策略

**策略：按时间顺序配对，合并连续的 AI 消息**

```go
// 配对逻辑：智能处理连续消息
func pairMessages(messages []*Message) []*ConversationTurn {
    var turns []*ConversationTurn
    var currentUserMsgs []*Message
    var currentAIMsgs []*Message
    
    for _, msg := range messages {
        if msg.Type == MessageTypeUser {
            // 如果之前有未配对的 AI 消息，先创建对话对
            if len(currentAIMsgs) > 0 {
                turn := createTurn(currentUserMsgs, currentAIMsgs)
                turns = append(turns, turn)
                currentUserMsgs = nil
                currentAIMsgs = nil
            }
            // 开始新的用户消息（通常只有一条）
            currentUserMsgs = []*Message{msg}
            
        } else if msg.Type == MessageTypeAI {
            // 累积 AI 消息（可能有多条连续）
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
```

### 配对示例

**场景 1：标准对话**
```
user: "如何设计 RAG？"
assistant: "需要向量库..."
→ Turn[0]: User=[msg0], AI=[msg1]
```

**场景 2：AI 连续回复多条**
```
user: "如何设计 RAG？"
assistant: "需要向量库..."
assistant: "还需要考虑跨平台..."
→ Turn[0]: User=[msg0], AI=[msg1, msg2]  // 合并两条 AI 消息
```

**场景 3：AI 使用工具后继续回复**
```
user: "如何设计 RAG？"
assistant: "让我查看代码..." [Tool call]
assistant: "根据代码，建议使用 Qdrant"
→ Turn[0]: User=[msg0], AI=[msg1, msg2]  // 合并，包含工具调用
```

**场景 4：用户连续发多条（少见但可能）**
```
user: "如何设计 RAG？"
user: "需要考虑哪些因素？"
assistant: "需要向量库和嵌入模型..."
→ Turn[0]: User=[msg0, msg1], AI=[msg2]  // 合并两条用户消息
```

## 工具调用的处理

### 当前实现

- **工具调用**：存储在 `msg.Tools` 字段，**不包含在文本中**
- **工具结果**：被跳过，不包含在消息中

### RAG 索引策略

**选项 1：只索引文本内容（推荐）**
- 工具调用不参与向量化
- 工具结果不参与向量化
- 只索引用户和 AI 的文本对话

**选项 2：包含工具调用描述**
- 将工具调用转换为文本描述
- 例如：`[调用了 read_file 工具，参数: path=xxx]`
- 增加上下文信息

**推荐：选项 1**，因为：
- 工具调用是执行细节，不是对话内容
- 用户搜索时更关心"说了什么"，而不是"调用了什么工具"
- 简化实现，减少噪音

## 更新后的配对逻辑

```go
// 创建对话对（合并多条连续消息）
func createTurn(userMsgs, aiMsgs []*Message) *ConversationTurn {
    // 合并用户消息文本
    userText := combineMessageTexts(userMsgs)
    
    // 合并 AI 消息文本（不包含工具调用）
    aiText := combineMessageTexts(aiMsgs)
    
    // 组合对话对文本
    combinedText := fmt.Sprintf("用户: %s\n\nAI: %s", userText, aiText)
    
    return &ConversationTurn{
        UserMessages: userMsgs,
        AIMessages:  aiMsgs,
        UserText:    userText,
        AIText:      aiText,
        CombinedText: combinedText,
        TurnIndex:   len(turns),
        Timestamp:   getFirstTimestamp(userMsgs, aiMsgs),
    }
}

// 合并消息文本（跳过工具调用）
func combineMessageTexts(messages []*Message) string {
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
