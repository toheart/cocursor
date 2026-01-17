package handler

import (
	"net/http"
	"os"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// StatsHandler 统计处理器
type StatsHandler struct {
	entropyService *appCursor.EntropyService
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(entropyService *appCursor.EntropyService) *StatsHandler {
	return &StatsHandler{
		entropyService: entropyService,
	}
}

// CurrentSession 获取当前会话的健康状态
// @Summary 获取当前会话的健康状态
// @Tags 统计
// @Accept json
// @Produce json
// @Param project_path query string false "项目路径，如 D:/code/cocursor（可选，如果不提供则尝试自动检测）"
// @Success 200 {object} response.Response{data=appCursor.HealthInfo}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /stats/current-session [get]
func (h *StatsHandler) CurrentSession(c *gin.Context) {
	// 获取项目路径（可选）
	projectPath := c.Query("project_path")
	if projectPath == "" {
		// 尝试从当前工作目录获取
		cwd, err := os.Getwd()
		if err != nil {
			response.Error(c, http.StatusBadRequest, 700001, "无法获取当前工作目录，请提供 project_path 参数")
			return
		}
		projectPath = cwd
	}

	// 创建路径解析器和数据库读取器
	pathResolver := infraCursor.NewPathResolver()
	dbReader := infraCursor.NewDBReader()

	// 根据项目路径查找工作区 ID
	workspaceID, err := pathResolver.GetWorkspaceIDByPath(projectPath)
	if err != nil {
		response.Error(c, http.StatusNotFound, 700001, "无法找到工作区: "+err.Error())
		return
	}

	// 获取工作区数据库路径
	workspaceDBPath, err := pathResolver.GetWorkspaceDBPath(workspaceID)
	if err != nil {
		response.Error(c, http.StatusNotFound, 700001, "无法找到工作区数据库: "+err.Error())
		return
	}

	// 读取 composer.composerData
	composerDataValue, err := dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
	if err != nil {
		// 如果没有活跃会话，返回健康状态
		healthInfo := appCursor.HealthInfo{
			Entropy: 0,
			Status:  appCursor.HealthStatusHealthy,
			Warning: "",
		}
		response.Success(c, healthInfo)
		return
	}

	// 解析 Composer 数据
	composers, err := domainCursor.ParseComposerData(string(composerDataValue))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 700001, "无法解析 composer 数据: "+err.Error())
		return
	}

	// 获取活跃的会话
	activeComposer := domainCursor.GetActiveComposer(composers)
	if activeComposer == nil {
		// 如果没有活跃会话，返回健康状态
		healthInfo := appCursor.HealthInfo{
			Entropy: 0,
			Status:  appCursor.HealthStatusHealthy,
			Warning: "",
		}
		response.Success(c, healthInfo)
		return
	}

	// 计算熵值
	entropy := h.entropyService.CalculateSessionEntropy(*activeComposer)

	// 获取健康状态
	status, warning := h.entropyService.GetHealthStatus(entropy)

	healthInfo := appCursor.HealthInfo{
		Entropy: entropy,
		Status:  status,
		Warning: warning,
	}

	response.Success(c, healthInfo)
}
