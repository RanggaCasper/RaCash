// Package repository berisi model GORM dan implementasi repository modul catalog.
// Model sinkron dengan database.dbml § MODUL: catalog.
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Category struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Slug        string
	Name        string
	IconMediaID *uuid.UUID `gorm:"type:uuid"`
	SortOrder   int32
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt
}

func (Category) TableName() string { return "categories" }

type Brand struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey"`
	CategoryID           uuid.UUID `gorm:"type:uuid"`
	Slug                 string
	Name                 string
	IconMediaID          *uuid.UUID `gorm:"type:uuid"`
	BannerMediaID        *uuid.UUID `gorm:"type:uuid"`
	Description          *string
	InputSchema          datatypes.JSON // [{key,label,type,required,pattern,options}]
	SupportsAccountCheck bool
	IsPopular            bool
	SortOrder            int32
	IsActive             bool
	SEOTitle             *string `gorm:"column:seo_title"`
	SEODescription       *string `gorm:"column:seo_description"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            gorm.DeletedAt
}

func (Brand) TableName() string { return "brands" }

type Product struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	BrandID       uuid.UUID `gorm:"type:uuid"`
	SKU           string    `gorm:"column:sku"`
	Name          string
	Description   *string
	Badge         *string
	FixedPrice    *int64
	OriginalPrice *int64
	SortOrder     int32
	IsActive      bool
	IsAvailable   bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt
}

func (Product) TableName() string { return "products" }

type Banner struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Title        string
	ImageMediaID uuid.UUID `gorm:"type:uuid"`
	LinkURL      *string
	SortOrder    int32
	StartsAt     *time.Time
	EndsAt       *time.Time
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (Banner) TableName() string { return "banners" }
