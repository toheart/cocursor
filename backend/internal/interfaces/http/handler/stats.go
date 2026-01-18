package handler

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// StatsHandler 统计处理器
type StatsHandler struct {
	statsService   *appCursor.StatsService
	projectManager *appCursor.ProjectManager
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(statsService *appCursor.StatsService, projectManager *appCursor.ProjectManager) *StatsHandler {
	return &StatsHandler{
		statsService:   statsService,
		projectManager: projectManager,
	}
}

// resolveWorkspaceID 解析工作区 ID
// 优先级：project_name > 当前目录自动检测
// 返回：workspaceID 和错误
func (h *StatsHandler) resolveWorkspaceID(c *gin.Context) (string, error) {
	// 获取项目名
	projectName := c.Query("project_name")

	if projectName != "" {
		// 使用项目名查询
		projectInfo := h.projectManager.GetProject(projectName)
		if projectInfo == nil {
			return "", fmt.Errorf("项目不存在: %s（可通过 GET /api/v1/project/list 查看所有项目）", projectName)
		}

		// 使用主工作区或第一个工作区
		var primaryWorkspace *domainCursor.WorkspaceInfo
		for _, ws := range projectInfo.Workspaces {
			if ws.IsPrimary {
				primaryWorkspace = ws
				break
			}
		}
		if primaryWorkspace == nil && len(projectInfo.Workspaces) > 0 {
			primaryWorkspace = projectInfo.Workspaces[0]
		}

		if primaryWorkspace == nil {
			return "", fmt.Errorf("项目没有可用的工作区")
		}

		return primaryWorkspace.WorkspaceID, nil
	}

	// 尝试从当前工作目录自动检测
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("无法获取当前工作目录，请提供 project_name 参数（可通过 GET /api/v1/project/list 查看所有项目）")
	}

	// 尝试通过路径查找项目
	_, workspaceInfo := h.projectManager.FindByPath(cwd)
	if workspaceInfo != nil {
		return workspaceInfo.WorkspaceID, nil
	}

	return "", fmt.Errorf("无法从当前目录自动检测项目，请提供 project_name 参数（可通过 GET /api/v1/project/list 查看所有项目）")
}

// CurrentSession 获取当前会话的健康状态
// @Summary 获取当前会话的健康状态
// @Description 使用 project_name 参数查询项目，如果不提供则尝试从当前工作目录自动检测
// @Tags 统计
// @Accept json
// @Produce json
// @Param project_name query string false "项目名称，如 cocursor（可通过 GET /api/v1/project/list 查看所有项目）"
// @Success 200 {object} response.Response{data=appCursor.HealthInfo}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /stats/current-session [get]
func (h *StatsHandler) CurrentSession(c *gin.Context) {
	projectName := c.Query("project_name")
	log.Printf("[StatsHandler.CurrentSession] 收到请求: project_name=%s", projectName)

	// 解析工作区 ID
	workspaceID, err := h.resolveWorkspaceID(c)
	if err != nil {
		// 根据错误类型返回不同的状态码
		errMsg := err.Error()
		log.Printf("[StatsHandler.CurrentSession] 解析工作区 ID 失败: project_name=%s, error=%v", projectName, err)
		if strings.Contains(errMsg, "项目不存在") || errMsg == "项目没有可用的工作区" {
			response.Error(c, http.StatusNotFound, 800001, errMsg)
		} else {
			response.Error(c, http.StatusBadRequest, 700001, errMsg)
		}
		return
	}

	log.Printf("[StatsHandler.CurrentSession] 解析成功: project_name=%s, workspaceID=%s", projectName, workspaceID)

	// 创建路径解析器和数据库读取器
	pathResolver := infraCursor.NewPathResolver()
	dbReader := infraCursor.NewDBReader()

	// 获取工作区数据库路径
	workspaceDBPath, err := pathResolver.GetWorkspaceDBPath(workspaceID)
	if err != nil {
		log.Printf("[StatsHandler.CurrentSession] 获取工作区数据库路径失败: workspaceID=%s, error=%v", workspaceID, err)
		response.Error(c, http.StatusNotFound, 700001, "无法找到工作区数据库: "+err.Error())
		return
	}

	log.Printf("[StatsHandler.CurrentSession] 工作区数据库路径: workspaceID=%s, path=%s", workspaceID, workspaceDBPath)

	// 读取 composer.composerData
	composerDataValue, err := dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
	if err != nil {
		// 如果没有活跃会话，返回健康状态
		log.Printf("[StatsHandler.CurrentSession] 读取 composer 数据失败（可能没有活跃会话）: workspaceID=%s, error=%v", workspaceID, err)
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
		log.Printf("[StatsHandler.CurrentSession] 解析 composer 数据失败: workspaceID=%s, error=%v", workspaceID, err)
		response.Error(c, http.StatusInternalServerError, 700001, "无法解析 composer 数据: "+err.Error())
		return
	}

	// 获取活跃的会话
	activeComposer := domainCursor.GetActiveComposer(composers)
	if activeComposer == nil {
		// 如果没有活跃会话，返回健康状态
		log.Printf("[StatsHandler.CurrentSession] 没有活跃会话: workspaceID=%s", workspaceID)
		healthInfo := appCursor.HealthInfo{
			Entropy: 0,
			Status:  appCursor.HealthStatusHealthy,
			Warning: "",
		}
		response.Success(c, healthInfo)
		return
	}

	// 计算熵值
	entropy := h.statsService.CalculateSessionEntropy(*activeComposer)

	// 获取健康状态
	status, warning := h.statsService.GetHealthStatus(entropy)

	healthInfo := appCursor.HealthInfo{
		Entropy: entropy,
		Status:  status,
		Warning: warning,
	}

	log.Printf("[StatsHandler.CurrentSession] 成功: workspaceID=%s, entropy=%.2f, status=%s", workspaceID, entropy, status)
	response.Success(c, healthInfo)
}

// AcceptanceRate 获取 AI 代码接受率统计
// @Summary 获取 AI 代码接受率统计
// @Tags 统计
// @Accept json
// @Produce json
// @Param start_date query string false "开始日期，格式 YYYY-MM-DD，默认最近 7 天"
// @Param end_date query string false "结束日期，格式 YYYY-MM-DD，默认今天"
// @Success 200 {object} response.Response{data=[]domainCursor.DailyAcceptanceStats}
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /stats/acceptance-rate [get]
func (h *StatsHandler) AcceptanceRate(c *gin.Context) {
	// 获取日期参数
	endDate := c.DefaultQuery("end_date", "")
	startDate := c.DefaultQuery("start_date", "")

	// 如果没有提供日期，默认最近 7 天
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}
	if startDate == "" {
		// 计算 7 天前
		end, _ := time.Parse("2006-01-02", endDate)
		start := end.AddDate(0, 0, -7)
		startDate = start.Format("2006-01-02")
	}

	// 调用服务获取统计数据
	stats, err := h.statsService.GetAcceptanceRateStats(startDate, endDate)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 700001, "获取接受率统计失败: "+err.Error())
		return
	}

	response.Success(c, stats)
}

// ConversationOverview 获取对话统计概览
// @Summary 获取对话统计概览
// @Description 使用 project_name 参数查询项目，如果不提供则尝试从当前工作目录自动检测
// @Tags 统计
// @Accept json
// @Produce json
// @Param project_name query string false "项目名称，如 cocursor（可通过 GET /api/v1/project/list 查看所有项目）"
// @Success 200 {object} response.Response{data=domainCursor.ConversationOverview}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /stats/conversation-overview [get]
func (h *StatsHandler) ConversationOverview(c *gin.Context) {
	// 解析工作区 ID
	workspaceID, err := h.resolveWorkspaceID(c)
	if err != nil {
		// 根据错误类型返回不同的状态码
		errMsg := err.Error()
		if strings.Contains(errMsg, "项目不存在") || errMsg == "项目没有可用的工作区" {
			response.Error(c, http.StatusNotFound, 800001, errMsg)
		} else {
			response.Error(c, http.StatusBadRequest, 700001, errMsg)
		}
		return
	}

	// 调用服务获取对话概览
	overview, err := h.statsService.GetConversationOverview(workspaceID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 700001, "获取对话概览失败: "+err.Error())
		return
	}

	response.Success(c, overview)
}

// FileReferences 获取文件引用分析
// @Summary 获取文件引用分析
// @Description 使用 project_name 参数查询项目，如果不提供则尝试从当前工作目录自动检测
// @Tags 统计
// @Accept json
// @Produce json
// @Param project_name query string false "项目名称，如 cocursor（可通过 GET /api/v1/project/list 查看所有项目）"
// @Param top_n query int false "返回前 N 个文件，默认 10，最大 50，最小 1"
// @Success 200 {object} response.Response{data=[]domainCursor.FileReference}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /stats/file-references [get]
func (h *StatsHandler) FileReferences(c *gin.Context) {
	// 解析工作区 ID
	workspaceID, err := h.resolveWorkspaceID(c)
	if err != nil {
		// 根据错误类型返回不同的状态码
		errMsg := err.Error()
		if strings.Contains(errMsg, "项目不存在") || errMsg == "项目没有可用的工作区" {
			response.Error(c, http.StatusNotFound, 800001, errMsg)
		} else {
			response.Error(c, http.StatusBadRequest, 700001, errMsg)
		}
		return
	}

	// 获取 top_n 参数
	topN := 10
	if topNStr := c.Query("top_n"); topNStr != "" {
		if n, err := strconv.Atoi(topNStr); err == nil {
			topN = n
		}
	}

	// 调用服务获取文件引用
	refs, err := h.statsService.GetFileReferences(workspaceID, topN)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 700001, "获取文件引用失败: "+err.Error())
		return
	}

	response.Success(c, refs)
}

// DailyReport 生成日报
// @Summary 生成日报
// @Description 使用 project_name 参数查询项目，如果不提供则尝试从当前工作目录自动检测
// @Tags 统计
// @Accept json
// @Produce json
// @Param project_name query string false "项目名称，如 cocursor（可通过 GET /api/v1/project/list 查看所有项目）"
// @Param date query string false "日期，格式 YYYY-MM-DD，默认今天"
// @Param top_n_sessions query int false "Top N 会话数，默认 5，最大 20"
// @Param top_n_files query int false "Top N 文件数，默认 10，最大 50"
// @Success 200 {object} response.Response{data=domainCursor.DailyReport}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /stats/daily-report [get]
func (h *StatsHandler) DailyReport(c *gin.Context) {
	// 解析工作区 ID
	workspaceID, err := h.resolveWorkspaceID(c)
	if err != nil {
		// 根据错误类型返回不同的状态码
		errMsg := err.Error()
		if strings.Contains(errMsg, "项目不存在") || errMsg == "项目没有可用的工作区" {
			response.Error(c, http.StatusNotFound, 800001, errMsg)
		} else {
			response.Error(c, http.StatusBadRequest, 700001, errMsg)
		}
		return
	}

	// 获取日期参数
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	// 获取 top_n 参数
	topNSessions := 5
	if topNSessionsStr := c.Query("top_n_sessions"); topNSessionsStr != "" {
		if n, err := strconv.Atoi(topNSessionsStr); err == nil {
			topNSessions = n
		}
	}
	topNFiles := 10
	if topNFilesStr := c.Query("top_n_files"); topNFilesStr != "" {
		if n, err := strconv.Atoi(topNFilesStr); err == nil {
			topNFiles = n
		}
	}

	// 调用服务生成日报
	report, err := h.statsService.GenerateDailyReport(workspaceID, date, topNSessions, topNFiles)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 700001, "生成日报失败: "+err.Error())
		return
	}

	response.Success(c, report)
}
