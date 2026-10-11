package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/rbac/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/uow"
)

// Repo adalah akses ke roles dan user_roles.
type Repo struct{ db uow.DBProvider }

func New(db uow.DBProvider) *Repo { return &Repo{db: db} }

// RolesOf mengembalikan role aktif (belum kedaluwarsa) milik user, terurut berdasarkan code.
func (r *Repo) RolesOf(ctx context.Context, userID uuid.UUID, now time.Time) ([]domain.RoleRef, error) {
	var rows []Role
	err := r.db.DB(ctx).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND (user_roles.expires_at IS NULL OR user_roles.expires_at > ?)", userID, now).
		Order("roles.code").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.RoleRef, len(rows))
	for i, ro := range rows {
		out[i] = domain.RoleRef{Code: ro.Code, Name: ro.Name, Scope: ro.Scope}
	}
	return out, nil
}

// AssignByCode memberi role ke user. Idempoten: role yang sudah dimiliki tidak diubah.
func (r *Repo) AssignByCode(ctx context.Context, userID uuid.UUID, code string, now time.Time) error {
	db := r.db.DB(ctx)
	var role Role
	if err := db.Where("code = ?", code).Take(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.New(errs.NotFound, "Role tidak ditemukan.")
		}
		return err
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&UserRole{UserID: userID, RoleID: role.ID, AssignedAt: now}).Error
}
