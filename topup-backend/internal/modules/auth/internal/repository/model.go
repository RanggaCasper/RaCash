// Package repository berisi model GORM dan implementasi repository modul auth.
// Model sinkron dengan database.dbml § MODUL: auth.
package repository

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"` // claim sid di JWT
	UserID            uuid.UUID `gorm:"type:uuid"`
	RefreshTokenHash  string
	PreviousTokenHash *string
	UserAgent         *string
	IP                *string `gorm:"type:inet"`
	ExpiresAt         time.Time
	LastUsedAt        *time.Time
	RevokedAt         *time.Time
	RevokedReason     *string
	CreatedAt         time.Time
}

func (Session) TableName() string { return "sessions" }

type EmailVerification struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid"`
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (EmailVerification) TableName() string { return "email_verifications" }

type PasswordReset struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid"`
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	IP        *string `gorm:"type:inet"`
	CreatedAt time.Time
}

func (PasswordReset) TableName() string { return "password_resets" }

type UserTOTP struct {
	UserID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	SecretEncrypted []byte
	SecretKeyID     string
	ConfirmedAt     *time.Time
	LastUsedStep    *int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (UserTOTP) TableName() string { return "user_totp" }

type UserRecoveryCode struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid"`
	CodeHash  string
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (UserRecoveryCode) TableName() string { return "user_recovery_codes" }
