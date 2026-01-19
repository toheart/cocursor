package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client Embedding API 客户端
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient 创建 Embedding 客户端
func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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

	// 构建请求
	reqBody := EmbeddingRequest{
		Model: c.model,
		Input: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	url := fmt.Sprintf("%s/v1/embeddings", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// 发送请求（带重试）
	var resp *http.Response
	maxRetries := 3
	for retry := 0; retry < maxRetries; retry++ {
		resp, err = c.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		if retry < maxRetries-1 {
			time.Sleep(time.Duration(retry+1) * time.Second) // 递增延迟
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var embeddingResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
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
	_, err := c.GetVectorDimension()
	return err
}
