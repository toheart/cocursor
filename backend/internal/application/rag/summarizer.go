package rag

import (
	"log/slog"

	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// Summarizer 对话总结服务
type Summarizer struct {
	llmClient *LLMClient
	logger    *slog.Logger
}

// NewSummarizer 创建总结服务
func NewSummarizer(llmClient *LLMClient) *Summarizer {
	return &Summarizer{
		llmClient: llmClient,
		logger:    log.NewModuleLogger("rag", "summarizer"),
	}
}

// SummarizeTurn 总结单个对话对
func (s *Summarizer) SummarizeTurn(turn *ConversationTurn) (*domainRAG.TurnSummary, error) {
	s.logger.Debug("Summarizing turn",
		"turn_index", turn.TurnIndex,
		"user_text_length", len(turn.UserText),
		"ai_text_length", len(turn.AIText),
	)

	summary, err := s.llmClient.SummarizeTurn(turn.UserText, turn.AIText)
	if err != nil {
		s.logger.Warn("Failed to summarize turn",
			"turn_index", turn.TurnIndex,
			"error", err,
		)
		return nil, err
	}

	s.logger.Debug("Turn summarization successful",
		"turn_index", turn.TurnIndex,
		"main_topic", summary.MainTopic,
		"summary", summary.Summary,
	)

	return summary, nil
}
