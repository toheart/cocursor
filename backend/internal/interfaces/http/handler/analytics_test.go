package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyticsHandler_WorkAnalysis(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockGlobalDBReader := infraCursor.NewMockGlobalDBReader()
	statsService := appCursor.NewStatsService(mockGlobalDBReader)
	mockGlobalDBReader2 := infraCursor.NewMockGlobalDBReader()
	dbReader := infraCursor.NewDBReader()
	dataMerger := appCursor.NewDataMerger(dbReader, mockGlobalDBReader2)
	// 创建 mock sessionRepo（测试中可能不需要实际数据）
	var sessionRepo storage.WorkspaceSessionRepository = nil
	workAnalysisService, _ := appCursor.NewWorkAnalysisService(statsService, appCursor.NewProjectManager(), sessionRepo, dataMerger)
	sessionService, _ := appCursor.NewSessionService(appCursor.NewProjectManager())
	handler := NewAnalyticsHandler(
		appCursor.NewTokenService(),
		workAnalysisService,
		sessionService,
	)

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
				// 验证返回数据包含 prompts 和 generations 字段
				if data, ok := body["data"].(map[string]interface{}); ok {
					if overview, ok := data["overview"].(map[string]interface{}); ok {
						// 验证 prompts 和 generations 字段存在
						_, hasPrompts := overview["total_prompts"]
						_, hasGenerations := overview["total_generations"]
						assert.True(t, hasPrompts || hasGenerations,
							"返回数据应该包含 prompts 和 generations 字段")
					}
				}
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/work-analysis"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/v1/stats/work-analysis", handler.WorkAnalysis)
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
