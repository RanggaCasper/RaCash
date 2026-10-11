// Package repository berisi model GORM dan implementasi repository modul provider.
// Model sinkron dengan database.dbml § MODUL: provider.
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Provider struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code                 string    // sama dengan Code() adapter
	Name                 string
	Mode                 string // vendor_mode
	CredentialsEncrypted []byte
	CredentialsKeyID     *string
	IsActive             bool
	Priority             int32
	Balance              *int64
	BalanceMinAlert      int64
	BalanceCheckedAt     *time.Time
	CircuitState         string // circuit_state (snapshot; realtime di Redis)
	CircuitOpenedAt      *time.Time
	LastPriceSyncAt      *time.Time
	Config               datatypes.JSON
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (Provider) TableName() string { return "providers" }

type ProductRouting struct {
	ProductID   uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Strategy    string     // routing_strategy
	MaxAttempts int16      // jumlah provider BERBEDA yang dicoba
	UpdatedBy   *uuid.UUID `gorm:"type:uuid"`
	UpdatedAt   time.Time
}

func (ProductRouting) TableName() string { return "product_routing" }

type ProductProviderMap struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	ProductID    uuid.UUID `gorm:"type:uuid"`
	ProviderID   uuid.UUID `gorm:"type:uuid"`
	ProviderSKU  string    `gorm:"column:provider_sku"`
	CostPrice    *int64
	IsAvailable  bool
	IsActive     bool
	Weight       *int32
	LastSyncedAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (ProductProviderMap) TableName() string { return "product_provider_map" }

type ProviderPriceHistory struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey"`
	ProductProviderMapID uuid.UUID `gorm:"type:uuid"`
	OldCostPrice         *int64
	NewCostPrice         int64
	ChangedAt            time.Time
}

func (ProviderPriceHistory) TableName() string { return "provider_price_histories" }
