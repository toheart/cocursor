package marketplace

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/infrastructure/p2p"
)

// TeamSkillValidator 团队技能验证器
type TeamSkillValidator struct {
	fileTransfer *p2p.FileTransfer
}

// NewTeamSkillValidator 创建验证器
func NewTeamSkillValidator() *TeamSkillValidator {
	return &TeamSkillValidator{
		fileTransfer: p2p.NewFileTransfer(),
	}
}

// SkillMetadataPrefill 预填充元数据（从 SKILL.md 或 plugin.json 解析）
type SkillMetadataPrefill struct {
	Name        string `json:"name"`
	NameZhCN    string `json:"name_zh_cn,omitempty"`
	Description string `json:"description"`
	DescZhCN    string `json:"description_zh_cn,omitempty"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Category    string `json:"category"`
}

// SkillValidationResult 技能验证结果
type SkillValidationResult struct {
	Valid         bool                 `json:"valid"`
	Error         string               `json:"error,omitempty"`
	SourceType    string               `json:"source_type"`              // "plugin" | "skill"
	Prefill       SkillMetadataPrefill `json:"prefill"`                  // 预填充数据
	MissingFields []string             `json:"missing_fields,omitempty"` // 缺失的必填字段
	Files         []string             `json:"files"`
	TotalSize     int64                `json:"total_size"`
	SkillPath     string               `json:"skill_path"`

	// 向后兼容字段（保留原有逻辑）
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// ValidateDirectory 验证技能目录
// 规则：
// 1. 目录必须存在
// 2. 必须包含 SKILL.md 文件
// 3. 如果存在 plugin.json，优先从中读取完整元数据
// 4. 返回预填充数据和缺失字段列表
func (v *TeamSkillValidator) ValidateDirectory(dirPath string) (*SkillValidationResult, error) {
	result := &SkillValidationResult{
		SkillPath:  dirPath,
		SourceType: "skill", // 默认为 skill 类型
	}

	// 检查目录是否存在
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Error = "directory does not exist"
			return result, nil
		}
		return nil, err
	}

	if !info.IsDir() {
		result.Error = "path is not a directory"
		return result, nil
	}

	// 检查 SKILL.md 是否存在
	skillMDPath := filepath.Join(dirPath, "SKILL.md")
	if _, err := os.Stat(skillMDPath); os.IsNotExist(err) {
		result.Error = "SKILL.md not found in directory"
		return result, nil
	}

	// 尝试从 plugin.json 读取（如果存在）
	pluginJSONPath := filepath.Join(dirPath, "plugin.json")
	if _, err := os.Stat(pluginJSONPath); err == nil {
		// plugin.json 存在，优先读取
		if prefill, err := v.parsePluginJSON(pluginJSONPath); err == nil {
			result.SourceType = "plugin"
			result.Prefill = *prefill
		}
	}

	// 如果没有从 plugin.json 读取，则从 SKILL.md frontmatter 读取
	if result.SourceType == "skill" {
		frontmatter, err := v.parseSkillMD(skillMDPath)
		if err != nil {
			result.Error = fmt.Sprintf("failed to parse SKILL.md: %v", err)
			return result, nil
		}

		result.Prefill = SkillMetadataPrefill{
			Name:        frontmatter["name"],
			Description: frontmatter["description"],
			Version:     frontmatter["version"],
			Author:      frontmatter["author"],
		}
	}

	// 如果没有名称，使用目录名
	if result.Prefill.Name == "" {
		result.Prefill.Name = filepath.Base(dirPath)
	}

	// 如果没有版本，默认 1.0.0
	if result.Prefill.Version == "" {
		result.Prefill.Version = "1.0.0"
	}

	// 检查缺失的必填字段
	result.MissingFields = v.checkMissingFields(&result.Prefill)

	// 获取文件列表和大小
	files, totalSize, err := v.fileTransfer.GetDirectoryInfo(dirPath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get directory info: %v", err)
		return result, nil
	}

	result.Files = files
	result.TotalSize = totalSize

	// 向后兼容：设置原有字段
	result.Name = result.Prefill.Name
	result.Description = result.Prefill.Description
	result.Version = result.Prefill.Version

	result.Valid = true
	return result, nil
}

// parsePluginJSON 解析 plugin.json 文件
func (v *TeamSkillValidator) parsePluginJSON(filePath string) (*SkillMetadataPrefill, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// plugin.json 结构
	var plugin struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		NameI18n    struct {
			ZhCN string `json:"zh-CN"`
			En   string `json:"en"`
		} `json:"name_i18n"`
		Description string `json:"description"`
		DescI18n    struct {
			ZhCN string `json:"zh-CN"`
			En   string `json:"en"`
		} `json:"description_i18n"`
		Author   string `json:"author"`
		Version  string `json:"version"`
		Category string `json:"category"`
	}

	if err := json.Unmarshal(data, &plugin); err != nil {
		return nil, err
	}

	prefill := &SkillMetadataPrefill{
		Name:        plugin.Name,
		Description: plugin.Description,
		Version:     plugin.Version,
		Author:      plugin.Author,
		Category:    plugin.Category,
	}

	// 优先使用 i18n 字段
	if plugin.NameI18n.En != "" {
		prefill.Name = plugin.NameI18n.En
	}
	if plugin.NameI18n.ZhCN != "" {
		prefill.NameZhCN = plugin.NameI18n.ZhCN
	}
	if plugin.DescI18n.En != "" {
		prefill.Description = plugin.DescI18n.En
	}
	if plugin.DescI18n.ZhCN != "" {
		prefill.DescZhCN = plugin.DescI18n.ZhCN
	}

	return prefill, nil
}

// checkMissingFields 检查缺失的必填字段
func (v *TeamSkillValidator) checkMissingFields(prefill *SkillMetadataPrefill) []string {
	var missing []string

	if prefill.Name == "" {
		missing = append(missing, "name")
	}
	if prefill.Description == "" {
		missing = append(missing, "description")
	}
	if prefill.Version == "" {
		missing = append(missing, "version")
	}
	if prefill.Category == "" {
		missing = append(missing, "category")
	}
	if prefill.Author == "" {
		missing = append(missing, "author")
	}

	return missing
}

// parseSkillMD 解析 SKILL.md 的 frontmatter
// 支持 YAML 格式的 frontmatter
func (v *TeamSkillValidator) parseSkillMD(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)

	// 查找 frontmatter 开始
	inFrontmatter := false
	lineNum := 0
	maxLines := 50 // 只扫描前 50 行

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		if lineNum > maxLines {
			break
		}

		// 检查 frontmatter 边界
		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				// frontmatter 结束
				break
			}
		}

		if inFrontmatter {
			// 解析 key: value
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// 移除引号
				value = strings.Trim(value, "\"'")
				result[key] = value
			}
		}
	}

	return result, scanner.Err()
}

// BuildSkillEntry 从验证结果构建技能条目（向后兼容）
func (v *TeamSkillValidator) BuildSkillEntry(
	result *SkillValidationResult,
	pluginID string,
	authorID, authorName, authorEndpoint string,
) (*domainTeam.TeamSkillEntry, error) {
	if !result.Valid {
		return nil, fmt.Errorf("invalid skill: %s", result.Error)
	}

	// 计算校验和
	checksum, err := v.fileTransfer.CalculateDirectoryChecksum(result.SkillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return &domainTeam.TeamSkillEntry{
		PluginID:       pluginID,
		Name:           result.Name,
		Description:    result.Description,
		Version:        result.Version,
		AuthorID:       authorID,
		AuthorName:     authorName,
		AuthorEndpoint: authorEndpoint,
		FileCount:      len(result.Files),
		TotalSize:      result.TotalSize,
		Checksum:       checksum,
	}, nil
}

// BuildSkillEntryFromMetadata 从用户元数据构建技能条目
func (v *TeamSkillValidator) BuildSkillEntryFromMetadata(
	metadata *domainTeam.SkillMetadata,
	skillPath string,
	authorID, authorEndpoint string,
) (*domainTeam.TeamSkillEntry, error) {
	// 验证元数据
	if err := metadata.Validate(); err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	// 获取文件信息
	files, totalSize, err := v.fileTransfer.GetDirectoryInfo(skillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get directory info: %w", err)
	}

	// 计算校验和
	checksum, err := v.fileTransfer.CalculateDirectoryChecksum(skillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return &domainTeam.TeamSkillEntry{
		PluginID:       metadata.PluginID,
		Name:           metadata.Name,
		NameZhCN:       metadata.NameZhCN,
		Description:    metadata.Description,
		DescZhCN:       metadata.DescZhCN,
		Version:        metadata.Version,
		Category:       metadata.Category,
		AuthorID:       authorID,
		AuthorName:     metadata.Author,
		AuthorEndpoint: authorEndpoint,
		FileCount:      len(files),
		TotalSize:      totalSize,
		Checksum:       checksum,
	}, nil
}
