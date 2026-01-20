package rag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigManager RAG 配置管理器
type ConfigManager struct {
	configPath string
	encryptKey *EncryptionKey
}

// NewConfigManager 创建 RAG 配置管理器
func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".cocursor", "rag_config.json")

	// 创建加密密钥管理器
	encryptKey, err := NewEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to create encryption key: %w", err)
	}

	return &ConfigManager{
		configPath: configPath,
		encryptKey: encryptKey,
	}, nil
}

// RAGConfig RAG 配置结构
type RAGConfig struct {
	// Embedding API 配置
	EmbeddingAPI struct {
		URL    string `json:"url"`     // API URL
		APIKey string `json:"api_key"` // API Key（加密存储）
		Model  string `json:"model"`   // 模型名称
	} `json:"embedding_api"`

	// LLM Chat API 配置
	LLMChatAPI struct {
		URL    string `json:"url"`     // API URL (如 https://api.openai.com/v1)
		APIKey string `json:"api_key"` // API Key（加密存储）
		Model  string `json:"model"`   // 模型名称 (如 gpt-4, gpt-3.5-turbo)
	} `json:"llm_chat_api"`

	// Qdrant 配置
	Qdrant struct {
		Version    string `json:"version"`     // 已下载的版本
		BinaryPath string `json:"binary_path"` // 二进制路径
		DataPath   string `json:"data_path"`   // 数据存储路径
	} `json:"qdrant"`

	// 索引配置（仅用于全量索引）
	IndexConfig struct {
		BatchSize   int `json:"batch_size"`  // 批量大小（默认 10）
		Concurrency int `json:"concurrency"` // 并发数（默认 3）
	} `json:"index_config"`

	// 元数据
	LastFullScan        int64 `json:"last_full_scan"`        // 最后全量扫描时间
	LastIncrementalScan int64 `json:"last_incremental_scan"` // 最后增量扫描时间
	TotalIndexed        int   `json:"total_indexed"`         // 已索引消息数
}

// ReadConfig 读取 RAG 配置
func (c *ConfigManager) ReadConfig() (*RAGConfig, error) {
	// 如果文件不存在，返回默认配置
	if _, err := os.Stat(c.configPath); os.IsNotExist(err) {
		return c.getDefaultConfig(), nil
	}

	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config RAGConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 解密 Embedding API Key
	if config.EmbeddingAPI.APIKey != "" {
		decrypted, err := c.encryptKey.Decrypt(config.EmbeddingAPI.APIKey)
		if err == nil {
			config.EmbeddingAPI.APIKey = decrypted
		}
		// 如果解密失败，保持原值（可能是未加密的旧数据）
	}

	// 解密 LLM Chat API Key
	if config.LLMChatAPI.APIKey != "" {
		decrypted, err := c.encryptKey.Decrypt(config.LLMChatAPI.APIKey)
		if err == nil {
			config.LLMChatAPI.APIKey = decrypted
		}
		// 如果解密失败，保持原值（可能是未加密的旧数据）
	}

	return &config, nil
}

// WriteConfig 写入 RAG 配置
func (c *ConfigManager) WriteConfig(config *RAGConfig) error {
	// 创建配置副本以避免修改原始配置
	configCopy := *config

	// 加密 Embedding API Key（如果未加密）
	if configCopy.EmbeddingAPI.APIKey != "" {
		// 检查是否已经是加密格式（base64）
		encrypted, err := c.encryptKey.Encrypt(configCopy.EmbeddingAPI.APIKey)
		if err == nil {
			configCopy.EmbeddingAPI.APIKey = encrypted
		}
		// 如果加密失败，保持原值
	}

	// 加密 LLM Chat API Key（如果未加密）
	if configCopy.LLMChatAPI.APIKey != "" {
		// 检查是否已经是加密格式（base64）
		encrypted, err := c.encryptKey.Encrypt(configCopy.LLMChatAPI.APIKey)
		if err == nil {
			configCopy.LLMChatAPI.APIKey = encrypted
		}
		// 如果加密失败，保持原值
	}

	// 确保目录存在
	dir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(configCopy, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(c.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getDefaultConfig 获取默认配置
func (c *ConfigManager) getDefaultConfig() *RAGConfig {
	return &RAGConfig{
		IndexConfig: struct {
			BatchSize   int `json:"batch_size"`
			Concurrency int `json:"concurrency"`
		}{
			BatchSize:   10,
			Concurrency: 3,
		},
	}
}
