// Package catalog adalah kontrak publik modul catalog: kategori, brand,
// produk, banner, dan skema input per brand (ARCHITECTURE §2.1).
package catalog

import (
	"context"

	"gorm.io/gorm"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/catalog/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/catalog/internal/repository"
)

// Seed menambah kategori dasar (Game, Voucher, Pulsa, Data, PLN, E-Wallet)
// yang belum ada. Dipanggil cmd/migrate setelah migrasi; aman dijalankan berulang.
func Seed(ctx context.Context, db *gorm.DB) error {
	return repository.SeedCategories(ctx, db, domain.BaseCategories)
}
