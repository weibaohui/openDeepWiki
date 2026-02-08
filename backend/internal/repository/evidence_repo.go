package repository

import (
	"strings"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

type evidenceRepository struct {
	db *gorm.DB
}

func NewEvidenceRepository(db *gorm.DB) EvidenceRepository {
	return &evidenceRepository{db: db}
}

func (r *evidenceRepository) CreateBatch(evidences []model.TaskEvidence) error {
	if len(evidences) == 0 {
		return nil
	}
	return r.db.Create(&evidences).Error
}

func (r *evidenceRepository) GetByTaskID(taskID uint) ([]model.TaskEvidence, error) {
	var evidences []model.TaskEvidence
	err := r.db.Where("task_id = ?", taskID).Order("id").Find(&evidences).Error
	return evidences, err
}

func (r *evidenceRepository) SearchInRepo(repoID uint, keywords []string) ([]model.TaskEvidence, error) {
	var evidences []model.TaskEvidence
	if repoID == 0 {
		return evidences, nil
	}
	if len(keywords) == 0 {
		return evidences, nil
	}

	tx := r.db.Model(&model.TaskEvidence{}).Where("repository_id = ?", repoID)

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
		return evidences, nil
	}

	if err := tx.Where(orCond).Order("id").Find(&evidences).Error; err != nil {
		return nil, err
	}
	return evidences, nil
}
