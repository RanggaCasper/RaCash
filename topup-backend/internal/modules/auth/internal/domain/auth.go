// Package domain berisi aturan bisnis murni modul auth: kebijakan password,
// masa berlaku token, dan entity sesi.
package domain

import (
	"time"
	"unicode"

	"github.com/google/uuid"
)

// Masa berlaku (API_CONTRACT §1.7).
const (
	AccessTokenTTL       = 15 * time.Minute
	RefreshTokenTTL      = 30 * 24 * time.Hour
	EmailVerificationTTL = 24 * time.Hour
	PasswordResetTTL     = time.Hour
)

// Awalan token opaque supaya mudah dikenali (dan dideteksi secret scanner).
const (
	RefreshTokenPrefix     = "rt_"
	EmailVerifyTokenPrefix = "ev_"
	PasswordResetPrefix    = "pr_"
)

// Alasan sesi dicabut (sessions.revoked_reason).
const (
	RevokedLogout      = "LOGOUT"
	RevokedLogoutAll   = "LOGOUT_ALL"
	RevokedUser        = "USER_REVOKED"
	RevokedReuse       = "REUSE_DETECTED"
	RevokedPasswordSet = "PASSWORD_RESET"
)

// Batas panjang password: argon2id tidak punya batas, tetapi input tak
// terbatas membuka celah DoS.
const (
	PasswordMinLen = 8
	PasswordMaxLen = 128
)

// ValidPassword: minimal 8 karakter, mengandung huruf dan angka (API_CONTRACT §4.1).
func ValidPassword(p string) bool {
	if n := len([]rune(p)); n < PasswordMinLen || n > PasswordMaxLen {
		return false
	}
	var letter, digit bool
	for _, r := range p {
		switch {
		case unicode.IsLetter(r):
			letter = true
		case unicode.IsDigit(r):
			digit = true
		}
	}
	return letter && digit
}

// Session adalah sesi login (satu refresh token aktif).
type Session struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	UserAgent  *string
	IP         *string
	ExpiresAt  time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

// Active menyatakan sesi belum dicabut dan belum kedaluwarsa pada waktu now.
func (s Session) Active(now time.Time) bool {
	return s.RevokedAt == nil && now.Before(s.ExpiresAt)
}

// Claims adalah isi access token (API_CONTRACT §1.7).
type Claims struct {
	UserID      uuid.UUID
	SessionID   uuid.UUID
	PermVersion int32
	IssuedAt    time.Time
	ExpiresAt   time.Time
}

// Principal adalah identitas hasil autentikasi sebuah request.
type Principal struct {
	UserID      uuid.UUID
	SessionID   uuid.UUID
	PermVersion int32
}

// SessionMeta adalah metadata klien saat sesi dibuat.
type SessionMeta struct {
	UserAgent string
	IP        string
}

// Tokens adalah pasangan token hasil login atau refresh.
type Tokens struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// AuthUser adalah ringkasan user di respons login.
type AuthUser struct {
	ID    uuid.UUID
	Name  string
	Email string
	Tier  string
	Roles []string
}

// LoginResult adalah hasil login atau refresh yang sukses.
type LoginResult struct {
	Tokens
	User AuthUser
}

// SessionView adalah sesi aktif untuk daftar sesi user.
type SessionView struct {
	Session
	IsCurrent bool
}

// Event yang diterbitkan modul auth (ARCHITECTURE §2.5). Payload memuat token
// mentah karena penerima (notification) yang mengirimkannya lewat email.
const (
	EventModule                = "auth"
	EventEmailVerificationReq  = "email_verification.requested"
	EventPasswordResetReq      = "password_reset.requested"
	AggregateEmailVerification = "email_verification"
	AggregatePasswordReset     = "password_reset"
)

// TokenEventPayload adalah payload kedua event di atas.
type TokenEventPayload struct {
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
