package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

// MockUserRequestService Mock用户需求服务接口
type MockUserRequestService struct {
	createFunc  func(repoID uint, content string) (*model.UserRequest, error)
	getFunc     func(id uint) (*model.UserRequest, error)
	listFunc    func(repoID uint, page, pageSize int, status string) ([]*model.UserRequest, int64, error)
	deleteFunc  func(id uint) error
	updateFunc  func(id uint, status string) error
}

func (m *MockUserRequestService) CreateRequest(repoID uint, content string) (*model.UserRequest, error) {
	if m.createFunc != nil {
		return m.createFunc(repoID, content)
	}
	return &model.UserRequest{
		ID:           1,
		RepositoryID: repoID,
		Content:      content,
		Status:       model.UserRequestStatusPending,
	}, nil
}

func (m *MockUserRequestService) GetRequest(id uint) (*model.UserRequest, error) {
	if m.getFunc != nil {
		return m.getFunc(id)
	}
	if id == 999 {
		return nil, errors.New("request not found")
	}
	return &model.UserRequest{
		ID:           id,
		RepositoryID: 1,
		Content:      "test content",
		Status:       model.UserRequestStatusPending,
	}, nil
}

func (m *MockUserRequestService) ListRequests(repoID uint, page, pageSize int, status string) ([]*model.UserRequest, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(repoID, page, pageSize, status)
	}
	requests := []*model.UserRequest{
		{ID: 1, RepositoryID: repoID, Content: "request 1", Status: model.UserRequestStatusPending},
		{ID: 2, RepositoryID: repoID, Content: "request 2", Status: model.UserRequestStatusCompleted},
	}
	return requests, int64(len(requests)), nil
}

func (m *MockUserRequestService) DeleteRequest(id uint) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *MockUserRequestService) UpdateStatus(id uint, status string) error {
	if m.updateFunc != nil {
		return m.updateFunc(id, status)
	}
	return nil
}

// MockTaskService Mock任务服务接口（用于用户需求handler，不需要实际实现CreateTask）
type MockTaskServiceForUserRequest struct {
}

// Implement minimal TaskService interface for mock

// Implement minimal TaskService interface for the mock
func (m *MockTaskServiceForUserRequest) Get(id uint) (*model.Task, error) {
	return nil, nil
}
func (m *MockTaskServiceForUserRequest) GetByRepository(repoID uint) ([]*model.Task, error) {
	return nil, nil
}
func (m *MockTaskServiceForUserRequest) GetTaskStats(repoID uint) (map[string]int, error) {
	return nil, nil
}
func (m *MockTaskServiceForUserRequest) Enqueue(id uint) error {
	return nil
}
func (m *MockTaskServiceForUserRequest) Reset(id uint) error {
	return nil
}
func (m *MockTaskServiceForUserRequest) ForceReset(id uint) error {
	return nil
}
func (m *MockTaskServiceForUserRequest) Retry(id uint) error {
	return nil
}
func (m *MockTaskServiceForUserRequest) ReGenByNewTask(id uint) error {
	return nil
}
func (m *MockTaskServiceForUserRequest) Cancel(id uint) error {
	return nil
}
func (m *MockTaskServiceForUserRequest) Delete(id uint) error {
	return nil
}
func (m *MockTaskServiceForUserRequest) GetStuckTasks(timeout string) ([]*model.Task, error) {
	return nil, nil
}
func (m *MockTaskServiceForUserRequest) CleanupStuckTasks(timeout string) (int, error) {
	return 0, nil
}
func (m *MockTaskServiceForUserRequest) GetOrchestratorStatus() map[string]interface{} {
	return nil
}
func (m *MockTaskServiceForUserRequest) GetGlobalMonitorData() (map[string]interface{}, error) {
	return nil, nil
}

// setupUserRequestRouter 创建测试路由
func setupUserRequestRouter(userRequestService service.UserRequestService, taskBus *eventbus.TaskEventBus) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	// Pass nil for task service
	var taskService *service.TaskService = nil

	h := NewUserRequestHandler(userRequestService, taskBus, taskService)

	// Register routes manually
	api := router.Group("/api")
	api.POST("/repositories/:id/user-requests", h.CreateUserRequest)
	api.GET("/repositories/:id/user-requests", h.ListUserRequests)
	api.GET("/user-requests/:id", h.GetUserRequest)
	api.DELETE("/user-requests/:id", h.DeleteUserRequest)
	api.PATCH("/user-requests/:id/status", h.UpdateUserRequestStatus)

	return router
}

// TestUserRequestHandler_CreateUserRequest 测试创建用户需求
func TestUserRequestHandler_CreateUserRequest(t *testing.T) {
	tests := []struct {
		name           string
		repoID         string
		requestBody    interface{}
		mockSetup      func(*MockUserRequestService)
		expectedStatus int
		verifyResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "成功创建用户需求",
			repoID: "1",
			requestBody: map[string]interface{}{
				"content": "test request content",
			},
			mockSetup: func(m *MockUserRequestService) {
				m.createFunc = func(repoID uint, content string) (*model.UserRequest, error) {
					return &model.UserRequest{
						ID:           1,
						RepositoryID: repoID,
						Content:      content,
						Status:       model.UserRequestStatusPending,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, float64(0), response["code"])
				assert.Contains(t, response["message"], "需求已提交")
			},
		},
		{
			name:   "无效的仓库ID",
			repoID: "invalid",
			requestBody: map[string]interface{}{
				"content": "test content",
			},
			mockSetup:      func(m *MockUserRequestService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "缺少content字段",
			repoID: "1",
			requestBody: map[string]interface{}{},
			mockSetup:      func(m *MockUserRequestService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "content为空字符串",
			repoID: "1",
			requestBody: map[string]interface{}{
				"content": "",
			},
			mockSetup: func(m *MockUserRequestService) {
				m.createFunc = func(repoID uint, content string) (*model.UserRequest, error) {
					return nil, errors.New("需求内容不能为空")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "content超过200字符",
			repoID: "1",
			requestBody: map[string]interface{}{
				"content": string(make([]byte, 201)),
			},
			mockSetup: func(m *MockUserRequestService) {
				m.createFunc = func(repoID uint, content string) (*model.UserRequest, error) {
					return nil, errors.New("需求内容不能超过200个字符")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserRequestService{}
			taskBus := eventbus.NewTaskEventBus()
			tt.mockSetup(mockService)

			router := setupUserRequestRouter(mockService, taskBus)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/repositories/"+tt.repoID+"/user-requests", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.verifyResponse != nil {
				tt.verifyResponse(t, w)
			}
		})
	}
}

// TestUserRequestHandler_ListUserRequests 测试获取用户需求列表
func TestUserRequestHandler_ListUserRequests(t *testing.T) {
	tests := []struct {
		name           string
		repoID         string
		query          string
		mockSetup      func(*MockUserRequestService)
		expectedStatus int
		verifyResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "成功获取列表",
			repoID: "1",
			query:  "",
			mockSetup: func(m *MockUserRequestService) {
				m.listFunc = func(repoID uint, page, pageSize int, status string) ([]*model.UserRequest, int64, error) {
					requests := []*model.UserRequest{
						{ID: 1, RepositoryID: repoID, Content: "request 1", Status: model.UserRequestStatusPending},
						{ID: 2, RepositoryID: repoID, Content: "request 2", Status: model.UserRequestStatusCompleted},
					}
					return requests, 2, nil
				}
			},
			expectedStatus: http.StatusOK,
			verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, float64(0), response["code"])
				data := response["data"].(map[string]interface{})
				assert.Equal(t, float64(2), data["total"])
			},
		},
		{
			name:           "无效的仓库ID",
			repoID:         "invalid",
			query:          "",
			mockSetup:      func(m *MockUserRequestService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "带分页参数",
			repoID: "1",
			query:  "?page=2&page_size=10",
			mockSetup: func(m *MockUserRequestService) {
				m.listFunc = func(repoID uint, page, pageSize int, status string) ([]*model.UserRequest, int64, error) {
					assert.Equal(t, 2, page)
					assert.Equal(t, 10, pageSize)
					return []*model.UserRequest{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "带状态过滤",
			repoID: "1",
			query:  "?status=completed",
			mockSetup: func(m *MockUserRequestService) {
				m.listFunc = func(repoID uint, page, pageSize int, status string) ([]*model.UserRequest, int64, error) {
					assert.Equal(t, "completed", status)
					return []*model.UserRequest{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserRequestService{}
			taskBus := eventbus.NewTaskEventBus()
			tt.mockSetup(mockService)

			router := setupUserRequestRouter(mockService, taskBus)

			req, _ := http.NewRequest(http.MethodGet, "/api/repositories/"+tt.repoID+"/user-requests"+tt.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.verifyResponse != nil {
				tt.verifyResponse(t, w)
			}
		})
	}
}

// TestUserRequestHandler_GetUserRequest 测试获取用户需求详情
func TestUserRequestHandler_GetUserRequest(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockSetup      func(*MockUserRequestService)
		expectedStatus int
	}{
		{
			name: "成功获取详情",
			id:   "1",
			mockSetup: func(m *MockUserRequestService) {
				m.getFunc = func(id uint) (*model.UserRequest, error) {
					return &model.UserRequest{
						ID:           id,
						RepositoryID: 1,
						Content:      "test content",
						Status:       model.UserRequestStatusPending,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "无效的ID",
			id:             "invalid",
			mockSetup:      func(m *MockUserRequestService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "需求不存在",
			id:   "999",
			mockSetup: func(m *MockUserRequestService) {
				m.getFunc = func(id uint) (*model.UserRequest, error) {
					return nil, errors.New("request not found")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserRequestService{}
			taskBus := eventbus.NewTaskEventBus()
			tt.mockSetup(mockService)

			router := setupUserRequestRouter(mockService, taskBus)

			req, _ := http.NewRequest(http.MethodGet, "/api/user-requests/"+tt.id, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestUserRequestHandler_DeleteUserRequest 测试删除用户需求
func TestUserRequestHandler_DeleteUserRequest(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockSetup      func(*MockUserRequestService)
		expectedStatus int
	}{
		{
			name: "成功删除",
			id:   "1",
			mockSetup: func(m *MockUserRequestService) {
				m.deleteFunc = func(id uint) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "无效的ID",
			id:             "invalid",
			mockSetup:      func(m *MockUserRequestService) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserRequestService{}
			taskBus := eventbus.NewTaskEventBus()
			tt.mockSetup(mockService)

			router := setupUserRequestRouter(mockService, taskBus)

			req, _ := http.NewRequest(http.MethodDelete, "/api/user-requests/"+tt.id, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestUserRequestHandler_UpdateUserRequestStatus 测试更新用户需求状态
func TestUserRequestHandler_UpdateUserRequestStatus(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		requestBody    interface{}
		mockSetup      func(*MockUserRequestService)
		expectedStatus int
	}{
		{
			name: "成功更新状态",
			id:   "1",
			requestBody: map[string]interface{}{
				"status": "completed",
			},
			mockSetup: func(m *MockUserRequestService) {
				m.updateFunc = func(id uint, status string) error {
					assert.Equal(t, "completed", status)
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "无效的ID",
			id:             "invalid",
			requestBody:    map[string]interface{}{"status": "completed"},
			mockSetup:      func(m *MockUserRequestService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "缺少status字段",
			id:             "1",
			requestBody:    map[string]interface{}{},
			mockSetup:      func(m *MockUserRequestService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "无效的状态值",
			id:     "1",
			requestBody: map[string]interface{}{
				"status": "invalid",
			},
			mockSetup: func(m *MockUserRequestService) {
				m.updateFunc = func(id uint, status string) error {
					return errors.New("无效的状态值")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserRequestService{}
			taskBus := eventbus.NewTaskEventBus()
			tt.mockSetup(mockService)

			router := setupUserRequestRouter(mockService, taskBus)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPatch, "/api/user-requests/"+tt.id+"/status", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestUserRequestHandler_ContentLengthValidation 测试内容长度验证
func TestUserRequestHandler_ContentLengthValidation(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedStatus int
	}{
		{
			name:           "内容为200字符",
			content:        string(make([]byte, 200)),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "内容为201字符",
			content:        string(make([]byte, 201)),
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserRequestService{}
			taskBus := eventbus.NewTaskEventBus()

			requestCount := 0
			mockService.createFunc = func(repoID uint, content string) (*model.UserRequest, error) {
				requestCount++
				if len(content) > 200 {
					return nil, errors.New("需求内容不能超过200个字符")
				}
				return &model.UserRequest{
					ID:           1,
					RepositoryID: repoID,
					Content:      content,
					Status:       model.UserRequestStatusPending,
				}, nil
			}

			router := setupUserRequestRouter(mockService, taskBus)

			requestBody := map[string]interface{}{
				"content": tt.content,
			}
			body, _ := json.Marshal(requestBody)

			req, _ := http.NewRequest(http.MethodPost, "/api/repositories/1/user-requests", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestUserRequestStatuses 测试用户需求状态常量
func TestUserRequestStatuses(t *testing.T) {
	assert.Equal(t, "pending", model.UserRequestStatusPending)
	assert.Equal(t, "processing", model.UserRequestStatusProcessing)
	assert.Equal(t, "completed", model.UserRequestStatusCompleted)
	assert.Equal(t, "rejected", model.UserRequestStatusRejected)
}

// TestUserRequestHandler_EventBus 测试事件总线集成
func TestUserRequestHandler_EventBus(t *testing.T) {
	mockService := &MockUserRequestService{}
	taskBus := eventbus.NewTaskEventBus()

	// Track events published
	var publishedEvent *eventbus.TaskEvent
	eventPublished := false

	taskBus.Subscribe(eventbus.TaskEventUserRequest, func(ctx context.Context, event eventbus.TaskEvent) error {
		publishedEvent = &event
		eventPublished = true
		return nil
	})

	mockService.createFunc = func(repoID uint, content string) (*model.UserRequest, error) {
		return &model.UserRequest{
			ID:           1,
			RepositoryID: repoID,
			Content:      content,
			Status:       model.UserRequestStatusPending,
		}, nil
	}

	router := setupUserRequestRouter(mockService, taskBus)

	requestBody := map[string]interface{}{
		"content": "test request",
	}
	body, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/repositories/1/user-requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify event was published
	assert.True(t, eventPublished, "Event should be published")
	assert.NotNil(t, publishedEvent, "Published event should not be nil")
	assert.Equal(t, eventbus.TaskEventUserRequest, publishedEvent.Type)
	assert.Equal(t, uint(1), publishedEvent.RepositoryID)
	assert.Equal(t, "test request", publishedEvent.Title)
	assert.Equal(t, domain.UserRequestWriter, publishedEvent.WriterName)
}
