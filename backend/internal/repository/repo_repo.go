package repository

import (
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

type repoRepository struct {
	db *gorm.DB
}

func NewRepoRepository(db *gorm.DB) RepoRepository {
	return &repoRepository{db: db}
}

func (r *repoRepository) Create(repo *model.Repository) error {
	return r.db.Create(repo).Error
}

func (r *repoRepository) List() ([]model.Repository, error) {
	var repos []model.Repository
	err := r.db.Order("created_at desc").Find(&repos).Error
	return repos, err
}

func (r *repoRepository) Get(id uint) (*model.Repository, error) {
	var repo model.Repository
	err := r.db.Preload("Tasks").Preload("Documents").First(&repo, id).Error
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (r *repoRepository) GetBasic(id uint) (*model.Repository, error) {
	var repo model.Repository
	err := r.db.First(&repo, id).Error
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (r *repoRepository) Save(repo *model.Repository) error {
	return r.db.Save(repo).Error
}

func (r *repoRepository) Delete(id uint) error {
	return r.db.Delete(&model.Repository{}, id).Error
}
