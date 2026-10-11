// Package repository berisi model GORM dan implementasi repository modul promo.
// Model sinkron dengan database.dbml § MODUL: promo.
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/money"
)

type Promo struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code           string    // uppercase
	Name           string
	Type           string        // amount_type
	Value          money.Percent // numeric(12,3), skala ×1000: rupiah bila FLAT, persen bila PERCENT
	MaxDiscount    *int64
	MinTransaction int64
	Quota          *int32
	UsedCount      int32
	PerUserLimit   *int32
	Rules          datatypes.JSON
	StartsAt       time.Time
	EndsAt         time.Time
	IsActive       bool
	CreatedBy      *uuid.UUID `gorm:"type:uuid"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (Promo) TableName() string { return "promos" }

type PromoUsage struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	PromoID     uuid.UUID  `gorm:"type:uuid"`
	OrderID     uuid.UUID  `gorm:"type:uuid"`
	UserID      *uuid.UUID `gorm:"type:uuid"`
	ContactHash *string
	Discount    int64
	ReleasedAt  *time.Time
	CreatedAt   time.Time
}

func (PromoUsage) TableName() string { return "promo_usages" }
