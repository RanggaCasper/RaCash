package app

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/captcha"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/httpx"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/logger"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/ratelimit"
)

// Batas default (API_CONTRACT §1.8).
const (
	publicRateLimit  = 120
	publicRateWindow = time.Minute
)

// HTTP membangun server Fiber: middleware global, health check, lalu route modul.
func (a *App) HTTP() *fiber.App {
	p := a.Platform
	srv := fiber.New(fiber.Config{
		AppName:               "racash-api",
		ErrorHandler:          httpx.ErrorHandler,
		BodyLimit:             p.Config.BodyLimitBytes,
		ReadTimeout:           15 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           60 * time.Second,
		DisableStartupMessage: true,
	})

	// Urutan penting: AccessLog merender error (termasuk panic yang ditangkap
	// recover di dalamnya) sehingga Metrics di luarnya membaca status final.
	srv.Use(
		httpx.RequestIDMiddleware(p.Log),
		p.Metrics.Middleware(),
		httpx.AccessLog(),
		recover.New(recover.Config{
			EnableStackTrace: true,
			StackTraceHandler: func(c *fiber.Ctx, e any) {
				logger.FromContext(c.UserContext()).Error("panic", "panic", e)
			},
		}),
	)

	srv.Get("/healthz", func(c *fiber.Ctx) error {
		return httpx.OK(c, fiber.Map{"status": "ok"})
	})
	srv.Get("/readyz", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := p.Ready(ctx); err != nil {
			return errs.Wrap(errs.ServiceUnavailable, err, "Dependensi belum siap.")
		}
		return httpx.OK(c, fiber.Map{"status": "ready"})
	})

	// Batas publik per IP untuk /api/v1, kecuali webhook vendor.
	publicLimit := p.RateLimit.Middleware("public", publicRateLimit, publicRateWindow, ratelimit.ByIP)
	srv.Use(httpx.PrefixPublic, func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), httpx.PrefixWebhook) {
			return c.Next()
		}
		return publicLimit(c)
	})

	mw := Middlewares{
		Idempotency: p.Idempotency.Middleware,
		RateLimitIP: func(scope string, limit int, window time.Duration) fiber.Handler {
			return p.RateLimit.Middleware(scope, limit, window, ratelimit.ByIP)
		},
		Captcha: captcha.Middleware(p.Captcha),
	}
	for _, m := range a.Registry.Modules() {
		if au, ok := m.(Authenticator); ok {
			mw.Authenticate = au.Authenticate()
			mw.StepUp = au.StepUp()
		}
	}
	for _, m := range a.Registry.Modules() {
		m.RegisterRoutes(srv, mw)
	}
	return srv
}

// MetricsServer melayani /metrics di port internal (METRICS_ADDR), terpisah
// dari port publik agar tidak terekspos lewat reverse proxy.
func (a *App) MetricsServer() *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", a.Platform.Metrics.Handler())
	return &http.Server{
		Addr:              a.Platform.Config.MetricsAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
