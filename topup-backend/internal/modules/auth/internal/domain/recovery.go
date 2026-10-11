package domain

import (
	"crypto/rand"
	"strings"
)

// Alfabet recovery code: tanpa karakter yang mudah tertukar (0/O, 1/I/L).
const recoveryAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// NewRecoveryCode membuat satu recovery code berformat XXXXX-XXXXX (~49 bit).
func NewRecoveryCode() (string, error) {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	var b strings.Builder
	for i, v := range buf {
		if i == 5 {
			b.WriteByte('-')
		}
		b.WriteByte(recoveryAlphabet[int(v)%len(recoveryAlphabet)])
	}
	return b.String(), nil
}

// NormalizeRecoveryCode menyeragamkan input pengguna (huruf besar, tanpa spasi/tanda hubung).
func NormalizeRecoveryCode(s string) string {
	s = strings.ToUpper(s)
	return strings.NewReplacer(" ", "", "-", "").Replace(s)
}

// IsTOTPCode menyatakan s berbentuk kode TOTP: tepat 6 digit.
func IsTOTPCode(s string) bool {
	if len(s) != 6 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
