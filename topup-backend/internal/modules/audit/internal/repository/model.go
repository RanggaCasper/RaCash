// Package repository berisi model GORM dan implementasi repository modul audit.
// Model sinkron dengan database.dbml § MODUL: audit.
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/database"
)

// AuditLog append-only: hanya INSERT (UPDATE/DELETE ditolak trigger DB).
type AuditLog struct {
	ID         uuid.UUID            `gorm:"type:uuid;primaryKey"`
	ActorID    *uuid.UUID           `gorm:"type:uuid"` // null = system
	ActorRoles database.StringArray `gorm:"type:varchar(64)[]"`
	Action     string
	Entity     string
	EntityID   *string
	Reason     *string
	Before     datatypes.JSON
	After      datatypes.JSON
	IP         *string `gorm:"type:inet"`
	UserAgent  *string
	RequestID  *string
	CreatedAt  time.Time
}

func (AuditLog) TableName() string { return "audit_logs" }
