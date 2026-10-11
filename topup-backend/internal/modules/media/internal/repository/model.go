// Package repository berisi model GORM dan implementasi repository modul media.
// Model sinkron dengan database.dbml § MODUL: media.
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Media struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	StorageKey     string
	URL            string
	OriginalName   *string
	MimeType       string
	SizeBytes      int32
	Width          *int32
	Height         *int32
	AltText        *string
	ChecksumSHA256 string     `gorm:"column:checksum_sha256"`
	UploadedBy     *uuid.UUID `gorm:"type:uuid"`
	CreatedAt      time.Time
	DeletedAt      gorm.DeletedAt
}

func (Media) TableName() string { return "media" }
