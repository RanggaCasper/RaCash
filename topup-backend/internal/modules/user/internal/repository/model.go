// Package repository berisi model GORM dan implementasi repository modul user.
// Model sinkron dengan database.dbml § MODUL: user dan tidak boleh keluar dari
// package ini (RULES §3.2). Default kolom di DB tidak dipakai GORM: isi nilai
// secara eksplisit saat Create.
package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/database"
)

type Tier struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code                 string
	Name                 string
	Rank                 int16
	IsDefault            bool
	AutoUpgradeMinVolume *int64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (Tier) TableName() string { return "tiers" }

type User struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name             string
	Email            string
	Phone            *string
	PasswordHash     string
	TierID           uuid.UUID `gorm:"type:uuid"`
	Status           string    // user_status
	Balance          int64
	PermVersion      int32
	TwoFactorEnabled bool
	EmailVerifiedAt  *time.Time
	LastLoginAt      *time.Time
	SuspendedReason  *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt
}

func (User) TableName() string { return "users" }

type StaffInvitation struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email          string
	RoleIDs        database.StringArray `gorm:"type:uuid[]"`
	TokenHash      string
	InvitedBy      uuid.UUID  `gorm:"type:uuid"`
	AcceptedUserID *uuid.UUID `gorm:"type:uuid"`
	ExpiresAt      time.Time
	AcceptedAt     *time.Time
	RevokedAt      *time.Time
	CreatedAt      time.Time
}

func (StaffInvitation) TableName() string { return "staff_invitations" }
