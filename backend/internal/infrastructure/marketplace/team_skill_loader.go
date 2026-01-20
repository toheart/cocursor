package marketplace

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/cocursor/backend/internal/domain/marketplace"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/infrastructure/log"
)

// TeamSkillLoader 团队技能加载器
type TeamSkillLoader struct {
	validator  *TeamSkillValidator
	downloader *TeamSkillDownloader
	logger     *slog.Logger
}

// NewTeamSkillLoader 创建加载器
func NewTeamSkillLoader() *TeamSkillLoader {
	return &TeamSkillLoader{
		validator:  NewTeamSkillValidator(),
		downloader: NewTeamSkillDownloader(),
		logger:     log.NewModuleLogger("marketplace", "team_skill_loader"),
	}
}

// LoadTeamSkills 从团队技能目录加载技能
// 返回的 Plugin 对象包含团队技能特有的字段
func (l *TeamSkillLoader) LoadTeamSkills(teamID string, index *domainTeam.TeamSkillIndex, team *domainTeam.Team) []*marketplace.Plugin {
	if index == nil || len(index.Skills) == 0 {
		return nil
	}

	var plugins []*marketplace.Plugin
	for _, entry := range index.Skills {
		plugin := l.convertEntryToPlugin(teamID, team, &entry)
		if plugin != nil {
			plugins = append(plugins, plugin)
		}
	}

	return plugins
}

// convertEntryToPlugin 将技能目录条目转换为 Plugin
func (l *TeamSkillLoader) convertEntryToPlugin(teamID string, team *domainTeam.Team, entry *domainTeam.TeamSkillEntry) *marketplace.Plugin {
	publishedAt := entry.PublishedAt
	
	// 检查是否已下载
	isDownloaded := l.downloader.IsDownloaded(teamID, entry.PluginID)
	
	// 检查是否已安装
	installed, installedVersion := l.checkInstalled(teamID, entry.PluginID)

	plugin := &marketplace.Plugin{
		ID:          entry.PluginID,
		FullID:      teamID + ":" + entry.PluginID,
		Name:        entry.Name,
		Description: entry.Description,
		Version:     entry.Version,
		Author:      entry.AuthorName,
		Category:    "team", // 团队技能分类
		
		// 来源信息
		Source:   marketplace.SourceTeamGlobal,
		TeamID:   teamID,
		TeamName: "",
		
		// 作者信息
		AuthorID:       entry.AuthorID,
		AuthorName:     entry.AuthorName,
		AuthorEndpoint: entry.AuthorEndpoint,
		AuthorOnline:   false, // 需要从成员列表获取
		
		// 发布时间
		PublishedAt: &publishedAt,
		
		// 下载状态
		IsDownloaded: isDownloaded,
		
		// 安装状态
		Installed:        installed,
		InstalledVersion: installedVersion,
		
		// Skill 组件
		Skill: marketplace.SkillComponent{
			SkillName: entry.PluginID,
		},
	}

	// 补充团队名称
	if team != nil {
		plugin.TeamName = team.Name
	}

	// 获取下载时间
	if isDownloaded {
		status := l.downloader.GetStatus(entry.PluginID)
		if status != nil && status.DownloadedAt != nil {
			plugin.DownloadedAt = status.DownloadedAt
		}
	}

	return plugin
}

// checkInstalled 检查技能是否已安装
func (l *TeamSkillLoader) checkInstalled(teamID, pluginID string) (bool, string) {
	// 检查 ~/.claude/skills/{teamID}-{pluginID}/ 是否存在
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, ""
	}

	installDirName := teamID + "-" + pluginID
	skillPath := filepath.Join(homeDir, ".claude", "skills", installDirName)

	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		return false, ""
	}

	// 尝试读取版本信息
	pluginJSONPath := filepath.Join(skillPath, "plugin.json")
	if data, err := os.ReadFile(pluginJSONPath); err == nil {
		var pluginInfo struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(data, &pluginInfo) == nil {
			return true, pluginInfo.Version
		}
	}

	return true, ""
}

// GetDownloadedSkills 获取已下载的技能列表
func (l *TeamSkillLoader) GetDownloadedSkills(teamID string) ([]string, error) {
	return l.downloader.ListDownloaded(teamID)
}

// LoadDownloadedSkillDetail 加载已下载技能的详细信息
func (l *TeamSkillLoader) LoadDownloadedSkillDetail(teamID, pluginID string) (*marketplace.Plugin, error) {
	localPath, err := l.downloader.GetDownloadedPath(teamID, pluginID)
	if err != nil {
		return nil, err
	}

	// 验证技能目录
	result, err := l.validator.ValidateDirectory(localPath)
	if err != nil {
		return nil, err
	}

	if !result.Valid {
		return nil, domainTeam.ErrInvalidSkillDirectory
	}

	// 检查安装状态
	installed, installedVersion := l.checkInstalled(teamID, pluginID)

	now := time.Now()
	return &marketplace.Plugin{
		ID:          pluginID,
		FullID:      teamID + ":" + pluginID,
		Name:        result.Name,
		Description: result.Description,
		Version:     result.Version,
		Category:    "team",
		
		Source:       marketplace.SourceTeamGlobal,
		TeamID:       teamID,
		IsDownloaded: true,
		DownloadedAt: &now,
		
		Installed:        installed,
		InstalledVersion: installedVersion,
		
		Skill: marketplace.SkillComponent{
			SkillName: pluginID,
		},
	}, nil
}

// FilterBySource 按来源筛选插件
func FilterBySource(plugins []*marketplace.Plugin, source marketplace.PluginSource) []*marketplace.Plugin {
	if source == "" {
		return plugins
	}

	var filtered []*marketplace.Plugin
	for _, p := range plugins {
		if p.Source == source {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// FilterByTeam 按团队筛选插件
func FilterByTeam(plugins []*marketplace.Plugin, teamID string) []*marketplace.Plugin {
	if teamID == "" {
		return plugins
	}

	var filtered []*marketplace.Plugin
	for _, p := range plugins {
		if p.TeamID == teamID {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// MergePlugins 合并不同来源的插件列表
// 处理可能的 ID 冲突，使用 FullID 区分
func MergePlugins(sources ...[]*marketplace.Plugin) []*marketplace.Plugin {
	seen := make(map[string]bool)
	var merged []*marketplace.Plugin

	for _, plugins := range sources {
		for _, p := range plugins {
			key := p.GetFullID()
			if !seen[key] {
				seen[key] = true
				merged = append(merged, p)
			}
		}
	}

	return merged
}
