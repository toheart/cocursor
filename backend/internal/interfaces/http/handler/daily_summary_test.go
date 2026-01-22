package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDailySummaryRepositoryForHandler 用于 handler 测试的 mock
type mockDailySummaryRepositoryForHandler struct {
	summaries map[string]*domainCursor.DailySummary
}

func newMockDailySummaryRepositoryForHandler() *mockDailySummaryRepositoryForHandler {
	return &mockDailySummaryRepositoryForHandler{
		summaries: map[string]*domainCursor.DailySummary{
			"2026-01-18": {
				ID:            "test-id-1",
				Date:          "2026-01-18",
				Summary:       "## 测试日报\n\n这是测试内容。",
				Language:      "zh-CN",
				TotalSessions: 5,
			},
			"2026-01-20": {
				ID:            "test-id-2",
				Date:          "2026-01-20",
				Summary:       "## Test Report\n\nThis is test content.",
				Language:      "en",
				TotalSessions: 3,
			},
		},
	}
}

func (m *mockDailySummaryRepositoryForHandler) Save(summary *domainCursor.DailySummary) error {
	m.summaries[summary.Date] = summary
	return nil
}

func (m *mockDailySummaryRepositoryForHandler) FindByDate(date string) (*domainCursor.DailySummary, error) {
	if summary, ok := m.summaries[date]; ok {
		return summary, nil
	}
	return nil, nil
}

func (m *mockDailySummaryRepositoryForHandler) FindDatesByRange(startDate, endDate string) (map[string]bool, error) {
	result := make(map[string]bool)
	for date := range m.summaries {
		if date >= startDate && date <= endDate {
			result[date] = true
		}
	}
	return result, nil
}

func (m *mockDailySummaryRepositoryForHandler) FindByDateRange(startDate, endDate string) ([]*domainCursor.DailySummary, error) {
	var result []*domainCursor.DailySummary
	for date, summary := range m.summaries {
		if date >= startDate && date <= endDate {
			result = append(result, summary)
		}
	}
	return result, nil
}

func TestDailySummaryHandler_GetBatchStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := newMockDailySummaryRepositoryForHandler()
	handler := NewDailySummaryHandler(mockRepo, nil)

	tests := []struct {
		name           string
		startDate      string
		endDate        string
		expectedStatus int
		validateFunc   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "正常查询日期范围",
			startDate:      "2026-01-14",
			endDate:        "2026-01-20",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(0), body["code"])
				data := body["data"].(map[string]interface{})
				statuses := data["statuses"].(map[string]interface{})
				// 应该有 2026-01-18 和 2026-01-20
				assert.True(t, statuses["2026-01-18"].(bool))
				assert.True(t, statuses["2026-01-20"].(bool))
			},
		},
		{
			name:           "缺少 start_date 参数",
			startDate:      "",
			endDate:        "2026-01-20",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				assert.NotEqual(t, float64(0), body["code"])
			},
		},
		{
			name:           "缺少 end_date 参数",
			startDate:      "2026-01-14",
			endDate:        "",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				assert.NotEqual(t, float64(0), body["code"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/api/v1/daily-summary/batch-status", handler.GetBatchStatus)

			url := "/api/v1/daily-summary/batch-status"
			if tt.startDate != "" || tt.endDate != "" {
				url += "?"
				if tt.startDate != "" {
					url += "start_date=" + tt.startDate
				}
				if tt.endDate != "" {
					if tt.startDate != "" {
						url += "&"
					}
					url += "end_date=" + tt.endDate
				}
			}

			req, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var body map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &body)
			require.NoError(t, err)

			if tt.validateFunc != nil {
				tt.validateFunc(t, body)
			}
		})
	}
}

func TestDailySummaryHandler_GetDailySummary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := newMockDailySummaryRepositoryForHandler()
	handler := NewDailySummaryHandler(mockRepo, nil)

	tests := []struct {
		name           string
		date           string
		expectedStatus int
		validateFunc   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "获取存在的日报",
			date:           "2026-01-18",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(0), body["code"])
				data := body["data"].(map[string]interface{})
				assert.Equal(t, "2026-01-18", data["date"])
				assert.Contains(t, data["summary"], "测试日报")
			},
		},
		{
			name:           "获取不存在的日报",
			date:           "2026-01-15",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				assert.NotEqual(t, float64(0), body["code"])
			},
		},
		{
			name:           "缺少 date 参数",
			date:           "",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, body map[string]interface{}) {
				assert.NotEqual(t, float64(0), body["code"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/api/v1/daily-summary", handler.GetDailySummary)

			url := "/api/v1/daily-summary"
			if tt.date != "" {
				url += "?date=" + tt.date
			}

			req, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var body map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &body)
			require.NoError(t, err)

			if tt.validateFunc != nil {
				tt.validateFunc(t, body)
			}
		})
	}
}
