package repository

import (
	"errors"

	"pos-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Merchant").First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Merchant").First(&user, "email = ?", email).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Merchant").First(&user, "username = ?", username).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) FindByGoogleID(googleID string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Merchant").First(&user, "google_id = ?", googleID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) FindByEmailOrUsername(identifier string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Merchant").
		Where("email = ? OR username = ?", identifier, identifier).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) UpdateFields(id uuid.UUID, fields map[string]interface{}) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Updates(fields).Error
}

func (r *UserRepository) List(page, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	offset := (page - 1) * limit

	r.db.Model(&models.User{}).Count(&total)
	err := r.db.Preload("Merchant").
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&users).Error

	return users, total, err
}

func (r *UserRepository) ListByStatus(status models.UserStatus, page, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	offset := (page - 1) * limit
	q := r.db.Model(&models.User{}).Where("status = ?", status)
	q.Count(&total)
	err := r.db.Preload("Merchant").
		Where("status = ?", status).
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&users).Error

	return users, total, err
}

func (r *UserRepository) ListByMerchant(merchantID uuid.UUID) ([]models.User, error) {
	var users []models.User
	err := r.db.Where("merchant_id = ? AND role = ?", merchantID, models.RoleUser).
		Order("created_at DESC").
		Find(&users).Error
	return users, err
}

func (r *UserRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.User{}, "id = ?", id).Error
}
