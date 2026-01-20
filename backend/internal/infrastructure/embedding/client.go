package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cocursor/backend/internal/infrastructure/log"
)

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Client Embedding API 客户端
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient 创建 Embedding 客户端
func NewClient(baseURL, apiKey, model string) *Client {
	// 规范化 baseURL：移除末尾斜杠
	normalizedURL := strings.TrimSuffix(baseURL, "/")

	return &Client{
		baseURL:    normalizedURL,
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: log.NewModuleLogger("embedding", "client"),
	}
}

// buildEmbeddingURL 构建 Embedding API URL
// 支持多种输入格式，智能拼接 /v1/embeddings 路径
func buildEmbeddingURL(baseURL string) string {
	// 1. 如果已经包含完整路径 /v1/embeddings，直接使用
	if strings.Contains(baseURL, "/v1/embeddings") {
		return baseURL
	}

	// 2. 如果以 /v1 结尾，只追加 /embeddings
	if strings.HasSuffix(baseURL, "/v1") {
		return baseURL + "/embeddings"
	}

	// 3. 如果以 /v1/ 结尾，追加 embeddings
	if strings.HasSuffix(baseURL, "/v1/") {
		return baseURL + "embeddings"
	}

	// 4. 其他情况，追加完整的 /v1/embeddings
	return fmt.Sprintf("%s/v1/embeddings", baseURL)
}

// EmbeddingRequest Embedding 请求
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse Embedding 响应
type EmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// EmbedTexts 批量向量化文本
func (c *Client) EmbedTexts(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	// OpenAI embeddings API 批量限制：每次最多 2048 个文本
	const maxBatchSize = 2048
	const maxRetriesPerBatch = 3

	// 如果文本数量小于等于限制，直接批量处理
	if len(texts) <= maxBatchSize {
		return c.embedTextsBatch(texts)
	}

	// 否则分批处理
	c.logger.Info("Splitting texts into batches",
		"total_texts", len(texts),
		"batch_limit", maxBatchSize,
	)

	allVectors := make([][]float32, 0, len(texts))

	for i := 0; i < len(texts); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		batchNum := (i / maxBatchSize) + 1
		totalBatches := (len(texts) + maxBatchSize - 1) / maxBatchSize

		c.logger.Debug("Processing batch",
			"batch", batchNum,
			"total_batches", totalBatches,
			"batch_size", len(batch),
		)

		vectors, err := c.embedTextsWithRetry(batch, maxRetriesPerBatch)
		if err != nil {
			c.logger.Error("Failed to embed batch",
				"batch", batchNum,
				"error", err,
			)
			return nil, fmt.Errorf("failed to embed batch %d: %w", batchNum, err)
		}

		allVectors = append(allVectors, vectors...)
	}

	c.logger.Info("Successfully embedded texts",
		"total_vectors", len(allVectors),
	)

	return allVectors, nil
}

// embedTextsBatch 处理单个批次
func (c *Client) embedTextsBatch(texts []string) ([][]float32, error) {
	return c.embedTextsWithRetry(texts, 3)
}

// embedTextsWithRetry 带重试的嵌入处理
func (c *Client) embedTextsWithRetry(texts []string, maxRetries int) ([][]float32, error) {
	// 构建请求
	reqBody := EmbeddingRequest{
		Model: c.model,
		Input: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 构建 URL（智能处理是否已包含路径）
	url := buildEmbeddingURL(c.baseURL)

	// API Key 脱敏
	apiKeyMasked := c.apiKey
	if len(apiKeyMasked) > 8 {
		apiKeyMasked = apiKeyMasked[:4] + "..." + apiKeyMasked[len(apiKeyMasked)-4:]
	} else {
		apiKeyMasked = "***"
	}

	c.logger.Debug("Sending embedding request",
		"url", url,
		"batch_size", len(texts),
		"model", c.model,
		"api_key", apiKeyMasked,
		"request_body_preview", string(jsonData[:min(200, len(jsonData))]),
	)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// 发送请求（带重试）
	var resp *http.Response
	for retry := 0; retry < maxRetries; retry++ {
		resp, err = c.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if resp != nil {
			c.logger.Warn("Embedding request failed, retrying",
				"attempt", retry+1,
				"max_retries", maxRetries,
				"status_code", resp.StatusCode,
			)
			resp.Body.Close()
		}
		if retry < maxRetries-1 {
			time.Sleep(time.Duration(retry+1) * time.Second) // 递增延迟
		}
	}

	if err != nil {
		c.logger.Error("Embedding request failed after all retries",
			"max_retries", maxRetries,
			"error", err,
		)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("Embedding request completed",
		"status_code", resp.StatusCode,
	)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("API returned error",
			"status_code", resp.StatusCode,
			"response_body", string(body),
		)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var embeddingResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		c.logger.Error("Failed to decode embedding response",
			"error", err,
		)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 提取向量
	vectors := make([][]float32, len(embeddingResp.Data))
	for _, data := range embeddingResp.Data {
		vectors[data.Index] = data.Embedding
	}

	return vectors, nil
}

// GetVectorDimension 获取向量维度（通过测试请求）
func (c *Client) GetVectorDimension() (int, error) {
	testTexts := []string{"test"}
	vectors, err := c.EmbedTexts(testTexts)
	if err != nil {
		return 0, err
	}

	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return 0, fmt.Errorf("invalid embedding response")
	}

	return len(vectors[0]), nil
}

// TestConnection 测试连接
func (c *Client) TestConnection() error {
	c.logger.Info("Testing embedding API connection",
		"base_url", c.baseURL,
		"model", c.model,
	)

	dimension, err := c.GetVectorDimension()
	if err != nil {
		c.logger.Error("Embedding API connection test failed",
			"error", err,
		)
		return err
	}

	c.logger.Info("Embedding API connection test successful",
		"vector_dimension", dimension,
	)

	return nil
}
