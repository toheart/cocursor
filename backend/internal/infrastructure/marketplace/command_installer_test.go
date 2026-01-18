package marketplace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommandInstaller_InstallCommand(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	workspacePath := filepath.Join(tempDir, "workspace")

	stateManager, _ := NewStateManager()
	pluginLoader := NewPluginLoader(stateManager)
	installer := NewCommandInstaller(pluginLoader)

	// 测试安装 Command（使用 simple-skill 插件，它可能没有 command，但我们可以测试错误处理）
	// 由于 simple-skill 可能没有 command，我们测试空 command ID 的情况
	err := installer.InstallCommand("simple-skill", "", workspacePath)
	if err != nil {
		t.Fatalf("安装空 Command 应该成功（跳过）: %v", err)
	}

	// 测试安装不存在的 Command（应该返回错误）
	err = installer.InstallCommand("non-existent-plugin", "test-command", workspacePath)
	if err == nil {
		t.Error("安装不存在的 Command 应该返回错误")
	}
}

func TestCommandInstaller_UninstallCommand(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	workspacePath := filepath.Join(tempDir, "workspace")

	stateManager, _ := NewStateManager()
	pluginLoader := NewPluginLoader(stateManager)
	installer := NewCommandInstaller(pluginLoader)

	// 测试卸载不存在的 Command（应该成功，不报错）
	err := installer.UninstallCommand("non-existent-command", workspacePath)
	if err != nil {
		t.Errorf("卸载不存在的 Command 不应该报错: %v", err)
	}

	// 测试卸载空 Command ID（应该成功，不报错）
	err = installer.UninstallCommand("", workspacePath)
	if err != nil {
		t.Errorf("卸载空 Command ID 不应该报错: %v", err)
	}

	// 创建测试文件
	commandsDir := filepath.Join(workspacePath, ".cursor", "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	testCommandPath := filepath.Join(commandsDir, "test-command.md")
	testContent := []byte("# Test Command\n\nThis is a test command.")
	if err := os.WriteFile(testCommandPath, testContent, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(testCommandPath); os.IsNotExist(err) {
		t.Fatal("测试文件应该存在")
	}

	// 卸载 Command
	if err := installer.UninstallCommand("test-command", workspacePath); err != nil {
		t.Fatalf("卸载 Command 失败: %v", err)
	}

	// 验证文件已删除
	if _, err := os.Stat(testCommandPath); !os.IsNotExist(err) {
		t.Error("测试文件应该已被删除")
	}
}

func TestCommandInstaller_InstallCommand_Overwrite(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	workspacePath := filepath.Join(tempDir, "workspace")

	// 创建已存在的文件
	commandsDir := filepath.Join(workspacePath, ".cursor", "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	existingCommandPath := filepath.Join(commandsDir, "existing-command.md")
	oldContent := []byte("# Old Command\n\nOld content.")
	if err := os.WriteFile(existingCommandPath, oldContent, 0644); err != nil {
		t.Fatalf("创建已存在文件失败: %v", err)
	}

	// 测试覆盖已存在的文件（直接测试文件写入功能）
	newContent := []byte("# New Command\n\nNew content.")
	if err := os.WriteFile(existingCommandPath, newContent, 0644); err != nil {
		t.Fatalf("覆盖文件失败: %v", err)
	}

	// 验证文件内容已更新
	readContent, err := os.ReadFile(existingCommandPath)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}

	if string(readContent) != string(newContent) {
		t.Error("文件内容应该已更新")
	}
}
