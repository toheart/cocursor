package rag

import (
	"path/filepath"
	"testing"
)

func TestConfigManager_ReadWrite(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rag_config.json")

	// 创建配置管理器（使用临时路径）
	manager := &ConfigManager{
		configPath: configPath,
	}

	// 测试写入配置
	config := manager.getDefaultConfig()
	config.EmbeddingAPI.URL = "https://api.example.com"
	config.EmbeddingAPI.APIKey = "test-key"
	config.EmbeddingAPI.Model = "text-embedding-ada-002"

	err := manager.WriteConfig(config)
	if err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	// 测试读取配置
	readConfig, err := manager.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig failed: %v", err)
	}

	// 验证配置
	if readConfig.EmbeddingAPI.URL != config.EmbeddingAPI.URL {
		t.Errorf("URL mismatch: got %s, want %s", readConfig.EmbeddingAPI.URL, config.EmbeddingAPI.URL)
	}
	if readConfig.EmbeddingAPI.APIKey != config.EmbeddingAPI.APIKey {
		t.Errorf("APIKey mismatch: got %s, want %s", readConfig.EmbeddingAPI.APIKey, config.EmbeddingAPI.APIKey)
	}
	if readConfig.EmbeddingAPI.Model != config.EmbeddingAPI.Model {
		t.Errorf("Model mismatch: got %s, want %s", readConfig.EmbeddingAPI.Model, config.EmbeddingAPI.Model)
	}
}

func TestConfigManager_ReadNonExistent(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "non_existent.json")

	manager := &ConfigManager{
		configPath: configPath,
	}

	// 读取不存在的配置应该返回默认配置
	config, err := manager.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig failed: %v", err)
	}

	// 验证返回的是默认配置
	if config.ScanConfig.Interval != "1h" {
		t.Errorf("Expected default interval 1h, got %s", config.ScanConfig.Interval)
	}
	if config.ScanConfig.BatchSize != 10 {
		t.Errorf("Expected default batch size 10, got %d", config.ScanConfig.BatchSize)
	}
}

func TestConfigManager_DefaultConfig(t *testing.T) {
	manager := &ConfigManager{}
	config := manager.getDefaultConfig()

	if config.ScanConfig.Enabled != false {
		t.Errorf("Expected Enabled=false, got %v", config.ScanConfig.Enabled)
	}
	if config.ScanConfig.Interval != "1h" {
		t.Errorf("Expected Interval=1h, got %s", config.ScanConfig.Interval)
	}
	if config.ScanConfig.BatchSize != 10 {
		t.Errorf("Expected BatchSize=10, got %d", config.ScanConfig.BatchSize)
	}
	if config.ScanConfig.Concurrency != 3 {
		t.Errorf("Expected Concurrency=3, got %d", config.ScanConfig.Concurrency)
	}
}
