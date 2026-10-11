// Package repository berisi model GORM dan implementasi repository modul payment.
// Model sinkron dengan database.dbml § MODUL: payment.
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/money"
)

type PaymentGateway struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code                 string    // sama dengan Code() adapter
	Name                 string
	Mode                 string // vendor_mode
	CredentialsEncrypted []byte
	CredentialsKeyID     *string
	IsActive             bool
	CircuitState         string // circuit_state
	Config               datatypes.JSON
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (PaymentGateway) TableName() string { return "payment_gateways" }

type PaymentMethod struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code             string
	Name             string
	Group            string     // payment_method_group
	IconMediaID      *uuid.UUID `gorm:"type:uuid"`
	FeeFlat          int64
	FeePercent       money.Percent
	FeeBearer        string         // fee_bearer
	FeeCustomerShare *money.Percent // wajib bila fee_bearer = SPLIT
	MinAmount        int64
	MaxAmount        *int64
	ExpiryMinutes    int32 // sumber tunggal masa berlaku invoice/deposit
	Instructions     datatypes.JSON
	SortOrder        int32
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (PaymentMethod) TableName() string { return "payment_methods" }

type PaymentMethodGatewayMap struct {
	PaymentMethodID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	GatewayID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	GatewayMethodCode string
	Priority          int32
	IsActive          bool
}

func (PaymentMethodGatewayMap) TableName() string { return "payment_method_gateway_map" }

type Payment struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Purpose         string     // payment_purpose
	OrderID         *uuid.UUID `gorm:"type:uuid"`
	DepositID       *uuid.UUID `gorm:"type:uuid"`
	PaymentMethodID uuid.UUID  `gorm:"type:uuid"`
	GatewayID       *uuid.UUID `gorm:"type:uuid"` // null untuk metode BALANCE
	ExternalID      *string
	Amount          int64
	Fee             int64
	Status          string // payment_status
	Instructions    datatypes.JSON
	ExpiresAt       *time.Time
	PaidAt          *time.Time
	RefundedAt      *time.Time
	RawResponse     datatypes.JSON // sudah dimasking
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (Payment) TableName() string { return "payments" }
