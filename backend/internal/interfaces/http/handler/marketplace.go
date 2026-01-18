package handler

import (
	"net/http"
	"strconv"

	appMarketplace "github.com/cocursor/backend/internal/application/marketplace"
	"github.com/cocursor/backend/internal/interfaces/http/response"
	"github.com/gin-gonic/gin"
)

// MarketplaceHandler 插件市场处理器
type MarketplaceHandler struct {
	pluginService *appMarketplace.PluginService
}

// NewMarketplaceHandler 创建插件市场处理器
func NewMarketplaceHandler(pluginService *appMarketplace.PluginService) *MarketplaceHandler {
	return &MarketplaceHandler{
		pluginService: pluginService,
	}
}

// ListPlugins 获取插件列表
// @Summary 获取插件列表
// @Description 支持分类、搜索、已安装筛选
// @Tags 插件市场
// @Accept json
// @Produce json
// @Param category query string false "分类筛选"
// @Param search query string false "搜索关键词"
// @Param installed query bool false "是否只显示已安装"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 500 {object} response.ErrorResponse
// @Router /marketplace/plugins [get]
func (h *MarketplaceHandler) ListPlugins(c *gin.Context) {
	category := c.Query("category")
	search := c.Query("search")

	var installed *bool
	if installedStr := c.Query("installed"); installedStr != "" {
		installedVal, err := strconv.ParseBool(installedStr)
		if err == nil {
			installed = &installedVal
		}
	}

	plugins, err := h.pluginService.ListPlugins(category, search, installed)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500001, "Failed to list plugins: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"plugins": plugins,
		"total":   len(plugins),
	})
}

// GetPlugin 获取插件详情
// @Summary 获取插件详情
// @Tags 插件市场
// @Accept json
// @Produce json
// @Param id path string true "插件 ID"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /marketplace/plugins/{id} [get]
func (h *MarketplaceHandler) GetPlugin(c *gin.Context) {
	id := c.Param("id")

	plugin, err := h.pluginService.GetPlugin(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, 404001, "Plugin not found: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"plugin": plugin,
	})
}

// GetInstalledPlugins 获取已安装插件列表
// @Summary 获取已安装插件列表
// @Tags 插件市场
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 500 {object} response.ErrorResponse
// @Router /marketplace/installed [get]
func (h *MarketplaceHandler) GetInstalledPlugins(c *gin.Context) {
	plugins, err := h.pluginService.GetInstalledPlugins()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500001, "Failed to get installed plugins: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"plugins": plugins,
		"total":   len(plugins),
	})
}

// InstallPluginRequest 安装插件请求
type InstallPluginRequest struct {
	WorkspacePath string `json:"workspace_path" binding:"required"`
}

// InstallPlugin 安装插件
// @Summary 安装插件
// @Tags 插件市场
// @Accept json
// @Produce json
// @Param id path string true "插件 ID"
// @Param request body InstallPluginRequest true "安装请求"
// @Success 200 {object} response.Response{data=appMarketplace.InstallPluginResult}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /marketplace/plugins/{id}/install [post]
func (h *MarketplaceHandler) InstallPlugin(c *gin.Context) {
	id := c.Param("id")

	var req InstallPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400001, "Invalid parameter: "+err.Error())
		return
	}

	result, err := h.pluginService.InstallPlugin(id, req.WorkspacePath)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500001, "Failed to install plugin: "+err.Error())
		return
	}

	response.Success(c, result)
}

// UninstallPluginRequest 卸载插件请求
type UninstallPluginRequest struct {
	WorkspacePath string `json:"workspace_path" binding:"required"`
}

// UninstallPlugin 卸载插件
// @Summary 卸载插件
// @Tags 插件市场
// @Accept json
// @Produce json
// @Param id path string true "插件 ID"
// @Param request body UninstallPluginRequest true "卸载请求"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /marketplace/plugins/{id}/uninstall [post]
func (h *MarketplaceHandler) UninstallPlugin(c *gin.Context) {
	id := c.Param("id")

	var req UninstallPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400001, "Invalid parameter: "+err.Error())
		return
	}

	if err := h.pluginService.UninstallPlugin(id, req.WorkspacePath); err != nil {
		response.Error(c, http.StatusInternalServerError, 500001, "Failed to uninstall plugin: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"success": true,
		"message": "Uninstall successful",
	})
}

// CheckPluginStatus 检查插件状态
// @Summary 检查插件状态
// @Tags 插件市场
// @Accept json
// @Produce json
// @Param id path string true "插件 ID"
// @Success 200 {object} response.Response{data=appMarketplace.PluginStatus}
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /marketplace/plugins/{id}/status [get]
func (h *MarketplaceHandler) CheckPluginStatus(c *gin.Context) {
	id := c.Param("id")

	status, err := h.pluginService.CheckPluginStatus(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, 404001, "Plugin not found: "+err.Error())
		return
	}

	response.Success(c, status)
}
