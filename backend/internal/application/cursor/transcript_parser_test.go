package cursor

import (
	"strings"
	"testing"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
)

// mockSessionService 创建用于测试的 mock service
func mockSessionService() *SessionService {
	return &SessionService{}
}

func TestTranscriptParser_Parse_BasicConversation(t *testing.T) {
	content := `user:
你好，帮我写一个 Hello World 程序
assistant:
好的，这是一个简单的 Hello World 程序：
` + "```go" + `
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
` + "```"

	service := mockSessionService()
	parser := newTranscriptParser(service, content, 1000000)
	messages, err := parser.parse()

	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("期望 2 条消息，实际 %d 条", len(messages))
	}

	// 验证用户消息
	if messages[0].Type != domainCursor.MessageTypeUser {
		t.Errorf("第一条消息类型应该是 user，实际是 %s", messages[0].Type)
	}
	if !strings.Contains(messages[0].Text, "Hello World") {
		t.Errorf("用户消息应包含 'Hello World'")
	}

	// 验证 AI 消息
	if messages[1].Type != domainCursor.MessageTypeAI {
		t.Errorf("第二条消息类型应该是 ai，实际是 %s", messages[1].Type)
	}
}

func TestTranscriptParser_Parse_ToolCalls(t *testing.T) {
	content := `user:
帮我读取 main.go 文件
assistant:
我来读取文件内容。
[Tool call] Read
path: /Users/test/main.go
[Tool result]
package main
func main() {}
assistant:
文件内容如上所示。`

	service := mockSessionService()
	parser := newTranscriptParser(service, content, 1000000)
	messages, err := parser.parse()

	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 应该有 3 条消息：user, assistant(带工具调用), assistant
	if len(messages) < 2 {
		t.Fatalf("期望至少 2 条消息，实际 %d 条", len(messages))
	}

	// 查找带工具调用的消息
	var toolMessage *domainCursor.Message
	for _, msg := range messages {
		if len(msg.Tools) > 0 {
			toolMessage = msg
			break
		}
	}

	if toolMessage == nil {
		t.Fatal("没有找到包含工具调用的消息")
	}

	if toolMessage.Tools[0].Name != "Read" {
		t.Errorf("工具名称应该是 'Read'，实际是 '%s'", toolMessage.Tools[0].Name)
	}

	if toolMessage.Tools[0].Arguments["path"] != "/Users/test/main.go" {
		t.Errorf("工具参数 path 不正确，实际是 '%s'", toolMessage.Tools[0].Arguments["path"])
	}
}

func TestTranscriptParser_Parse_XMLTags(t *testing.T) {
	// 测试 user_query 标签内容提取
	t.Run("user_query标签", func(t *testing.T) {
		content := `user:
<user_query>这是用户的问题</user_query>
assistant:
这是 AI 的回答`

		service := mockSessionService()
		parser := newTranscriptParser(service, content, 1000000)
		messages, err := parser.parse()

		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}

		if len(messages) != 2 {
			t.Fatalf("期望 2 条消息，实际 %d 条", len(messages))
		}

		// 用户消息应该包含 user_query 的内容
		if !strings.Contains(messages[0].Text, "这是用户的问题") {
			t.Errorf("用户消息应包含 'user_query' 标签的内容，实际：%s", messages[0].Text)
		}
	})

	// 测试 think 标签跳过（think 单独成行）
	t.Run("think标签跳过", func(t *testing.T) {
		content := `user:
你好
assistant:
<think>
这是思考内容
</think>
这是正式回答`

		service := mockSessionService()
		parser := newTranscriptParser(service, content, 1000000)
		messages, err := parser.parse()

		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}

		if len(messages) != 2 {
			t.Fatalf("期望 2 条消息，实际 %d 条", len(messages))
		}

		// AI 消息不应该包含 think 标签的内容
		if strings.Contains(messages[1].Text, "思考内容") {
			t.Errorf("AI 消息不应包含 'think' 标签的内容")
		}

		// AI 消息应该包含正式回答
		if !strings.Contains(messages[1].Text, "正式回答") {
			t.Errorf("AI 消息应包含正式回答内容，实际：%s", messages[1].Text)
		}
	})
}

func TestTranscriptParser_Parse_MultilineToolArgs(t *testing.T) {
	content := `assistant:
[Tool call] Write
path: /Users/test/file.go
content: package main

import "fmt"

func main() {
    fmt.Println("Hello")
}
user:
好的`

	service := mockSessionService()
	parser := newTranscriptParser(service, content, 1000000)
	messages, err := parser.parse()

	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 查找带工具调用的消息
	var toolMessage *domainCursor.Message
	for _, msg := range messages {
		if len(msg.Tools) > 0 {
			toolMessage = msg
			break
		}
	}

	if toolMessage == nil {
		t.Fatal("没有找到包含工具调用的消息")
	}

	if toolMessage.Tools[0].Name != "Write" {
		t.Errorf("工具名称应该是 'Write'，实际是 '%s'", toolMessage.Tools[0].Name)
	}

	// 验证多行参数被正确解析
	if toolMessage.Tools[0].Arguments["path"] != "/Users/test/file.go" {
		t.Errorf("path 参数不正确")
	}
}

func TestTranscriptParser_Parse_EmptyContent(t *testing.T) {
	content := ""

	service := mockSessionService()
	parser := newTranscriptParser(service, content, 1000000)
	messages, err := parser.parse()

	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("空内容应该返回 0 条消息，实际 %d 条", len(messages))
	}
}

func TestTranscriptParser_Parse_OnlyRoles(t *testing.T) {
	content := `user:
assistant:
user:`

	service := mockSessionService()
	parser := newTranscriptParser(service, content, 1000000)
	messages, err := parser.parse()

	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 没有实际内容的角色不应产生消息
	if len(messages) != 0 {
		t.Errorf("没有内容的角色不应产生消息，实际 %d 条", len(messages))
	}
}

func TestTranscriptParser_HandleRoleSwitch(t *testing.T) {
	service := mockSessionService()
	parser := newTranscriptParser(service, "", 1000000)

	tests := []struct {
		line    string
		handled bool
		role    string
	}{
		{"user:", true, "user"},
		{"assistant:", true, "assistant"},
		{"other:", false, ""},
		{"user", false, ""},
		{"", false, ""},
	}

	for _, tt := range tests {
		parser.currentRole = ""
		result := parser.handleRoleSwitch(tt.line)

		if result.handled != tt.handled {
			t.Errorf("handleRoleSwitch(%q): handled = %v, want %v", tt.line, result.handled, tt.handled)
		}

		if tt.handled && parser.currentRole != tt.role {
			t.Errorf("handleRoleSwitch(%q): role = %q, want %q", tt.line, parser.currentRole, tt.role)
		}
	}
}

func TestTranscriptParser_IsTerminatorLine(t *testing.T) {
	service := mockSessionService()
	parser := newTranscriptParser(service, "", 1000000)

	tests := []struct {
		line     string
		expected bool
	}{
		{"", true},
		{"user:", true},
		{"assistant:", true},
		{"[Tool call]", true},
		{"[Tool call] Read", true},
		{"[Tool result]", true},
		{"[Tool result] success", true},
		{"普通文本", false},
		{"  ", false},
		{"path: /test", false},
	}

	for _, tt := range tests {
		result := parser.isTerminatorLine(tt.line)
		if result != tt.expected {
			t.Errorf("isTerminatorLine(%q) = %v, want %v", tt.line, result, tt.expected)
		}
	}
}

func TestTranscriptParser_IsNewParameter(t *testing.T) {
	service := mockSessionService()
	parser := newTranscriptParser(service, "", 1000000)

	tests := []struct {
		line     string
		trimmed  string
		expected bool
	}{
		{"path: /test", "path: /test", true},
		{"  path: /test", "path: /test", false}, // 以空格开头
		{"\tpath: /test", "path: /test", false}, // 以 tab 开头
		{"content", "content", false},           // 不含冒号
		{"key=value", "key=value", false},       // 不含冒号
	}

	for _, tt := range tests {
		result := parser.isNewParameter(tt.line, tt.trimmed)
		if result != tt.expected {
			t.Errorf("isNewParameter(%q, %q) = %v, want %v", tt.line, tt.trimmed, result, tt.expected)
		}
	}
}

func TestTranscriptParser_ExtractTagContent(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		openTag      string
		closeTag     string
		wantContent  string
		wantSkipLine int
	}{
		{
			name:         "同一行标签",
			content:      "<user_query>问题内容</user_query>",
			openTag:      "<user_query>",
			closeTag:     "</user_query>",
			wantContent:  "问题内容",
			wantSkipLine: 0,
		},
		{
			name:         "跨行标签",
			content:      "<user_query>第一行\n第二行\n</user_query>",
			openTag:      "<user_query>",
			closeTag:     "</user_query>",
			wantContent:  "第一行\n第二行",
			wantSkipLine: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := mockSessionService()
			parser := newTranscriptParser(service, tt.content, 1000000)

			content, skipLines := parser.extractTagContent(0, tt.openTag, tt.closeTag)

			if content != tt.wantContent {
				t.Errorf("extractTagContent() content = %q, want %q", content, tt.wantContent)
			}

			if skipLines != tt.wantSkipLine {
				t.Errorf("extractTagContent() skipLines = %d, want %d", skipLines, tt.wantSkipLine)
			}
		})
	}
}

func TestTranscriptParser_SkipTagContent(t *testing.T) {
	content := `<think>
这是思考内容
第二行
</think>
正常内容`

	service := mockSessionService()
	parser := newTranscriptParser(service, content, 1000000)

	skipLines := parser.skipTagContent(0, "</think>")

	// 应该跳过到 </think> 所在行
	if skipLines != 3 {
		t.Errorf("skipTagContent() = %d, want 3", skipLines)
	}
}

func TestTranscriptParser_Timestamps(t *testing.T) {
	content := `user:
消息1
assistant:
消息2
user:
消息3`

	baseTimestamp := int64(1000000)
	service := mockSessionService()
	parser := newTranscriptParser(service, content, baseTimestamp)
	messages, err := parser.parse()

	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("期望 3 条消息，实际 %d 条", len(messages))
	}

	// 验证时间戳递增
	for i := 0; i < len(messages); i++ {
		expectedTimestamp := baseTimestamp + int64(i*1000)
		if messages[i].Timestamp != expectedTimestamp {
			t.Errorf("消息 %d 时间戳应为 %d，实际 %d", i, expectedTimestamp, messages[i].Timestamp)
		}
	}
}

func TestTranscriptParser_MultipleToolCalls(t *testing.T) {
	content := `assistant:
我需要读取多个文件
[Tool call] Read
path: /file1.go
[Tool call] Read
path: /file2.go
user:
好的`

	service := mockSessionService()
	parser := newTranscriptParser(service, content, 1000000)
	messages, err := parser.parse()

	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 查找带工具调用的消息
	var toolMessage *domainCursor.Message
	for _, msg := range messages {
		if len(msg.Tools) > 0 {
			toolMessage = msg
			break
		}
	}

	if toolMessage == nil {
		t.Fatal("没有找到包含工具调用的消息")
	}

	// 应该有 2 个工具调用
	if len(toolMessage.Tools) != 2 {
		t.Errorf("期望 2 个工具调用，实际 %d 个", len(toolMessage.Tools))
	}
}

// BenchmarkTranscriptParser 性能基准测试
func BenchmarkTranscriptParser(b *testing.B) {
	// 构建一个较大的测试内容
	var builder strings.Builder
	for i := 0; i < 100; i++ {
		builder.WriteString("user:\n")
		builder.WriteString("这是用户的问题，可能很长很长\n")
		builder.WriteString("assistant:\n")
		builder.WriteString("这是 AI 的回答\n")
		builder.WriteString("[Tool call] Read\n")
		builder.WriteString("path: /test/file.go\n")
		builder.WriteString("[Tool result]\n")
		builder.WriteString("文件内容\n")
	}
	content := builder.String()

	service := mockSessionService()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := newTranscriptParser(service, content, 1000000)
		_, _ = parser.parse()
	}
}
