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
	return &TeamSkillPublisher{
		validator:       NewTeamSkillValidator(),
		fileTransfer:    p2p.NewFileTransfer(),
		publishedSkills: make(map[string]string),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: log.NewModuleLogger("marketplace", "team_skill_publisher"),
	}
}

// PublishRequest 发布请求
type PublishRequest struct {
	TeamID      string `json:"team_id"`
	PluginID    string `json:"plugin_id"`
	LocalPath   string `json:"local_path"`   // 本地技能目录
	AuthorID    string `json:"author_id"`
	AuthorName  string `json:"author_name"`
	Endpoint    string `json:"endpoint"`     // 本机端点（用于 P2P 下载）
}

// PublishResponse 发布响应
type PublishResponse struct {
	Success   bool                      `json:"success"`
	Error     string                    `json:"error,omitempty"`
	Entry     *domainTeam.TeamSkillEntry `json:"entry,omitempty"`
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

// GetStoragePath 获取团队技能存储路径
func GetTeamSkillStoragePath(teamID, pluginID string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".cocursor", "team-skills", teamID, pluginID), nil
}
