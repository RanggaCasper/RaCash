// Package repository berisi model GORM dan implementasi repository modul pricing.
// Model sinkron dengan database.dbml § MODUL: pricing.
package repository

import (
	"time"

	"github.com/google/uuid"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/money"
)

type PricingRule struct {
	ID          uuid.UUID     `gorm:"type:uuid;primaryKey"`
	Scope       string        // pricing_scope
	ScopeID     *uuid.UUID    `gorm:"type:uuid"` // null bila scope = global (polimorfik, tanpa FK)
	TierID      *uuid.UUID    `gorm:"type:uuid"` // null = semua tier
	MarginType  string        // amount_type
	MarginValue money.Percent // numeric(12,3), skala ×1000: rupiah bila FLAT, persen bila PERCENT
	MinMargin   int64
	IsActive    bool
	CreatedBy   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (PricingRule) TableName() string { return "pricing_rules" }
