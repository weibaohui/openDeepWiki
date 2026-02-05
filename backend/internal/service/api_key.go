package service

import (
	"context"
	"fmt"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

// APIKeyService API Key 服务接口
type APIKeyService interface {
	// CreateAPIKey 创建 API Key 配置
	CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*model.APIKey, error)

	// UpdateAPIKey 更新 API Key 配置
	UpdateAPIKey(ctx context.Context, id uint, req *UpdateAPIKeyRequest) (*model.APIKey, error)

	// DeleteAPIKey 删除 API Key 配置
	DeleteAPIKey(ctx context.Context, id uint) error

	// GetAPIKey 获取 API Key 配置
	GetAPIKey(ctx context.Context, id uint) (*model.APIKey, error)

	// ListAPIKeys 列出所有 API Key 配置
	ListAPIKeys(ctx context.Context) ([]*model.APIKey, error)

	// UpdateAPIKeyStatus 更新状态
	UpdateAPIKeyStatus(ctx context.Context, id uint, status string) error

	// GetStats 获取统计信息
	GetStats(ctx context.Context) (map[string]interface{}, error)

	// RecordRequest 记录请求
	RecordRequest(ctx context.Context, apiKeyID uint, success bool) error

	// MarkUnavailable 标记为不可用
	MarkUnavailable(ctx context.Context, apiKeyID uint, resetTime time.Time) error

	// GetAPIKeyByName 根据名称获取
	GetAPIKeyByName(ctx context.Context, name string) (*model.APIKey, error)

	// GetAPIKeysByNames 根据名称列表获取
	GetAPIKeysByNames(ctx context.Context, names []string) ([]*model.APIKey, error)
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	Name     string `json:"name" binding:"required"`
	Provider string `json:"provider" binding:"required"`
	BaseURL  string `json:"base_url" binding:"required"`
	APIKey   string `json:"api_key" binding:"required"`
	Model    string `json:"model" binding:"required"`
	Priority int    `json:"priority"`
}

// UpdateAPIKeyRequest 更新 API Key 请求
type UpdateAPIKeyRequest struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	Priority int    `json:"priority"`
}

// apiKeyService API Key 服务实现
type apiKeyService struct {
	repo repository.APIKeyRepository
}

// NewAPIKeyService 创建 API Key 服务
func NewAPIKeyService(repo repository.APIKeyRepository) APIKeyService {
	return &apiKeyService{repo: repo}
}

// CreateAPIKey 创建 API Key 配置
func (s *apiKeyService) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*model.APIKey, error) {
	klog.V(6).Infof("CreateAPIKey: creating API Key with name=%s", req.Name)

	// 校验名称唯一性
	existing, err := s.repo.GetByName(ctx, req.Name)
	if err == nil && existing != nil {
		klog.Warningf("CreateAPIKey: API Key name %s already exists", req.Name)
		return nil, repository.ErrAPIKeyDuplicate
	}

	apiKey := &model.APIKey{
		Name:     req.Name,
		Provider: req.Provider,
		BaseURL:  req.BaseURL,
		APIKey:   req.APIKey,
		Model:    req.Model,
		Priority: req.Priority,
		Status:   "enabled",
	}

	if err := s.repo.Create(ctx, apiKey); err != nil {
		klog.Errorf("CreateAPIKey: failed to create API Key: %v", err)
		return nil, err
	}

	klog.V(6).Infof("CreateAPIKey: successfully created API Key with id=%d", apiKey.ID)
	return apiKey, nil
}

// UpdateAPIKey 更新 API Key 配置
func (s *apiKeyService) UpdateAPIKey(ctx context.Context, id uint, req *UpdateAPIKeyRequest) (*model.APIKey, error) {
	klog.V(6).Infof("UpdateAPIKey: updating API Key with id=%d", id)

	apiKey, err := s.repo.GetByID(ctx, id)
	if err != nil {
		klog.Errorf("UpdateAPIKey: failed to get API Key: %v", err)
		return nil, err
	}

	// 更新字段
	originalName := apiKey.Name
	if req.Name != "" && req.Name != originalName {
		// 校验新名称是否已存在
		existing, err := s.repo.GetByName(ctx, req.Name)
		if err == nil && existing != nil && existing.ID != id {
			klog.Warningf("UpdateAPIKey: API Key name %s already exists", req.Name)
			return nil, repository.ErrAPIKeyDuplicate
		}
		apiKey.Name = req.Name
	}
	if req.Provider != "" {
		apiKey.Provider = req.Provider
	}
	if req.BaseURL != "" {
		apiKey.BaseURL = req.BaseURL
	}
	if req.APIKey != "" {
		apiKey.APIKey = req.APIKey
	}
	if req.Model != "" {
		apiKey.Model = req.Model
	}
	if req.Priority > 0 {
		apiKey.Priority = req.Priority
	}

	if err := s.repo.Update(ctx, apiKey); err != nil {
		klog.Errorf("UpdateAPIKey: failed to update API Key: %v", err)
		return nil, err
	}

	klog.V(6).Infof("UpdateAPIKey: successfully updated API Key with id=%d", id)
	return apiKey, nil
}

// DeleteAPIKey 删除 API Key 配置
func (s *apiKeyService) DeleteAPIKey(ctx context.Context, id uint) error {
	klog.V(6).Infof("DeleteAPIKey: deleting API Key with id=%d", id)

	if err := s.repo.Delete(ctx, id); err != nil {
		klog.Errorf("DeleteAPIKey: failed to delete API Key: %v", err)
		return err
	}

	klog.V(6).Infof("DeleteAPIKey: successfully deleted API Key with id=%d", id)
	return nil
}

// GetAPIKey 获取 API Key 配置
func (s *apiKeyService) GetAPIKey(ctx context.Context, id uint) (*model.APIKey, error) {
	return s.repo.GetByID(ctx, id)
}

// ListAPIKeys 列出所有 API Key 配置
func (s *apiKeyService) ListAPIKeys(ctx context.Context) ([]*model.APIKey, error) {
	return s.repo.List(ctx)
}

// UpdateAPIKeyStatus 更新状态
func (s *apiKeyService) UpdateAPIKeyStatus(ctx context.Context, id uint, status string) error {
	klog.V(6).Infof("UpdateAPIKeyStatus: updating status to %s for API Key with id=%d", status, id)

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		klog.Errorf("UpdateAPIKeyStatus: failed to update status: %v", err)
		return err
	}

	klog.V(6).Infof("UpdateAPIKeyStatus: successfully updated status for API Key with id=%d", id)
	return nil
}

// GetStats 获取统计信息
func (s *apiKeyService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return s.repo.GetStats(ctx)
}

// RecordRequest 记录请求
func (s *apiKeyService) RecordRequest(ctx context.Context, apiKeyID uint, success bool) error {
	requestCount := 1
	errorCount := 0
	if !success {
		errorCount = 1
	}

	klog.V(6).Infof("RecordRequest: recording request for API Key id=%d, success=%v", apiKeyID, success)
	return s.repo.IncrementStats(ctx, apiKeyID, requestCount, errorCount)
}

// MarkUnavailable 标记为不可用
func (s *apiKeyService) MarkUnavailable(ctx context.Context, apiKeyID uint, resetTime time.Time) error {
	klog.Warningf("MarkUnavailable: marking API Key id=%d as unavailable, reset at %v", apiKeyID, resetTime)

	if err := s.repo.SetRateLimitReset(ctx, apiKeyID, resetTime); err != nil {
		klog.Errorf("MarkUnavailable: failed to mark unavailable: %v", err)
		return err
	}

	return nil
}

// GetAPIKeyByName 根据名称获取
func (s *apiKeyService) GetAPIKeyByName(ctx context.Context, name string) (*model.APIKey, error) {
	return s.repo.GetByName(ctx, name)
}

// GetAPIKeysByNames 根据名称列表获取
func (s *apiKeyService) GetAPIKeysByNames(ctx context.Context, names []string) ([]*model.APIKey, error) {
	klog.V(6).Infof("GetAPIKeysByNames: getting API Keys for names %v", names)
	return s.repo.ListByNames(ctx, names)
}

// ErrInvalidStatus 无效的状态
var ErrInvalidStatus = fmt.Errorf("invalid status")
