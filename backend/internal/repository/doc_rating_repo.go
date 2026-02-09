package repository

import (
	"errors"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

type documentRatingRepository struct {
	db *gorm.DB
}

// NewDocumentRatingRepository 创建文档评分数据仓库
func NewDocumentRatingRepository(db *gorm.DB) DocumentRatingRepository {
	return &documentRatingRepository{db: db}
}

// Create 创建文档评分记录
func (r *documentRatingRepository) Create(rating *model.DocumentRating) error {
	return r.db.Create(rating).Error
}

// GetLatestByDocumentID 获取文档最新一条评分记录
func (r *documentRatingRepository) GetLatestByDocumentID(documentID uint) (*model.DocumentRating, error) {
	var rating model.DocumentRating
	err := r.db.Where("document_id = ?", documentID).
		Order("id DESC").
		First(&rating).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rating, nil
}

// GetStatsByDocumentID 获取文档评分统计信息
func (r *documentRatingRepository) GetStatsByDocumentID(documentID uint) (*model.DocumentRatingStats, error) {
	var stats model.DocumentRatingStats
	err := r.db.Model(&model.DocumentRating{}).
		Where("document_id = ?", documentID).
		Select("AVG(score) as average_score, COUNT(*) as rating_count").
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
