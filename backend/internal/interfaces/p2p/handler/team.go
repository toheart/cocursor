package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appTeam "github.com/cocursor/backend/internal/application/team"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
	infraP2P "github.com/cocursor/backend/internal/infrastructure/p2p"
)

// P2PTeamHandler P2P 团队处理器
// 提供给其他成员调用的 P2P 接口（Leader 暴露）
type P2PTeamHandler struct {
	teamService *appTeam.TeamService
	wsServer    *infraP2P.WebSocketServer
}

// NewP2PTeamHandler 创建 P2P 团队处理器
func NewP2PTeamHandler(teamService *appTeam.TeamService, wsServer *infraP2P.WebSocketServer) *P2PTeamHandler {
	return &P2PTeamHandler{
		teamService: teamService,
		wsServer:    wsServer,
	}
}

// SetWebSocketServer 设置 WebSocket 服务端
func (h *P2PTeamHandler) SetWebSocketServer(server *infraP2P.WebSocketServer) {
	h.wsServer = server
}

// GetTeamInfo 获取团队信息
// 路由: GET /team/info 或 GET /team/:id/info
func (h *P2PTeamHandler) GetTeamInfo(c *gin.Context) {
	teamID := c.Param("id")

	// 如果没有指定团队 ID，获取 Leader 团队
	if teamID == "" {
		leaderTeam := h.getLeaderTeam()
		if leaderTeam == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not a team leader"})
			return
		}
		teamID = leaderTeam.ID
	}

	team, err := h.teamService.GetTeam(teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}

	members, _ := h.teamService.GetTeamMembers(teamID)

	c.JSON(http.StatusOK, gin.H{
		"team":    team,
		"members": members,
	})
}

// getLeaderTeam 获取 Leader 团队
func (h *P2PTeamHandler) getLeaderTeam() *domainTeam.Team {
	teams := h.teamService.GetTeamList()
	for _, team := range teams {
		if team.IsLeader {
			return team
		}
	}
	return nil
}

// HandleJoin 处理加入请求
// 路由: POST /team/:id/join
func (h *P2PTeamHandler) HandleJoin(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id required"})
		return
	}

	var req domainTeam.JoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	resp, err := h.teamService.HandleJoinRequest(teamID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// HandleLeave 处理离开请求
// 路由: POST /team/:id/leave
func (h *P2PTeamHandler) HandleLeave(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id required"})
		return
	}

	var req struct {
		MemberID string `json:"member_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.teamService.HandleLeaveRequest(teamID, req.MemberID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetMembers 获取成员列表
// 路由: GET /team/:id/members
func (h *P2PTeamHandler) GetMembers(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id required"})
		return
	}

	members, err := h.teamService.GetTeamMembers(teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}

	c.JSON(http.StatusOK, members)
}

// GetSkills 获取技能目录
// 路由: GET /team/:id/skills
func (h *P2PTeamHandler) GetSkills(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id required"})
		return
	}

	index, err := h.teamService.GetSkillIndex(teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "skill index not found"})
		return
	}

	c.JSON(http.StatusOK, index)
}

// PublishSkill 接收技能发布请求
// 路由: POST /team/:id/skills
func (h *P2PTeamHandler) PublishSkill(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id required"})
		return
	}

	// 验证是否是 Leader
	leaderTeam := h.getLeaderTeam()
	if leaderTeam == nil || leaderTeam.ID != teamID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not the leader of this team"})
		return
	}

	var entry domainTeam.TeamSkillEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill entry"})
		return
	}

	// 这里应该调用 TeamService 添加技能到目录
	// 暂时返回成功
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"entry":   entry,
	})
}

// DeleteSkill 删除技能
// 路由: DELETE /team/:id/skills/:skillId
func (h *P2PTeamHandler) DeleteSkill(c *gin.Context) {
	teamID := c.Param("id")
	skillID := c.Param("skillId")

	if teamID == "" || skillID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id and skill id required"})
		return
	}

	// 验证是否是 Leader
	leaderTeam := h.getLeaderTeam()
	if leaderTeam == nil || leaderTeam.ID != teamID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not the leader of this team"})
		return
	}

	// 这里应该调用 TeamService 从目录删除技能
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// HandleWebSocket WebSocket 端点
// 路由: GET /team/:id/ws
func (h *P2PTeamHandler) HandleWebSocket(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team id required"})
		return
	}

	if h.wsServer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "websocket server not available"})
		return
	}

	h.wsServer.HandleConnection(c.Writer, c.Request, teamID)
}

// RegisterRoutes 注册路由
func (h *P2PTeamHandler) RegisterRoutes(router *gin.Engine) {
	// 团队信息（不需要团队 ID）
	router.GET("/team/info", h.GetTeamInfo)

	// 团队相关路由
	team := router.Group("/team/:id")
	{
		team.GET("/info", h.GetTeamInfo)
		team.POST("/join", h.HandleJoin)
		team.POST("/leave", h.HandleLeave)
		team.GET("/members", h.GetMembers)
		team.GET("/skills", h.GetSkills)
		team.POST("/skills", h.PublishSkill)
		team.DELETE("/skills/:skillId", h.DeleteSkill)
		team.GET("/ws", h.HandleWebSocket)
	}
}
