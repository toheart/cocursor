package watcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cocursor/backend/internal/infrastructure/config"
)

// ScanMetadata 扫描元数据管理
// 用于记录上次扫描时间，决定启动时是否需要全量扫描
type ScanMetadata struct {
	mu           sync.RWMutex
	lastScanTime time.Time
	filePath     string
}

// scanMetadataData 元数据文件结构
type scanMetadataData struct {
	LastScanTime time.Time `json:"last_scan_time"`
}

// NewScanMetadata 创建扫描元数据管理器
func NewScanMetadata() *ScanMetadata {
	filePath := filepath.Join(config.GetDataDir(), "scan_metadata.json")

	sm := &ScanMetadata{
		filePath: filePath,
	}

	// 从文件加载
	sm.load()

	return sm
}

// GetLastScanTime 获取上次扫描时间
func (sm *ScanMetadata) GetLastScanTime() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.lastScanTime
}

// SetLastScanTime 设置上次扫描时间
func (sm *ScanMetadata) SetLastScanTime(t time.Time) {
	sm.mu.Lock()
	sm.lastScanTime = t
	sm.mu.Unlock()

	// 持久化到文件
	sm.save()
}

// load 从文件加载元数据
func (sm *ScanMetadata) load() {
	data, err := os.ReadFile(sm.filePath)
	if err != nil {
		return // 文件不存在或无法读取，使用默认值
	}

	var metadata scanMetadataData
	if err := json.Unmarshal(data, &metadata); err != nil {
		return
	}

	sm.mu.Lock()
	sm.lastScanTime = metadata.LastScanTime
	sm.mu.Unlock()
}

// save 保存元数据到文件
func (sm *ScanMetadata) save() {
	sm.mu.RLock()
	metadata := scanMetadataData{
		LastScanTime: sm.lastScanTime,
	}
	sm.mu.RUnlock()

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return
	}

	// 确保目录存在
	dir := filepath.Dir(sm.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	// 写入文件
	_ = os.WriteFile(sm.filePath, data, 0644)
}
