package marketplace

import (
	"fmt"
	"os"
	"path/filepath"

	"log/slog"

	domainMarketplace "github.com/cocursor/backend/internal/domain/marketplace"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// SkillInstaller Skill 安装器
type SkillInstaller struct {
	pluginLoader  *PluginLoader
	stateManager  *StateManager
	agentsUpdater *AgentsUpdater
	logger        *slog.Logger
}

// NewSkillInstaller 创建 Skill 安装器
func NewSkillInstaller(pluginLoader *PluginLoader, stateManager *StateManager, agentsUpdater *AgentsUpdater) *SkillInstaller {
	return &SkillInstaller{
		pluginLoader:  pluginLoader,
		stateManager:  stateManager,
		agentsUpdater: agentsUpdater,
		logger:        log.NewModuleLogger("marketplace", "skill_installer"),
	}
}

// SkillConflictError Skill 名称冲突错误
type SkillConflictError struct {
	SkillName string
	PluginID  string
	Message   string
}

func (e *SkillConflictError) Error() string {
	return e.Message
}

// CheckSkillConflict 检查 Skill 名称冲突
// 返回冲突信息，如果无冲突返回 nil
func (s *SkillInstaller) CheckSkillConflict(skillName string, pluginID string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	skillDir := filepath.Join(homeDir, ".claude", "skills", skillName)

	// 检查目录是否存在
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		// 目录不存在，无冲突
		return nil
	}

	// 目录存在，检查是否为同一插件
	state, err := s.stateManager.ReadState()
	if err != nil {
		// 如果无法读取状态，认为有冲突（安全起见）
		return &SkillConflictError{
			SkillName: skillName,
			PluginID:  pluginID,
			Message:   fmt.Sprintf("skill '%s' already exists and cannot verify if it belongs to the same plugin", skillName),
		}
	}

	// 查找使用相同 skillName 的已安装插件
	for installedPluginID := range state.InstalledPlugins {
		if installedPluginID == pluginID {
			// 是同一插件，允许覆盖（版本更新）
			return nil
		}

		// 检查已安装的插件是否使用相同的 skillName
		installedPluginData, err := s.pluginLoader.LoadPlugin(installedPluginID)
		if err != nil {
			// 无法加载插件信息，跳过
			continue
		}

		if installedPluginData.Skill.SkillName == skillName {
			// 发现冲突：不同的插件使用了相同的 skillName
			return &SkillConflictError{
				SkillName: skillName,
				PluginID:  pluginID,
				Message:   fmt.Sprintf("skill '%s' is already used by plugin '%s'", skillName, installedPluginID),
			}
		}
	}

	// 目录存在但状态文件中没有记录，可能是手动安装的，认为有冲突
	return &SkillConflictError{
		SkillName: skillName,
		PluginID:  pluginID,
		Message:   fmt.Sprintf("skill '%s' already exists (possibly manually installed)", skillName),
	}
}

// InstallSkill 安装 Skill
// workspacePath: 工作区路径（由前端传递），用于更新该工作区的 AGENTS.md
func (s *SkillInstaller) InstallSkill(pluginID string, skill *domainMarketplace.SkillComponent, workspacePath string) error {
	// 检查冲突
	if err := s.CheckSkillConflict(skill.SkillName, pluginID); err != nil {
		if conflictErr, ok := err.(*SkillConflictError); ok {
			return conflictErr
		}
		return err
	}

	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// 构建目标目录路径
	targetDir := filepath.Join(homeDir, ".claude", "skills", skill.SkillName)

	// 读取 Skill 文件
	files, err := s.pluginLoader.ReadSkillFiles(pluginID)
	if err != nil {
		return fmt.Errorf("failed to read skill files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no skill files found for plugin %s", pluginID)
	}

	// 创建目标目录
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// 复制所有文件
	var skillMDContent []byte
	for relPath, content := range files {
		// relPath 使用正斜杠，需要转换为系统路径分隔符
		// 构建目标文件路径
		targetPath := filepath.Join(targetDir, filepath.FromSlash(relPath))

		// 确保目标文件的目录存在
		targetFileDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetFileDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for file %s: %w", relPath, err)
		}

		// 写入文件
		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", relPath, err)
		}

		// 保存 SKILL.md 内容用于解析 frontmatter
		if relPath == "SKILL.md" {
			skillMDContent = content
		}
	}

	// 更新工作区的 AGENTS.md（如果提供了工作区路径）
	if len(skillMDContent) > 0 && workspacePath != "" {
		metadata, err := s.agentsUpdater.ParseSkillFrontmatter(skillMDContent)
		if err == nil {
			// 尝试查找并更新 AGENTS.md（如果找不到，静默失败，不影响安装）
			agentsPath, err := s.agentsUpdater.FindAgentsMDFile(workspacePath)
			if err == nil {
				if err := s.agentsUpdater.AddSkillToAgentsMD(agentsPath, metadata); err != nil {
					// 记录错误但不影响安装流程
					s.logger.Warn("Failed to add skill to AGENTS.md",
						"skill_name", skill.SkillName,
						"workspace_path", workspacePath,
						"error", err,
					)
				}
			}
		}
	}

	return nil
}

// UninstallSkill 卸载 Skill
// workspacePath: 工作区路径（由前端传递），用于更新该工作区的 AGENTS.md
func (s *SkillInstaller) UninstallSkill(skillName string, workspacePath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	skillDir := filepath.Join(homeDir, ".claude", "skills", skillName)

	// 检查目录是否存在
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		// 目录不存在，认为已经卸载
		return nil
	}

	// 删除整个目录
	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to remove skill directory: %w", err)
	}

	// 从工作区的 AGENTS.md 中移除技能条目（如果提供了工作区路径）
	if workspacePath != "" {
		agentsPath, err := s.agentsUpdater.FindAgentsMDFile(workspacePath)
		if err == nil {
			if err := s.agentsUpdater.RemoveSkillFromAgentsMD(agentsPath, skillName); err != nil {
				// 记录错误但不影响卸载流程
				s.logger.Warn("Failed to remove skill from AGENTS.md",
					"skill_name", skillName,
					"workspace_path", workspacePath,
					"error", err,
				)
			}
		}
	}

	return nil
}

// SyncSkillToAgentsMD 同步技能到工作区的 AGENTS.md
// 用于在工作区激活时，确保已安装插件的技能都在 AGENTS.md 中
func (s *SkillInstaller) SyncSkillToAgentsMD(pluginID string, skillName string, workspacePath string, skillMDContent []byte) error {
	if workspacePath == "" {
		return fmt.Errorf("workspace path is required")
	}

	if len(skillMDContent) == 0 {
		return fmt.Errorf("skill content is required")
	}

	// 解析 frontmatter 获取技能元数据
	metadata, err := s.agentsUpdater.ParseSkillFrontmatter(skillMDContent)
	if err != nil {
		return fmt.Errorf("failed to parse skill frontmatter: %w", err)
	}

	// 查找或创建 AGENTS.md
	agentsPath, err := s.agentsUpdater.FindAgentsMDFile(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to find or create AGENTS.md: %w", err)
	}

	// 添加技能到 AGENTS.md（如果已存在会跳过）
	if err := s.agentsUpdater.AddSkillToAgentsMD(agentsPath, metadata); err != nil {
		return fmt.Errorf("failed to add skill to AGENTS.md: %w", err)
	}

	return nil
}
