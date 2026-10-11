// Package repository berisi model GORM dan implementasi repository modul wallet.
// Model sinkron dengan database.dbml § MODUL: wallet.
package repository

import (
	"time"

	"github.com/google/uuid"
)

type Deposit struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID          uuid.UUID `gorm:"type:uuid"`
	Amount          int64
	Status          string    // deposit_status
	PaymentMethodID uuid.UUID `gorm:"type:uuid"`
	ExpiresAt       time.Time
	PaidAt          *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (Deposit) TableName() string { return "deposits" }

// BalanceLedger append-only: hanya INSERT (UPDATE/DELETE ditolak trigger DB).
type BalanceLedger struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID        uuid.UUID `gorm:"type:uuid"`
	Type          string    // ledger_type
	Amount        int64     // positif = masuk, negatif = keluar
	BalanceBefore int64
	BalanceAfter  int64
	ReferenceType string    // order, deposit, adjustment
	ReferenceID   uuid.UUID `gorm:"type:uuid"`
	Description   string
	ActorID       *uuid.UUID `gorm:"type:uuid"`
	Reason        *string
	CreatedAt     time.Time
}

func (BalanceLedger) TableName() string { return "balance_ledger" }
