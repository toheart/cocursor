package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	appTeam "github.com/cocursor/backend/internal/application/team"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
	"github.com/cocursor/backend/internal/infrastructure/marketplace"
	"github.com/cocursor/backend/internal/interfaces/http/response"
)

// TeamHandler 团队管理处理器
type TeamHandler struct {
	teamService     *appTeam.TeamService
	skillPublisher  *marketplace.TeamSkillPublisher
	skillDownloader *marketplace.TeamSkillDownloader
}

// NewTeamHandler 创建团队管理处理器
func NewTeamHandler(
	teamService *appTeam.TeamService,
	skillPublisher *marketplace.TeamSkillPublisher,
	skillDownloader *marketplace.TeamSkillDownloader,
) *TeamHandler {
	return &TeamHandler{
		teamService:     teamService,
		skillPublisher:  skillPublisher,
		skillDownloader: skillDownloader,
	}
}

// GetIdentity 获取本机身份
// @Summary 获取本机身份
// @Tags 团队
// @Produce json
// @Success 200 {object} response.Response
// @Router /team/identity [get]
func (h *TeamHandler) GetIdentity(c *gin.Context) {
	identity, err := h.teamService.GetIdentity()
	if err != nil {
		if err == domainTeam.ErrIdentityNotFound {
			response.Success(c, gin.H{
				"exists":   false,
				"identity": nil,
			})
			return
		}
		response.Error(c, http.StatusInternalServerError, 600001, "Failed to get identity: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"exists":   true,
		"identity": identity,
	})
}

// CreateOrUpdateIdentity 创建或更新身份
// @Summary 创建或更新身份
// @Tags 团队
// @Accept json
// @Produce json
// @Param body body CreateIdentityRequest true "身份信息"
// @Success 200 {object} response.Response
// @Router /team/identity [post]
func (h *TeamHandler) CreateOrUpdateIdentity(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 600002, "Invalid request: "+err.Error())
		return
	}

	identity, err := h.teamService.EnsureIdentity(req.Name)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 600003, "Failed to create identity: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"identity": identity,
	})
}

// GetNetworkInterfaces 获取可用网卡列表
// @Summary 获取可用网卡列表
// @Tags 团队
// @Produce json
// @Success 200 {object} response.Response
// @Router /team/network [get]
func (h *TeamHandler) GetNetworkInterfaces(c *gin.Context) {
	interfaces, err := h.teamService.GetNetworkInterfaces()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 600004, "Failed to get network interfaces: "+err.Error())
		return
	}

	config := h.teamService.GetNetworkConfig()

	response.Success(c, gin.H{
		"interfaces": interfaces,
		"config":     config,
	})
}

// CreateTeam 创建团队
// @Summary 创建团队
// @Tags 团队
// @Accept json
// @Produce json
// @Param body body CreateTeamRequest true "团队信息"
// @Success 200 {object} response.Response
// @Router /team/create [post]
func (h *TeamHandler) CreateTeam(c *gin.Context) {
	var req struct {
		Name               string `json:"name" binding:"required"`
		PreferredInterface string `json:"preferred_interface"`
		PreferredIP        string `json:"preferred_ip"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 600005, "Invalid request: "+err.Error())
		return
	}

	team, err := h.teamService.CreateTeam(req.Name, req.PreferredInterface, req.PreferredIP)
	if err != nil {
		if err == domainTeam.ErrTeamAlreadyExists {
			response.Error(c, http.StatusConflict, 600006, "Already created a team as leader")
			return
		}
		if err == domainTeam.ErrIdentityNotFound {
			response.Error(c, http.StatusBadRequest, 600007, "Please set your identity first")
			return
		}
		response.Error(c, http.StatusInternalServerError, 600008, "Failed to create team: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"team": team,
	})
}

// DiscoverTeams 发现局域网团队
// @Summary 发现局域网团队
// @Tags 团队
// @Produce json
// @Param timeout query int false "超时时间（秒），默认 3"
// @Success 200 {object} response.Response
// @Router /team/discover [get]
func (h *TeamHandler) DiscoverTeams(c *gin.Context) {
	timeout := 3 * time.Second
	if timeoutStr := c.Query("timeout"); timeoutStr != "" {
		if t, err := time.ParseDuration(timeoutStr + "s"); err == nil {
			timeout = t
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout+time.Second)
	defer cancel()

	teams, err := h.teamService.DiscoverTeams(ctx, timeout)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 600009, "Failed to discover teams: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"teams": teams,
	})
}

// JoinTeam 加入团队
// @Summary 加入团队
// @Tags 团队
// @Accept json
// @Produce json
// @Param body body JoinTeamRequest true "加入信息"
// @Success 200 {object} response.Response
// @Router /team/join [post]
func (h *TeamHandler) JoinTeam(c *gin.Context) {
	var req struct {
		Endpoint string `json:"endpoint" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 600010, "Invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	team, err := h.teamService.JoinTeam(ctx, req.Endpoint)
	if err != nil {
		if err == domainTeam.ErrIdentityNotFound {
			response.Error(c, http.StatusBadRequest, 600011, "Please set your identity first")
			return
		}
		response.Error(c, http.StatusInternalServerError, 600012, "Failed to join team: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"team": team,
	})
}

// ListTeams 获取已加入团队列表
// @Summary 获取已加入团队列表
// @Tags 团队
// @Produce json
// @Success 200 {object} response.Response
// @Router /team/list [get]
func (h *TeamHandler) ListTeams(c *gin.Context) {
	teams := h.teamService.GetTeamList()

	response.Success(c, gin.H{
		"teams": teams,
		"total": len(teams),
	})
}

// GetTeamMembers 获取团队成员列表
// @Summary 获取团队成员列表
// @Tags 团队
// @Produce json
// @Param id path string true "团队 ID"
// @Success 200 {object} response.Response
// @Router /team/{id}/members [get]
func (h *TeamHandler) GetTeamMembers(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		response.Error(c, http.StatusBadRequest, 600013, "Team ID is required")
		return
	}

	members, err := h.teamService.GetTeamMembers(teamID)
	if err != nil {
		if err == domainTeam.ErrTeamNotFound {
			response.Error(c, http.StatusNotFound, 600014, "Team not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, 600015, "Failed to get members: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"members": members,
		"total":   len(members),
	})
}

// LeaveTeam 离开团队
// @Summary 离开团队
// @Tags 团队
// @Produce json
// @Param id path string true "团队 ID"
// @Success 200 {object} response.Response
// @Router /team/{id}/leave [post]
func (h *TeamHandler) LeaveTeam(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		response.Error(c, http.StatusBadRequest, 600016, "Team ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	err := h.teamService.LeaveTeam(ctx, teamID)
	if err != nil {
		if err == domainTeam.ErrTeamNotFound {
			response.Error(c, http.StatusNotFound, 600017, "Team not found")
			return
		}
		if err == domainTeam.ErrIsTeamLeader {
			response.Error(c, http.StatusBadRequest, 600018, "Leader cannot leave team, use dissolve instead")
			return
		}
		response.Error(c, http.StatusInternalServerError, 600019, "Failed to leave team: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"message": "Left team successfully",
	})
}

// DissolveTeam 解散团队
// @Summary 解散团队
// @Tags 团队
// @Produce json
// @Param id path string true "团队 ID"
// @Success 200 {object} response.Response
// @Router /team/{id}/dissolve [post]
func (h *TeamHandler) DissolveTeam(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		response.Error(c, http.StatusBadRequest, 600020, "Team ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	err := h.teamService.DissolveTeam(ctx, teamID)
	if err != nil {
		if err == domainTeam.ErrTeamNotFound {
			response.Error(c, http.StatusNotFound, 600021, "Team not found")
			return
		}
		if err == domainTeam.ErrNotTeamLeader {
			response.Error(c, http.StatusForbidden, 600022, "Only team leader can dissolve team")
			return
		}
		response.Error(c, http.StatusInternalServerError, 600023, "Failed to dissolve team: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"message": "Team dissolved successfully",
	})
}

// ValidateSkill 验证技能目录
// @Summary 验证技能目录
// @Tags 团队技能
// @Accept json
// @Produce json
// @Param body body ValidateSkillRequest true "技能目录路径"
// @Success 200 {object} response.Response
// @Router /team/skills/validate [post]
func (h *TeamHandler) ValidateSkill(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 600024, "Invalid request: "+err.Error())
		return
	}

	result, err := h.skillPublisher.ValidateAndPreview(req.Path)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 600025, "Failed to validate skill: "+err.Error())
		return
	}

	response.Success(c, result)
}

// PublishSkill 发布技能到团队
// @Summary 发布技能到团队
// @Tags 团队技能
// @Accept json
// @Produce json
// @Param id path string true "团队 ID"
// @Param body body PublishSkillRequest true "发布信息"
// @Success 200 {object} response.Response
// @Router /team/{id}/skills/publish [post]
func (h *TeamHandler) PublishSkill(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		response.Error(c, http.StatusBadRequest, 600026, "Team ID is required")
		return
	}

	var req struct {
		PluginID  string `json:"plugin_id" binding:"required"`
		LocalPath string `json:"local_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 600027, "Invalid request: "+err.Error())
		return
	}

	// 获取身份
	identity, err := h.teamService.GetIdentity()
	if err != nil {
		response.Error(c, http.StatusBadRequest, 600028, "Please set your identity first")
		return
	}

	// 获取团队信息
	team, err := h.teamService.GetTeam(teamID)
	if err != nil {
		response.Error(c, http.StatusNotFound, 600029, "Team not found")
		return
	}

	// 构建端点
	interfaces, _ := h.teamService.GetNetworkInterfaces()
	var endpoint string
	if len(interfaces) > 0 && len(interfaces[0].Addresses) > 0 {
		endpoint = interfaces[0].Addresses[0] + ":19960"
	}

	publishReq := &marketplace.PublishRequest{
		TeamID:     teamID,
		PluginID:   req.PluginID,
		LocalPath:  req.LocalPath,
		AuthorID:   identity.ID,
		AuthorName: identity.Name,
		Endpoint:   endpoint,
	}

	// 如果是 Leader，直接发布到本地
	if team.IsLeader {
		entry, err := h.skillPublisher.PublishLocal(publishReq)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 600030, "Failed to publish skill: "+err.Error())
			return
		}
		response.Success(c, gin.H{
			"entry": entry,
		})
		return
	}

	// 否则发布到 Leader
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result, err := h.skillPublisher.PublishToLeader(ctx, publishReq, team.LeaderEndpoint)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 600031, "Failed to publish skill: "+err.Error())
		return
	}

	if !result.Success {
		response.Error(c, http.StatusBadRequest, 600032, result.Error)
		return
	}

	response.Success(c, gin.H{
		"entry": result.Entry,
	})
}

// DownloadSkill 下载团队技能
// @Summary 下载团队技能
// @Tags 团队技能
// @Accept json
// @Produce json
// @Param body body DownloadSkillRequest true "下载信息"
// @Success 200 {object} response.Response
// @Router /team/skills/{id}/download [post]
func (h *TeamHandler) DownloadSkill(c *gin.Context) {
	pluginID := c.Param("id")
	if pluginID == "" {
		response.Error(c, http.StatusBadRequest, 600033, "Plugin ID is required")
		return
	}

	var req struct {
		TeamID           string `json:"team_id" binding:"required"`
		AuthorEndpoint   string `json:"author_endpoint" binding:"required"`
		ExpectedChecksum string `json:"expected_checksum"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 600034, "Invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	downloadReq := &marketplace.DownloadRequest{
		TeamID:           req.TeamID,
		PluginID:         pluginID,
		AuthorEndpoint:   req.AuthorEndpoint,
		ExpectedChecksum: req.ExpectedChecksum,
	}

	result, err := h.skillDownloader.Download(ctx, downloadReq)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 600035, "Failed to download skill: "+err.Error())
		return
	}

	if !result.Success {
		response.Error(c, http.StatusBadRequest, 600036, result.Error)
		return
	}

	response.Success(c, gin.H{
		"local_path": result.LocalPath,
	})
}

// GetSkillIndex 获取团队技能目录
// @Summary 获取团队技能目录
// @Tags 团队技能
// @Produce json
// @Param id path string true "团队 ID"
// @Success 200 {object} response.Response
// @Router /team/{id}/skills [get]
func (h *TeamHandler) GetSkillIndex(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		response.Error(c, http.StatusBadRequest, 600037, "Team ID is required")
		return
	}

	index, err := h.teamService.GetSkillIndex(teamID)
	if err != nil {
		if err == domainTeam.ErrTeamNotFound {
			response.Error(c, http.StatusNotFound, 600038, "Team not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, 600039, "Failed to get skill index: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"skill_index": index,
	})
}
