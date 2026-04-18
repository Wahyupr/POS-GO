package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"pos-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (r *TokenRepository) Save(userID uuid.UUID, rawToken string, expiresAt time.Time) error {
	rt := models.RefreshToken{
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: expiresAt,
	}
	return r.db.Create(&rt).Error
}

func (r *TokenRepository) FindByToken(rawToken string) (*models.RefreshToken, error) {
	var rt models.RefreshToken
	err := r.db.Where("token_hash = ? AND expires_at > NOW()", hashToken(rawToken)).First(&rt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rt, err
}

func (r *TokenRepository) DeleteByToken(rawToken string) error {
	return r.db.Where("token_hash = ?", hashToken(rawToken)).Delete(&models.RefreshToken{}).Error
}

func (r *TokenRepository) DeleteByUserID(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error
}
