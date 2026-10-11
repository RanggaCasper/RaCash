// Package rbac adalah kontrak publik modul rbac: role, permission, assignment,
// Authorizer, dan middleware Require (ARCHITECTURE §2.1, PRD §7).
package rbac

import (
	"context"

	"gorm.io/gorm"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/rbac/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/rbac/internal/repository"
)

// Seed menyinkronkan katalog permission dan role sistem beserta matriks
// awalnya (RBAC-01, PRD §7.3–7.5). Dipanggil cmd/migrate setelah migrasi;
// aman dijalankan berulang.
func Seed(ctx context.Context, db *gorm.DB) error {
	return repository.Seed(ctx, db, domain.Permissions, domain.SystemRoles())
}
