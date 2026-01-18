package marketplace

import (
	"os"
	"testing"

	domainMarketplace "github.com/cocursor/backend/internal/domain/marketplace"
)

func TestPluginLoader_LoadPlugins(t *testing.T) {
	// 使用临时目录作为状态文件路径
	tempDir := t.TempDir()
	originalStatePath := os.Getenv("HOME")
	defer os.Setenv("HOME", originalStatePath)

	// 设置临时目录为 HOME
	os.Setenv("HOME", tempDir)

	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	loader := NewPluginLoader(stateManager)

	plugins, err := loader.LoadPlugins()
	if err != nil {
		t.Fatalf("加载插件失败: %v", err)
	}

	if len(plugins) == 0 {
		t.Fatal("没有加载到任何插件")
	}

	// 验证 simple-skill 插件
	found := false
	for _, plugin := range plugins {
		if plugin.ID == "simple-skill" {
			found = true
			if plugin.Name != "简单技能" {
				t.Errorf("插件名称不匹配: 期望 '简单技能', 得到 '%s'", plugin.Name)
			}
			if plugin.Skill.SkillName != "simple-skill" {
				t.Errorf("Skill 名称不匹配: 期望 'simple-skill', 得到 '%s'", plugin.Skill.SkillName)
			}
			break
		}
	}

	if !found {
		t.Fatal("未找到 simple-skill 插件")
	}
}

func TestPluginLoader_LoadPlugin(t *testing.T) {
	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	loader := NewPluginLoader(stateManager)

	plugin, err := loader.LoadPlugin("simple-skill")
	if err != nil {
		t.Fatalf("加载插件失败: %v", err)
	}

	if plugin.ID != "simple-skill" {
		t.Errorf("插件 ID 不匹配: 期望 'simple-skill', 得到 '%s'", plugin.ID)
	}

	if err := plugin.Validate(); err != nil {
		t.Errorf("插件验证失败: %v", err)
	}
}

func TestPluginLoader_ReadSkillFiles(t *testing.T) {
	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	loader := NewPluginLoader(stateManager)

	files, err := loader.ReadSkillFiles("simple-skill")
	if err != nil {
		t.Fatalf("读取 Skill 文件失败: %v", err)
	}

	// 验证 SKILL.md 文件存在
	if _, exists := files["SKILL.md"]; !exists {
		t.Fatal("未找到 SKILL.md 文件")
	}

	// 验证文件内容
	skillContent := string(files["SKILL.md"])
	if len(skillContent) == 0 {
		t.Error("SKILL.md 文件内容为空")
	}
}

func TestPluginLoader_ExtractEnvVars(t *testing.T) {
	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	loader := NewPluginLoader(stateManager)

	tests := []struct {
		name     string
		headers  map[string]string
		expected []string
	}{
		{
			name: "单个环境变量",
			headers: map[string]string{
				"Authorization": "Bearer ${env:TOKEN}",
			},
			expected: []string{"TOKEN"},
		},
		{
			name: "多个环境变量",
			headers: map[string]string{
				"Authorization": "Bearer ${env:TOKEN}",
				"X-API-Key":     "${env:API_KEY}",
			},
			expected: []string{"TOKEN", "API_KEY"},
		},
		{
			name: "同一变量多次出现",
			headers: map[string]string{
				"Authorization": "Bearer ${env:TOKEN}",
				"X-Token":       "${env:TOKEN}",
			},
			expected: []string{"TOKEN"},
		},
		{
			name: "无环境变量",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars := loader.ExtractEnvVars(tt.headers)

			if len(envVars) != len(tt.expected) {
				t.Errorf("环境变量数量不匹配: 期望 %d, 得到 %d", len(tt.expected), len(envVars))
			}

			// 验证变量名
			expectedMap := make(map[string]bool)
			for _, v := range tt.expected {
				expectedMap[v] = true
			}

			for _, v := range envVars {
				if !expectedMap[v] {
					t.Errorf("意外的环境变量: %s", v)
				}
			}
		})
	}
}

func TestStateManager_ReadWriteState(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Setenv("HOME", tempDir)

	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	// 测试写入状态
	state := domainMarketplace.NewPluginState()
	state.SetInstalled("test-plugin", "1.0.0")

	if err := stateManager.WriteState(state); err != nil {
		t.Fatalf("写入状态失败: %v", err)
	}

	// 测试读取状态
	readState, err := stateManager.ReadState()
	if err != nil {
		t.Fatalf("读取状态失败: %v", err)
	}

	if !readState.IsInstalled("test-plugin") {
		t.Error("插件未标记为已安装")
	}

	if readState.GetInstalledVersion("test-plugin") != "1.0.0" {
		t.Errorf("版本不匹配: 期望 '1.0.0', 得到 '%s'", readState.GetInstalledVersion("test-plugin"))
	}
}

func TestStateManager_UpdateRemoveInstalledPlugin(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Setenv("HOME", tempDir)

	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	// 测试更新安装状态
	if err := stateManager.UpdateInstalledPlugin("test-plugin", "1.0.0"); err != nil {
		t.Fatalf("更新安装状态失败: %v", err)
	}

	state, err := stateManager.ReadState()
	if err != nil {
		t.Fatalf("读取状态失败: %v", err)
	}

	if !state.IsInstalled("test-plugin") {
		t.Error("插件未标记为已安装")
	}

	// 测试移除安装状态
	if err := stateManager.RemoveInstalledPlugin("test-plugin"); err != nil {
		t.Fatalf("移除安装状态失败: %v", err)
	}

	state, err = stateManager.ReadState()
	if err != nil {
		t.Fatalf("读取状态失败: %v", err)
	}

	if state.IsInstalled("test-plugin") {
		t.Error("插件仍标记为已安装")
	}
}

func TestStateManager_ReadState_FileNotExists(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Setenv("HOME", tempDir)

	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	// 文件不存在时应该返回空状态
	state, err := stateManager.ReadState()
	if err != nil {
		t.Fatalf("读取状态失败: %v", err)
	}

	if state == nil {
		t.Fatal("状态为 nil")
	}

	if len(state.InstalledPlugins) != 0 {
		t.Errorf("期望空状态，但得到 %d 个已安装插件", len(state.InstalledPlugins))
	}
}

func TestPluginLoader_LoadPlugins_WithInstalledState(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Setenv("HOME", tempDir)

	stateManager, err := NewStateManager()
	if err != nil {
		t.Fatalf("创建 StateManager 失败: %v", err)
	}

	// 先标记插件为已安装
	if err := stateManager.UpdateInstalledPlugin("simple-skill", "1.0.0"); err != nil {
		t.Fatalf("更新安装状态失败: %v", err)
	}

	loader := NewPluginLoader(stateManager)

	plugins, err := loader.LoadPlugins()
	if err != nil {
		t.Fatalf("加载插件失败: %v", err)
	}

	// 查找 simple-skill 插件
	var plugin *domainMarketplace.Plugin
	for _, p := range plugins {
		if p.ID == "simple-skill" {
			plugin = p
			break
		}
	}

	if plugin == nil {
		t.Fatal("未找到 simple-skill 插件")
	}

	if !plugin.Installed {
		t.Error("插件未标记为已安装")
	}

	if plugin.InstalledVersion != "1.0.0" {
		t.Errorf("已安装版本不匹配: 期望 '1.0.0', 得到 '%s'", plugin.InstalledVersion)
	}
}
