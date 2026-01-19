package rag

import (
	"log/slog"

	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	"github.com/cocursor/backend/internal/infrastructure/llm"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// LLMClient LLM 总结客户端（应用层 wrapper）
type LLMClient struct {
	client *llm.Client
	logger *slog.Logger
}

// NewLLMClient 创建 LLM 客户端
func NewLLMClient(url, apiKey, model string) *LLMClient {
	client := llm.NewClient(url, apiKey, model)
	return &LLMClient{
		client: client,
		logger: log.NewModuleLogger("rag", "llm_client"),
	}
}

// SummarizeTurn 总结对话对
func (c *LLMClient) SummarizeTurn(userText, aiText string) (*domainRAG.TurnSummary, error) {
	return c.client.SummarizeTurn(userText, aiText)
}

// TestConnection 测试 LLM API 连接
func (c *LLMClient) TestConnection() error {
	return c.client.TestConnection()
}
