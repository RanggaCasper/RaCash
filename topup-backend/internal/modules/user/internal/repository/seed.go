package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/user/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/id"
)

// SeedTiers menambah tier bawaan yang belum ada (berdasarkan code). Idempoten.
func SeedTiers(ctx context.Context, db *gorm.DB, tiers []domain.Tier) error {
	rows := make([]Tier, len(tiers))
	for i, t := range tiers {
		rows[i] = Tier{ID: id.New(), Code: t.Code, Name: t.Name, Rank: t.Rank, IsDefault: t.IsDefault}
	}
	err := db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "code"}}, DoNothing: true}).
		Create(&rows).Error
	if err != nil {
		return fmt.Errorf("seed tiers: %w", err)
	}
	return nil
}
