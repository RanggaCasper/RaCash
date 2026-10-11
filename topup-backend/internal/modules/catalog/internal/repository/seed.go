package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/catalog/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/id"
)

// SeedCategories menambah kategori yang slug-nya belum ada (termasuk yang
// sudah di-soft-delete). Idempoten.
func SeedCategories(ctx context.Context, db *gorm.DB, cats []domain.Category) error {
	rows := make([]Category, len(cats))
	for i, c := range cats {
		rows[i] = Category{ID: id.New(), Slug: c.Slug, Name: c.Name, SortOrder: c.SortOrder, IsActive: true}
	}
	err := db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "slug"}}, DoNothing: true}).
		Create(&rows).Error
	if err != nil {
		return fmt.Errorf("seed categories: %w", err)
	}
	return nil
}
