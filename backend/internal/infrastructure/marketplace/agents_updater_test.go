package marketplace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentsUpdater_ParseSkillFrontmatter(t *testing.T) {
	updater := NewAgentsUpdater()

	skillContent := `---
name: test-skill
description: This is a test skill for testing purposes.
---

# Test Skill

This is the skill content.
`

	metadata, err := updater.ParseSkillFrontmatter([]byte(skillContent))
	if err != nil {
		t.Fatalf("解析 frontmatter 失败: %v", err)
	}

	if metadata.Name != "test-skill" {
		t.Errorf("技能名称不匹配: 期望 'test-skill', 得到 '%s'", metadata.Name)
	}

	if metadata.Description != "This is a test skill for testing purposes." {
		t.Errorf("技能描述不匹配: 期望 'This is a test skill for testing purposes.', 得到 '%s'", metadata.Description)
	}
}

func TestAgentsUpdater_AddRemoveSkillFromAgentsMD(t *testing.T) {
	// 创建临时目录和 AGENTS.md 文件
	tempDir := t.TempDir()
	agentsPath := filepath.Join(tempDir, "AGENTS.md")

	// 创建初始 AGENTS.md 内容
	initialContent := `# AGENTS

<skills_system priority="1">

## Available Skills

<!-- SKILLS_TABLE_START -->
<usage>
How to use skills...
</usage>

<available_skills>

<skill>
<name>existing-skill</name>
<description>An existing skill</description>
<location>global</location>
</skill>

</available_skills>
<!-- SKILLS_TABLE_END -->

</skills_system>
`

	if err := os.WriteFile(agentsPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("创建 AGENTS.md 失败: %v", err)
	}

	// 切换到临时目录
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前目录失败: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}

	updater := NewAgentsUpdater()

	// 测试添加技能
	metadata := &SkillMetadata{
		Name:        "new-skill",
		Description: "A new skill added by marketplace",
	}

	if err := updater.AddSkillToAgentsMD(agentsPath, metadata); err != nil {
		t.Fatalf("添加技能失败: %v", err)
	}

	// 验证技能已添加
	content, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("读取 AGENTS.md 失败: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "<name>new-skill</name>") {
		t.Error("技能未添加到 AGENTS.md")
	}

	if !strings.Contains(contentStr, "<name>existing-skill</name>") {
		t.Error("原有技能被意外删除")
	}

	// 测试移除技能
	if err := updater.RemoveSkillFromAgentsMD(agentsPath, "new-skill"); err != nil {
		t.Fatalf("移除技能失败: %v", err)
	}

	// 验证技能已移除
	content, err = os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("读取 AGENTS.md 失败: %v", err)
	}

	contentStr = string(content)
	if strings.Contains(contentStr, "<name>new-skill</name>") {
		t.Error("技能未从 AGENTS.md 中移除")
	}

	if !strings.Contains(contentStr, "<name>existing-skill</name>") {
		t.Error("原有技能被意外删除")
	}
}

func TestAgentsUpdater_FindAgentsMDFile(t *testing.T) {
	// 创建临时目录结构
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	agentsPath := filepath.Join(tempDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n"), 0644); err != nil {
		t.Fatalf("创建 AGENTS.md 失败: %v", err)
	}

	// 切换到子目录
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前目录失败: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}

	updater := NewAgentsUpdater()

	// 应该能找到父目录的 AGENTS.md
	foundPath, err := updater.FindAgentsMDFile(subDir)
	if err != nil {
		t.Fatalf("查找 AGENTS.md 失败: %v", err)
	}

	if foundPath != agentsPath {
		t.Errorf("找到的路径不匹配: 期望 '%s', 得到 '%s'", agentsPath, foundPath)
	}
}

func TestAgentsUpdater_FindAgentsMDFile_CreateIfNotExists(t *testing.T) {
	// 创建临时目录（不包含 AGENTS.md）
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	updater := NewAgentsUpdater()

	// 应该在工作区根目录创建 AGENTS.md
	expectedPath := filepath.Join(workspaceDir, "AGENTS.md")
	foundPath, err := updater.FindAgentsMDFile(workspaceDir)
	if err != nil {
		t.Fatalf("创建 AGENTS.md 失败: %v", err)
	}

	if foundPath != expectedPath {
		t.Errorf("创建的路径不匹配: 期望 '%s', 得到 '%s'", expectedPath, foundPath)
	}

	// 验证文件已创建
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("AGENTS.md 文件未创建")
	}

	// 验证文件内容包含必要的标记
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("读取 AGENTS.md 失败: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "<!-- SKILLS_TABLE_START -->") {
		t.Error("AGENTS.md 缺少 SKILLS_TABLE_START 标记")
	}
	if !strings.Contains(contentStr, "<!-- SKILLS_TABLE_END -->") {
		t.Error("AGENTS.md 缺少 SKILLS_TABLE_END 标记")
	}
	if !strings.Contains(contentStr, "<available_skills>") {
		t.Error("AGENTS.md 缺少 available_skills 标记")
	}
}
