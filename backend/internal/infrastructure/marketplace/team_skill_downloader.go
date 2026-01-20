package marketplace

import (
	"context"
	"fmt"
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

// TeamSkillDownloader 团队技能下载器
type TeamSkillDownloader struct {
	mu           sync.RWMutex
	fileTransfer *p2p.FileTransfer
	httpClient   *http.Client
	logger       *slog.Logger

	// 下载状态缓存
	// pluginID -> DownloadStatus
	downloadStatus map[string]*DownloadStatus
}

// DownloadStatus 下载状态
type DownloadStatus struct {
	PluginID     string     `json:"plugin_id"`
	TeamID       string     `json:"team_id"`
	Status       string     `json:"status"` // downloading, completed, failed
	Progress     float64    `json:"progress"`
	Error        string     `json:"error,omitempty"`
	DownloadedAt *time.Time `json:"downloaded_at,omitempty"`
	LocalPath    string     `json:"local_path,omitempty"`
}

// NewTeamSkillDownloader 创建下载器
func NewTeamSkillDownloader() *TeamSkillDownloader {
	return &TeamSkillDownloader{
		fileTransfer:   p2p.NewFileTransfer(),
		downloadStatus: make(map[string]*DownloadStatus),
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // 下载超时 5 分钟
		},
		logger: log.NewModuleLogger("marketplace", "team_skill_downloader"),
	}
}

// DownloadRequest 下载请求
type DownloadRequest struct {
	TeamID         string `json:"team_id"`
	PluginID       string `json:"plugin_id"`
	AuthorEndpoint string `json:"author_endpoint"`
	ExpectedChecksum string `json:"expected_checksum"`
}

// DownloadResult 下载结果
type DownloadResult struct {
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	LocalPath string `json:"local_path,omitempty"`
}

// Download 下载技能
func (d *TeamSkillDownloader) Download(ctx context.Context, req *DownloadRequest) (*DownloadResult, error) {
	d.logger.Info("starting skill download",
		"team_id", req.TeamID,
		"plugin_id", req.PluginID,
		"author", req.AuthorEndpoint,
	)

	// 更新下载状态
	d.setStatus(req.PluginID, &DownloadStatus{
		PluginID: req.PluginID,
		TeamID:   req.TeamID,
		Status:   "downloading",
		Progress: 0,
	})

	// 获取存储路径
	localPath, err := GetTeamSkillStoragePath(req.TeamID, req.PluginID)
	if err != nil {
		d.setStatusError(req.PluginID, err.Error())
		return &DownloadResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		d.setStatusError(req.PluginID, err.Error())
		return &DownloadResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// 下载文件
	downloadURL := fmt.Sprintf("http://%s/p2p/skills/%s/download", req.AuthorEndpoint, req.PluginID)
	data, err := d.downloadWithContext(ctx, downloadURL)
	if err != nil {
		d.setStatusError(req.PluginID, err.Error())
		return &DownloadResult{
			Success: false,
			Error:   fmt.Sprintf("download failed: %v", err),
		}, nil
	}

	d.setStatusProgress(req.PluginID, 0.5)

	// 验证校验和
	if req.ExpectedChecksum != "" {
		if !d.fileTransfer.VerifyChecksum(data, req.ExpectedChecksum) {
			d.setStatusError(req.PluginID, "checksum mismatch")
			return &DownloadResult{
				Success: false,
				Error:   "checksum mismatch, file may be corrupted",
			}, nil
		}
	}

	d.setStatusProgress(req.PluginID, 0.7)

	// 清空目标目录（如果存在）
	os.RemoveAll(localPath)

	// 解包文件
	if err := d.fileTransfer.UnpackArchive(data, localPath); err != nil {
		d.setStatusError(req.PluginID, err.Error())
		return &DownloadResult{
			Success: false,
			Error:   fmt.Sprintf("unpack failed: %v", err),
		}, nil
	}

	// 更新状态为完成
	now := time.Now()
	d.setStatus(req.PluginID, &DownloadStatus{
		PluginID:     req.PluginID,
		TeamID:       req.TeamID,
		Status:       "completed",
		Progress:     1.0,
		DownloadedAt: &now,
		LocalPath:    localPath,
	})

	d.logger.Info("skill download completed",
		"team_id", req.TeamID,
		"plugin_id", req.PluginID,
		"local_path", localPath,
	)

	return &DownloadResult{
		Success:   true,
		LocalPath: localPath,
	}, nil
}

// downloadWithContext 带上下文的下载
func (d *TeamSkillDownloader) downloadWithContext(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 读取响应体
	var data []byte
	buf := make([]byte, 32*1024) // 32KB 缓冲区
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
	}

	return data, nil
}

// GetStatus 获取下载状态
func (d *TeamSkillDownloader) GetStatus(pluginID string) *DownloadStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if status, exists := d.downloadStatus[pluginID]; exists {
		statusCopy := *status
		return &statusCopy
	}
	return nil
}

// IsDownloaded 检查是否已下载
func (d *TeamSkillDownloader) IsDownloaded(teamID, pluginID string) bool {
	localPath, err := GetTeamSkillStoragePath(teamID, pluginID)
	if err != nil {
		return false
	}

	// 检查目录和 SKILL.md 是否存在
	skillMDPath := filepath.Join(localPath, "SKILL.md")
	_, err = os.Stat(skillMDPath)
	return err == nil
}

// GetDownloadedPath 获取已下载技能的路径
func (d *TeamSkillDownloader) GetDownloadedPath(teamID, pluginID string) (string, error) {
	localPath, err := GetTeamSkillStoragePath(teamID, pluginID)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return "", domainTeam.ErrSkillNotFound
	}

	return localPath, nil
}

// DeleteDownloaded 删除已下载的技能
func (d *TeamSkillDownloader) DeleteDownloaded(teamID, pluginID string) error {
	localPath, err := GetTeamSkillStoragePath(teamID, pluginID)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(localPath); err != nil {
		return fmt.Errorf("failed to delete skill: %w", err)
	}

	// 清除状态
	d.mu.Lock()
	delete(d.downloadStatus, pluginID)
	d.mu.Unlock()

	d.logger.Info("downloaded skill deleted",
		"team_id", teamID,
		"plugin_id", pluginID,
	)

	return nil
}

// ListDownloaded 列出已下载的技能
func (d *TeamSkillDownloader) ListDownloaded(teamID string) ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	teamSkillsDir := filepath.Join(homeDir, ".cocursor", "team-skills", teamID)
	
	if _, err := os.Stat(teamSkillsDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(teamSkillsDir)
	if err != nil {
		return nil, err
	}

	var skills []string
	for _, entry := range entries {
		if entry.IsDir() {
			// 检查是否有 SKILL.md
			skillMDPath := filepath.Join(teamSkillsDir, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillMDPath); err == nil {
				skills = append(skills, entry.Name())
			}
		}
	}

	return skills, nil
}

// setStatus 设置状态
func (d *TeamSkillDownloader) setStatus(pluginID string, status *DownloadStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.downloadStatus[pluginID] = status
}

// setStatusProgress 更新进度
func (d *TeamSkillDownloader) setStatusProgress(pluginID string, progress float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if status, exists := d.downloadStatus[pluginID]; exists {
		status.Progress = progress
	}
}

// setStatusError 设置错误状态
func (d *TeamSkillDownloader) setStatusError(pluginID string, errMsg string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if status, exists := d.downloadStatus[pluginID]; exists {
		status.Status = "failed"
		status.Error = errMsg
	}
}
