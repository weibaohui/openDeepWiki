package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

// UserRequestService 用户需求服务接口
type UserRequestService interface {
	// CreateRequest 创建用户需求
	CreateRequest(repoID uint, content string) (*model.UserRequest, error)
	// GetRequest 获取用户需求详情
	GetRequest(id uint) (*model.UserRequest, error)
	// ListRequests 获取用户需求列表
	ListRequests(repoID uint, page, pageSize int, status string) ([]*model.UserRequest, int64, error)
	// DeleteRequest 删除用户需求
	DeleteRequest(id uint) error
	// UpdateStatus 更新用户需求状态
	UpdateStatus(id uint, status string) error
}

// userRequestService 用户需求服务实现
type userRequestService struct {
	userRequestRepo repository.UserRequestRepository
	repoRepo        repository.RepoRepository
}

// NewUserRequestService 创建用户需求服务实例
func NewUserRequestService(userRequestRepo repository.UserRequestRepository, repoRepo repository.RepoRepository) UserRequestService {
	klog.V(6).Infof("[service] 创建 UserRequestService")
	return &userRequestService{
		userRequestRepo: userRequestRepo,
		repoRepo:        repoRepo,
	}
}

// CreateRequest 创建用户需求
// 验证输入内容，验证仓库存在性，然后创建用户需求记录
func (s *userRequestService) CreateRequest(repoID uint, content string) (*model.UserRequest, error) {
	klog.V(6).Infof("[service] 创建用户需求: repoID=%d, content=%s", repoID, content)

	// 验证内容
	if content == "" {
		klog.Warningf("[service] 创建用户需求失败: 内容为空")
		return nil, errors.New("需求内容不能为空")
	}

	if len(content) > 200 {
		klog.Warningf("[service] 创建用户需求失败: 内容过长, length=%d", len(content))
		return nil, errors.New("需求内容不能超过200个字符")
	}

	// 验证仓库是否存在
	_, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		klog.Errorf("[service] 创建用户需求失败: 仓库不存在, repoID=%d, error=%v", repoID, err)
		return nil, fmt.Errorf("仓库不存在: %w", err)
	}

	// 创建用户需求
	now := time.Now()
	request := &model.UserRequest{
		RepositoryID: repoID,
		Content:      content,
		Status:       model.UserRequestStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRequestRepo.Create(request); err != nil {
		klog.Errorf("[service] 创建用户需求失败: 保存失败, error=%v", err)
		return nil, fmt.Errorf("保存需求失败: %w", err)
	}

	klog.V(6).Infof("[service] 用户需求创建成功: id=%d", request.ID)
	return request, nil
}

// GetRequest 获取用户需求详情
func (s *userRequestService) GetRequest(id uint) (*model.UserRequest, error) {
	klog.V(6).Infof("[service] 获取用户需求: id=%d", id)
	request, err := s.userRequestRepo.GetByID(id)
	if err != nil {
		klog.Errorf("[service] 获取用户需求失败: id=%d, error=%v", id, err)
		return nil, fmt.Errorf("获取需求失败: %w", err)
	}
	return request, nil
}

// ListRequests 获取用户需求列表
// 支持分页和状态过滤
func (s *userRequestService) ListRequests(repoID uint, page, pageSize int, status string) ([]*model.UserRequest, int64, error) {
	klog.V(6).Infof("[service] 获取用户需求列表: repoID=%d, page=%d, pageSize=%d, status=%s", repoID, page, pageSize, status)

	requests, total, err := s.userRequestRepo.ListByRepository(repoID, page, pageSize, status)
	if err != nil {
		klog.Errorf("[service] 获取用户需求列表失败: error=%v", err)
		return nil, 0, fmt.Errorf("获取需求列表失败: %w", err)
	}

	klog.V(6).Infof("[service] 获取用户需求列表成功: total=%d, returned=%d", total, len(requests))
	return requests, total, nil
}

// DeleteRequest 删除用户需求
func (s *userRequestService) DeleteRequest(id uint) error {
	klog.V(6).Infof("[service] 删除用户需求: id=%d", id)

	// 验证需求是否存在
	_, err := s.userRequestRepo.GetByID(id)
	if err != nil {
		klog.Errorf("[service] 删除用户需求失败: 需求不存在, id=%d, error=%v", id, err)
		return fmt.Errorf("需求不存在: %w", err)
	}

	if err := s.userRequestRepo.Delete(id); err != nil {
		klog.Errorf("[service] 删除用户需求失败: error=%v", err)
		return fmt.Errorf("删除需求失败: %w", err)
	}

	klog.V(6).Infof("[service] 用户需求删除成功: id=%d", id)
	return nil
}

// UpdateStatus 更新用户需求状态
func (s *userRequestService) UpdateStatus(id uint, status string) error {
	klog.V(6).Infof("[service] 更新用户需求状态: id=%d, status=%s", id, status)

	// 验证状态值是否有效
	validStatuses := map[string]bool{
		model.UserRequestStatusPending:    true,
		model.UserRequestStatusProcessing: true,
		model.UserRequestStatusCompleted:  true,
		model.UserRequestStatusRejected:   true,
	}
	if !validStatuses[status] {
		klog.Warningf("[service] 更新用户需求状态失败: 无效状态, status=%s", status)
		return errors.New("无效的状态值")
	}

	if err := s.userRequestRepo.UpdateStatus(id, status); err != nil {
		klog.Errorf("[service] 更新用户需求状态失败: error=%v", err)
		return fmt.Errorf("更新需求状态失败: %w", err)
	}

	klog.V(6).Infof("[service] 用户需求状态更新成功: id=%d, status=%s", id, status)
	return nil
}
