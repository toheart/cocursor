package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cocursor/backend/internal/infrastructure/config"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupSettingsRouter 创建测试路由
func setupSettingsRouter() *gin.Engine {
	router := gin.New()
	handler := NewSettingsHandler()

	settings := router.Group("/api/v1/settings")
	{
		settings.GET("/cursor-paths", handler.GetPathStatus)
		settings.POST("/cursor-paths/validate", handler.ValidatePath)
		settings.POST("/cursor-paths", handler.SetPaths)
	}

	return router
}

// TestSettingsHandler_GetPathStatus 测试获取路径状态
func TestSettingsHandler_GetPathStatus(t *testing.T) {
	// 保存原始配置
	originalConfig := infraCursor.GetGlobalCursorConfig()
	defer infraCursor.SetGlobalCursorConfig(originalConfig)

	// 清除配置以触发错误状态
	infraCursor.SetGlobalCursorConfig(nil)

	router := setupSettingsRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/cursor-paths", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 验证响应结构
	data, ok := response["data"].(map[string]interface{})
	require.True(t, ok, "响应应包含 data 字段")

	status, ok := data["status"].(map[string]interface{})
	require.True(t, ok, "data 应包含 status 字段")

	// status 应该包含必要的字段
	_, hasUserDataDirOK := status["user_data_dir_ok"]
	_, hasProjectsDirOK := status["projects_dir_ok"]
	_, hasIsWSL := status["is_wsl"]

	assert.True(t, hasUserDataDirOK, "status 应包含 user_data_dir_ok")
	assert.True(t, hasProjectsDirOK, "status 应包含 projects_dir_ok")
	assert.True(t, hasIsWSL, "status 应包含 is_wsl")
}

// TestSettingsHandler_GetPathStatus_WithValidConfig 测试有效配置时的路径状态
func TestSettingsHandler_GetPathStatus_WithValidConfig(t *testing.T) {
	// 保存原始配置
	originalConfig := infraCursor.GetGlobalCursorConfig()
	defer infraCursor.SetGlobalCursorConfig(originalConfig)

	// 创建临时目录
	tmpDir := t.TempDir()
	userDataDir := filepath.Join(tmpDir, "Cursor", "User")
	projectsDir := filepath.Join(tmpDir, ".cursor", "projects")

	// 创建必要的目录结构
	require.NoError(t, os.MkdirAll(filepath.Join(userDataDir, "globalStorage"), 0755))
	require.NoError(t, os.MkdirAll(projectsDir, 0755))

	// 设置配置
	infraCursor.SetGlobalCursorConfig(&config.CursorConfig{
		UserDataDir: userDataDir,
		ProjectsDir: projectsDir,
	})

	router := setupSettingsRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/cursor-paths", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})

	// 应该不需要配置
	needsConfig, _ := data["needs_config"].(bool)
	assert.False(t, needsConfig, "有效配置时不需要额外配置")
}

// TestSettingsHandler_ValidatePath 测试路径验证
func TestSettingsHandler_ValidatePath(t *testing.T) {
	router := setupSettingsRouter()

	t.Run("验证存在的目录", func(t *testing.T) {
		tmpDir := t.TempDir()

		reqBody := map[string]string{
			"path_type": "projects_dir",
			"path":      tmpDir,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/cursor-paths/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		valid, _ := data["valid"].(bool)
		assert.True(t, valid)
	})

	t.Run("验证不存在的目录", func(t *testing.T) {
		reqBody := map[string]string{
			"path_type": "projects_dir",
			"path":      "/nonexistent/path/12345",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/cursor-paths/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		valid, _ := data["valid"].(bool)
		assert.False(t, valid)
		assert.NotEmpty(t, data["error"])
	})

	t.Run("验证 user_data_dir 目录结构", func(t *testing.T) {
		// 创建有效的 Cursor User 目录结构
		tmpDir := t.TempDir()
		globalStorageDir := filepath.Join(tmpDir, "globalStorage")
		require.NoError(t, os.MkdirAll(globalStorageDir, 0755))

		reqBody := map[string]string{
			"path_type": "user_data_dir",
			"path":      tmpDir,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/cursor-paths/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		valid, _ := data["valid"].(bool)
		assert.True(t, valid)
	})

	t.Run("验证 user_data_dir 目录结构不正确", func(t *testing.T) {
		// 创建没有 globalStorage 的目录
		tmpDir := t.TempDir()

		reqBody := map[string]string{
			"path_type": "user_data_dir",
			"path":      tmpDir,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/cursor-paths/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		valid, _ := data["valid"].(bool)
		assert.False(t, valid)
		assert.Contains(t, data["error"], "目录结构不正确")
	})

	t.Run("验证文件而非目录", func(t *testing.T) {
		// 创建临时文件
		tmpFile, err := os.CreateTemp("", "test-file-*")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		reqBody := map[string]string{
			"path_type": "projects_dir",
			"path":      tmpFile.Name(),
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/cursor-paths/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		valid, _ := data["valid"].(bool)
		assert.False(t, valid)
		assert.Contains(t, data["error"], "不是目录")
	})

	t.Run("缺少必要参数", func(t *testing.T) {
		reqBody := map[string]string{
			"path_type": "projects_dir",
			// 缺少 path
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/cursor-paths/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestSettingsHandler_SetPaths 测试设置路径
func TestSettingsHandler_SetPaths(t *testing.T) {
	router := setupSettingsRouter()

	t.Run("设置有效路径", func(t *testing.T) {
		userDataDir := t.TempDir()
		projectsDir := t.TempDir()

		reqBody := map[string]string{
			"user_data_dir": userDataDir,
			"projects_dir":  projectsDir,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/cursor-paths", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})

		// 应该返回配置说明
		assert.NotEmpty(t, data["message"])
		assert.NotNil(t, data["instructions"])
		assert.NotNil(t, data["env_vars"])

		envVars := data["env_vars"].(map[string]interface{})
		assert.Equal(t, userDataDir, envVars["CURSOR_USER_DATA_DIR"])
		assert.Equal(t, projectsDir, envVars["CURSOR_PROJECTS_DIR"])
	})

	t.Run("设置无效路径", func(t *testing.T) {
		reqBody := map[string]string{
			"user_data_dir": "/nonexistent/path/12345",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/cursor-paths", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("设置部分路径", func(t *testing.T) {
		projectsDir := t.TempDir()

		reqBody := map[string]string{
			"projects_dir": projectsDir,
			// user_data_dir 为空
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/cursor-paths", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		envVars := data["env_vars"].(map[string]interface{})
		assert.Equal(t, projectsDir, envVars["CURSOR_PROJECTS_DIR"])
	})
}

// TestGetConfigMethods 测试获取配置方法
func TestGetConfigMethods(t *testing.T) {
	t.Run("非 WSL 环境", func(t *testing.T) {
		methods := getConfigMethods(false)
		assert.GreaterOrEqual(t, len(methods), 2, "至少应该有两种配置方法")

		// 验证基本方法存在
		hasEnvMethod := false
		for _, method := range methods {
			if method["method"] == "环境变量" {
				hasEnvMethod = true
				break
			}
		}
		assert.True(t, hasEnvMethod, "应该包含环境变量配置方法")
	})

	t.Run("WSL 环境", func(t *testing.T) {
		methods := getConfigMethods(true)
		assert.GreaterOrEqual(t, len(methods), 3, "WSL 环境应该有更多配置方法")

		// 验证 WSL 特定方法存在
		hasWSLMethod := false
		for _, method := range methods {
			if method["method"] == "WSL 配置" {
				hasWSLMethod = true
				// 验证 WSL 配置包含示例路径
				example, ok := method["example"].(gin.H)
				if ok {
					assert.Contains(t, example["user_data_dir"], "/mnt/c/")
					assert.Contains(t, example["projects_dir"], "/mnt/c/")
				}
				break
			}
		}
		assert.True(t, hasWSLMethod, "WSL 环境应该包含 WSL 配置方法")
	})
}

// TestGenerateConfigInstructions 测试生成配置说明
func TestGenerateConfigInstructions(t *testing.T) {
	instructions := generateConfigInstructions("/path/to/user/data", "/path/to/projects")

	assert.NotEmpty(t, instructions, "应该生成配置说明")

	// 验证包含 bash/zsh 配置
	hasBashConfig := false
	hasSystemdConfig := false
	for _, inst := range instructions {
		shell, _ := inst["shell"].(string)
		if shell == "bash/zsh" {
			hasBashConfig = true
			content, _ := inst["content"].(string)
			assert.Contains(t, content, "CURSOR_USER_DATA_DIR")
			assert.Contains(t, content, "CURSOR_PROJECTS_DIR")
		}
		if shell == "systemd" {
			hasSystemdConfig = true
		}
	}

	assert.True(t, hasBashConfig, "应该包含 bash/zsh 配置")
	assert.True(t, hasSystemdConfig, "应该包含 systemd 配置")
}
