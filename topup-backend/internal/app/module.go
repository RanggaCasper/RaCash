// Package app merangkai platform dan modul: kontrak Module, registry,
// bootstrap (wiring DI manual), server HTTP, dan worker.
package app

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/event"
)

// Module adalah kontrak setiap modul domain (ARCHITECTURE §2.4).
type Module interface {
	Name() string
	RegisterRoutes(r fiber.Router, mw Middlewares) // dipanggil di cmd/api
	RegisterJobs(mux *asynq.ServeMux)              // dipanggil di cmd/worker
	RegisterSubscribers(bus event.Bus)             // listener domain event
}

// Scheduled opsional: modul yang punya job terjadwal (cron, UTC).
type Scheduled interface {
	RegisterSchedules(s *asynq.Scheduler) error
}

// Middlewares adalah middleware bersama yang bisa dipasang modul per route.
// Auth & RBAC ditambahkan oleh modul auth/rbac (issue #5, #7).
type Middlewares struct {
	// Idempotency(true) mewajibkan header Idempotency-Key (endpoint 🔁).
	Idempotency func(required bool) fiber.Handler
	// RateLimitIP membatasi per IP dengan scope tertentu.
	RateLimitIP func(scope string, limit int, window time.Duration) fiber.Handler
}

// Base dapat di-embed modul agar cukup mengimplementasikan hook yang dipakai.
type Base struct{}

func (Base) RegisterRoutes(fiber.Router, Middlewares) {}
func (Base) RegisterJobs(*asynq.ServeMux)             {}
func (Base) RegisterSubscribers(event.Bus)            {}
