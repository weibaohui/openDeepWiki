package repository

import (
	"context"
	"errors"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

type syncTargetRepository struct {
	db *gorm.DB
}

func NewSyncTargetRepository(db *gorm.DB) SyncTargetRepository {
	return &syncTargetRepository{db: db}
}

func (r *syncTargetRepository) List(ctx context.Context) ([]model.SyncTarget, error) {
	var targets []model.SyncTarget
	err := r.db.WithContext(ctx).Order("updated_at desc, id desc").Find(&targets).Error
	return targets, err
}

func (r *syncTargetRepository) Upsert(ctx context.Context, url string) (*model.SyncTarget, error) {
	var target model.SyncTarget
	err := r.db.WithContext(ctx).Where("url = ?", url).First(&target).Error
	if err == nil {
		target.UpdatedAt = time.Now()
		if err := r.db.WithContext(ctx).Save(&target).Error; err != nil {
			return nil, err
		}
		return &target, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	target = model.SyncTarget{
		URL: url,
	}
	if err := r.db.WithContext(ctx).Create(&target).Error; err != nil {
		return nil, err
	}
	return &target, nil
}

func (r *syncTargetRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.SyncTarget{}, id).Error
}

func (r *syncTargetRepository) TrimExcess(ctx context.Context, max int) error {
	if max <= 0 {
		return nil
	}
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.SyncTarget{}).Count(&count).Error; err != nil {
		return err
	}
	if int(count) <= max {
		return nil
	}
	trim := int(count) - max
	var targets []model.SyncTarget
	if err := r.db.WithContext(ctx).Order("updated_at asc, id asc").Limit(trim).Find(&targets).Error; err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(targets))
	for _, target := range targets {
		ids = append(ids, target.ID)
	}
	return r.db.WithContext(ctx).Delete(&model.SyncTarget{}, ids).Error
}
