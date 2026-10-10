// Package crypto: AES-GCM dengan rotasi kunci (key_id), HMAC-SHA256, hashing,
// dan pembuatan token acak.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnknownKey  = errors.New("crypto: key_id tidak dikenal")
	ErrCiphertext  = errors.New("crypto: ciphertext tidak valid")
	ErrKeyringSpec = errors.New("crypto: format ENCRYPTION_KEYS tidak valid")
)

// Keyring menyimpan beberapa kunci AES-256 agar data lama tetap bisa dibuka
// setelah rotasi. Enkripsi selalu memakai kunci aktif.
type Keyring struct {
	keys   map[string]cipher.AEAD
	active string
}

// ParseKeyring membaca format "k1:<base64 32 byte>,k2:<base64 32 byte>".
func ParseKeyring(spec, activeID string) (*Keyring, error) {
	k := &Keyring{keys: map[string]cipher.AEAD{}, active: activeID}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kid, b64, ok := strings.Cut(part, ":")
		if !ok || kid == "" {
			return nil, ErrKeyringSpec
		}
		raw, err := base64.StdEncoding.DecodeString(b64)
		if err != nil || len(raw) != 32 {
			return nil, fmt.Errorf("%w: kunci %q harus 32 byte base64", ErrKeyringSpec, kid)
		}
		block, err := aes.NewCipher(raw)
		if err != nil {
			return nil, err
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}
		k.keys[kid] = aead
	}
	if _, ok := k.keys[activeID]; !ok {
		return nil, fmt.Errorf("%w: kunci aktif %q tidak ada", ErrKeyringSpec, activeID)
	}
	return k, nil
}

// Encrypt mengembalikan nonce||ciphertext dan key_id yang dipakai.
func (k *Keyring) Encrypt(plain []byte) ([]byte, string, error) {
	aead := k.keys[k.active]
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, "", err
	}
	return aead.Seal(nonce, nonce, plain, nil), k.active, nil
}

// Decrypt membuka data yang dienkripsi dengan key_id tertentu.
func (k *Keyring) Decrypt(data []byte, keyID string) ([]byte, error) {
	aead, ok := k.keys[keyID]
	if !ok {
		return nil, ErrUnknownKey
	}
	if len(data) < aead.NonceSize() {
		return nil, ErrCiphertext
	}
	nonce, ct := data[:aead.NonceSize()], data[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, ErrCiphertext
	}
	return plain, nil
}

// ActiveKeyID mengembalikan id kunci yang dipakai untuk enkripsi baru.
func (k *Keyring) ActiveKeyID() string { return k.active }

// HMACSHA256Hex menghitung hex(HMAC_SHA256(secret, msg)).
func HMACSHA256Hex(secret, msg []byte) string {
	m := hmac.New(sha256.New, secret)
	m.Write(msg)
	return hex.EncodeToString(m.Sum(nil))
}

// VerifyHMACSHA256Hex membandingkan signature secara constant-time.
func VerifyHMACSHA256Hex(secret, msg []byte, sigHex string) bool {
	want, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	m := hmac.New(sha256.New, secret)
	m.Write(msg)
	return hmac.Equal(m.Sum(nil), want)
}

// SHA256Hex mengembalikan hex(SHA256(b)).
func SHA256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// RandomToken membuat token acak base64url dengan prefix, mis. "rt_...".
func RandomToken(prefix string, nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}
