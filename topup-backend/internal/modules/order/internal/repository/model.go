// Package repository berisi model GORM dan implementasi repository modul order.
// Model sinkron dengan database.dbml § MODUL: order.
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Order struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey"`
	InvoiceNo             string
	Source                string     // order_source
	UserID                *uuid.UUID `gorm:"type:uuid"` // null = guest
	ResellerRefID         *string
	ProductID             uuid.UUID `gorm:"type:uuid"`
	ProductSnapshot       datatypes.JSON
	TargetData            datatypes.JSON
	TargetDisplay         string
	ContactEmail          *string
	ContactWhatsapp       *string
	TierID                *uuid.UUID `gorm:"type:uuid"`
	PaymentMethodID       uuid.UUID  `gorm:"type:uuid"`
	PromoID               *uuid.UUID `gorm:"type:uuid"`
	Price                 int64
	EstimatedCost         int64
	ActualCost            *int64
	Discount              int64
	Fee                   int64
	Total                 int64
	Status                string // order_status
	SerialNumberEncrypted []byte
	SerialNumberKeyID     *string
	DeliveryMessage       *string
	FailureReason         *string
	RefundDestination     *string // refund_destination
	Version               int32   // optimistic locking
	ExpiresAt             time.Time
	PaidAt                *time.Time
	CompletedAt           *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (Order) TableName() string { return "orders" }

type OrderStatusHistory struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrderID    uuid.UUID `gorm:"type:uuid"`
	FromStatus *string   // order_status
	ToStatus   string
	ActorType  string     // system, user, admin, provider, gateway
	ActorID    *uuid.UUID `gorm:"type:uuid"`
	Reason     *string
	CreatedAt  time.Time
}

func (OrderStatusHistory) TableName() string { return "order_status_histories" }
