// Package captcha memverifikasi token captcha (Cloudflare Turnstile) untuk
// endpoint publik yang rawan spam (API_CONTRACT §1.5, header X-Captcha-Token).
package captcha

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/logger"
)

const (
	HeaderToken = "X-Captcha-Token"
	// TurnstileURL adalah endpoint siteverify Cloudflare Turnstile.
	TurnstileURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
)

// ErrRejected berarti vendor menolak token (salah, kedaluwarsa, atau dipakai ulang).
var ErrRejected = errors.New("captcha: token ditolak")

// Verifier memeriksa satu token captcha. Error selain ErrRejected berarti
// vendor tidak dapat dihubungi.
type Verifier interface {
	Verify(ctx context.Context, token, ip string) error
}

// Disabled menerima semua request; hanya untuk APP_ENV=local.
type Disabled struct{}

func (Disabled) Verify(context.Context, string, string) error { return nil }

// Turnstile memverifikasi token ke Cloudflare.
type Turnstile struct {
	secret string
	url    string
	client *http.Client
}

func NewTurnstile(secret, verifyURL string) *Turnstile {
	if verifyURL == "" {
		verifyURL = TurnstileURL
	}
	return &Turnstile{secret: secret, url: verifyURL, client: &http.Client{Timeout: 5 * time.Second}}
}

func (t *Turnstile) Verify(ctx context.Context, token, ip string) error {
	form := url.Values{"secret": {t.secret}, "response": {token}}
	if ip != "" {
		form.Set("remoteip", ip)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("captcha: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("captcha: vendor membalas status %d", resp.StatusCode)
	}
	var out struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("captcha: respons tidak valid: %w", err)
	}
	if !out.Success {
		return ErrRejected
	}
	return nil
}

// Middleware menolak request tanpa token captcha yang valid. Token wajib ada
// (422 VALIDATION_ERROR); vendor yang tidak bisa dihubungi menghasilkan 503
// (fail closed) agar captcha tidak bisa dilewati dengan memutus koneksi.
func Middleware(v Verifier) fiber.Handler {
	_, off := v.(Disabled)
	return func(c *fiber.Ctx) error {
		if off {
			return c.Next()
		}
		token := strings.TrimSpace(c.Get(HeaderToken))
		if token == "" {
			return errs.Validation(errs.Detail{Field: HeaderToken, Code: "captcha_required", Message: "Captcha wajib diisi."})
		}
		switch err := v.Verify(c.UserContext(), token, c.IP()); {
		case err == nil:
			return c.Next()
		case errors.Is(err, ErrRejected):
			return errs.Validation(errs.Detail{Field: HeaderToken, Code: "captcha_invalid", Message: "Captcha tidak valid atau kedaluwarsa."})
		default:
			logger.FromContext(c.UserContext()).Error("verifikasi captcha gagal", "err", err)
			return errs.Wrap(errs.ServiceUnavailable, err, "")
		}
	}
}
