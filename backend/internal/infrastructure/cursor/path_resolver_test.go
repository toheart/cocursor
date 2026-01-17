package cursor

import (
	"testing"
)

// TestGetWorkspaceIDByPath 测试根据项目路径查找工作区 ID
// 验收标准：输入项目路径 D:/code/cocursor，能输出工作区 ID（如 d4b798d4...）
func TestGetWorkspaceIDByPath(t *testing.T) {
	resolver := NewPathResolver()

	// 测试路径：D:/code/cocursor
	testPath := "D:/code/cocursor"

	workspaceID, err := resolver.GetWorkspaceIDByPath(testPath)
	if err != nil {
		t.Fatalf("GetWorkspaceIDByPath failed: %v", err)
	}

	if workspaceID == "" {
		t.Fatal("workspaceID is empty")
	}

	t.Logf("项目路径: %s", testPath)
	t.Logf("工作区 ID: %s", workspaceID)

	// 验证工作区 ID 格式（应该是 32 位十六进制字符串）
	if len(workspaceID) != 32 {
		t.Errorf("workspaceID length should be 32, got %d", len(workspaceID))
	}
}

// TestGetGlobalStoragePath 测试获取全局存储路径
func TestGetGlobalStoragePath(t *testing.T) {
	resolver := NewPathResolver()

	path, err := resolver.GetGlobalStoragePath()
	if err != nil {
		t.Fatalf("GetGlobalStoragePath failed: %v", err)
	}

	if path == "" {
		t.Fatal("path is empty")
	}

	t.Logf("全局存储路径: %s", path)
}

// TestGetWorkspaceStorageDir 测试获取工作区存储目录
func TestGetWorkspaceStorageDir(t *testing.T) {
	resolver := NewPathResolver()

	dir, err := resolver.GetWorkspaceStorageDir()
	if err != nil {
		t.Fatalf("GetWorkspaceStorageDir failed: %v", err)
	}

	if dir == "" {
		t.Fatal("dir is empty")
	}

	t.Logf("工作区存储目录: %s", dir)
}
