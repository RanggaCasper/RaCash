// Package repository berisi model GORM dan implementasi repository modul setting.
// Model sinkron dengan database.dbml § MODUL: setting.
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Setting struct {
	Key         string `gorm:"primaryKey"`
	Value       datatypes.JSON
	IsSecret    bool // nilai disamarkan di respons API
	Description *string
	UpdatedBy   *uuid.UUID `gorm:"type:uuid"`
	UpdatedAt   time.Time
}

func (Setting) TableName() string { return "settings" }
