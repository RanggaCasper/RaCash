// Package repository berisi model GORM dan implementasi repository modul reseller.
// Model sinkron dengan database.dbml § MODUL: reseller.
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/database"
)

type APIKey struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID          uuid.UUID `gorm:"type:uuid"`
	Name            *string
	KeyPrefix       string
	KeyHash         string
	SecretEncrypted []byte
	SecretKeyID     string
	LastUsedAt      *time.Time
	LastUsedIP      *string `gorm:"type:inet"`
	RevokedAt       *time.Time
	CreatedAt       time.Time
}

func (APIKey) TableName() string { return "api_keys" }

type ResellerSetting struct {
	UserID                  uuid.UUID `gorm:"type:uuid;primaryKey"`
	CallbackURL             *string
	CallbackSecretEncrypted []byte
	CallbackSecretKeyID     *string
	CallbackSecretRotatedAt *time.Time
	IPWhitelist             database.StringArray `gorm:"column:ip_whitelist;type:cidr[]"`
	RateLimitPerMinute      int32
	IsH2HEnabled            bool `gorm:"column:is_h2h_enabled"`
	UpdatedAt               time.Time
}

func (ResellerSetting) TableName() string { return "reseller_settings" }

type ResellerCallback struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID         uuid.UUID `gorm:"type:uuid"`
	OrderID        uuid.UUID `gorm:"type:uuid"`
	Event          string    // order.succeeded, order.failed, order.refunded
	EventID        string
	URL            string
	Payload        datatypes.JSON
	Status         string // delivery_status
	Attempts       int16
	LastHTTPStatus *int16 `gorm:"column:last_http_status"`
	LastError      *string
	NextRetryAt    *time.Time
	DeliveredAt    *time.Time
	CreatedAt      time.Time
}

func (ResellerCallback) TableName() string { return "reseller_callbacks" }
