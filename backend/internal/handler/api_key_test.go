package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

// MockAPIKeyService Mock服务接口
type MockAPIKeyService struct {
	mock.Mock
}

func (m *MockAPIKeyService) CreateAPIKey(ctx context.Context, req *service.CreateAPIKeyRequest) (*model.APIKey, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.APIKey), args.Error(1)
}

func (m *MockAPIKeyService) UpdateAPIKey(ctx context.Context, id uint, req *service.UpdateAPIKeyRequest) (*model.APIKey, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.APIKey), args.Error(1)
}

func (m *MockAPIKeyService) DeleteAPIKey(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAPIKeyService) GetAPIKey(ctx context.Context, id uint) (*model.APIKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.APIKey), args.Error(1)
}

func (m *MockAPIKeyService) ListAPIKeys(ctx context.Context) ([]*model.APIKey, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.APIKey), args.Error(1)
}

func (m *MockAPIKeyService) UpdateAPIKeyStatus(ctx context.Context, id uint, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockAPIKeyService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockAPIKeyService) RecordRequest(ctx context.Context, apiKeyID uint, success bool) error {
	args := m.Called(ctx, apiKeyID, success)
	return args.Error(0)
}

func (m *MockAPIKeyService) MarkUnavailable(ctx context.Context, apiKeyID uint, resetTime time.Time) error {
	args := m.Called(ctx, apiKeyID, resetTime)
	return args.Error(0)
}

func (m *MockAPIKeyService) GetAPIKeyByName(ctx context.Context, name string) (*model.APIKey, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.APIKey), args.Error(1)
}

func (m *MockAPIKeyService) GetAPIKeysByNames(ctx context.Context, names []string) ([]*model.APIKey, error) {
	args := m.Called(ctx, names)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.APIKey), args.Error(1)
}

// setupRouter 设置测试路由
func setupRouter(service *MockAPIKeyService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	h := &APIKeyHandler{service: service}

	api := router.Group("/api/v1")
	h.RegisterRoutes(api)

	return router
}

// TestAPIKeyHandler_CreateAPIKey 测试创建 API Key
func TestAPIKeyHandler_CreateAPIKey(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAPIKeyService)
		expectedStatus int
		verifyResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "成功创建 API Key",
			requestBody: map[string]interface{}{
				"name":     "test-key",
				"provider": "openai",
				"base_url": "https://api.openai.com/v1",
				"api_key":  "sk-test123456789",
				"model":    "gpt-4",
				"priority": 10,
			},
			mockSetup: func(m *MockAPIKeyService) {
				now := time.Now()
				apiKey := &model.APIKey{
					ID:        1,
					Name:      "test-key",
					Provider:  "openai",
					BaseURL:   "https://api.openai.com/v1",
					APIKey:    "sk-test123456789",
					Model:     "gpt-4",
					Priority:  10,
					Status:    "enabled",
					CreatedAt: now,
					UpdatedAt: now,
				}
				m.On("CreateAPIKey", mock.Anything, mock.AnythingOfType("*service.CreateAPIKeyRequest")).Return(apiKey, nil)
			},
			expectedStatus: http.StatusCreated,
			verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, uint(1), uint(response["id"].(float64)))
				assert.Equal(t, "test-key", response["name"])
				// API Key 应该被脱敏
				assert.Equal(t, "sk-***6789", response["api_key"])
			},
		},
		{
			name: "缺少必填字段",
			requestBody: map[string]interface{}{
				"name": "test-key",
			},
			mockSetup:      func(m *MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, "error")
			},
		},
		{
			name: "名称已存在",
			requestBody: map[string]interface{}{
				"name":     "duplicate",
				"provider": "openai",
				"base_url": "https://api.openai.com/v1",
				"api_key":  "sk-test",
				"model":    "gpt-4",
			},
			mockSetup: func(m *MockAPIKeyService) {
				m.On("CreateAPIKey", mock.Anything, mock.AnythingOfType("*service.CreateAPIKeyRequest")).
					Return(nil, repository.ErrAPIKeyDuplicate)
			},
			expectedStatus: http.StatusInternalServerError,
			verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAPIKeyService)
			tt.mockSetup(mockService)

			router := setupRouter(mockService)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/api-keys", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.verifyResponse(t, w)
			mockService.AssertExpectations(t)
		})
	}
}

// TestAPIKeyHandler_GetAPIKey 测试获取 API Key
func TestAPIKeyHandler_GetAPIKey(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockSetup      func(*MockAPIKeyService)
		expectedStatus int
		verifyResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "成功获取",
			id:   "1",
			mockSetup: func(m *MockAPIKeyService) {
				apiKey := &model.APIKey{
					ID:       1,
					Name:     "test-key",
					Provider: "openai",
					BaseURL:  "https://api.openai.com/v1",
					APIKey:   "sk-test123456789",
					Model:    "gpt-4",
					Status:   "enabled",
				}
				m.On("GetAPIKey", mock.Anything, uint(1)).Return(apiKey, nil)
			},
			expectedStatus: http.StatusOK,
			verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, uint(1), uint(response["id"].(float64)))
				assert.Equal(t, "test-key", response["name"])
				// API Key 应该被脱敏
				assert.Equal(t, "sk-***6789", response["api_key"])
			},
		},
		{
			name:           "无效的 ID",
			id:             "invalid",
			mockSetup:      func(m *MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, "error")
				assert.Contains(t, response["error"], "invalid id")
			},
		},
		{
			name: "ID 为 0",
			id:   "0",
			mockSetup: func(m *MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
			verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, "error")
				assert.Contains(t, response["error"], "invalid id")
			},
		},
		{
			name: "API Key 不存在",
			id:   "999",
			mockSetup: func(m *MockAPIKeyService) {
				m.On("GetAPIKey", mock.Anything, uint(999)).Return(nil, repository.ErrAPIKeyNotFound)
			},
			expectedStatus: http.StatusNotFound,
			verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAPIKeyService)
			tt.mockSetup(mockService)

			router := setupRouter(mockService)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/api-keys/"+tt.id, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.verifyResponse(t, w)
			mockService.AssertExpectations(t)
		})
	}
}

// TestAPIKeyHandler_ListAPIKeys 测试列出所有 API Key
func TestAPIKeyHandler_ListAPIKeys(t *testing.T) {
	tests := []struct {
		name           string
		mockSetup      func(*MockAPIKeyService)
		expectedStatus int
		verifyResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "成功列出",
			mockSetup: func(m *MockAPIKeyService) {
				keys := []*model.APIKey{
					{ID: 1, Name: "key1", Provider: "openai", APIKey: "sk-123", Status: "enabled"},
					{ID: 2, Name: "key2", Provider: "anthropic", APIKey: "sk-456", Status: "enabled"},
				}
				m.On("ListAPIKeys", mock.Anything).Return(keys, nil)
			},
			expectedStatus: http.StatusOK,
			verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, float64(2), response["total"])
				data := response["data"].([]interface{})
				assert.Len(t, data, 2)
				// 所有 API Key 都应该被脱敏
				for _, item := range data {
					key := item.(map[string]interface{})
					assert.Contains(t, key["api_key"], "***")
				}
			},
		},
		{
			name: "返回空列表",
			mockSetup: func(m *MockAPIKeyService) {
				m.On("ListAPIKeys", mock.Anything).Return([]*model.APIKey{}, nil)
			},
			expectedStatus: http.StatusOK,
			verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, float64(0), response["total"])
				assert.Len(t, response["data"].([]interface{}), 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAPIKeyService)
			tt.mockSetup(mockService)

			router := setupRouter(mockService)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/api-keys", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.verifyResponse(t, w)
			mockService.AssertExpectations(t)
		})
	}
}

// TestAPIKeyHandler_UpdateAPIKey 测试更新 API Key
func TestAPIKeyHandler_UpdateAPIKey(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		requestBody    interface{}
		mockSetup      func(*MockAPIKeyService)
		expectedStatus int
		verifyResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "成功更新",
			id:   "1",
			requestBody: map[string]interface{}{
				"name":     "updated-key",
				"provider": "anthropic",
				"priority": 20,
			},
			mockSetup: func(m *MockAPIKeyService) {
				existing := &model.APIKey{
					ID:       1,
					Name:     "old-key",
					Provider: "openai",
					Priority: 10,
				}
				m.On("UpdateAPIKey", mock.Anything, uint(1), mock.AnythingOfType("*service.UpdateAPIKeyRequest")).
					Return(existing, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "无效的 ID",
			id:          "invalid",
			requestBody:  map[string]interface{}{},
			mockSetup:    func(m *MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "JSON 格式错误",
			id:          "1",
			requestBody:  "{invalid json",
			mockSetup:    func(m *MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAPIKeyService)
			tt.mockSetup(mockService)

			router := setupRouter(mockService)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req, _ := http.NewRequest(http.MethodPut, "/api/v1/api-keys/"+tt.id, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.verifyResponse != nil {
				tt.verifyResponse(t, w)
			}
			mockService.AssertExpectations(t)
		})
	}
}

// TestAPIKeyHandler_DeleteAPIKey 测试删除 API Key
func TestAPIKeyHandler_DeleteAPIKey(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockSetup      func(*MockAPIKeyService)
		expectedStatus int
	}{
		{
			name: "成功删除",
			id:   "1",
			mockSetup: func(m *MockAPIKeyService) {
				m.On("DeleteAPIKey", mock.Anything, uint(1)).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "无效的 ID",
			id:          "invalid",
			mockSetup:    func(m *MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "删除失败",
			id:   "1",
			mockSetup: func(m *MockAPIKeyService) {
				m.On("DeleteAPIKey", mock.Anything, uint(1)).Return(errors.New("delete failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAPIKeyService)
			tt.mockSetup(mockService)

			router := setupRouter(mockService)

			req, _ := http.NewRequest(http.MethodDelete, "/api/v1/api-keys/"+tt.id, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// TestAPIKeyHandler_UpdateStatus 测试更新状态
func TestAPIKeyHandler_UpdateStatus(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		requestBody    interface{}
		mockSetup      func(*MockAPIKeyService)
		expectedStatus int
	}{
		{
			name: "成功更新状态",
			id:   "1",
			requestBody: map[string]interface{}{
				"status": "disabled",
			},
			mockSetup: func(m *MockAPIKeyService) {
				m.On("UpdateAPIKeyStatus", mock.Anything, uint(1), "disabled").Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "无效的 ID",
			id:          "invalid",
			requestBody:  map[string]interface{}{"status": "disabled"},
			mockSetup:    func(m *MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "缺少 status 字段",
			id:          "1",
			requestBody:  map[string]interface{}{},
			mockSetup:    func(m *MockAPIKeyService) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAPIKeyService)
			tt.mockSetup(mockService)

			router := setupRouter(mockService)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPatch, "/api/v1/api-keys/"+tt.id+"/status", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// TestAPIKeyHandler_GetStats 测试获取统计信息
func TestAPIKeyHandler_GetStats(t *testing.T) {
	tests := []struct {
		name           string
		mockSetup      func(*MockAPIKeyService)
		expectedStatus int
		verifyResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "成功获取统计",
			mockSetup: func(m *MockAPIKeyService) {
				stats := map[string]interface{}{
					"total_count":    10,
					"enabled_count":  8,
					"disabled_count": 2,
					"total_requests": 1000,
				}
				m.On("GetStats", mock.Anything).Return(stats, nil)
			},
			expectedStatus: http.StatusOK,
			verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, float64(10), response["total_count"])
				assert.Equal(t, float64(8), response["enabled_count"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAPIKeyService)
			tt.mockSetup(mockService)

			router := setupRouter(mockService)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/api-keys/stats", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.verifyResponse(t, w)
			mockService.AssertExpectations(t)
		})
	}
}

// TestAPIKeyHandler_toResponse 测试响应转换（脱敏）
func TestAPIKeyHandler_toResponse(t *testing.T) {
	handler := &APIKeyHandler{}

	apiKey := &model.APIKey{
		ID:               1,
		Name:             "test-key",
		Provider:         "openai",
		BaseURL:          "https://api.openai.com/v1",
		APIKey:           "sk-test123456789",
		Model:            "gpt-4",
		Priority:         10,
		Status:           "enabled",
		RequestCount:     100,
		ErrorCount:       5,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	response := handler.toResponse(apiKey)

	assert.Equal(t, uint(1), response.ID)
	assert.Equal(t, "test-key", response.Name)
	assert.Equal(t, "openai", response.Provider)
	assert.Equal(t, "https://api.openai.com/v1", response.BaseURL)
	assert.Equal(t, "sk-***6789", response.APIKey) // 应该被脱敏
	assert.Equal(t, "gpt-4", response.Model)
	assert.Equal(t, 10, response.Priority)
	assert.Equal(t, "enabled", response.Status)
	assert.Equal(t, 100, response.RequestCount)
	assert.Equal(t, 5, response.ErrorCount)
}
