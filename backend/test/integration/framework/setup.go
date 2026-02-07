//go:build integration
// +build integration

// 测试框架的全局设置和清理
package framework

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var (
	// BinaryPath 编译后的守护进程二进制路径
	BinaryPath string
)

// BuildDaemon 编译 cocursor-daemon 二进制（在 TestMain 中调用一次）
func BuildDaemon() error {
	// 获取项目根目录
	_, currentFile, _, _ := runtime.Caller(0)
	backendDir := filepath.Join(filepath.Dir(currentFile), "..", "..", "..")

	// 构建输出路径
	tmpDir, err := os.MkdirTemp("", "cocursor-test-bin-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	binaryName := "cocursor-daemon"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	BinaryPath = filepath.Join(tmpDir, binaryName)

	// 编译二进制
	cmd := exec.Command("go", "build", "-o", BinaryPath, "./cmd/server")
	cmd.Dir = backendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build daemon binary: %w", err)
	}

	return nil
}

// Cleanup 清理构建的二进制（在 TestMain 结束时调用）
func Cleanup() {
	if BinaryPath != "" {
		os.RemoveAll(filepath.Dir(BinaryPath))
	}
}

// RequireDaemonBinary 检查二进制是否已构建
func RequireDaemonBinary(t *testing.T) {
	t.Helper()
	if BinaryPath == "" {
		t.Fatal("daemon binary not built, call BuildDaemon() in TestMain first")
	}
	if _, err := os.Stat(BinaryPath); os.IsNotExist(err) {
		t.Fatal("daemon binary not found at: " + BinaryPath)
	}
}
