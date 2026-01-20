package marketplace

import (
	"bufio"
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

// SkillValidationResult 技能验证结果
type SkillValidationResult struct {
	Valid       bool     `json:"valid"`
	Error       string   `json:"error,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Files       []string `json:"files"`
	TotalSize   int64    `json:"total_size"`
	SkillPath   string   `json:"skill_path"`
}

// ValidateDirectory 验证技能目录
// 规则：
// 1. 目录必须存在
// 2. 必须包含 SKILL.md 文件
// 3. SKILL.md 必须包含有效的 frontmatter（可选但推荐）
func (v *TeamSkillValidator) ValidateDirectory(dirPath string) (*SkillValidationResult, error) {
	result := &SkillValidationResult{
		SkillPath: dirPath,
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

	// 解析 SKILL.md
	frontmatter, err := v.parseSkillMD(skillMDPath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to parse SKILL.md: %v", err)
		return result, nil
	}

	result.Name = frontmatter["name"]
	result.Description = frontmatter["description"]
	result.Version = frontmatter["version"]

	// 如果没有名称，使用目录名
	if result.Name == "" {
		result.Name = filepath.Base(dirPath)
	}

	// 获取文件列表和大小
	files, totalSize, err := v.fileTransfer.GetDirectoryInfo(dirPath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get directory info: %v", err)
		return result, nil
	}

	result.Files = files
	result.TotalSize = totalSize
	result.Valid = true

	return result, nil
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

// BuildSkillEntry 从验证结果构建技能条目
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
