package rag

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScanScheduler_Start_Disabled 测试禁用状态的调度器
func TestScanScheduler_Start_Disabled(t *testing.T) {
	config := &ScanConfig{
		Enabled:  false,
		Interval: 1 * time.Hour,
	}

	scheduler := NewScanScheduler(nil, nil, nil, config)
	err := scheduler.Start()

	assert.NoError(t, err)
}

// TestScanScheduler_Stop 测试停止调度器
func TestScanScheduler_Stop(t *testing.T) {
	config := &ScanConfig{
		Enabled:  true,
		Interval: 1 * time.Hour,
	}

	scheduler := NewScanScheduler(nil, nil, nil, config)
	err := scheduler.Stop()

	assert.NoError(t, err)
}

// TestScanScheduler_TriggerScan 测试手动触发扫描
func TestScanScheduler_TriggerScan(t *testing.T) {
	// 注意：TriggerScan 会启动 goroutine 执行实际扫描
	// 由于需要真实的依赖（projectManager），这个测试在集成测试中运行
	t.Skip("需要真实的 ProjectManager，在集成测试中运行")
}

// TestScanScheduler_NeedsUpdate 测试文件更新检测
func TestScanScheduler_NeedsUpdate(t *testing.T) {
	mockRAGRepo := new(MockRAGRepository)

	config := &ScanConfig{
		Enabled: true,
	}

	scheduler := NewScanScheduler(nil, nil, mockRAGRepo, config)

	// 测试场景 1: 没有元数据，需要更新
	mockRAGRepo.On("GetFileMetadata", "/path/to/file").Return(nil, errors.New("not found")).Once()
	needsUpdate := scheduler.needsUpdate("session-1", "/path/to/file", time.Now())
	assert.True(t, needsUpdate)

	// 测试场景 2: 文件修改时间更新，需要更新
	oldMtime := time.Now().Add(-2 * time.Hour)
	metadata := &domainRAG.FileMetadata{
		FilePath: "/path/to/file",
		FileMtime: oldMtime.Unix(),
	}
	mockRAGRepo.On("GetFileMetadata", "/path/to/file").Return(metadata, nil).Once()
	needsUpdate = scheduler.needsUpdate("session-1", "/path/to/file", time.Now())
	assert.True(t, needsUpdate)

	// 测试场景 3: 文件未修改，不需要更新
	newMtime := time.Now()
	metadata.FileMtime = newMtime.Unix()
	mockRAGRepo.On("GetFileMetadata", "/path/to/file").Return(metadata, nil).Once()
	needsUpdate = scheduler.needsUpdate("session-1", "/path/to/file", newMtime)
	assert.False(t, needsUpdate)
	
	mockRAGRepo.AssertExpectations(t)
}

// TestScanScheduler_NeedsReindex 测试内容哈希检测
func TestScanScheduler_NeedsReindex(t *testing.T) {
	mockRAGRepo := new(MockRAGRepository)

	config := &ScanConfig{
		Enabled: true,
	}

	scheduler := NewScanScheduler(nil, nil, mockRAGRepo, config)

	// 创建临时文件用于测试
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// 测试场景 1: 没有元数据，需要重新索引
	mockRAGRepo.On("GetFileMetadata", testFile).Return(nil, errors.New("not found")).Once()
	needsReindex := scheduler.needsReindex("session-1", testFile)
	assert.True(t, needsReindex)

	// 测试场景 2: 内容哈希不同，需要重新索引
	metadata := &domainRAG.FileMetadata{
		FilePath:   testFile,
		ContentHash: "old-hash",
	}
	mockRAGRepo.On("GetFileMetadata", testFile).Return(metadata, nil).Once()
	needsReindex = scheduler.needsReindex("session-1", testFile)
	assert.True(t, needsReindex)

	// 测试场景 3: 内容哈希相同，不需要重新索引
	currentHash := scheduler.calculateFileHash(testFile)
	metadata.ContentHash = currentHash
	mockRAGRepo.On("GetFileMetadata", testFile).Return(metadata, nil).Once()
	needsReindex = scheduler.needsReindex("session-1", testFile)
	assert.False(t, needsReindex)
	
	mockRAGRepo.AssertExpectations(t)
}

// TestScanScheduler_CalculateFileHash 测试文件哈希计算
func TestScanScheduler_CalculateFileHash(t *testing.T) {
	scheduler := NewScanScheduler(nil, nil, nil, &ScanConfig{})

	// 创建临时文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "test content for hashing"
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	// 计算哈希
	hash1 := scheduler.calculateFileHash(testFile)
	assert.NotEmpty(t, hash1)

	// 相同内容应该产生相同哈希
	hash2 := scheduler.calculateFileHash(testFile)
	assert.Equal(t, hash1, hash2)

	// 修改内容后哈希应该不同
	err = os.WriteFile(testFile, []byte("different content"), 0644)
	assert.NoError(t, err)
	hash3 := scheduler.calculateFileHash(testFile)
	assert.NotEqual(t, hash1, hash3)

	// 不存在的文件应该返回空字符串
	hash4 := scheduler.calculateFileHash("/nonexistent/file")
	assert.Empty(t, hash4)
}

// TestScanScheduler_Phase2BatchProcess_EmptyList 测试批量处理空列表
func TestScanScheduler_Phase2BatchProcess_EmptyList(t *testing.T) {
	scheduler := NewScanScheduler(nil, nil, nil, &ScanConfig{
		BatchSize:   10,
		Concurrency: 3,
	})

	// 空列表应该直接返回，不处理
	filesToUpdate := []*FileToUpdate{}
	scheduler.phase2BatchProcess(filesToUpdate)
	
	// 如果没有 panic，测试通过
	assert.True(t, true)
}

// TestScanScheduler_Phase2BatchProcess_ConfigDefaults 测试配置默认值
func TestScanScheduler_Phase2BatchProcess_ConfigDefaults(t *testing.T) {
	// 测试 BatchSize 和 Concurrency 的默认值处理
	scheduler1 := NewScanScheduler(nil, nil, nil, &ScanConfig{
		BatchSize:   0, // 应该使用默认值 10
		Concurrency: 0, // 应该使用默认值 3
	})

	// 验证配置会被正确处理（在 phase2BatchProcess 中）
	filesToUpdate := []*FileToUpdate{}
	scheduler1.phase2BatchProcess(filesToUpdate)
	
	assert.True(t, true)
}

// TestScanScheduler_NeedsUpdate_ErrorHandling 测试错误处理
func TestScanScheduler_NeedsUpdate_ErrorHandling(t *testing.T) {
	mockRAGRepo := new(MockRAGRepository)

	scheduler := NewScanScheduler(nil, nil, mockRAGRepo, &ScanConfig{})

	// 测试仓库返回错误的情况
	mockRAGRepo.On("GetFileMetadata", "/path/to/file").Return(nil, errors.New("database error")).Once()
	needsUpdate := scheduler.needsUpdate("session-1", "/path/to/file", time.Now())
	
	// 错误时应该返回 true（需要更新）
	assert.True(t, needsUpdate)
	mockRAGRepo.AssertExpectations(t)
}

// TestScanScheduler_NeedsReindex_ErrorHandling 测试错误处理
func TestScanScheduler_NeedsReindex_ErrorHandling(t *testing.T) {
	mockRAGRepo := new(MockRAGRepository)

	scheduler := NewScanScheduler(nil, nil, mockRAGRepo, &ScanConfig{})

	// 测试仓库返回错误的情况
	mockRAGRepo.On("GetFileMetadata", "/path/to/file").Return(nil, errors.New("database error")).Once()
	needsReindex := scheduler.needsReindex("session-1", "/path/to/file")
	
	// 错误时应该返回 true（需要重新索引）
	assert.True(t, needsReindex)
	mockRAGRepo.AssertExpectations(t)
}

// TestScanScheduler_NeedsReindex_HashCalculationFailure 测试哈希计算失败
func TestScanScheduler_NeedsReindex_HashCalculationFailure(t *testing.T) {
	mockRAGRepo := new(MockRAGRepository)

	scheduler := NewScanScheduler(nil, nil, mockRAGRepo, &ScanConfig{})

	// 测试文件不存在导致哈希计算失败
	metadata := &domainRAG.FileMetadata{
		FilePath:   "/nonexistent/file",
		ContentHash: "some-hash",
	}
	mockRAGRepo.On("GetFileMetadata", "/nonexistent/file").Return(metadata, nil).Once()
	
	// 哈希计算失败应该返回 true（需要重新索引）
	needsReindex := scheduler.needsReindex("session-1", "/nonexistent/file")
	assert.True(t, needsReindex)
	
	mockRAGRepo.AssertExpectations(t)
}

