# Issues and Optimizations Analysis

Extract issues and optimization opportunities from conversation content.

## Issue Categories

### 1. Context Loss (context_loss)

**Severity Levels:**
- **High**: Frequent context loss, major rework needed
- **Medium**: Occasional context loss, minor rework
- **Low**: Rare context loss, quick recovery

**Patterns:**
- "忘记之前的讨论"
- "需要重新解释"
- "之前说过..."
- "重复一遍"
- "Lost context"
- "Forget what we discussed"

**Example Extraction:**
```
User: "忘记之前的讨论了，能再解释一下吗？"
→ Issue: {
    category: "context_loss",
    description: "用户需要重新解释需求，上下文丢失",
    severity: "medium",
    session_ids: ["session-123"],
    occurrences: 1,
    example: "忘记之前的讨论了，能再解释一下吗？"
}
```

### 2. Repeated Problem (repeated_problem)

**Severity Levels:**
- **High**: Same problem in 3+ sessions
- **Medium**: Same problem in 2 sessions
- **Low**: Similar but not identical problems

**Patterns:**
- "又遇到这个问题"
- "之前也这样"
- "Same error again"
- "This happened before"
- Cross-session pattern matching

**Example Extraction:**
```
Session 1: "遇到类型错误 TypeError"
Session 2: "又遇到同样的类型错误"
Session 3: "还是这个类型错误"
→ Issue: {
    category: "repeated_problem",
    description: "类型错误在多个会话中重复出现",
    severity: "high",
    session_ids: ["session-1", "session-2", "session-3"],
    occurrences: 3,
    example: "又遇到同样的类型错误"
}
```

### 3. Stuck (stuck)

**Severity Levels:**
- **High**: 10+ messages on same problem, user frustrated
- **Medium**: 5-9 messages, some difficulty
- **Low**: 3-4 messages, minor delay

**Patterns:**
- "还是不行"
- "试了很多次"
- "Still not working"
- "Tried many approaches"
- Long conversation threads on same topic
- Multiple failed attempts

**Example Extraction:**
```
User: "Fix the login bug"
AI: [Solution 1]
User: "Still not working"
AI: [Solution 2]
User: "还是不行"
AI: [Solution 3]
User: "Finally works"
→ Issue: {
    category: "stuck",
    description: "登录 bug 修复花费多次尝试",
    severity: "high",
    session_ids: ["session-456"],
    occurrences: 1,
    example: "试了很多次还是不行"
}
```

### 4. Quality Issue (quality_issue)

**Severity Levels:**
- **High**: Critical quality problems, security issues
- **Medium**: Code quality concerns, maintainability
- **Low**: Minor style issues

**Patterns:**
- "代码有问题"
- "需要重构"
- "Not following best practices"
- "Performance issue"
- "Security concern"

**Example Extraction:**
```
User: "这个代码需要重构，太复杂了"
→ Issue: {
    category: "quality_issue",
    description: "代码复杂度高，需要重构",
    severity: "medium",
    session_ids: ["session-789"],
    occurrences: 1,
    example: "这个代码需要重构，太复杂了"
}
```

### 5. Tool Usage (tool_usage)

**Severity Levels:**
- **High**: Tool failures, incorrect usage blocking work
- **Medium**: Tool usage inefficiencies
- **Low**: Minor tool usage improvements

**Patterns:**
- "工具调用失败"
- "Tool not working"
- "Wrong tool used"
- "Should use different tool"

## Optimization Categories

### 1. Workflow Optimization (workflow)

**Priority Levels:**
- **High**: Major efficiency gains
- **Medium**: Moderate improvements
- **Low**: Minor optimizations

**Impact Types:**
- **efficiency**: Improves work speed
- **quality**: Improves output quality
- **experience**: Improves user experience

**Patterns:**
- "如果一开始就..."
- "应该先..."
- "流程可以优化"
- "Should have done X first"
- "Better approach would be..."

**Example Extraction:**
```
User: "如果一开始就明确需求，就不会有这么多修改了"
→ Optimization: {
    category: "workflow",
    suggestion: "在开始编码前更明确地定义需求，减少后续修改",
    priority: "high",
    impact: "efficiency",
    session_ids: ["session-101"]
}
```

### 2. Prompt Quality (prompt_quality)

**Patterns:**
- "提示不够清晰"
- "应该更具体"
- "描述不清楚"
- "Prompt was too vague"
- "Should have been more specific"

**Example Extraction:**
```
User: "我的提示可能不够清晰，导致生成了错误的代码"
→ Optimization: {
    category: "prompt_quality",
    suggestion: "提供更具体和详细的初始提示，包含明确的约束和期望",
    priority: "high",
    impact: "quality",
    session_ids: ["session-202"]
}
```

### 3. Tool Usage Optimization (tool_usage)

**Patterns:**
- "应该使用工具 X"
- "Better tool for this"
- "工具组合可以优化"

**Example Extraction:**
```
User: "应该使用 git 工具而不是手动检查提交"
→ Optimization: {
    category: "tool_usage",
    suggestion: "使用自动化工具（如 git 分析）而不是手动检查",
    priority: "medium",
    impact: "efficiency",
    session_ids: ["session-303"]
}
```

### 4. Context Management (context_management)

**Patterns:**
- "应该及时总结"
- "Better context organization"
- "Should have saved context"

**Example Extraction:**
```
User: "如果及时总结会话，就不会丢失上下文了"
→ Optimization: {
    category: "context_management",
    suggestion: "定期总结会话内容，避免上下文丢失",
    priority: "medium",
    impact: "experience",
    session_ids: ["session-404"]
}
```

## Analysis Workflow

1. **Scan all conversations** for issue and optimization patterns
2. **Group similar issues** across sessions
3. **Count occurrences** for repeated problems
4. **Assess severity/priority** based on impact and frequency
5. **Extract examples** from actual conversation snippets
6. **Generate actionable suggestions** for optimizations

## Best Practices

1. **Be Specific**: Include concrete examples from conversations
2. **Link to Sessions**: Always include session IDs for traceability
3. **Prioritize**: Focus on high-severity issues and high-priority optimizations
4. **Actionable**: Optimizations should be specific and implementable
5. **Context-Aware**: Consider the full conversation context, not just keywords
