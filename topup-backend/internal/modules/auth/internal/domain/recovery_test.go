package domain

import (
	"regexp"
	"testing"
)

func TestRecoveryCodes(t *testing.T) {
	re := regexp.MustCompile(`^[A-HJKMN-Z2-9]{5}-[A-HJKMN-Z2-9]{5}$`)
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		c, err := NewRecoveryCode()
		if err != nil || !re.MatchString(c) {
			t.Fatalf("format salah: %q err=%v", c, err)
		}
		if seen[c] {
			t.Fatalf("recovery code duplikat: %s", c)
		}
		seen[c] = true
	}
	if got := NormalizeRecoveryCode(" ab3de-fgh4j "); got != "AB3DEFGH4J" {
		t.Fatalf("normalize: %q", got)
	}
}

func TestIsTOTPCode(t *testing.T) {
	for in, want := range map[string]bool{"123456": true, "000000": true, "12345": false, "1234567": false, "12345a": false, "": false, "AB3DE-FGH4J": false} {
		if IsTOTPCode(in) != want {
			t.Errorf("IsTOTPCode(%q) = %v", in, !want)
		}
	}
}
