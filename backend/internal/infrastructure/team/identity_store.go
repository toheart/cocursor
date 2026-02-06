package team

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cocursor/backend/internal/infrastructure/config"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

// IdentityStore 身份存储
type IdentityStore struct {
	mu       sync.RWMutex
	filePath string
	identity *domainTeam.Identity
}

// NewIdentityStore 创建身份存储
func NewIdentityStore() (*IdentityStore, error) {
	filePath := filepath.Join(config.GetDataDir(), "team", "identity.json")

	store := &IdentityStore{
		filePath: filePath,
	}

	// 尝试加载现有身份
	identity, err := store.load()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	store.identity = identity

	return store, nil
}

// load 从文件加载身份
func (s *IdentityStore) load() (*domainTeam.Identity, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	var identity domainTeam.Identity
	if err := json.Unmarshal(data, &identity); err != nil {
		return nil, fmt.Errorf("failed to parse identity file: %w", err)
	}

	return &identity, nil
}

// save 保存身份到文件
func (s *IdentityStore) save(identity *domainTeam.Identity) error {
	// 确保目录存在
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal identity: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write identity file: %w", err)
	}

	return nil
}

// Get 获取当前身份
// 当内存缓存为空时，尝试从文件重新加载（支持多实例间的数据共享）
func (s *IdentityStore) Get() (*domainTeam.Identity, error) {
	s.mu.RLock()
	if s.identity != nil {
		identityCopy := *s.identity
		s.mu.RUnlock()
		return &identityCopy, nil
	}
	s.mu.RUnlock()

	// 内存缓存为空，尝试从文件重新加载
	s.mu.Lock()
	defer s.mu.Unlock()

	// 双重检查
	if s.identity != nil {
		identityCopy := *s.identity
		return &identityCopy, nil
	}

	identity, err := s.load()
	if err != nil {
		return nil, domainTeam.ErrIdentityNotFound
	}
	s.identity = identity

	identityCopy := *s.identity
	return &identityCopy, nil
}

// Exists 检查身份是否存在
func (s *IdentityStore) Exists() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.identity != nil
}

// Create 创建新身份
func (s *IdentityStore) Create(name string) (*domainTeam.Identity, error) {
	if name == "" {
		return nil, domainTeam.ErrIdentityNameRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	identity := &domainTeam.Identity{
		ID:        uuid.New().String(),
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.save(identity); err != nil {
		return nil, err
	}

	s.identity = identity
	return identity, nil
}

// UpdateName 更新名称
func (s *IdentityStore) UpdateName(name string) (*domainTeam.Identity, error) {
	if name == "" {
		return nil, domainTeam.ErrIdentityNameRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.identity == nil {
		return nil, domainTeam.ErrIdentityNotFound
	}

	s.identity.Name = name
	s.identity.UpdatedAt = time.Now()

	if err := s.save(s.identity); err != nil {
		return nil, err
	}

	identityCopy := *s.identity
	return &identityCopy, nil
}

// Delete 删除身份
func (s *IdentityStore) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.identity = nil

	// 删除文件（忽略不存在的错误）
	if err := os.Remove(s.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete identity file: %w", err)
	}

	return nil
}
