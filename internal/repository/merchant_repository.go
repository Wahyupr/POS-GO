package repository

import (
	"errors"

	"pos-go/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MerchantRepository struct {
	db *gorm.DB
}

func NewMerchantRepository(db *gorm.DB) *MerchantRepository {
	return &MerchantRepository{db: db}
}

func (r *MerchantRepository) Create(merchant *models.Merchant) error {
	return r.db.Create(merchant).Error
}

func (r *MerchantRepository) FindByID(id uuid.UUID) (*models.Merchant, error) {
	var merchant models.Merchant
	err := r.db.First(&merchant, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &merchant, err
}

func (r *MerchantRepository) List() ([]models.Merchant, error) {
	var merchants []models.Merchant
	err := r.db.Order("created_at DESC").Find(&merchants).Error
	return merchants, err
}

func (r *MerchantRepository) Update(merchant *models.Merchant) error {
	return r.db.Save(merchant).Error
}
