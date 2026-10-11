// Package user adalah kontrak publik modul user: profil, status, tier, dan
// undangan staf (ARCHITECTURE §2.1).
package user

import (
	"context"

	"gorm.io/gorm"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/user/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/user/internal/repository"
)

// Seed menambah tier bawaan (GUEST, MEMBER, RESELLER_*) yang belum ada.
// Dipanggil cmd/migrate setelah migrasi; aman dijalankan berulang.
func Seed(ctx context.Context, db *gorm.DB) error {
	return repository.SeedTiers(ctx, db, domain.SystemTiers)
}
