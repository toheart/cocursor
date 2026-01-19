package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsHandler_AcceptanceRate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockGlobalDBReader := infraCursor.NewMockGlobalDBReader()
	statsService := appCursor.NewStatsService(mockGlobalDBReader)
	handler := NewStatsHandler(statsService, appCursor.NewProjectManager())

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateFunc   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "默认参数（最近7天）",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, 0, int(body["code"].(float64)), "应该返回成功")
				// 数据可能为空（测试环境没有真实数据），这是正常的
			},
		},
		{
			name:           "指定日期范围",
			queryParams:    "?start_date=2026-01-10&end_date=2026-01-17",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, 0, int(body["code"].(float64)), "应该返回成功")
			},
		},
		{
			name:           "无效的日期格式",
			queryParams:    "?start_date=invalid&end_date=2026-01-17",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				assert.NotEqual(t, 0, int(body["code"].(float64)), "应该返回错误")
			},
		},
		{
			name:           "日期范围超过90天",
			queryParams:    "?start_date=2025-01-01&end_date=2026-01-17",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				assert.NotEqual(t, 0, int(body["code"].(float64)), "应该返回错误")
				assert.Contains(t, body["message"].(string), "90", "错误信息应该提到90天限制")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/acceptance-rate"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/v1/stats/acceptance-rate", handler.AcceptanceRate)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "HTTP 状态码应该正确")

			var body map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &body)
			require.NoError(t, err, "响应应该是有效的 JSON")

			if tt.validateFunc != nil {
				tt.validateFunc(t, body)
			}
		})
	}
}

func TestStatsHandler_FileReferences(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockGlobalDBReader := infraCursor.NewMockGlobalDBReader()
	statsService := appCursor.NewStatsService(mockGlobalDBReader)
	handler := NewStatsHandler(statsService, appCursor.NewProjectManager())

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateFunc   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "默认 top_n=10",
			queryParams:    "?project_name=nonexistent",
			expectedStatus: http.StatusNotFound, // 项目不存在时返回 404
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				// 项目不存在时返回 404
				code := int(body["code"].(float64))
				assert.Equal(t, 800001, code, "应该返回资源不存在错误")
			},
		},
		{
			name:           "指定 top_n=5",
			queryParams:    "?project_name=nonexistent&top_n=5",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				code := int(body["code"].(float64))
				assert.Equal(t, 800001, code, "应该返回资源不存在错误")
			},
		},
		{
			name:           "top_n 超过最大值（50）",
			queryParams:    "?project_name=nonexistent&top_n=100",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				// 参数验证在服务层，404 是因为项目不存在
				code := int(body["code"].(float64))
				assert.Equal(t, 800001, code, "应该返回资源不存在错误")
			},
		},
		{
			name:           "top_n 小于最小值（1）",
			queryParams:    "?project_name=nonexistent&top_n=0",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				// 参数验证在服务层，404 是因为项目不存在
				code := int(body["code"].(float64))
				assert.Equal(t, 800001, code, "应该返回资源不存在错误")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/file-references"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/v1/stats/file-references", handler.FileReferences)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "HTTP 状态码应该正确")

			var body map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &body)
			require.NoError(t, err, "响应应该是有效的 JSON")

			if tt.validateFunc != nil {
				tt.validateFunc(t, body)
			}
		})
	}
}

func TestStatsHandler_DailyReport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockGlobalDBReader := infraCursor.NewMockGlobalDBReader()
	statsService := appCursor.NewStatsService(mockGlobalDBReader)
	handler := NewStatsHandler(statsService, appCursor.NewProjectManager())

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateFunc   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "默认参数",
			queryParams:    "?project_name=nonexistent",
			expectedStatus: http.StatusNotFound, // 项目不存在时返回 404
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				code := int(body["code"].(float64))
				assert.Equal(t, 800001, code, "应该返回资源不存在错误")
			},
		},
		{
			name:           "指定日期和 top_n 参数",
			queryParams:    "?project_name=nonexistent&date=2026-01-17&top_n_sessions=3&top_n_files=5",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				code := int(body["code"].(float64))
				assert.Equal(t, 800001, code, "应该返回资源不存在错误")
			},
		},
		{
			name:           "top_n_sessions 超过最大值",
			queryParams:    "?project_name=nonexistent&top_n_sessions=100",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				// 参数验证在服务层，404 是因为项目不存在
				code := int(body["code"].(float64))
				assert.Equal(t, 800001, code, "应该返回资源不存在错误")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/daily-report"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/v1/stats/daily-report", handler.DailyReport)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "HTTP 状态码应该正确")

			var body map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &body)
			require.NoError(t, err, "响应应该是有效的 JSON")

			if tt.validateFunc != nil {
				tt.validateFunc(t, body)
			}
		})
	}
}
