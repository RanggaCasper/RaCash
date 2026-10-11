package httpx

import (
	"log/slog"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/crypto"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/logger"
)

const (
	HeaderRequestID = "X-Request-Id"
	localsRequestID = "request_id"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_\-]{8,64}$`)

// RequestIDMiddleware memakai X-Request-Id dari klien bila valid, atau membuat
// yang baru; selalu dikembalikan di header respons dan disisipkan ke logger context.
func RequestIDMiddleware(base *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rid := c.Get(HeaderRequestID)
		if !requestIDPattern.MatchString(rid) {
			rid, _ = crypto.RandomToken("req_", 12)
		}
		c.Locals(localsRequestID, rid)
		c.Set(HeaderRequestID, rid)
		ctx := logger.WithContext(c.UserContext(), base.With("request_id", rid))
		c.SetUserContext(ctx)
		return c.Next()
	}
}

// RequestID mengembalikan request id milik request ini.
func RequestID(c *fiber.Ctx) string {
	if v, ok := c.Locals(localsRequestID).(string); ok {
		return v
	}
	return ""
}

// AccessLog mencatat satu baris per request (tanpa body). Error dari handler
// langsung dirender lewat ErrorHandler di sini, sehingga status yang dicatat
// (dan dibaca middleware metrik di luarnya) sama dengan yang diterima klien.
func AccessLog() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		if err := c.Next(); err != nil {
			if herr := c.App().ErrorHandler(c, err); herr != nil {
				return herr
			}
		}
		status := c.Response().StatusCode()
		l := logger.FromContext(c.UserContext())
		attrs := []any{
			"method", c.Method(),
			"route", routePath(c),
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", c.IP(),
		}
		if status >= 500 {
			l.Error("http request", attrs...)
		} else {
			l.Info("http request", attrs...)
		}
		return nil
	}
}

// routePath mengembalikan pola route (mis. /api/v1/orders/:invoice_no) agar
// label metrik & log tidak meledak oleh ID. Request tanpa route cocok → "unmatched".
func routePath(c *fiber.Ctx) string {
	if c.Response().StatusCode() == fiber.StatusNotFound {
		if r := c.Route(); r == nil || r.Path == "/" || r.Path == "" || isMiddlewarePath(r.Path) {
			return "unmatched"
		}
	}
	if r := c.Route(); r != nil && r.Path != "" {
		return r.Path
	}
	return "unmatched"
}

// isMiddlewarePath: prefix grup yang hanya dipasangi middleware (app.Use).
func isMiddlewarePath(p string) bool {
	switch p {
	case PrefixPublic, PrefixAdmin, PrefixH2H, PrefixWebhook:
		return true
	}
	return false
}

// RoutePath diekspor untuk package metrik.
func RoutePath(c *fiber.Ctx) string { return routePath(c) }
