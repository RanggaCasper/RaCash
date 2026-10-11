// Package user adalah kontrak publik modul user: profil, status, tier, dan
// undangan staf (ARCHITECTURE §2.1).
package user

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/user/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/user/internal/repository"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/user/internal/service"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/event"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/httpx"
)

// Tipe publik; implementasinya ada di internal/domain.
type (
	User        = domain.User
	Credentials = domain.Credentials
	NewUser     = domain.NewUser
)

// Status akun.
const (
	StatusPendingVerification = domain.StatusPendingVerification
	StatusActive              = domain.StatusActive
	StatusSuspended           = domain.StatusSuspended
)

// Service adalah kontrak yang dipakai modul lain (auth, rbac, ...).
// Error "tidak ada" dikembalikan sebagai errs.NotFound.
type Service interface {
	// Create membuat user PENDING_VERIFICATION dengan tier default.
	// Email yang sudah terdaftar menghasilkan errs.Conflict.
	Create(ctx context.Context, in NewUser) (User, error)
	Get(ctx context.Context, userID uuid.UUID) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	// CredentialsByEmail menyertakan hash password; hanya untuk modul auth.
	CredentialsByEmail(ctx context.Context, email string) (Credentials, error)
	// MarkEmailVerified mengisi email_verified_at dan mengaktifkan akun PENDING.
	MarkEmailVerified(ctx context.Context, userID uuid.UUID) error
	SetPasswordHash(ctx context.Context, userID uuid.UUID, hash string) error
	RecordLogin(ctx context.Context, userID uuid.UUID) error
}

// Seed menambah tier bawaan (GUEST, MEMBER, RESELLER_*) yang belum ada.
// Dipanggil cmd/migrate setelah migrasi; aman dijalankan berulang.
func Seed(ctx context.Context, db *gorm.DB) error {
	return repository.SeedTiers(ctx, db, domain.SystemTiers)
}

// Module mengimplementasikan app.Module. Modul user belum punya route sendiri.
type Module struct{ svc *service.Service }

func New(p *platform.Platform) *Module {
	return &Module{svc: service.New(repository.New(p.UoW), p.Clock)}
}

func (*Module) Name() string { return "user" }

// Service mengembalikan kontrak publik untuk modul yang bergantung pada user.
func (m *Module) Service() Service { return m.svc }

func (*Module) RegisterRoutes(fiber.Router, httpx.Middlewares) {}
func (*Module) RegisterJobs(*asynq.ServeMux)                   {}
func (*Module) RegisterSubscribers(event.Bus)                  {}
