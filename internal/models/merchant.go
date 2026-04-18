package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MerchantStatus string

const (
	MerchantStatusActive   MerchantStatus = "ACTIVE"
	MerchantStatusInactive MerchantStatus = "INACTIVE"
)

type Merchant struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string         `gorm:"not null"                                       json:"name"`
	Status    MerchantStatus `gorm:"type:varchar(20);not null;default:'ACTIVE'"     json:"status"`
	Users     []User         `gorm:"foreignKey:MerchantID"                          json:"users,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

func (m *Merchant) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
