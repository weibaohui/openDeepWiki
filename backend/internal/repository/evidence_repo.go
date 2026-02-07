package repository

import (
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
