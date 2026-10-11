// Package repository berisi model GORM dan implementasi repository modul rbac.
// Model sinkron dengan database.dbml § MODUL: rbac.
package repository

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code        string
	Name        string
	Description *string
	Scope       string // role_scope
	IsSystem    bool
	CreatedBy   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Role) TableName() string { return "roles" }

type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code        string
	Resource    string
	Action      string
	Description string
	IsSensitive bool
	CreatedAt   time.Time
}

func (Permission) TableName() string { return "permissions" }

type RolePermission struct {
	RoleID       uuid.UUID  `gorm:"type:uuid;primaryKey"`
	PermissionID uuid.UUID  `gorm:"type:uuid;primaryKey"`
	GrantedBy    *uuid.UUID `gorm:"type:uuid"`
	GrantedAt    time.Time
}

func (RolePermission) TableName() string { return "role_permissions" }

type UserRole struct {
	UserID     uuid.UUID  `gorm:"type:uuid;primaryKey"`
	RoleID     uuid.UUID  `gorm:"type:uuid;primaryKey"`
	AssignedBy *uuid.UUID `gorm:"type:uuid"`
	AssignedAt time.Time
	ExpiresAt  *time.Time
}

func (UserRole) TableName() string { return "user_roles" }
