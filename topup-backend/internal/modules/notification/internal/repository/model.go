// Package repository berisi model GORM dan implementasi repository modul notification.
// Model sinkron dengan database.dbml § MODUL: notification.
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type NotificationLog struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	Channel           string    // notification_channel
	Recipient         string
	Template          string
	Subject           *string
	Payload           datatypes.JSON
	RelatedType       *string    // order, deposit, user, system
	RelatedID         *uuid.UUID `gorm:"type:uuid"`
	Status            string     // delivery_status
	Attempts          int16
	ProviderMessageID *string
	LastError         *string
	SentAt            *time.Time
	CreatedAt         time.Time
}

func (NotificationLog) TableName() string { return "notification_logs" }
