// Package repository berisi model GORM dan implementasi repository modul fulfillment.
// Model sinkron dengan database.dbml § MODUL: fulfillment.
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type OrderAttempt struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrderID         uuid.UUID `gorm:"type:uuid"`
	AttemptNo       int16
	ProviderID      uuid.UUID `gorm:"type:uuid"`
	ProviderSKU     string    `gorm:"column:provider_sku"`
	RefID           string    // invoice_no-attempt_no
	ProviderTrxID   *string
	CostPrice       int64
	Status          string // attempt_status
	ErrorCode       *string
	VendorErrorCode *string
	VendorMessage   *string
	IsRetryable     *bool
	RequestPayload  datatypes.JSON // sudah dimasking
	ResponsePayload datatypes.JSON // sudah dimasking
	TriggeredBy     *uuid.UUID     `gorm:"type:uuid"`
	LatencyMs       *int32
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (OrderAttempt) TableName() string { return "order_attempts" }
