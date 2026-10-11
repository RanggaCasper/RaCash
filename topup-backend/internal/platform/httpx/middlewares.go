package httpx

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// Middlewares adalah middleware bersama yang bisa dipasang modul per route.
// Didefinisikan di sini (bukan di internal/app) agar modul bisa memakainya
// tanpa mengimpor app, yang dilarang aturan arsitektur.
type Middlewares struct {
	// Idempotency(true) mewajibkan header Idempotency-Key (endpoint 🔁).
	Idempotency func(required bool) fiber.Handler
	// RateLimitIP membatasi per IP dengan scope tertentu.
	RateLimitIP func(scope string, limit int, window time.Duration) fiber.Handler
	// Captcha memverifikasi header X-Captcha-Token (endpoint publik rawan spam).
	Captcha fiber.Handler
	// Authenticate memverifikasi Bearer JWT beserta sesinya (diisi modul auth).
	Authenticate fiber.Handler
}
