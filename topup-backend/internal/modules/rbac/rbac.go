// Package rbac adalah kontrak publik modul rbac: role, permission, assignment,
// Authorizer, dan middleware Require (ARCHITECTURE §2.1, PRD §7).
package rbac

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/rbac/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/rbac/internal/repository"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/clock"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/event"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/httpx"
)

// Tipe dan konstanta publik; implementasinya ada di internal/domain.
type RoleRef = domain.RoleRef

const (
	RoleMember = domain.RoleMember

	ScopeCustomer = domain.ScopeCustomer
	ScopeInternal = domain.ScopeInternal
)

// Service adalah kontrak yang dipakai modul lain (auth).
type Service interface {
	// RolesOf mengembalikan role aktif milik user.
	RolesOf(ctx context.Context, userID uuid.UUID) ([]RoleRef, error)
	// AssignRole memberi role sistem berdasarkan code. Idempoten.
	// Jalankan di dalam transaksi pemanggil bila harus atomik dengan pembuatan user.
	AssignRole(ctx context.Context, userID uuid.UUID, roleCode string) error
}

// Seed menyinkronkan katalog permission dan role sistem beserta matriks
// awalnya (RBAC-01, PRD §7.3–7.5). Dipanggil cmd/migrate setelah migrasi;
// aman dijalankan berulang.
func Seed(ctx context.Context, db *gorm.DB) error {
	return repository.Seed(ctx, db, domain.Permissions, domain.SystemRoles())
}

type service struct {
	repo  *repository.Repo
	clock clock.Clock
}

func (s *service) RolesOf(ctx context.Context, userID uuid.UUID) ([]RoleRef, error) {
	return s.repo.RolesOf(ctx, userID, s.clock.Now())
}

func (s *service) AssignRole(ctx context.Context, userID uuid.UUID, roleCode string) error {
	return s.repo.AssignByCode(ctx, userID, roleCode, s.clock.Now())
}

// Module mengimplementasikan app.Module. Route dan middleware Require menyusul (issue #7).
type Module struct{ svc *service }

func New(p *platform.Platform) *Module {
	return &Module{svc: &service{repo: repository.New(p.UoW), clock: p.Clock}}
}

func (*Module) Name() string { return "rbac" }

// Service mengembalikan kontrak publik untuk modul yang bergantung pada rbac.
func (m *Module) Service() Service { return m.svc }

func (*Module) RegisterRoutes(fiber.Router, httpx.Middlewares) {}
func (*Module) RegisterJobs(*asynq.ServeMux)                   {}
func (*Module) RegisterSubscribers(event.Bus)                  {}
