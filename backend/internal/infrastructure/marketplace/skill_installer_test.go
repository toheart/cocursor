package marketplace

import (
	"os"
	"path/filepath"
	"testing"

	domainMarketplace "github.com/cocursor/backend/internal/domain/marketplace"
)

func TestSkillInstaller_InstallSkill(t *testing.T) {
	tempDir := t.TempDir()
	// Windows 上使用 USERPROFILE，Unix 上使用 HOME
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
		if originalUserProfile != "" {
			os.Setenv("USERPROFILE", originalUserProfile)
		}
	}()

	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)

	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	pluginLoader := NewPluginLoader(stateManager)
	agentsUpdater := NewAgentsUpdater()
	installer := NewSkillInstaller(pluginLoader, stateManager, agentsUpdater)

	// 先检查 ReadSkillFiles 是否返回文件
	files, err := pluginLoader.ReadSkillFiles("simple-skill")
	if err != nil {
		t.Fatalf("读取 Skill 文件失败: %v", err)
	}
	t.Logf("读取到的文件数量: %d", len(files))
	for path := range files {
		t.Logf("文件路径: %s", path)
	}

	// 安装 Skill
	skill := &domainMarketplace.SkillComponent{
		SkillName: "test-skill",
	}

	if err := installer.InstallSkill("simple-skill", skill, "", false); err != nil {
		t.Fatalf("安装 Skill 失败: %v", err)
	}

	// 验证文件是否存在
	skillDir := filepath.Join(tempDir, ".claude", "skills", "test-skill")
	t.Logf("Skill 目录路径: %s", skillDir)

	// 检查目录是否存在
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		t.Fatalf("Skill 目录不存在: %s", skillDir)
	}

	skillFile := filepath.Join(skillDir, "SKILL.md")
	t.Logf("Skill 文件路径: %s", skillFile)

	// 列出目录内容用于调试
	entries, _ := os.ReadDir(skillDir)
	t.Logf("Skill 目录内容: %v", entries)

	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		t.Fatalf("SKILL.md 文件不存在，路径: %s", skillFile)
	}

	// 验证文件内容
	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}

	if len(content) == 0 {
		t.Error("文件内容为空")
	}
}

func TestSkillInstaller_UninstallSkill(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
		if originalUserProfile != "" {
			os.Setenv("USERPROFILE", originalUserProfile)
		}
	}()

	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)

	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	pluginLoader := NewPluginLoader(stateManager)
	agentsUpdater := NewAgentsUpdater()
	installer := NewSkillInstaller(pluginLoader, stateManager, agentsUpdater)

	// 先安装 Skill
	skill := &domainMarketplace.SkillComponent{
		SkillName: "test-skill",
	}

	if err := installer.InstallSkill("simple-skill", skill, "", false); err != nil {
		t.Fatalf("安装 Skill 失败: %v", err)
	}

	// 卸载 Skill
	if err := installer.UninstallSkill("test-skill", ""); err != nil {
		t.Fatalf("卸载 Skill 失败: %v", err)
	}

	// 验证目录已删除
	skillDir := filepath.Join(tempDir, ".claude", "skills", "test-skill")
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Error("Skill 目录未被删除")
	}
}

func TestSkillInstaller_UninstallSkill_NotExists(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
		if originalUserProfile != "" {
			os.Setenv("USERPROFILE", originalUserProfile)
		}
	}()

	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)

	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	pluginLoader := NewPluginLoader(stateManager)
	agentsUpdater := NewAgentsUpdater()
	installer := NewSkillInstaller(pluginLoader, stateManager, agentsUpdater)

	// 卸载不存在的 Skill（应该不报错）
	if err := installer.UninstallSkill("non-existent-skill", ""); err != nil {
		t.Errorf("卸载不存在的 Skill 应该不报错，但得到: %v", err)
	}
}

func TestSkillInstaller_CheckSkillConflict_NoConflict(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
		if originalUserProfile != "" {
			os.Setenv("USERPROFILE", originalUserProfile)
		}
	}()

	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)

	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	pluginLoader := NewPluginLoader(stateManager)
	agentsUpdater := NewAgentsUpdater()
	installer := NewSkillInstaller(pluginLoader, stateManager, agentsUpdater)

	// 检查不存在的 Skill（应该无冲突）
	if err := installer.CheckSkillConflict("new-skill", "new-plugin"); err != nil {
		t.Errorf("不存在的 Skill 应该无冲突，但得到: %v", err)
	}
}

func TestSkillInstaller_CheckSkillConflict_SamePlugin(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
		if originalUserProfile != "" {
			os.Setenv("USERPROFILE", originalUserProfile)
		}
	}()

	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)

	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	pluginLoader := NewPluginLoader(stateManager)
	agentsUpdater := NewAgentsUpdater()
	installer := NewSkillInstaller(pluginLoader, stateManager, agentsUpdater)

	// 先安装 Skill
	skill := &domainMarketplace.SkillComponent{
		SkillName: "test-skill",
	}

	if err := installer.InstallSkill("simple-skill", skill, "", false); err != nil {
		t.Fatalf("安装 Skill 失败: %v", err)
	}

	// 标记为已安装
	if err := stateManager.UpdateInstalledPlugin("simple-skill", "1.0.0"); err != nil {
		t.Fatalf("更新安装状态失败: %v", err)
	}

	// 检查同一插件的 Skill（应该无冲突，允许版本更新）
	if err := installer.CheckSkillConflict("test-skill", "simple-skill"); err != nil {
		t.Errorf("同一插件的 Skill 应该无冲突（允许版本更新），但得到: %v", err)
	}
}

func TestSkillInstaller_CheckSkillConflict_DifferentPlugin(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
		if originalUserProfile != "" {
			os.Setenv("USERPROFILE", originalUserProfile)
		}
	}()

	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)

	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	pluginLoader := NewPluginLoader(stateManager)
	agentsUpdater := NewAgentsUpdater()
	installer := NewSkillInstaller(pluginLoader, stateManager, agentsUpdater)

	// 先安装 Skill（来自 simple-skill 插件）
	skill := &domainMarketplace.SkillComponent{
		SkillName: "test-skill",
	}

	if err := installer.InstallSkill("simple-skill", skill, "", false); err != nil {
		t.Fatalf("安装 Skill 失败: %v", err)
	}

	// 标记 simple-skill 为已安装
	if err := stateManager.UpdateInstalledPlugin("simple-skill", "1.0.0"); err != nil {
		t.Fatalf("更新安装状态失败: %v", err)
	}

	// 检查不同插件的相同 Skill 名称（应该有冲突）
	err = installer.CheckSkillConflict("test-skill", "different-plugin")
	if err == nil {
		t.Error("不同插件使用相同 Skill 名称应该有冲突")
	}

	if conflictErr, ok := err.(*SkillConflictError); !ok {
		t.Errorf("期望 SkillConflictError，但得到: %T", err)
	} else {
		if conflictErr.SkillName != "test-skill" {
			t.Errorf("冲突的 Skill 名称不匹配: 期望 'test-skill', 得到 '%s'", conflictErr.SkillName)
		}
	}
}

func TestSkillInstaller_InstallSkill_WithSubdirectories(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
		if originalUserProfile != "" {
			os.Setenv("USERPROFILE", originalUserProfile)
		}
	}()

	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)

	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	pluginLoader := NewPluginLoader(stateManager)
	agentsUpdater := NewAgentsUpdater()
	installer := NewSkillInstaller(pluginLoader, stateManager, agentsUpdater)

	// 安装 Skill
	skill := &domainMarketplace.SkillComponent{
		SkillName: "test-skill",
	}

	if err := installer.InstallSkill("simple-skill", skill, "", false); err != nil {
		t.Fatalf("安装 Skill 失败: %v", err)
	}

	// 验证主文件存在
	skillDir := filepath.Join(tempDir, ".claude", "skills", "test-skill")
	skillFile := filepath.Join(skillDir, "SKILL.md")

	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		t.Fatal("SKILL.md 文件不存在")
	}
}
