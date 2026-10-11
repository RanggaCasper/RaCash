// Package app merangkai platform dan modul: kontrak Module, registry,
// bootstrap (wiring DI manual), server HTTP, dan worker.
package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/event"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/httpx"
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

// Authenticator diimplementasikan modul yang menyediakan middleware Bearer JWT
// (modul auth). App memasangnya ke Middlewares.Authenticate untuk modul lain.
type Authenticator interface {
	Authenticate() fiber.Handler
	StepUp() fiber.Handler
}

// Middlewares adalah middleware bersama yang bisa dipasang modul per route.
// Tipenya ada di httpx supaya modul tidak perlu mengimpor app.
type Middlewares = httpx.Middlewares

// Base dapat di-embed modul agar cukup mengimplementasikan hook yang dipakai.
type Base struct{}

func (Base) RegisterRoutes(fiber.Router, Middlewares) {}
func (Base) RegisterJobs(*asynq.ServeMux)             {}
func (Base) RegisterSubscribers(event.Bus)            {}
