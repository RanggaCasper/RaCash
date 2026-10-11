// Package totp mengimplementasikan TOTP (RFC 6238) dengan HMAC-SHA1, 6 digit,
// periode 30 detik: parameter yang didukung semua aplikasi autentikator.
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // SHA-1 adalah algoritma default RFC 6238 yang dipakai aplikasi autentikator.
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"time"
)

const (
	Period     = 30 * time.Second
	Digits     = 6
	SecretSize = 20 // 160 bit, sesuai rekomendasi RFC 4226
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// GenerateSecret membuat secret acak.
func GenerateSecret() ([]byte, error) {
	s := make([]byte, SecretSize)
	if _, err := rand.Read(s); err != nil {
		return nil, err
	}
	return s, nil
}

// EncodeSecret mengubah secret ke base32 tanpa padding (format input manual di aplikasi autentikator).
func EncodeSecret(secret []byte) string { return b32.EncodeToString(secret) }

// Step mengembalikan nomor langkah waktu untuk t.
func Step(t time.Time) int64 { return t.Unix() / int64(Period/time.Second) }

// Code menghitung kode 6 digit untuk satu langkah waktu (RFC 4226 §5.3).
func Code(secret []byte, step int64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step))
	mac := hmac.New(sha1.New, secret)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	bin := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", Digits, bin%1_000_000)
}

// Match memeriksa code terhadap langkah waktu di sekitar now (±skew langkah, untuk
// selisih jam). Mengembalikan langkah yang cocok supaya pemanggil bisa menolak
// pemakaian ulang kode yang sama (replay).
func Match(secret []byte, code string, now time.Time, skew int) (int64, bool) {
	if len(code) != Digits {
		return 0, false
	}
	cur := Step(now)
	var matched int64
	ok := false
	// Semua langkah diperiksa tanpa berhenti lebih awal agar waktu respons tetap.
	for d := -skew; d <= skew; d++ {
		step := cur + int64(d)
		if subtle.ConstantTimeCompare([]byte(Code(secret, step)), []byte(code)) == 1 {
			matched, ok = step, true
		}
	}
	return matched, ok
}

// URL membuat otpauth:// URI untuk QR code / tautan aplikasi autentikator.
func URL(issuer, account string, secret []byte) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{
		"secret":    {EncodeSecret(secret)},
		"issuer":    {issuer},
		"algorithm": {"SHA1"},
		"digits":    {fmt.Sprint(Digits)},
		"period":    {fmt.Sprint(int(Period / time.Second))},
	}
	return "otpauth://totp/" + label + "?" + q.Encode()
}
