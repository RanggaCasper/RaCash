package token

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/clock"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
)

const secret = "0123456789abcdef0123456789abcdef"

func TestSignParseRoundTrip(t *testing.T) {
	clk := clock.NewFixed(time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC))
	iss := NewIssuer(secret, clk)
	uid, sid := uuid.New(), uuid.New()

	raw, exp, err := iss.Sign(uid, sid, 7)
	if err != nil {
		t.Fatal(err)
	}
	if want := clk.Now().Add(15 * time.Minute); !exp.Equal(want) {
		t.Fatalf("exp = %v, want %v", exp, want)
	}
	c, err := iss.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if c.UserID != uid || c.SessionID != sid || c.PermVersion != 7 {
		t.Fatalf("claims salah: %+v", c)
	}
}

func TestParseExpired(t *testing.T) {
	clk := clock.NewFixed(time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC))
	iss := NewIssuer(secret, clk)
	raw, _, _ := iss.Sign(uuid.New(), uuid.New(), 1)
	clk.Advance(15*time.Minute + time.Second)
	if _, err := iss.Parse(raw); !errs.Is(err, errs.TokenExpired) {
		t.Fatalf("err = %v, want TOKEN_EXPIRED", err)
	}
}

func TestParseRejectsTamperedAndForeignTokens(t *testing.T) {
	clk := clock.NewFixed(time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC))
	iss := NewIssuer(secret, clk)
	raw, _, _ := iss.Sign(uuid.New(), uuid.New(), 1)

	other := NewIssuer("ffffffffffffffffffffffffffffffff", clk)
	foreign, _, _ := other.Sign(uuid.New(), uuid.New(), 1)

	// alg=none dengan claim valid.
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": uuid.NewString(), "sid": uuid.NewString(), "pv": 1,
		"iat": clk.Now().Unix(), "exp": clk.Now().Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)

	// HS512 dengan secret yang benar: algoritma di luar HS256 harus ditolak.
	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"sub": uuid.NewString(), "sid": uuid.NewString(), "pv": 1,
		"iat": clk.Now().Unix(), "exp": clk.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(secret))

	for name, tok := range map[string]string{
		"dirusak":      raw[:len(raw)-2] + "xx",
		"secret lain":  foreign,
		"alg none":     none,
		"alg HS512":    hs512,
		"bukan jwt":    "abc",
		"kosong":       "",
		"tanpa bagian": strings.Repeat(".", 2),
	} {
		if _, err := iss.Parse(tok); !errs.Is(err, errs.Unauthenticated) {
			t.Errorf("%s: err = %v, want UNAUTHENTICATED", name, err)
		}
	}
}
