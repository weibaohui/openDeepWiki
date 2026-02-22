package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

// MockAgentService Mock 接口
type MockAgentService struct {
	mock.Mock
}

func (m *MockAgentService) ListAgents(ctx context.Context) ([]*service.AgentInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*service.AgentInfo), args.Error(1)
}

func (m *MockAgentService) GetAgent(ctx context.Context, fileName string) (*service.AgentDTO, error) {
	args := m.Called(ctx, fileName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentDTO), args.Error(1)
}

func (m *MockAgentService) SaveAgent(ctx context.Context, fileName, content, source string, restoreFrom *int) (*service.SaveResultDTO, error) {
	args := m.Called(ctx, fileName, content, source, restoreFrom)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.SaveResultDTO), args.Error(1)
}

func (m *MockAgentService) GetVersions(ctx context.Context, fileName string) ([]*service.Version, error) {
	args := m.Called(ctx, fileName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*service.Version), args.Error(1)
}

func (m *MockAgentService) GetVersionContent(ctx context.Context, fileName string, version int) (*service.VersionContentDTO, error) {
	args := m.Called(ctx, fileName, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.VersionContentDTO), args.Error(1)
}

func (m *MockAgentService) RestoreVersion(ctx context.Context, fileName string, version int) (*service.SaveResultDTO, error) {
	args := m.Called(ctx, fileName, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.SaveResultDTO), args.Error(1)
}

func (m *MockAgentService) DeleteVersion(ctx context.Context, fileName string, version int) error {
	args := m.Called(ctx, fileName, version)
	return args.Error(0)
}

func (m *MockAgentService) DeleteVersions(ctx context.Context, fileName string, versions []int) error {
	args := m.Called(ctx, fileName, versions)
	return args.Error(0)
}

func (m *MockAgentService) RecordFileChange(ctx context.Context, fileName, content string) error {
	args := m.Called(ctx, fileName, content)
	return args.Error(0)
}

func setupAgentRouter(mockService *MockAgentService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 将 mockService 转换为接口类型
	var agentService service.AgentServiceAgentService = mockService
	handler := NewAgentHandler(agentService)
	handler.RegisterRoutes(router.Group("/api"))

	return router
}

func TestAgentHandler_ListAgents(t *testing.T) {
	mockService := new(MockAgentService)
	agents := []*service.AgentInfo{
		{FileName: "test1.yaml", Name: "test1", Description: "Test 1"},
		{FileName: "test2.yaml", Name: "test2", Description: "Test 2"},
	}

	mockService.On("ListAgents", mock.Anything).Return(agents, nil)

	router := setupAgentRouter(mockService)

	req, _ := http.NewRequest("GET", "/api/agents", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	assert.Len(t, data, 2)
}

func TestAgentHandler_GetAgent(t *testing.T) {
	mockService := new(MockAgentService)
	agent := &service.AgentDTO{
		FileName:       "test.yaml",
		Content:        "name: test",
		CurrentVersion: 1,
	}

	mockService.On("GetAgent", mock.Anything, "test.yaml").Return(agent, nil)

	router := setupAgentRouter(mockService)

	req, _ := http.NewRequest("GET", "/api/agents/test.yaml", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test.yaml", response["file_name"])
	assert.Equal(t, "name: test", response["content"])
}

func TestAgentHandler_GetAgent_InvalidFilename(t *testing.T) {
	mockService := new(MockAgentService)

	router := setupAgentRouter(mockService)

	// 测试路径遍历攻击
	req, _ := http.NewRequest("GET", "/api/agents/../test.yaml", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Gin 的路由解析会拒绝包含 .. 的路径，返回 404
	// 这也是有效的安全防护
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAgentHandler_SaveAgent(t *testing.T) {
	mockService := new(MockAgentService)
	result := &service.SaveResultDTO{
		FileName: "test.yaml",
		Version: 2,
		SavedAt: "2026-02-22T10:00:00Z",
	}

	mockService.On("SaveAgent", mock.Anything, "test.yaml", "name: test", "web", (*int)(nil)).Return(result, nil)

	router := setupAgentRouter(mockService)

	requestBody := map[string]string{
		"content": "name: test",
	}
	body, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("PUT", "/api/agents/test.yaml", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test.yaml", response["file_name"])
	assert.Equal(t, float64(2), response["version"])
}

func TestAgentHandler_GetVersions(t *testing.T) {
	mockService := new(MockAgentService)
	versions := []*service.Version{
		{ID: 1, Version: 1, SavedAt: "2026-02-20T10:00:00Z", Source: "web"},
		{ID: 2, Version: 2, SavedAt: "2026-02-21T15:30:00Z", Source: "web"},
	}

	mockService.On("GetVersions", mock.Anything, "test.yaml").Return(versions, nil)

	router := setupAgentRouter(mockService)

	req, _ := http.NewRequest("GET", "/api/agents/test.yaml/versions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test.yaml", response["file_name"])
	data := response["versions"].([]interface{})
	assert.Len(t, data, 2)
}

func TestAgentHandler_RestoreVersion(t *testing.T) {
	mockService := new(MockAgentService)
	v1 := 1
	result := &service.SaveResultDTO{
		FileName:    "test.yaml",
		RestoredFrom: &v1,
		Version:     3,
		SavedAt:     "2026-02-22T10:00:00Z",
	}

	mockService.On("RestoreVersion", mock.Anything, "test.yaml", 1).Return(result, nil)

	router := setupAgentRouter(mockService)

	req, _ := http.NewRequest("POST", "/api/agents/test.yaml/versions/1/restore", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test.yaml", response["file_name"])
	assert.Equal(t, float64(1), response["restored_from"])
}

func TestAgentHandler_GetVersionContent(t *testing.T) {
	mockService := new(MockAgentService)
	versionContent := &service.VersionContentDTO{
		FileName: "test.yaml",
		Version:  1,
		Content:  "name: test",
	}

	mockService.On("GetVersionContent", mock.Anything, "test.yaml", 1).Return(versionContent, nil)

	router := setupAgentRouter(mockService)

	req, _ := http.NewRequest("GET", "/api/agents/test.yaml/versions/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test.yaml", response["file_name"])
	assert.Equal(t, float64(1), response["version"])
	assert.Equal(t, "name: test", response["content"])
}

func TestAgentHandler_DeleteVersion(t *testing.T) {
	mockService := new(MockAgentService)
	mockService.On("DeleteVersion", mock.Anything, "test.yaml", 1).Return(nil)

	router := setupAgentRouter(mockService)

	req, _ := http.NewRequest("DELETE", "/api/agents/test.yaml/versions/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "deleted", response["message"])
}

func TestAgentHandler_DeleteVersions(t *testing.T) {
	mockService := new(MockAgentService)
	mockService.On("DeleteVersions", mock.Anything, "test.yaml", []int{1, 2}).Return(nil)

	router := setupAgentRouter(mockService)

	requestBody := map[string][]int{
		"versions": {1, 2},
	}
	body, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("DELETE", "/api/agents/test.yaml/versions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(2), response["deleted"])
}
