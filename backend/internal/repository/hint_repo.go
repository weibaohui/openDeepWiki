package repository

import (
	"strings"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

type hintRepository struct {
	db *gorm.DB
}

func NewHintRepository(db *gorm.DB) HintRepository {
	return &hintRepository{db: db}
}

func (r *hintRepository) CreateBatch(hints []model.TaskHint) error {
	if len(hints) == 0 {
		return nil
	}
	return r.db.Create(&hints).Error
}

func (r *hintRepository) GetByTaskID(taskID uint) ([]model.TaskHint, error) {
	var hints []model.TaskHint
	err := r.db.Where("task_id = ?", taskID).Order("id").Find(&hints).Error
	return hints, err
}

func (r *hintRepository) SearchInRepo(repoID uint, keywords []string) ([]model.TaskHint, error) {
	var hints []model.TaskHint
	if repoID == 0 {
		return hints, nil
	}
	if len(keywords) == 0 {
		return hints, nil
	}

	tx := r.db.Model(&model.TaskHint{}).Where("repository_id = ?", repoID)

	var orCond *gorm.DB
	for _, kw := range keywords {
		keyword := strings.TrimSpace(kw)
		if keyword == "" {
			continue
		}
		pat := "%" + keyword + "%"
		nextCond := r.db.
			Where("title LIKE ?", pat).
			Or("aspect LIKE ?", pat).
			Or("source LIKE ?", pat).
			Or("detail LIKE ?", pat)
		if orCond == nil {
			orCond = nextCond
		} else {
			orCond = orCond.Or(nextCond)
		}
	}

	if orCond == nil {
		return hints, nil
	}

	if err := tx.Where(orCond).Order("id").Find(&hints).Error; err != nil {
		return nil, err
	}
	return hints, nil
}
