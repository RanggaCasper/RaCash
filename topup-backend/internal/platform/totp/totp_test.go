package totp

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

// Vektor uji RFC 6238 Appendix B (SHA-1, secret "12345678901234567890").
// RFC memberi 8 digit; 6 digit terakhirnya adalah kode yang kita pakai.
func TestCodeRFC6238Vectors(t *testing.T) {
	secret := []byte("12345678901234567890")
	cases := []struct {
		unix int64
		want string
	}{
		{59, "287082"},
		{1111111109, "081804"},
		{1111111111, "050471"},
		{1234567890, "005924"},
		{2000000000, "279037"},
		{20000000000, "353130"},
	}
	for _, c := range cases {
		if got := Code(secret, Step(time.Unix(c.unix, 0))); got != c.want {
			t.Errorf("t=%d: %s, want %s", c.unix, got, c.want)
		}
	}
}

func TestMatchWindowAndStep(t *testing.T) {
	secret := []byte("12345678901234567890")
	now := time.Unix(1111111109, 0)
	cur := Step(now)

	if step, ok := Match(secret, Code(secret, cur), now, 1); !ok || step != cur {
		t.Fatalf("kode saat ini: step=%d ok=%v", step, ok)
	}
	if step, ok := Match(secret, Code(secret, cur-1), now, 1); !ok || step != cur-1 {
		t.Fatalf("kode langkah sebelumnya: step=%d ok=%v", step, ok)
	}
	if step, ok := Match(secret, Code(secret, cur+1), now, 1); !ok || step != cur+1 {
		t.Fatalf("kode langkah berikutnya: step=%d ok=%v", step, ok)
	}
	if _, ok := Match(secret, Code(secret, cur-2), now, 1); ok {
		t.Fatal("kode di luar jendela harus ditolak")
	}
	for _, bad := range []string{"", "12345", "1234567", "abcdef", "000000"} {
		if bad == Code(secret, cur) || bad == Code(secret, cur-1) || bad == Code(secret, cur+1) {
			continue
		}
		if _, ok := Match(secret, bad, now, 1); ok {
			t.Errorf("kode %q seharusnya ditolak", bad)
		}
	}
}

func TestGenerateSecretAndURL(t *testing.T) {
	a, err := GenerateSecret()
	if err != nil || len(a) != SecretSize {
		t.Fatalf("secret: len=%d err=%v", len(a), err)
	}
	b, _ := GenerateSecret()
	if string(a) == string(b) {
		t.Fatal("secret harus acak")
	}

	raw := URL("RaCash", "user+tag@example.com", a)
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "otpauth" || u.Host != "totp" {
		t.Fatalf("url: %s", raw)
	}
	if got := u.Query().Get("secret"); got != EncodeSecret(a) || strings.ContainsAny(got, "=") {
		t.Fatalf("secret di url: %q", got)
	}
	if u.Query().Get("issuer") != "RaCash" || u.Query().Get("digits") != "6" || u.Query().Get("period") != "30" {
		t.Fatalf("parameter url: %s", u.RawQuery)
	}
	if !strings.HasPrefix(u.Path, "/RaCash:") {
		t.Fatalf("label: %s", u.Path)
	}
}
