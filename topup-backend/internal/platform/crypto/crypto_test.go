package crypto

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func key(b byte) string { return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{b}, 32)) }

func TestKeyringRotation(t *testing.T) {
	old, err := ParseKeyring("k1:"+key(1), "k1")
	if err != nil {
		t.Fatal(err)
	}
	ct, kid, err := old.Encrypt([]byte("rahasia"))
	if err != nil || kid != "k1" {
		t.Fatalf("encrypt: %v kid=%s", err, kid)
	}

	rotated, err := ParseKeyring("k1:"+key(1)+",k2:"+key(2), "k2")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := rotated.Decrypt(ct, kid)
	if err != nil || string(plain) != "rahasia" {
		t.Fatalf("decrypt data lama: %q %v", plain, err)
	}
	if _, kid2, _ := rotated.Encrypt([]byte("x")); kid2 != "k2" {
		t.Fatalf("enkripsi baru harus memakai kunci aktif, got %s", kid2)
	}
	if _, err := rotated.Decrypt(ct, "k9"); err != ErrUnknownKey {
		t.Fatalf("want ErrUnknownKey, got %v", err)
	}
}

func TestParseKeyringInvalid(t *testing.T) {
	for _, spec := range []string{"", "k1", "k1:short", "k1:" + key(1)} {
		if _, err := ParseKeyring(spec, "k2"); err == nil {
			t.Errorf("spec %q harus ditolak", spec)
		}
	}
}

func TestHMAC(t *testing.T) {
	sig := HMACSHA256Hex([]byte("secret"), []byte("msg"))
	if !VerifyHMACSHA256Hex([]byte("secret"), []byte("msg"), sig) {
		t.Fatal("signature valid ditolak")
	}
	if VerifyHMACSHA256Hex([]byte("secret"), []byte("msg2"), sig) || VerifyHMACSHA256Hex([]byte("secret"), []byte("msg"), "zz") {
		t.Fatal("signature tidak valid diterima")
	}
}

func TestRandomToken(t *testing.T) {
	a, _ := RandomToken("rt_", 32)
	b, _ := RandomToken("rt_", 32)
	if a == b || !strings.HasPrefix(a, "rt_") {
		t.Fatalf("token tidak acak/prefix salah: %s %s", a, b)
	}
}
