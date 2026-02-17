package repository

import (
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

// userRequestRepository 用户需求仓库实现
type userRequestRepository struct {
	db *gorm.DB
}

// NewUserRequestRepository 创建用户需求仓库实例
func NewUserRequestRepository(db *gorm.DB) UserRequestRepository {
	klog.V(6).Infof("[repository] 创建 UserRequestRepository")
	return &userRequestRepository{db: db}
}

// Create 创建用户需求
func (r *userRequestRepository) Create(request *model.UserRequest) error {
	klog.V(6).Infof("[repository] 创建用户需求: repoID=%d, content=%s", request.RepositoryID, request.Content)
	if err := r.db.Create(request).Error; err != nil {
		klog.Errorf("[repository] 创建用户需求失败: %v", err)
		return err
	}
	return nil
}

// GetByID 根据 ID 获取用户需求
func (r *userRequestRepository) GetByID(id uint) (*model.UserRequest, error) {
	klog.V(6).Infof("[repository] 获取用户需求: id=%d", id)
	var request model.UserRequest
	err := r.db.Preload("Repository").First(&request, id).Error
	if err != nil {
		klog.Errorf("[repository] 获取用户需求失败: id=%d, error=%v", id, err)
		return nil, err
	}
	return &request, nil
}

// ListByRepository 获取仓库的用户需求列表
// 支持分页和状态过滤
func (r *userRequestRepository) ListByRepository(repoID uint, page, pageSize int, status string) ([]*model.UserRequest, int64, error) {
	klog.V(6).Infof("[repository] 获取用户需求列表: repoID=%d, page=%d, pageSize=%d, status=%s", repoID, page, pageSize, status)
	var requests []*model.UserRequest
	var total int64

	query := r.db.Model(&model.UserRequest{}).Where("repository_id = ?", repoID)

	// 状态过滤
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		klog.Errorf("[repository] 获取用户需求总数失败: %v", err)
		return nil, 0, err
	}

	// 分页查询，按创建时间倒序
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&requests).Error; err != nil {
		klog.Errorf("[repository] 获取用户需求列表失败: %v", err)
		return nil, 0, err
	}

	klog.V(6).Infof("[repository] 获取用户需求列表成功: total=%d, returned=%d", total, len(requests))
	return requests, total, nil
}

// Delete 删除用户需求
func (r *userRequestRepository) Delete(id uint) error {
	klog.V(6).Infof("[repository] 删除用户需求: id=%d", id)
	if err := r.db.Delete(&model.UserRequest{}, id).Error; err != nil {
		klog.Errorf("[repository] 删除用户需求失败: %v", err)
		return err
	}
	return nil
}

// UpdateStatus 更新用户需求状态
func (r *userRequestRepository) UpdateStatus(id uint, status string) error {
	klog.V(6).Infof("[repository] 更新用户需求状态: id=%d, status=%s", id, status)
	result := r.db.Model(&model.UserRequest{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		klog.Errorf("[repository] 更新用户需求状态失败: %v", result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		klog.Warningf("[repository] 更新用户需求状态: 未找到记录 id=%d", id)
		return gorm.ErrRecordNotFound
	}
	return nil
}
