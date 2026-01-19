# RAG 上下文策略讨论

## 问题分析

### 用户场景
- 在与 Cursor 聊天过程中会产生很多**架构和原理讨论**
- 在进行功能迭代时，希望**参考以往的聊天内容**
- 架构讨论通常是**多轮对话**，需要上下文才能理解

### 当前设计的问题
- **每条消息单独索引** → 丢失对话上下文
- **搜索返回单条消息** → 无法理解完整的讨论过程
- **架构讨论被拆分** → 难以理解整体思路

### 示例场景

```
用户: 如何设计 RAG 功能？
AI: 需要向量库和嵌入模型，推荐使用 Qdrant...
用户: Qdrant 如何与现有系统集成？
AI: 可以使用嵌入式模式，通过 Go SDK...
用户: 如何保证跨平台兼容？
AI: 需要自动下载对应平台的二进制...
```

**问题**：如果每条消息单独索引，搜索"RAG 设计"可能只返回第一条消息，丢失了后续的集成和兼容性讨论。

## 解决方案对比

### 方案 1：对话对（Turn）索引 ⭐ 推荐

**策略**：将用户消息 + 对应的 AI 回复作为一个单元索引

**实现**：
```go
// 将消息配对
type ConversationTurn struct {
    UserMessage    *Message  // 用户消息
    AIMessage      *Message  // AI 回复
    TurnIndex      int       // 对话轮次
}

// 索引时组合文本
turnText := fmt.Sprintf("用户: %s\n\nAI: %s", 
    turn.UserMessage.Text, 
    turn.AIMessage.Text)

// 向量化整个对话对
vector := embedText(turnText)
```

**优点**：
- ✅ 保持对话完整性
- ✅ 理解用户问题和 AI 回答的对应关系
- ✅ 适合架构讨论场景
- ✅ 实现简单

**缺点**：
- ❌ 如果用户连续发多条消息，配对可能不准确
- ❌ AI 回复很长时，向量可能不够精确

**适用场景**：架构讨论、问题解答、设计决策

---

### 方案 2：滑动窗口索引

**策略**：将连续的 N 条消息作为一个块索引（带重叠）

**实现**：
```go
// 滑动窗口，窗口大小 = 4 条消息，重叠 = 2 条
windowSize := 4
overlap := 2

for i := 0; i < len(messages); i += (windowSize - overlap) {
    window := messages[i:min(i+windowSize, len(messages))]
    windowText := combineMessages(window)
    vector := embedText(windowText)
}
```

**优点**：
- ✅ 保持上下文窗口
- ✅ 不丢失边界信息（通过重叠）
- ✅ 适合长对话

**缺点**：
- ❌ 存储空间增加（重叠部分重复索引）
- ❌ 边界处理复杂
- ❌ 可能包含不相关的消息

**适用场景**：长对话、多主题讨论

---

### 方案 3：混合策略 ⭐⭐ 最佳

**策略**：同时索引消息级别和对话级别

**实现**：
```go
// 1. 消息级别索引（精确匹配）
for _, msg := range messages {
    indexMessage(msg)  // 单条消息向量
}

// 2. 对话对级别索引（上下文理解）
turns := pairMessages(messages)
for _, turn := range turns {
    indexTurn(turn)  // 对话对向量
}
```

**存储结构**：
- Qdrant Collection: `cursor_sessions_messages` (消息级别)
- Qdrant Collection: `cursor_sessions_turns` (对话对级别)

**搜索策略**：
- 先搜索对话对级别（获取上下文）
- 再搜索消息级别（精确匹配）
- 合并结果，去重排序

**优点**：
- ✅ 兼顾精确匹配和上下文理解
- ✅ 灵活，可以根据查询类型选择
- ✅ 适合多种场景

**缺点**：
- ❌ 存储空间增加（约 2 倍）
- ❌ 实现复杂度提高
- ❌ 需要维护两套索引

**适用场景**：通用场景，需要兼顾多种查询需求

---

### 方案 4：分层索引

**策略**：消息级别 + 对话对级别 + 会话摘要级别

**实现**：
```go
// 1. 消息级别
indexMessages(messages)

// 2. 对话对级别
indexTurns(pairMessages(messages))

// 3. 会话摘要级别
summary := summarizeSession(session)
indexSummary(summary)
```

**优点**：
- ✅ 多层次理解
- ✅ 支持不同粒度的搜索

**缺点**：
- ❌ 存储空间大幅增加
- ❌ 实现复杂度很高
- ❌ 会话摘要需要额外的 LLM 调用

**适用场景**：需要非常全面的搜索能力

---

## 推荐方案：混合策略（方案 3）

### 设计细节

#### 1. 索引结构

**消息级别索引**：
```json
{
  "collection": "cursor_sessions_messages",
  "point": {
    "id": "composer-xxx:msg-0",
    "vector": [...],
    "payload": {
      "session_id": "composer-xxx",
      "message_id": "msg-0",
      "message_type": "user",
      "content": "如何设计 RAG 功能？",
      "turn_index": 0,  // 所属对话轮次
      ...
    }
  }
}
```

**对话对级别索引**：
```json
{
  "collection": "cursor_sessions_turns",
  "point": {
    "id": "composer-xxx:turn-0",
    "vector": [...],
    "payload": {
      "session_id": "composer-xxx",
      "turn_index": 0,
      "user_message": "如何设计 RAG 功能？",
      "ai_message": "需要向量库和嵌入模型...",
      "combined_text": "用户: 如何设计 RAG 功能？\n\nAI: 需要向量库和嵌入模型...",
      "message_count": 2,
      ...
    }
  }
}
```

#### 2. 消息配对逻辑

```go
func pairMessages(messages []*Message) []*ConversationTurn {
    var turns []*ConversationTurn
    var currentUserMsg *Message
    
    for _, msg := range messages {
        if msg.Type == MessageTypeUser {
            // 如果之前有未配对的用户消息，创建一个只有用户消息的 turn
            if currentUserMsg != nil {
                turns = append(turns, &ConversationTurn{
                    UserMessage: currentUserMsg,
                    AIMessage:  nil,  // 可能没有 AI 回复
                    TurnIndex:  len(turns),
                })
            }
            currentUserMsg = msg
        } else if msg.Type == MessageTypeAI {
            // 配对用户消息和 AI 回复
            turn := &ConversationTurn{
                UserMessage: currentUserMsg,
                AIMessage:  msg,
                TurnIndex:  len(turns),
            }
            turns = append(turns, turn)
            currentUserMsg = nil  // 已配对，清空
        }
    }
    
    // 处理最后一条未配对的用户消息
    if currentUserMsg != nil {
        turns = append(turns, &ConversationTurn{
            UserMessage: currentUserMsg,
            AIMessage:  nil,
            TurnIndex:  len(turns),
        })
    }
    
    return turns
}
```

#### 3. 搜索策略

```go
func (s *RAGService) Search(req SearchRequest) (*SearchResponse, error) {
    // 1. 对话对级别搜索（优先，获取上下文）
    turnResults, err := s.searchTurns(req.Query, req.Limit*2)  // 获取更多结果
    
    // 2. 消息级别搜索（精确匹配）
    messageResults, err := s.searchMessages(req.Query, req.Limit*2)
    
    // 3. 合并和去重
    merged := mergeResults(turnResults, messageResults)
    
    // 4. 重新排序（对话对优先，因为包含更多上下文）
    sorted := sortByRelevance(merged)
    
    // 5. 返回 Top-K
    return &SearchResponse{
        Results: sorted[:req.Limit],
        Total:   len(sorted),
    }, nil
}
```

#### 4. 结果展示

**对话对结果**：
```
┌─────────────────────────────────────────┐
│  会话: RAG 功能设计讨论                   │
│  项目: cocursor                          │
│  相似度: 0.85                            │
├─────────────────────────────────────────┤
│  用户: 如何设计 RAG 功能？               │
│                                          │
│  AI: 需要向量库和嵌入模型，推荐使用      │
│      Qdrant 嵌入式模式，通过 Go SDK      │
│      集成，支持跨平台自动下载...         │
└─────────────────────────────────────────┘
```

**消息结果**（如果对话对未匹配）：
```
┌─────────────────────────────────────────┐
│  会话: RAG 功能设计讨论                   │
│  项目: cocursor                          │
│  相似度: 0.78                            │
├─────────────────────────────────────────┤
│  用户: 如何设计 RAG 功能？               │
│                                          │
│  [查看完整对话]                          │
└─────────────────────────────────────────┘
```

---

## 方案对比总结

| 方案 | 上下文保持 | 实现复杂度 | 存储空间 | 搜索精度 | 推荐度 |
|------|-----------|-----------|---------|---------|--------|
| 单消息索引 | ❌ | ⭐ | ⭐ | ⭐⭐⭐ | ❌ |
| 对话对索引 | ⭐⭐ | ⭐⭐ | ⭐⭐ | ⭐⭐ | ⭐⭐⭐ |
| 滑动窗口 | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ | ⭐⭐ |
| **混合策略** | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| 分层索引 | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |

---

## 建议

### 阶段 1：先实现对话对索引（MVP）
- 快速验证效果
- 实现简单
- 满足架构讨论场景

### 阶段 2：升级到混合策略（优化）
- 如果对话对索引效果不够好
- 需要更精确的匹配
- 存储和计算资源充足

### 配置选项
允许用户选择索引策略：
- 仅对话对索引（节省空间）
- 仅消息索引（精确匹配）
- 混合索引（最佳效果，默认）

---

## 需要讨论的问题

1. **存储空间**：混合策略需要约 2 倍存储空间，是否可以接受？
2. **实现优先级**：先实现对话对索引，还是直接实现混合策略？
3. **配对准确性**：如果用户连续发多条消息，如何配对？
4. **搜索权重**：对话对和消息的搜索结果如何合并和排序？
