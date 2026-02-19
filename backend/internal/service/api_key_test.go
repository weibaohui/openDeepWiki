package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
)

// MockAPIKeyRepository Mock仓库接口
type MockAPIKeyRepository struct {
	mock.Mock
}

func (m *MockAPIKeyRepository) Create(ctx context.Context, apiKey *model.APIKey) error {
	args := m.Called(ctx, apiKey)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) Update(ctx context.Context, apiKey *model.APIKey) error {
	args := m.Called(ctx, apiKey)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) GetByID(ctx context.Context, id uint) (*model.APIKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) GetByName(ctx context.Context, name string) (*model.APIKey, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) List(ctx context.Context) ([]*model.APIKey, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) ListByProvider(ctx context.Context, provider string) ([]*model.APIKey, error) {
	args := m.Called(ctx, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) ListByNames(ctx context.Context, names []string) ([]*model.APIKey, error) {
	args := m.Called(ctx, names)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) GetHighestPriority(ctx context.Context) (*model.APIKey, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) IncrementStats(ctx context.Context, id uint, requestCount int, errorCount int) error {
	args := m.Called(ctx, id, requestCount, errorCount)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) UpdateLastUsedAt(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) SetRateLimitReset(ctx context.Context, id uint, resetTime time.Time) error {
	args := m.Called(ctx, id, resetTime)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// TestAPIKeyService_Create 测试创建 API Key
func TestAPIKeyService_Create(t *testing.T) {
	tests := []struct {
		name        string
		req         *CreateAPIKeyRequest
		mockSetup   func(*MockAPIKeyRepository)
		expectedErr error
		verify      func(*testing.T, *model.APIKey, error)
	}{
		{
			name: "成功创建 API Key",
			req: &CreateAPIKeyRequest{
				Name:     "test-key",
				Provider: "openai",
				BaseURL:  "https://api.openai.com/v1",
				APIKey:   "sk-test123456789",
				Model:    "gpt-4",
				Priority: 10,
			},
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("GetByName", mock.Anything, "test-key").Return(nil, repository.ErrAPIKeyNotFound)
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.APIKey")).Return(nil)
			},
			verify: func(t *testing.T, result *model.APIKey, err error) {
				require.NoError(t, err)
				assert.Equal(t, "test-key", result.Name)
				assert.Equal(t, "openai", result.Provider)
				assert.Equal(t, "https://api.openai.com/v1", result.BaseURL)
				assert.Equal(t, "sk-test123456789", result.APIKey)
				assert.Equal(t, "gpt-4", result.Model)
				assert.Equal(t, 10, result.Priority)
				assert.Equal(t, "enabled", result.Status)
			},
		},
		{
			name: "名称已存在",
			req: &CreateAPIKeyRequest{
				Name:     "duplicate",
				Provider: "openai",
				BaseURL:  "https://api.openai.com/v1",
				APIKey:   "sk-test",
				Model:    "gpt-4",
			},
			mockSetup: func(m *MockAPIKeyRepository) {
				existing := &model.APIKey{ID: 1, Name: "duplicate"}
				m.On("GetByName", mock.Anything, "duplicate").Return(existing, nil)
			},
			expectedErr: repository.ErrAPIKeyDuplicate,
			verify: func(t *testing.T, result *model.APIKey, err error) {
				assert.Error(t, err)
				assert.Equal(t, repository.ErrAPIKeyDuplicate, err)
				assert.Nil(t, result)
			},
		},
		{
			name: "数据库创建失败",
			req: &CreateAPIKeyRequest{
				Name:     "test-key",
				Provider: "openai",
				BaseURL:  "https://api.openai.com/v1",
				APIKey:   "sk-test",
				Model:    "gpt-4",
			},
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("GetByName", mock.Anything, "test-key").Return(nil, repository.ErrAPIKeyNotFound)
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.APIKey")).Return(errors.New("db error"))
			},
			expectedErr: errors.New("db error"),
			verify: func(t *testing.T, result *model.APIKey, err error) {
				assert.Error(t, err)
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAPIKeyRepository)
			tt.mockSetup(mockRepo)

			service := NewAPIKeyService(mockRepo)
			result, err := service.CreateAPIKey(context.Background(), tt.req)

			if tt.verify != nil {
				tt.verify(t, result, err)
			} else {
				if tt.expectedErr != nil {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedErr.Error())
				} else {
					require.NoError(t, err)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// TestAPIKeyService_Update 测试更新 API Key
func TestAPIKeyService_Update(t *testing.T) {
	tests := []struct {
		name        string
		id          uint
		req         *UpdateAPIKeyRequest
		mockSetup   func(*MockAPIKeyRepository)
		expectedErr error
		verify      func(*testing.T, *model.APIKey, error)
	}{
		{
			name: "成功更新所有字段",
			id:   1,
			req: &UpdateAPIKeyRequest{
				Name:     "updated-name",
				Provider: "anthropic",
				BaseURL:  "https://api.anthropic.com/v1",
				APIKey:   "sk-new",
				Model:    "claude-3",
				Priority: 20,
			},
			mockSetup: func(m *MockAPIKeyRepository) {
				existing := &model.APIKey{
					ID:       1,
					Name:     "old-name",
					Provider: "openai",
					BaseURL:  "https://api.openai.com/v1",
					APIKey:   "sk-old",
					Model:    "gpt-4",
					Priority: 10,
				}
				m.On("GetByID", mock.Anything, uint(1)).Return(existing, nil)
				m.On("GetByName", mock.Anything, "updated-name").Return(nil, repository.ErrAPIKeyNotFound)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.APIKey")).Return(nil)
			},
			verify: func(t *testing.T, result *model.APIKey, err error) {
				require.NoError(t, err)
				assert.Equal(t, "updated-name", result.Name)
				assert.Equal(t, "anthropic", result.Provider)
				assert.Equal(t, "https://api.anthropic.com/v1", result.BaseURL)
				assert.Equal(t, "sk-new", result.APIKey)
				assert.Equal(t, "claude-3", result.Model)
				assert.Equal(t, 20, result.Priority)
			},
		},
		{
			name: "更新时名称已存在（其他key）",
			id:   1,
			req: &UpdateAPIKeyRequest{
				Name: "existing-name",
			},
			mockSetup: func(m *MockAPIKeyRepository) {
				existing := &model.APIKey{
					ID:   1,
					Name: "old-name",
				}
				otherKey := &model.APIKey{
					ID:   2,
					Name: "existing-name",
				}
				m.On("GetByID", mock.Anything, uint(1)).Return(existing, nil)
				m.On("GetByName", mock.Anything, "existing-name").Return(otherKey, nil)
			},
			expectedErr: repository.ErrAPIKeyDuplicate,
		},
		{
			name: "API Key 不存在",
			id:   999,
			req: &UpdateAPIKeyRequest{
				Name: "test",
			},
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("GetByID", mock.Anything, uint(999)).Return(nil, repository.ErrAPIKeyNotFound)
			},
			expectedErr: repository.ErrAPIKeyNotFound,
		},
		{
			name: "更新相同名称（不检查）",
			id:   1,
			req: &UpdateAPIKeyRequest{
				Name: "same-name",
			},
			mockSetup: func(m *MockAPIKeyRepository) {
				existing := &model.APIKey{
					ID:   1,
					Name: "same-name",
				}
				m.On("GetByID", mock.Anything, uint(1)).Return(existing, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*model.APIKey")).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAPIKeyRepository)
			tt.mockSetup(mockRepo)

			service := NewAPIKeyService(mockRepo)
			result, err := service.UpdateAPIKey(context.Background(), tt.id, tt.req)

			if tt.verify != nil {
				tt.verify(t, result, err)
			} else {
				if tt.expectedErr != nil {
					assert.Error(t, err)
					assert.Equal(t, tt.expectedErr, err)
				} else {
					require.NoError(t, err)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// TestAPIKeyService_Delete 测试删除 API Key
func TestAPIKeyService_Delete(t *testing.T) {
	tests := []struct {
		name        string
		id          uint
		mockSetup   func(*MockAPIKeyRepository)
		expectedErr error
	}{
		{
			name: "成功删除",
			id:   1,
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("Delete", mock.Anything, uint(1)).Return(nil)
			},
		},
		{
			name: "删除失败",
			id:   1,
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("Delete", mock.Anything, uint(1)).Return(errors.New("delete failed"))
			},
			expectedErr: errors.New("delete failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAPIKeyRepository)
			tt.mockSetup(mockRepo)

			service := NewAPIKeyService(mockRepo)
			err := service.DeleteAPIKey(context.Background(), tt.id)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// TestAPIKeyService_GetAPIKey 测试获取 API Key
func TestAPIKeyService_GetAPIKey(t *testing.T) {
	tests := []struct {
		name        string
		id          uint
		mockSetup   func(*MockAPIKeyRepository)
		expectedErr error
		verify      func(*testing.T, *model.APIKey, error)
	}{
		{
			name: "成功获取",
			id:   1,
			mockSetup: func(m *MockAPIKeyRepository) {
				apiKey := &model.APIKey{
					ID:       1,
					Name:     "test-key",
					Provider: "openai",
				}
				m.On("GetByID", mock.Anything, uint(1)).Return(apiKey, nil)
			},
			verify: func(t *testing.T, result *model.APIKey, err error) {
				require.NoError(t, err)
				assert.Equal(t, uint(1), result.ID)
				assert.Equal(t, "test-key", result.Name)
			},
		},
		{
			name: "API Key 不存在",
			id:   999,
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("GetByID", mock.Anything, uint(999)).Return(nil, repository.ErrAPIKeyNotFound)
			},
			expectedErr: repository.ErrAPIKeyNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAPIKeyRepository)
			tt.mockSetup(mockRepo)

			service := NewAPIKeyService(mockRepo)
			result, err := service.GetAPIKey(context.Background(), tt.id)

			if tt.verify != nil {
				tt.verify(t, result, err)
			} else {
				if tt.expectedErr != nil {
					assert.Error(t, err)
					assert.Equal(t, tt.expectedErr, err)
				} else {
					require.NoError(t, err)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// TestAPIKeyService_ListAPIKeys 测试列出所有 API Key
func TestAPIKeyService_ListAPIKeys(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(*MockAPIKeyRepository)
		verify    func(*testing.T, []*model.APIKey, error)
	}{
		{
			name: "成功列出",
			mockSetup: func(m *MockAPIKeyRepository) {
				keys := []*model.APIKey{
					{ID: 1, Name: "key1"},
					{ID: 2, Name: "key2"},
				}
				m.On("List", mock.Anything).Return(keys, nil)
			},
			verify: func(t *testing.T, result []*model.APIKey, err error) {
				require.NoError(t, err)
				assert.Len(t, result, 2)
				assert.Equal(t, "key1", result[0].Name)
				assert.Equal(t, "key2", result[1].Name)
			},
		},
		{
			name: "返回空列表",
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("List", mock.Anything).Return([]*model.APIKey{}, nil)
			},
			verify: func(t *testing.T, result []*model.APIKey, err error) {
				require.NoError(t, err)
				assert.Len(t, result, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAPIKeyRepository)
			tt.mockSetup(mockRepo)

			service := NewAPIKeyService(mockRepo)
			result, err := service.ListAPIKeys(context.Background())

			tt.verify(t, result, err)
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestAPIKeyService_UpdateAPIKeyStatus 测试更新状态
func TestAPIKeyService_UpdateAPIKeyStatus(t *testing.T) {
	tests := []struct {
		name        string
		id          uint
		status      string
		mockSetup   func(*MockAPIKeyRepository)
		expectedErr error
	}{
		{
			name:   "成功启用",
			id:     1,
			status: "enabled",
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("UpdateStatus", mock.Anything, uint(1), "enabled").Return(nil)
			},
		},
		{
			name:   "成功禁用",
			id:     1,
			status: "disabled",
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("UpdateStatus", mock.Anything, uint(1), "disabled").Return(nil)
			},
		},
		{
			name:   "更新失败",
			id:     1,
			status: "enabled",
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("UpdateStatus", mock.Anything, uint(1), "enabled").Return(errors.New("update failed"))
			},
			expectedErr: errors.New("update failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAPIKeyRepository)
			tt.mockSetup(mockRepo)

			service := NewAPIKeyService(mockRepo)
			err := service.UpdateAPIKeyStatus(context.Background(), tt.id, tt.status)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// TestAPIKeyService_GetStats 测试获取统计信息
func TestAPIKeyService_GetStats(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(*MockAPIKeyRepository)
		verify    func(*testing.T, map[string]interface{}, error)
	}{
		{
			name: "成功获取统计",
			mockSetup: func(m *MockAPIKeyRepository) {
				stats := map[string]interface{}{
					"total_count":    int64(10),
					"enabled_count":  int64(8),
					"disabled_count": int64(2),
					"total_requests": int64(1000),
					"total_errors":   int64(50),
				}
				m.On("GetStats", mock.Anything).Return(stats, nil)
			},
			verify: func(t *testing.T, result map[string]interface{}, err error) {
				require.NoError(t, err)
				assert.Equal(t, int64(10), result["total_count"])
				assert.Equal(t, int64(8), result["enabled_count"])
				assert.Equal(t, int64(1000), result["total_requests"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAPIKeyRepository)
			tt.mockSetup(mockRepo)

			service := NewAPIKeyService(mockRepo)
			result, err := service.GetStats(context.Background())

			tt.verify(t, result, err)
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestAPIKeyService_RecordRequest 测试记录请求
func TestAPIKeyService_RecordRequest(t *testing.T) {
	tests := []struct {
		name        string
		apiKeyID    uint
		success     bool
		mockSetup   func(*MockAPIKeyRepository)
		expectedErr error
	}{
		{
			name:     "记录成功请求",
			apiKeyID: 1,
			success:  true,
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("IncrementStats", mock.Anything, uint(1), 1, 0).Return(nil)
			},
		},
		{
			name:     "记录失败请求",
			apiKeyID: 1,
			success:  false,
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("IncrementStats", mock.Anything, uint(1), 1, 1).Return(nil)
			},
		},
		{
			name:     "记录失败",
			apiKeyID: 1,
			success:  true,
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("IncrementStats", mock.Anything, uint(1), 1, 0).Return(errors.New("record failed"))
			},
			expectedErr: errors.New("record failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAPIKeyRepository)
			tt.mockSetup(mockRepo)

			service := NewAPIKeyService(mockRepo)
			err := service.RecordRequest(context.Background(), tt.apiKeyID, tt.success)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// TestAPIKeyService_MarkUnavailable 测试标记为不可用
func TestAPIKeyService_MarkUnavailable(t *testing.T) {
	tests := []struct {
		name        string
		apiKeyID    uint
		resetTime   time.Time
		mockSetup   func(*MockAPIKeyRepository)
		expectedErr error
	}{
		{
			name:     "成功标记",
			apiKeyID: 1,
			resetTime: time.Now().Add(5 * time.Minute),
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("SetRateLimitReset", mock.Anything, uint(1), mock.AnythingOfType("time.Time")).Return(nil)
			},
		},
		{
			name:     "标记失败",
			apiKeyID: 1,
			resetTime: time.Now(),
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("SetRateLimitReset", mock.Anything, uint(1), mock.AnythingOfType("time.Time")).Return(errors.New("mark failed"))
			},
			expectedErr: errors.New("mark failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAPIKeyRepository)
			tt.mockSetup(mockRepo)

			service := NewAPIKeyService(mockRepo)
			err := service.MarkUnavailable(context.Background(), tt.apiKeyID, tt.resetTime)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// TestAPIKeyService_GetAPIKeyByName 测试根据名称获取
func TestAPIKeyService_GetAPIKeyByName(t *testing.T) {
	tests := []struct {
		name        string
		nameStr     string
		mockSetup   func(*MockAPIKeyRepository)
		expectedErr error
		verify      func(*testing.T, *model.APIKey, error)
	}{
		{
			name:    "成功获取",
			nameStr: "test-key",
			mockSetup: func(m *MockAPIKeyRepository) {
				apiKey := &model.APIKey{ID: 1, Name: "test-key"}
				m.On("GetByName", mock.Anything, "test-key").Return(apiKey, nil)
			},
			verify: func(t *testing.T, result *model.APIKey, err error) {
				require.NoError(t, err)
				assert.Equal(t, "test-key", result.Name)
			},
		},
		{
			name:    "不存在",
			nameStr: "non-existent",
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("GetByName", mock.Anything, "non-existent").Return(nil, repository.ErrAPIKeyNotFound)
			},
			expectedErr: repository.ErrAPIKeyNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAPIKeyRepository)
			tt.mockSetup(mockRepo)

			service := NewAPIKeyService(mockRepo)
			result, err := service.GetAPIKeyByName(context.Background(), tt.nameStr)

			if tt.verify != nil {
				tt.verify(t, result, err)
			} else {
				if tt.expectedErr != nil {
					assert.Error(t, err)
					assert.Equal(t, tt.expectedErr, err)
				} else {
					require.NoError(t, err)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// TestAPIKeyService_GetAPIKeysByNames 测试根据名称列表获取
func TestAPIKeyService_GetAPIKeysByNames(t *testing.T) {
	tests := []struct {
		name      string
		names     []string
		mockSetup func(*MockAPIKeyRepository)
		verify    func(*testing.T, []*model.APIKey, error)
	}{
		{
			name:  "成功获取",
			names: []string{"key1", "key2"},
			mockSetup: func(m *MockAPIKeyRepository) {
				keys := []*model.APIKey{
					{ID: 1, Name: "key1"},
					{ID: 2, Name: "key2"},
				}
				m.On("ListByNames", mock.Anything, []string{"key1", "key2"}).Return(keys, nil)
			},
			verify: func(t *testing.T, result []*model.APIKey, err error) {
				require.NoError(t, err)
				assert.Len(t, result, 2)
			},
		},
		{
			name:  "空名称列表",
			names: []string{},
			mockSetup: func(m *MockAPIKeyRepository) {
				m.On("ListByNames", mock.Anything, []string{}).Return([]*model.APIKey{}, nil)
			},
			verify: func(t *testing.T, result []*model.APIKey, err error) {
				require.NoError(t, err)
				assert.Len(t, result, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAPIKeyRepository)
			tt.mockSetup(mockRepo)

			service := NewAPIKeyService(mockRepo)
			result, err := service.GetAPIKeysByNames(context.Background(), tt.names)

			tt.verify(t, result, err)
			mockRepo.AssertExpectations(t)
		})
	}
}
