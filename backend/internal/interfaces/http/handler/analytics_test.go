package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appCursor "github.com/cocursor/backend/internal/application/cursor"
	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	"github.com/cocursor/backend/internal/infrastructure/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockWorkspaceSessionRepository 模拟 WorkspaceSessionRepository
type mockWorkspaceSessionRepository struct{}

func (m *mockWorkspaceSessionRepository) Save(session *storage.WorkspaceSession) error {
	return nil
}

func (m *mockWorkspaceSessionRepository) FindByWorkspaceID(workspaceID string) ([]*storage.WorkspaceSession, error) {
	return []*storage.WorkspaceSession{}, nil
}

func (m *mockWorkspaceSessionRepository) FindByWorkspaceIDAndComposerID(workspaceID, composerID string) (*storage.WorkspaceSession, error) {
	return nil, nil
}

func (m *mockWorkspaceSessionRepository) FindByWorkspacesAndDateRange(workspaceIDs []string, startDate, endDate string) ([]*storage.WorkspaceSession, error) {
	return []*storage.WorkspaceSession{}, nil
}

func (m *mockWorkspaceSessionRepository) FindByWorkspaces(workspaceIDs []string, search string, limit, offset int) ([]*storage.WorkspaceSession, int, error) {
	return []*storage.WorkspaceSession{}, 0, nil
}

func (m *mockWorkspaceSessionRepository) GetCachedComposerIDs(workspaceID string) ([]string, error) {
	return []string{}, nil
}

func (m *mockWorkspaceSessionRepository) GetDailyTokenUsage(workspaceIDs []string, startDate, endDate string) ([]*storage.DailyTokenUsage, error) {
	return []*storage.DailyTokenUsage{}, nil
}

func (m *mockWorkspaceSessionRepository) UpdateRuntimeState(workspaceID string, updates []*storage.RuntimeStateUpdate) error {
	return nil
}

func (m *mockWorkspaceSessionRepository) ResetRuntimeState(workspaceID string) error {
	return nil
}

func (m *mockWorkspaceSessionRepository) FindActiveByWorkspaceID(workspaceID string) ([]*storage.WorkspaceSession, error) {
	// 返回模拟的活跃会话数据
	return []*storage.WorkspaceSession{
		{
			ComposerID:          "test-composer-1",
			Name:                "测试会话1",
			IsVisible:           true,
			IsFocused:           true,
			ActiveLevel:         0, // Focused
			ContextUsagePercent: 30.0,
		},
		{
			ComposerID:          "test-composer-2",
			Name:                "测试会话2",
			IsVisible:           true,
			IsFocused:           false,
			ActiveLevel:         1, // Open
			ContextUsagePercent: 50.0,
		},
	}, nil
}

// mockDailySummaryRepository 模拟 DailySummaryRepository
type mockDailySummaryRepository struct{}

func (m *mockDailySummaryRepository) Save(summary *domainCursor.DailySummary) error {
	return nil
}

func (m *mockDailySummaryRepository) FindByDate(date string) (*domainCursor.DailySummary, error) {
	return nil, nil
}

func (m *mockDailySummaryRepository) FindDatesByRange(startDate, endDate string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (m *mockDailySummaryRepository) FindByDateRange(startDate, endDate string) ([]*domainCursor.DailySummary, error) {
	return []*domainCursor.DailySummary{}, nil
}

func TestAnalyticsHandler_WorkAnalysis(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockGlobalDBReader := infraCursor.NewMockGlobalDBReader()
	statsService := appCursor.NewStatsService(mockGlobalDBReader)
	// 创建 mock repositories（测试中可能不需要实际数据）
	sessionRepo := &mockWorkspaceSessionRepository{}
	summaryRepo := &mockDailySummaryRepository{}
	tokenService := appCursor.NewTokenService()
	workAnalysisService := appCursor.NewWorkAnalysisService(statsService, appCursor.NewProjectManager(), sessionRepo, summaryRepo, tokenService)
	sessionService := appCursor.NewSessionService(appCursor.NewProjectManager(), sessionRepo)
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

func TestAnalyticsHandler_ActiveSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockGlobalDBReader := infraCursor.NewMockGlobalDBReader()
	statsService := appCursor.NewStatsService(mockGlobalDBReader)
	sessionRepo := &mockWorkspaceSessionRepository{}
	summaryRepo := &mockDailySummaryRepository{}
	tokenService := appCursor.NewTokenService()
	workAnalysisService := appCursor.NewWorkAnalysisService(statsService, appCursor.NewProjectManager(), sessionRepo, summaryRepo, tokenService)
	sessionService := appCursor.NewSessionService(appCursor.NewProjectManager(), sessionRepo)
	handler := NewAnalyticsHandler(
		appCursor.NewTokenService(),
		workAnalysisService,
		sessionService,
	)

	// 测试当 WorkspaceCacheService 未设置时的错误处理
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateFunc   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "未设置缓存服务时返回错误",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				// 验证返回错误码
				code, ok := body["code"].(float64)
				assert.True(t, ok, "应该有 code 字段")
				assert.Equal(t, float64(700001), code, "应该返回服务未初始化错误")
			},
		},
		{
			name:           "指定工作区ID但缓存服务未设置",
			queryParams:    "?workspace_id=test-workspace",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				// 验证返回错误
				code, ok := body["code"].(float64)
				assert.True(t, ok, "应该有 code 字段")
				assert.NotEqual(t, float64(0), code, "应该返回错误")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/active"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/v1/sessions/active", handler.ActiveSessions)
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
