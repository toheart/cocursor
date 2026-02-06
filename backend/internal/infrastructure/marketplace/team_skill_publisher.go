package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/infrastructure/config"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/cocursor/backend/internal/infrastructure/p2p"
)

// TeamSkillPublisher 团队技能发布器
type TeamSkillPublisher struct {
	mu           sync.RWMutex
	validator    *TeamSkillValidator
	fileTransfer *p2p.FileTransfer
	httpClient   *http.Client
	logger       *slog.Logger

	// 已发布技能的本地路径映射
	// pluginID -> local directory path
	publishedSkills map[string]string
}

// NewTeamSkillPublisher 创建发布器
func NewTeamSkillPublisher() *TeamSkillPublisher {
	p := &TeamSkillPublisher{
		validator:       NewTeamSkillValidator(),
		fileTransfer:    p2p.NewFileTransfer(),
		publishedSkills: make(map[string]string),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: log.NewModuleLogger("marketplace", "team_skill_publisher"),
	}
	
	// 服务启动时恢复已发布技能
	if err := p.RestorePublishedSkills(); err != nil {
		p.logger.Warn("failed to restore published skills", "error", err)
	}
	
	return p
}

// PublishRequest 发布请求（向后兼容）
type PublishRequest struct {
	TeamID     string `json:"team_id"`
	PluginID   string `json:"plugin_id"`
	LocalPath  string `json:"local_path"`  // 本地技能目录
	AuthorID   string `json:"author_id"`
	AuthorName string `json:"author_name"`
	Endpoint   string `json:"endpoint"` // 本机端点（用于 P2P 下载）
}

// PublishWithMetadataRequest 带元数据的发布请求
type PublishWithMetadataRequest struct {
	TeamID    string                    `json:"team_id"`
	LocalPath string                    `json:"local_path"` // 本地技能目录
	Metadata  *domainTeam.SkillMetadata `json:"metadata"`   // 用户填写的元数据
	AuthorID  string                    `json:"author_id"`
	Endpoint  string                    `json:"endpoint"` // 本机端点（用于 P2P 下载）
}

// PublishResponse 发布响应
type PublishResponse struct {
	Success bool                       `json:"success"`
	Error   string                     `json:"error,omitempty"`
	Entry   *domainTeam.TeamSkillEntry `json:"entry,omitempty"`
}

// PublishedSkillMeta 已发布技能的元数据（存储在 metadata.json）
type PublishedSkillMeta struct {
	PluginID    string    `json:"plugin_id"`
	Name        string    `json:"name"`
	NameZhCN    string    `json:"name_zh_cn,omitempty"`
	Description string    `json:"description"`
	DescZhCN    string    `json:"description_zh_cn,omitempty"`
	Version     string    `json:"version"`
	Category    string    `json:"category"`
	Author      string    `json:"author"`
	SourcePath  string    `json:"source_path"`  // 原始目录路径
	PublishedAt time.Time `json:"published_at"` // 发布时间
}

// ValidateAndPreview 验证并预览技能
func (p *TeamSkillPublisher) ValidateAndPreview(localPath string) (*SkillValidationResult, error) {
	return p.validator.ValidateDirectory(localPath)
}

// PublishToLeader 发布技能到 Leader
func (p *TeamSkillPublisher) PublishToLeader(ctx context.Context, req *PublishRequest, leaderEndpoint string) (*PublishResponse, error) {
	// 验证本地目录
	validationResult, err := p.validator.ValidateDirectory(req.LocalPath)
	if err != nil {
		return nil, err
	}

	if !validationResult.Valid {
		return &PublishResponse{
			Success: false,
			Error:   validationResult.Error,
		}, nil
	}

	// 构建技能条目
	entry, err := p.validator.BuildSkillEntry(
		validationResult,
		req.PluginID,
		req.AuthorID,
		req.AuthorName,
		req.Endpoint,
	)
	if err != nil {
		return &PublishResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	entry.PublishedAt = time.Now()

	// 发送发布请求到 Leader
	publishURL := fmt.Sprintf("http://%s/team/%s/skills", leaderEndpoint, req.TeamID)
	
	bodyData, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", publishURL, bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return &PublishResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to connect to leader: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return &PublishResponse{
			Success: false,
			Error:   fmt.Sprintf("leader rejected: %s", string(body)),
		}, nil
	}

	// 保存本地路径映射（用于 P2P 下载）
	p.mu.Lock()
	p.publishedSkills[req.PluginID] = req.LocalPath
	p.mu.Unlock()

	p.logger.Info("skill published to leader",
		"team_id", req.TeamID,
		"plugin_id", req.PluginID,
		"name", entry.Name,
	)

	return &PublishResponse{
		Success: true,
		Entry:   entry,
	}, nil
}

// PublishLocal 发布到本地（Leader 自己发布）
func (p *TeamSkillPublisher) PublishLocal(req *PublishRequest) (*domainTeam.TeamSkillEntry, error) {
	// 验证本地目录
	validationResult, err := p.validator.ValidateDirectory(req.LocalPath)
	if err != nil {
		return nil, err
	}

	if !validationResult.Valid {
		return nil, fmt.Errorf("invalid skill: %s", validationResult.Error)
	}

	// 构建技能条目
	entry, err := p.validator.BuildSkillEntry(
		validationResult,
		req.PluginID,
		req.AuthorID,
		req.AuthorName,
		req.Endpoint,
	)
	if err != nil {
		return nil, err
	}
	entry.PublishedAt = time.Now()

	// 保存本地路径映射
	p.mu.Lock()
	p.publishedSkills[req.PluginID] = req.LocalPath
	p.mu.Unlock()

	p.logger.Info("skill published locally",
		"plugin_id", req.PluginID,
		"name", entry.Name,
	)

	return entry, nil
}

// DeleteFromLeader 从 Leader 删除技能
func (p *TeamSkillPublisher) DeleteFromLeader(ctx context.Context, teamID, pluginID, leaderEndpoint string) error {
	deleteURL := fmt.Sprintf("http://%s/team/%s/skills/%s", leaderEndpoint, teamID, pluginID)
	
	req, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
	if err != nil {
		return err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to leader: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("leader rejected: %s", string(body))
	}

	// 移除本地路径映射
	p.mu.Lock()
	delete(p.publishedSkills, pluginID)
	p.mu.Unlock()

	p.logger.Info("skill deleted from leader",
		"team_id", teamID,
		"plugin_id", pluginID,
	)

	return nil
}

// DeleteLocal 从本地删除（Leader 删除）
func (p *TeamSkillPublisher) DeleteLocal(pluginID string) {
	p.mu.Lock()
	delete(p.publishedSkills, pluginID)
	p.mu.Unlock()
}

// GetSkillPath 获取已发布技能的本地路径
func (p *TeamSkillPublisher) GetSkillPath(pluginID string) (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	path, exists := p.publishedSkills[pluginID]
	return path, exists
}

// GetSkillMeta 获取技能元数据（用于 P2P 下载）
func (p *TeamSkillPublisher) GetSkillMeta(pluginID string) (*p2p.SkillMeta, error) {
	p.mu.RLock()
	localPath, exists := p.publishedSkills[pluginID]
	p.mu.RUnlock()

	if !exists {
		return nil, domainTeam.ErrSkillNotFound
	}

	validationResult, err := p.validator.ValidateDirectory(localPath)
	if err != nil {
		return nil, err
	}

	if !validationResult.Valid {
		return nil, fmt.Errorf("invalid skill: %s", validationResult.Error)
	}

	checksum, _ := p.fileTransfer.CalculateDirectoryChecksum(localPath)

	return &p2p.SkillMeta{
		PluginID:    pluginID,
		Name:        validationResult.Name,
		Description: validationResult.Description,
		Version:     validationResult.Version,
		Files:       validationResult.Files,
		TotalSize:   validationResult.TotalSize,
		Checksum:    checksum,
	}, nil
}

// GetSkillArchive 获取技能文件打包（用于 P2P 下载）
func (p *TeamSkillPublisher) GetSkillArchive(pluginID string) ([]byte, error) {
	p.mu.RLock()
	localPath, exists := p.publishedSkills[pluginID]
	p.mu.RUnlock()

	if !exists {
		return nil, domainTeam.ErrSkillNotFound
	}

	// 检查目录是否存在
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return nil, domainTeam.ErrSkillNotFound
	}

	// 打包目录
	data, _, err := p.fileTransfer.PackDirectory(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to pack skill: %w", err)
	}

	return data, nil
}

// ListPublishedSkills 列出已发布的技能
func (p *TeamSkillPublisher) ListPublishedSkills() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	skills := make([]string, 0, len(p.publishedSkills))
	for pluginID := range p.publishedSkills {
		skills = append(skills, pluginID)
	}
	return skills
}

// RegisterPublishedSkill 注册已发布技能（服务启动时恢复）
func (p *TeamSkillPublisher) RegisterPublishedSkill(pluginID, localPath string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 验证路径是否有效
	if _, err := os.Stat(localPath); err == nil {
		p.publishedSkills[pluginID] = localPath
	}
}

// GetTeamSkillStoragePath 获取团队技能下载存储路径
func GetTeamSkillStoragePath(teamID, pluginID string) (string, error) {
	return filepath.Join(config.GetDataDir(), "team-skills", teamID, pluginID), nil
}

// GetPublishedStoragePath 获取已发布技能的缓存存储路径
func GetPublishedStoragePath(teamID, pluginID string) (string, error) {
	return filepath.Join(config.GetDataDir(), "team-skills-published", teamID, pluginID), nil
}

// PublishWithMetadata 带元数据的发布（复制到缓存目录）
func (p *TeamSkillPublisher) PublishWithMetadata(ctx context.Context, req *PublishWithMetadataRequest, leaderEndpoint string) (*PublishResponse, error) {
	// 验证元数据
	if err := req.Metadata.Validate(); err != nil {
		return &PublishResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid metadata: %v", err),
		}, nil
	}

	// 验证本地目录
	validationResult, err := p.validator.ValidateDirectory(req.LocalPath)
	if err != nil {
		return nil, err
	}

	if !validationResult.Valid {
		return &PublishResponse{
			Success: false,
			Error:   validationResult.Error,
		}, nil
	}

	// 获取缓存目录路径
	cachePath, err := GetPublishedStoragePath(req.TeamID, req.Metadata.PluginID)
	if err != nil {
		return nil, err
	}

	// 复制目录到缓存
	if err := p.copyDirectoryToCache(req.LocalPath, cachePath); err != nil {
		return &PublishResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to copy to cache: %v", err),
		}, nil
	}

	// 保存 metadata.json
	if err := p.saveMetadataJSON(cachePath, req.Metadata, req.LocalPath); err != nil {
		return &PublishResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to save metadata: %v", err),
		}, nil
	}

	// 从缓存目录构建技能条目
	entry, err := p.validator.BuildSkillEntryFromMetadata(
		req.Metadata,
		cachePath,
		req.AuthorID,
		req.Endpoint,
	)
	if err != nil {
		return &PublishResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	entry.PublishedAt = time.Now()

	// 发送发布请求到 Leader
	publishURL := fmt.Sprintf("http://%s/team/%s/skills", leaderEndpoint, req.TeamID)

	bodyData, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", publishURL, bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return &PublishResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to connect to leader: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return &PublishResponse{
			Success: false,
			Error:   fmt.Sprintf("leader rejected: %s", string(body)),
		}, nil
	}

	// 保存缓存路径映射（用于 P2P 下载）
	p.mu.Lock()
	p.publishedSkills[req.Metadata.PluginID] = cachePath
	p.mu.Unlock()

	p.logger.Info("skill published with metadata",
		"team_id", req.TeamID,
		"plugin_id", req.Metadata.PluginID,
		"name", entry.Name,
		"cache_path", cachePath,
	)

	return &PublishResponse{
		Success: true,
		Entry:   entry,
	}, nil
}

// PublishLocalWithMetadata 本地发布带元数据（Leader 自己发布）
func (p *TeamSkillPublisher) PublishLocalWithMetadata(req *PublishWithMetadataRequest) (*domainTeam.TeamSkillEntry, error) {
	// 验证元数据
	if err := req.Metadata.Validate(); err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	// 验证本地目录
	validationResult, err := p.validator.ValidateDirectory(req.LocalPath)
	if err != nil {
		return nil, err
	}

	if !validationResult.Valid {
		return nil, fmt.Errorf("invalid skill: %s", validationResult.Error)
	}

	// 获取缓存目录路径
	cachePath, err := GetPublishedStoragePath(req.TeamID, req.Metadata.PluginID)
	if err != nil {
		return nil, err
	}

	// 复制目录到缓存
	if err := p.copyDirectoryToCache(req.LocalPath, cachePath); err != nil {
		return nil, fmt.Errorf("failed to copy to cache: %w", err)
	}

	// 保存 metadata.json
	if err := p.saveMetadataJSON(cachePath, req.Metadata, req.LocalPath); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	// 从缓存目录构建技能条目
	entry, err := p.validator.BuildSkillEntryFromMetadata(
		req.Metadata,
		cachePath,
		req.AuthorID,
		req.Endpoint,
	)
	if err != nil {
		return nil, err
	}
	entry.PublishedAt = time.Now()

	// 保存缓存路径映射
	p.mu.Lock()
	p.publishedSkills[req.Metadata.PluginID] = cachePath
	p.mu.Unlock()

	p.logger.Info("skill published locally with metadata",
		"plugin_id", req.Metadata.PluginID,
		"name", entry.Name,
		"cache_path", cachePath,
	)

	return entry, nil
}

// copyDirectoryToCache 复制目录到缓存位置
func (p *TeamSkillPublisher) copyDirectoryToCache(srcPath, dstPath string) error {
	// 清空目标目录（如果存在）
	if err := os.RemoveAll(dstPath); err != nil {
		return fmt.Errorf("failed to clean cache directory: %w", err)
	}

	// 创建目标目录
	if err := os.MkdirAll(dstPath, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// 遍历源目录并复制文件
	return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dstPath, relPath)

		if info.IsDir() {
			// 创建目录
			return os.MkdirAll(targetPath, info.Mode())
		}

		// 复制文件
		return p.copyFile(path, targetPath)
	})
}

// copyFile 复制单个文件
func (p *TeamSkillPublisher) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// saveMetadataJSON 保存 metadata.json
func (p *TeamSkillPublisher) saveMetadataJSON(cachePath string, metadata *domainTeam.SkillMetadata, sourcePath string) error {
	meta := PublishedSkillMeta{
		PluginID:    metadata.PluginID,
		Name:        metadata.Name,
		NameZhCN:    metadata.NameZhCN,
		Description: metadata.Description,
		DescZhCN:    metadata.DescZhCN,
		Version:     metadata.Version,
		Category:    metadata.Category,
		Author:      metadata.Author,
		SourcePath:  sourcePath,
		PublishedAt: time.Now(),
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	metaPath := filepath.Join(cachePath, "metadata.json")
	return os.WriteFile(metaPath, data, 0644)
}

// DeleteLocalWithCache 删除本地发布（包括缓存目录）
func (p *TeamSkillPublisher) DeleteLocalWithCache(teamID, pluginID string) error {
	p.mu.Lock()
	delete(p.publishedSkills, pluginID)
	p.mu.Unlock()

	// 删除缓存目录
	cachePath, err := GetPublishedStoragePath(teamID, pluginID)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(cachePath); err != nil && !os.IsNotExist(err) {
		p.logger.Warn("failed to delete cache directory",
			"path", cachePath,
			"error", err,
		)
	}

	return nil
}

// RestorePublishedSkills 服务重启时恢复已发布技能映射
func (p *TeamSkillPublisher) RestorePublishedSkills() error {
	publishedDir := filepath.Join(config.GetDataDir(), "team-skills-published")

	// 检查目录是否存在
	if _, err := os.Stat(publishedDir); os.IsNotExist(err) {
		return nil // 目录不存在，无需恢复
	}

	// 遍历 team 目录
	teamDirs, err := os.ReadDir(publishedDir)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, teamDir := range teamDirs {
		if !teamDir.IsDir() {
			continue
		}

		teamPath := filepath.Join(publishedDir, teamDir.Name())
		pluginDirs, err := os.ReadDir(teamPath)
		if err != nil {
			continue
		}

		for _, pluginDir := range pluginDirs {
			if !pluginDir.IsDir() {
				continue
			}

			pluginPath := filepath.Join(teamPath, pluginDir.Name())

			// 检查是否有 SKILL.md（有效的技能目录）
			skillMDPath := filepath.Join(pluginPath, "SKILL.md")
			if _, err := os.Stat(skillMDPath); os.IsNotExist(err) {
				continue
			}

			pluginID := pluginDir.Name()
			p.publishedSkills[pluginID] = pluginPath

			p.logger.Info("restored published skill",
				"plugin_id", pluginID,
				"path", pluginPath,
			)
		}
	}

	return nil
}
