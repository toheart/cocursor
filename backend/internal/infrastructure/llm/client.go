package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"log/slog"

	"github.com/cocursor/backend/internal/domain/rag"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// Client LLM Chat 客户端
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	language   string
	httpClient *http.Client
	logger     *slog.Logger
}

// ChatRequest Chat API 请求
type ChatRequest struct {
	Messages []Message `json:"messages"`
	Model    string    `json:"model,omitempty"`
}

// Message Chat 消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse Chat API 响应
type ChatResponse struct {
	ID      string `json:"id,omitempty"`
	Object  string `json:"object,omitempty"`
	Created int64  `json:"created,omitempty"`
	Model   string `json:"model,omitempty"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewClient 创建 LLM 客户端
func NewClient(baseURL, apiKey, model string) *Client {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &Client{
		baseURL:  baseURL,
		apiKey:   apiKey,
		model:    model,
		language: "zh-CN", // 默认使用中文
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: log.NewModuleLogger("llm", "client"),
	}
}

// SummarizeTurn 总结对话对
func (c *Client) SummarizeTurn(userText, aiText string) (*rag.TurnSummary, error) {
	prompt := c.buildPrompt(userText, aiText)

	reqBody := ChatRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Model: c.model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	c.logger.Debug("Sending LLM summarization request",
		"url", url,
		"model", c.model,
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LLM API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := c.readResponseBody(resp)
		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, body)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode LLM response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("LLM API returned no choices")
	}

	content := chatResp.Choices[0].Message.Content
	summary, err := c.parseSummary(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse summary JSON: %w", err)
	}

	c.logger.Info("LLM summarization successful",
		"model", c.model,
		"language", c.language,
		"tokens", chatResp.Usage.TotalTokens,
	)

	return summary, nil
}

// TestConnection 测试 LLM API 连接
func (c *Client) TestConnection() error {
	testPrompt := "This is a test. Please respond with 'OK' in JSON format: {\"status\": \"OK\"}"

	reqBody := ChatRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: testPrompt,
			},
		},
		Model: c.model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal test request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	c.logger.Debug("Testing LLM connection",
		"url", url,
		"model", c.model,
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("LLM connection test failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := c.readResponseBody(resp)
		return fmt.Errorf("LLM connection test failed with status %d: %s", resp.StatusCode, body)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return fmt.Errorf("failed to decode test response: %w", err)
	}

	c.logger.Info("LLM connection test successful",
		"model", c.model,
	)

	return nil
}

// buildPrompt 构建总结 Prompt
func (c *Client) buildPrompt(userText, aiText string) string {
	if c.language == "en-US" {
		return c.buildEnglishPrompt(userText, aiText)
	}
	return c.buildChinesePrompt(userText, aiText)
}

// buildChinesePrompt 构建中文 Prompt
func (c *Client) buildChinesePrompt(userText, aiText string) string {
	return fmt.Sprintf(`你是一个代码对话知识提取专家。请分析以下对话，生成结构化总结。

对话内容：
用户: %s
AI: %s

请提取以下信息并以 JSON 格式返回：

1. main_topic: 这段对话主要讨论什么主题？（简短）
2. problem: 用户遇到的问题或需求是什么？
3. solution: AI 提供的解决方案或答案是什么？
4. tech_stack: 涉及哪些技术栈？（数组，如 ["React", "Go"]）
5. code_snippets: 提取所有代码片段（数组）
6. key_points: 提取 3-5 个关键知识点（数组）
7. lessons: 提取经验教训或最佳实践（数组，可选）
8. tags: 生成 3-5 个标签用于检索（数组，如 ["react", "组件", "性能优化"]）
9. summary: 用一句话总结这段对话的核心价值
10. context: 保留对话的精华部分（100 字左右）

JSON 格式要求：
- 所有字段必须存在
- 数组字段为空时返回 []
- 字符串字段为空时返回 ""
- 请以纯 JSON 格式返回，不要包含其他文本

返回 JSON。`, userText, aiText)
}

// buildEnglishPrompt 构建英文 Prompt
func (c *Client) buildEnglishPrompt(userText, aiText string) string {
	return fmt.Sprintf(`You are a code conversation knowledge extraction expert. Please analyze the following conversation and generate a structured summary.

Conversation content:
User: %s
AI: %s

Please extract the following information and return it in JSON format:

1. main_topic: What is the main topic of this conversation? (Short)
2. problem: What problem or requirement does the user have?
3. solution: What solution or answer does the AI provide?
4. tech_stack: What technology stacks are involved? (Array, e.g. ["React", "Go"])
5. code_snippets: Extract all code snippets (Array)
6. key_points: Extract 3-5 key points (Array)
7. lessons: Extract lessons learned or best practices (Array, optional)
8. tags: Generate 3-5 tags for retrieval (Array, e.g. ["react", "component", "performance"])
9. summary: Summarize the core value of this conversation in one sentence
10. context: Keep the essence of the conversation (about 100 characters)

JSON format requirements:
- All fields must be present
- Array fields return [] when empty
- String fields return "" when empty
- Please return in pure JSON format, do not include other text

Return JSON.`, userText, aiText)
}

// parseSummary 解析 LLM 返回的总结 JSON
func (c *Client) parseSummary(content string) (*rag.TurnSummary, error) {
	var summary rag.TurnSummary
	if err := json.Unmarshal([]byte(content), &summary); err != nil {
		return nil, err
	}
	return &summary, nil
}

// readResponseBody 读取响应体
func (c *Client) readResponseBody(resp *http.Response) (string, error) {
	if resp.Body == nil {
		return "", nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
