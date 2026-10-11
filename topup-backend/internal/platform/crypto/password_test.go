package crypto

import (
	"strings"
	"testing"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	h, err := HashPassword("S3cure!pass")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$") {
		t.Fatalf("format hash: %s", h)
	}
	if ok, err := VerifyPassword("S3cure!pass", h); err != nil || !ok {
		t.Fatalf("password benar ditolak: ok=%v err=%v", ok, err)
	}
	if ok, err := VerifyPassword("salah123", h); err != nil || ok {
		t.Fatalf("password salah diterima: ok=%v err=%v", ok, err)
	}
	h2, _ := HashPassword("S3cure!pass")
	if h == h2 {
		t.Fatal("salt harus acak: dua hash identik")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	for _, h := range []string{"", "plain", "$bcrypt$x$y$z$w", "$argon2id$v=19$m=1,t=1,p=1$!!$!!"} {
		if ok, err := VerifyPassword("x", h); ok || err == nil {
			t.Errorf("hash %q: ok=%v err=%v, harus error", h, ok, err)
		}
	}
}
