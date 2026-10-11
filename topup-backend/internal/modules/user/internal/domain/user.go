package domain

import (
	"time"

	"github.com/google/uuid"
)

// Status akun (enum user_status).
const (
	StatusPendingVerification = "PENDING_VERIFICATION"
	StatusActive              = "ACTIVE"
	StatusSuspended           = "SUSPENDED"
)

// User adalah profil user tanpa kredensial.
type User struct {
	ID               uuid.UUID
	Name             string
	Email            string
	Phone            *string
	TierCode         string
	Status           string
	PermVersion      int32
	TwoFactorEnabled bool
	EmailVerifiedAt  *time.Time
	CreatedAt        time.Time
}

// IsVerified menyatakan email sudah diverifikasi.
func (u User) IsVerified() bool { return u.EmailVerifiedAt != nil }

// Credentials adalah user beserta hash password; hanya untuk modul auth.
type Credentials struct {
	User
	PasswordHash string
}

// NewUser adalah masukan pembuatan user baru. Password sudah di-hash oleh pemanggil.
type NewUser struct {
	Name         string
	Email        string
	Phone        *string
	PasswordHash string
}
