package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string
type UserStatus string

const (
	RoleNone     Role = "NONE"
	RoleAdmin    Role = "ADMIN"
	RoleMerchant Role = "MERCHANT"
	RoleUser     Role = "USER"
)

const (
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusInactive UserStatus = "INACTIVE"
	UserStatusPending  UserStatus = "PENDING" // Google login, belum di-assign admin
)

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email        string     `gorm:"uniqueIndex;not null"                           json:"email"`
	Username     *string    `gorm:"uniqueIndex"                                    json:"username,omitempty"`
	PasswordHash *string    `gorm:"column:password_hash"                           json:"-"`
	GoogleID     *string    `gorm:"uniqueIndex;column:google_id"                   json:"-"`
	GoogleAvatar *string    `gorm:"column:google_avatar"                           json:"avatar,omitempty"`
	Name         string     `gorm:"not null"                                       json:"name"`
	Role         Role       `gorm:"type:varchar(20);not null;default:'NONE'"       json:"role"`
	MerchantID   *uuid.UUID `gorm:"type:uuid;column:merchant_id"                   json:"merchant_id,omitempty"`
	Merchant     *Merchant  `gorm:"foreignKey:MerchantID"                          json:"merchant,omitempty"`
	Status       UserStatus `gorm:"type:varchar(20);not null;default:'ACTIVE'"     json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"uniqueIndex;not null;column:token_hash"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (r *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
