// Package repository berisi model GORM dan implementasi repository modul report.
// Model sinkron dengan database.dbml § MODUL: report.
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ReconciliationRun struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey"`
	TargetType    string     // reconciliation_target
	ProviderID    *uuid.UUID `gorm:"type:uuid"`
	GatewayID     *uuid.UUID `gorm:"type:uuid"`
	PeriodDate    time.Time  `gorm:"type:date"`
	Status        string     // reconciliation_status
	TotalInternal *int32
	TotalExternal *int32
	MismatchCount *int32
	TriggeredBy   *uuid.UUID `gorm:"type:uuid"` // null = terjadwal
	StartedAt     time.Time
	FinishedAt    *time.Time
	Error         *string
}

func (ReconciliationRun) TableName() string { return "reconciliation_runs" }

type ReconciliationItem struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	RunID          uuid.UUID `gorm:"type:uuid"`
	Type           string    // reconciliation_item_type
	Reference      string
	OrderID        *uuid.UUID `gorm:"type:uuid"`
	InternalData   datatypes.JSON
	ExternalData   datatypes.JSON
	ResolvedAt     *time.Time
	ResolvedBy     *uuid.UUID `gorm:"type:uuid"`
	ResolutionNote *string
}

func (ReconciliationItem) TableName() string { return "reconciliation_items" }
