package marketplace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// ConflictType 冲突类型
type ConflictType string

const (
	// ConflictTypeOtherPlugin 被其他插件占用，不允许覆盖
	ConflictTypeOtherPlugin ConflictType = "other_plugin"
	// ConflictTypeManualInstall 手动安装的，可以询问用户是否覆盖
	ConflictTypeManualInstall ConflictType = "manual_install"
	// ConflictTypeUnknown 未知冲突（无法读取状态文件等）
	ConflictTypeUnknown ConflictType = "unknown"
)

// SkillConflictError Skill 名称冲突错误
type SkillConflictError struct {
	SkillName    string       `json:"skill_name"`
	PluginID     string       `json:"plugin_id"`
	Message      string       `json:"message"`
	ConflictType ConflictType `json:"conflict_type"`
}

func (e *SkillConflictError) Error() string {
	return e.Message
}

// CheckSkillConflict 检查 Skill 名称冲突
// skillName: 目录名（用于检查目录是否存在）
// pluginID: 完整 ID（用于状态检查，可以是 marketplace ID 或 teamID:pluginID 格式）
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
			SkillName:    skillName,
			PluginID:     pluginID,
			Message:      fmt.Sprintf("skill '%s' already exists and cannot verify if it belongs to the same plugin", skillName),
			ConflictType: ConflictTypeUnknown,
		}
	}

	// 查找使用相同 skillName 的已安装插件
	for installedPluginID := range state.InstalledPlugins {
		if installedPluginID == pluginID {
			// 是同一插件，允许覆盖（版本更新）
			return nil
		}

		// 从 installedPluginID 中提取 skillName（目录名）
		// 格式可能是：
		// 1. marketplace 插件：直接是 pluginID（如 "simple-skill"）
		// 2. 团队技能：格式为 "teamID:pluginID"（如 "abc123:my-skill"）
		installedSkillName := installedPluginID
		if colonIdx := strings.LastIndex(installedPluginID, ":"); colonIdx != -1 {
			// 团队技能格式，提取 pluginID 部分作为目录名
			installedSkillName = installedPluginID[colonIdx+1:]
		}

		// 检查目录名是否相同
		if installedSkillName == skillName {
			// 发现冲突：不同的插件使用了相同的 skillName（目录名）
			return &SkillConflictError{
				SkillName:    skillName,
				PluginID:     pluginID,
				Message:      fmt.Sprintf("skill '%s' is already installed by '%s'", skillName, installedPluginID),
				ConflictType: ConflictTypeOtherPlugin,
			}
		}

		// 对于 marketplace 插件，还需要检查其 skill.SkillName 配置
		// 因为 marketplace 插件的 pluginID 和 skillName 可能不同
		if colonIdx := strings.LastIndex(installedPluginID, ":"); colonIdx == -1 {
			// 这是一个 marketplace 插件
			installedPluginData, err := s.pluginLoader.LoadPlugin(installedPluginID)
			if err == nil && installedPluginData.Skill.SkillName == skillName {
				return &SkillConflictError{
					SkillName:    skillName,
					PluginID:     pluginID,
					Message:      fmt.Sprintf("skill '%s' is already used by marketplace plugin '%s'", skillName, installedPluginID),
					ConflictType: ConflictTypeOtherPlugin,
				}
			}
		}
	}

	// 目录存在但状态文件中没有记录，可能是手动安装的
	return &SkillConflictError{
		SkillName:    skillName,
		PluginID:     pluginID,
		Message:      fmt.Sprintf("skill '%s' already exists (possibly manually installed)", skillName),
		ConflictType: ConflictTypeManualInstall,
	}
}

// InstallSkill 安装 Skill
// workspacePath: 工作区路径（由前端传递），用于更新该工作区的 AGENTS.md
// force: 强制覆盖（当检测到手动安装的同名 skill 时）
func (s *SkillInstaller) InstallSkill(pluginID string, skill *domainMarketplace.SkillComponent, workspacePath string, force bool) error {
	// 检查冲突
	if err := s.checkConflictWithForce(skill.SkillName, pluginID, force); err != nil {
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
	s.tryUpdateAgentsMD(skillMDContent, workspacePath, skill.SkillName)

	s.logger.Info("Skill installed",
		"plugin_id", pluginID,
		"skill_name", skill.SkillName,
		"target_dir", targetDir,
	)

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
	// 传递 skillName 作为目录名
	if err := s.agentsUpdater.AddSkillToAgentsMD(agentsPath, metadata, skillName); err != nil {
		return fmt.Errorf("failed to add skill to AGENTS.md: %w", err)
	}

	return nil
}

// InstallSkillFromPath 从本地路径安装技能（团队技能使用）
// sourcePath: 技能源目录（下载目录）
// installDirName: 安装目录名（格式：{team_id}-{plugin_id}）
// fullID: 完整 ID（格式：{team_id}:{plugin_id}），用于冲突检测
// workspacePath: 工作区路径，用于更新 AGENTS.md
// force: 强制覆盖
func (s *SkillInstaller) InstallSkillFromPath(sourcePath, installDirName, fullID, workspacePath string, force bool) error {
	// 检查冲突
	if err := s.checkConflictWithForce(installDirName, fullID, force); err != nil {
		return err
	}

	// 读取源目录文件
	files, err := ReadSkillFilesFromPath(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read skill files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no skill files found in %s", sourcePath)
	}

	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// 构建目标目录路径
	targetDir := filepath.Join(homeDir, ".claude", "skills", installDirName)

	// 清空目标目录（如果存在）
	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("failed to clean target directory: %w", err)
	}

	// 创建目标目录
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// 复制所有文件
	var skillMDContent []byte
	for relPath, content := range files {
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
	s.tryUpdateAgentsMD(skillMDContent, workspacePath, installDirName)

	s.logger.Info("Skill installed from path",
		"source", sourcePath,
		"target", targetDir,
		"full_id", fullID,
	)

	return nil
}

// checkConflictWithForce 检查冲突，如果 force=true 则允许覆盖手动安装的 skill
func (s *SkillInstaller) checkConflictWithForce(skillName, pluginID string, force bool) error {
	err := s.CheckSkillConflict(skillName, pluginID)
	if err == nil {
		return nil
	}

	conflictErr, ok := err.(*SkillConflictError)
	if !ok {
		return err
	}

	// 如果是手动安装的冲突，且 force=true，则允许覆盖
	if conflictErr.ConflictType == ConflictTypeManualInstall && force {
		s.logger.Info("Force overwriting manually installed skill",
			"skill_name", skillName,
			"plugin_id", pluginID,
		)
		return nil
	}

	return conflictErr
}

// tryUpdateAgentsMD 尝试更新工作区的 AGENTS.md（失败不影响安装）
func (s *SkillInstaller) tryUpdateAgentsMD(skillMDContent []byte, workspacePath, skillName string) {
	if len(skillMDContent) == 0 || workspacePath == "" {
		return
	}

	metadata, err := s.agentsUpdater.ParseSkillFrontmatter(skillMDContent)
	if err != nil {
		return
	}

	agentsPath, err := s.agentsUpdater.FindAgentsMDFile(workspacePath)
	if err != nil {
		return
	}

	if err := s.agentsUpdater.AddSkillToAgentsMD(agentsPath, metadata, skillName); err != nil {
		s.logger.Warn("Failed to add skill to AGENTS.md",
			"skill_name", skillName,
			"workspace_path", workspacePath,
			"error", err,
		)
	}
}

// ReadSkillFilesFromPath 从本地路径读取技能文件
// 返回文件映射：相对路径 -> 文件内容
func ReadSkillFilesFromPath(dirPath string) (map[string][]byte, error) {
	files := make(map[string][]byte)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 跳过 metadata.json（这是我们的内部文件）
		if info.Name() == "metadata.json" {
			return nil
		}

		// 计算相对路径
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		// 读取文件内容
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// 统一使用正斜杠作为路径分隔符（跨平台兼容）
		relPath = filepath.ToSlash(relPath)
		files[relPath] = content

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk skill directory: %w", err)
	}

	return files, nil
}
