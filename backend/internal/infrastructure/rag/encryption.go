package rag

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EncryptionKey 加密密钥管理器
type EncryptionKey struct {
	keyPath string
	key     []byte
}

// NewEncryptionKey 创建加密密钥管理器
func NewEncryptionKey() (*EncryptionKey, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	keyPath := filepath.Join(homeDir, ".cocursor", ".rag_key")

	ek := &EncryptionKey{
		keyPath: keyPath,
	}

	// 加载或生成密钥
	if err := ek.loadOrGenerateKey(); err != nil {
		return nil, fmt.Errorf("failed to load or generate key: %w", err)
	}

	return ek, nil
}

// loadOrGenerateKey 加载或生成加密密钥
func (ek *EncryptionKey) loadOrGenerateKey() error {
	// 尝试读取现有密钥
	if data, err := os.ReadFile(ek.keyPath); err == nil {
		// 密钥已存在，使用它
		ek.key = data
		return nil
	}

	// 生成新密钥（32 字节，用于 AES-256）
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(ek.keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// 保存密钥（仅所有者可读写）
	if err := os.WriteFile(ek.keyPath, key, 0600); err != nil {
		return fmt.Errorf("failed to save key: %w", err)
	}

	ek.key = key
	return nil
}

// Encrypt 加密文本
func (ek *EncryptionKey) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(ek.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// 使用 GCM 模式
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机 nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// Base64 编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密文本
func (ek *EncryptionKey) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		// 如果不是 base64 编码，可能是未加密的旧数据，直接返回
		return ciphertext, nil
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(ek.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// 使用 GCM 模式
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 提取 nonce
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		// 数据太短，可能是未加密的旧数据
		return ciphertext, nil
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// 解密
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		// 解密失败，可能是未加密的旧数据
		return ciphertext, nil
	}

	return string(plaintext), nil
}
