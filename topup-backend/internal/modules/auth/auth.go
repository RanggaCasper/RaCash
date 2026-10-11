// Package auth adalah kontrak publik modul auth: registrasi, login, JWT, sesi
// & refresh token, verifikasi email, dan reset password (ARCHITECTURE §2.1).
package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/handler"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/repository"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/service"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/token"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/rbac"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/user"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/event"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/httpx"
)

// Principal adalah identitas request yang sudah terautentikasi.
type Principal = domain.Principal

// Nama event yang diterbitkan modul auth (ARCHITECTURE §2.5).
const (
	EventEmailVerificationRequested = domain.EventEmailVerificationReq
	EventPasswordResetRequested     = domain.EventPasswordResetReq
)

// TokenEventPayload adalah payload kedua event di atas.
type TokenEventPayload = domain.TokenEventPayload

// PrincipalFrom mengembalikan identitas request yang lolos middleware
// Authenticate; ok=false bila route tidak memakainya.
func PrincipalFrom(c *fiber.Ctx) (Principal, bool) { return handler.LookupPrincipal(c) }

// Module mengimplementasikan app.Module.
type Module struct {
	svc          *service.Service
	handler      *handler.Handler
	authenticate fiber.Handler
	stepUp       fiber.Handler
}

// New merangkai modul auth. users dan roles adalah kontrak publik modul user dan rbac.
func New(p *platform.Platform, users user.Service, roles rbac.Service) (*Module, error) {
	svc, err := service.New(service.Deps{
		Repo:    repository.New(p.UoW),
		Users:   users,
		Roles:   roles,
		Tx:      p.UoW,
		Outbox:  p.Outbox,
		Tokens:  token.NewIssuer(p.Config.JWTSecret, p.Clock),
		Limiter: p.RateLimit,
		Keyring: p.Keyring,
		Redis:   p.Redis,
		Clock:   p.Clock,
		Log:     p.Log,
	})
	if err != nil {
		return nil, err
	}
	h, err := handler.New(svc, p.Validator)
	if err != nil {
		return nil, err
	}
	return &Module{
		svc: svc, handler: h,
		authenticate: handler.Authenticate(svc, false),
		stepUp:       handler.StepUp(svc),
	}, nil
}

func (*Module) Name() string { return "auth" }

// Authenticate adalah middleware Bearer JWT untuk modul lain (dipasang ke
// httpx.Middlewares.Authenticate oleh app).
func (m *Module) Authenticate() fiber.Handler { return m.authenticate }

// StepUp adalah middleware untuk endpoint sensitif ★: mewajibkan X-Step-Up-Token
// yang valid. Dipasang setelah Authenticate.
func (m *Module) StepUp() fiber.Handler { return m.stepUp }

func (m *Module) RegisterRoutes(r fiber.Router, mw httpx.Middlewares) { m.handler.Register(r, mw) }
func (*Module) RegisterJobs(*asynq.ServeMux)                          {}
func (*Module) RegisterSubscribers(event.Bus)                         {}
