package syncservice

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

// TargetManager 管理同步目标配置
type TargetManager struct {
	syncTargetRepo repository.SyncTargetRepository
}

// NewTargetManager 创建新的目标管理器
func NewTargetManager(syncTargetRepo repository.SyncTargetRepository) *TargetManager {
	return &TargetManager{
		syncTargetRepo: syncTargetRepo,
	}
}

// ValidateTargetServer 验证目标服务器地址格式
func (m *TargetManager) ValidateTargetServer(value string) (string, error) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", errors.New("目标服务器地址不能为空")
	}
	normalized = strings.TrimSuffix(normalized, "/")
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("目标服务器地址格式不正确")
	}
	if !strings.HasSuffix(parsed.Path, "/api/sync") {
		return "", errors.New("目标服务器地址格式不正确")
	}
	return normalized, nil
}

// List 获取所有同步目标
func (m *TargetManager) List(ctx context.Context) ([]model.SyncTarget, error) {
	return m.syncTargetRepo.List(ctx)
}

// Save 保存同步目标
func (m *TargetManager) Save(ctx context.Context, target string) (*model.SyncTarget, error) {
	normalized, err := m.ValidateTargetServer(target)
	if err != nil {
		return nil, err
	}
	targetModel, err := m.syncTargetRepo.Upsert(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if err := m.syncTargetRepo.TrimExcess(ctx, 20); err != nil {
		return nil, err
	}
	klog.V(6).Infof("同步地址已保存: id=%d, url=%s", targetModel.ID, targetModel.URL)
	return targetModel, nil
}

// Delete 删除同步目标
func (m *TargetManager) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return errors.New("地址ID不能为空")
	}
	if err := m.syncTargetRepo.Delete(ctx, id); err != nil {
		return err
	}
	klog.V(6).Infof("同步地址已删除: id=%d", id)
	return nil
}
