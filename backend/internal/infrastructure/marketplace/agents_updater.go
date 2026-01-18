package marketplace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillMetadata Skill 元数据（从 SKILL.md frontmatter 提取）
type SkillMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// AgentsUpdater AGENTS.md 更新器
type AgentsUpdater struct{}

// NewAgentsUpdater 创建 AGENTS.md 更新器
func NewAgentsUpdater() *AgentsUpdater {
	return &AgentsUpdater{}
}

// ParseSkillFrontmatter 解析 SKILL.md 的 frontmatter
func (a *AgentsUpdater) ParseSkillFrontmatter(skillContent []byte) (*SkillMetadata, error) {
	content := string(skillContent)

	// 查找 frontmatter（--- 之间的内容）
	frontmatterRegex := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---`)
	matches := frontmatterRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, fmt.Errorf("no frontmatter found in skill file")
	}

	var metadata SkillMetadata
	if err := yaml.Unmarshal([]byte(matches[1]), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if metadata.Name == "" {
		return nil, fmt.Errorf("skill name is required in frontmatter")
	}
	if metadata.Description == "" {
		return nil, fmt.Errorf("skill description is required in frontmatter")
	}

	return &metadata, nil
}

// FindAgentsMDFile 查找项目的 AGENTS.md 文件
// workspacePath: 工作区路径（由前端传递）
// 从工作区路径向上查找，直到找到包含 AGENTS.md 的目录
// 如果找不到，在工作区根目录创建 AGENTS.md 文件
func (a *AgentsUpdater) FindAgentsMDFile(workspacePath string) (string, error) {
	if workspacePath == "" {
		return "", fmt.Errorf("workspace path is required")
	}

	// 规范化工作区路径
	absPath, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 确定工作区根目录（如果是文件，取其父目录）
	workspaceRoot := absPath
	if info, err := os.Stat(workspaceRoot); err == nil && !info.IsDir() {
		workspaceRoot = filepath.Dir(workspaceRoot)
	}

	// 向上查找 AGENTS.md（从工作区根目录开始）
	dir := workspaceRoot
	for {
		agentsPath := filepath.Join(dir, "AGENTS.md")
		if _, err := os.Stat(agentsPath); err == nil {
			return agentsPath, nil
		}

		// 到达根目录，停止查找
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// 如果找不到，在工作区根目录创建 AGENTS.md

	agentsPath := filepath.Join(workspaceRoot, "AGENTS.md")

	// 创建默认的 AGENTS.md 文件
	defaultContent := `# AGENTS

<skills_system priority="1">

## Available Skills

<!-- SKILLS_TABLE_START -->
<usage>
When users ask you to perform tasks, check if any of the available skills below can help complete the task more effectively. Skills provide specialized capabilities and domain knowledge.

How to use skills:
- Invoke: Bash("openskills read <skill-name>")
- The skill content will load with detailed instructions on how to complete the task
- Base directory provided in output for resolving bundled resources (references/, scripts/, assets/)

Usage notes:
- Only use skills listed in <available_skills> below
- Do not invoke a skill that is already loaded in your context
- Each skill invocation is stateless
</usage>

<available_skills>

</available_skills>
<!-- SKILLS_TABLE_END -->

</skills_system>
`

	// 确保目录存在
	if err := os.MkdirAll(workspaceRoot, 0755); err != nil {
		return "", fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// 创建文件
	if err := os.WriteFile(agentsPath, []byte(defaultContent), 0644); err != nil {
		return "", fmt.Errorf("failed to create AGENTS.md: %w", err)
	}

	return agentsPath, nil
}

// AddSkillToAgentsMD 在 AGENTS.md 中添加技能条目
func (a *AgentsUpdater) AddSkillToAgentsMD(agentsPath string, metadata *SkillMetadata) error {
	content, err := os.ReadFile(agentsPath)
	if err != nil {
		return fmt.Errorf("failed to read AGENTS.md: %w", err)
	}

	contentStr := string(content)

	// 检查技能是否已存在
	skillPattern := regexp.MustCompile(fmt.Sprintf(`<skill>\s*<name>%s</name>`, regexp.QuoteMeta(metadata.Name)))
	if skillPattern.MatchString(contentStr) {
		// 技能已存在，跳过
		return nil
	}

	// 查找 `<!-- SKILLS_TABLE_END -->` 标记
	endMarker := "<!-- SKILLS_TABLE_END -->"
	endIndex := strings.Index(contentStr, endMarker)
	if endIndex == -1 {
		return fmt.Errorf("SKILLS_TABLE_END marker not found in AGENTS.md")
	}

	// 构建新的 skill 条目
	skillEntry := fmt.Sprintf(`<skill>
<name>%s</name>
<description>%s</description>
<location>global</location>
</skill>

`, metadata.Name, metadata.Description)

	// 在 endMarker 之前插入 skill 条目
	newContent := contentStr[:endIndex] + skillEntry + contentStr[endIndex:]

	// 写入文件
	if err := os.WriteFile(agentsPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write AGENTS.md: %w", err)
	}

	return nil
}

// RemoveSkillFromAgentsMD 从 AGENTS.md 中移除技能条目
func (a *AgentsUpdater) RemoveSkillFromAgentsMD(agentsPath string, skillName string) error {
	content, err := os.ReadFile(agentsPath)
	if err != nil {
		return fmt.Errorf("failed to read AGENTS.md: %w", err)
	}

	contentStr := string(content)

	// 查找并移除 skill 条目（包括前后的空行）
	// 匹配整个 skill 块，包括可能的空行
	skillPattern := regexp.MustCompile(
		fmt.Sprintf(`(?s)<skill>\s*<name>%s</name>\s*<description>.*?</description>\s*<location>.*?</location>\s*</skill>\s*\n?`,
			regexp.QuoteMeta(skillName)),
	)

	if !skillPattern.MatchString(contentStr) {
		// 技能不存在，跳过
		return nil
	}

	// 移除 skill 条目
	newContent := skillPattern.ReplaceAllString(contentStr, "")

	// 写入文件
	if err := os.WriteFile(agentsPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write AGENTS.md: %w", err)
	}

	return nil
}
