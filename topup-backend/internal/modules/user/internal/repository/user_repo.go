package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/user/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/id"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/uow"
)

// Repo adalah satu-satunya akses ke tabel users dan tiers.
type Repo struct{ db uow.DBProvider }

func New(db uow.DBProvider) *Repo { return &Repo{db: db} }

// userRow adalah hasil join users dengan tier untuk memetakan ke domain.
// Join di dalam modul pemilik tabel, jadi diperbolehkan.
type userRow struct {
	User
	TierCode string
}

func (r *Repo) base(ctx context.Context) *gorm.DB {
	return r.db.DB(ctx).Table("users").
		Select("users.*, tiers.code AS tier_code").
		Joins("JOIN tiers ON tiers.id = users.tier_id").
		Where("users.deleted_at IS NULL")
}

func toCredentials(row userRow) domain.Credentials {
	return domain.Credentials{
		User: domain.User{
			ID: row.ID, Name: row.Name, Email: row.Email, Phone: row.Phone,
			TierCode: row.TierCode, Status: row.Status, PermVersion: row.PermVersion,
			TwoFactorEnabled: row.TwoFactorEnabled, EmailVerifiedAt: row.EmailVerifiedAt,
			CreatedAt: row.CreatedAt,
		},
		PasswordHash: row.PasswordHash,
	}
}

// Create membuat user baru dengan tier default. Email yang sudah ada
// dipetakan ke errs.Conflict.
func (r *Repo) Create(ctx context.Context, n domain.NewUser, now time.Time) (uuid.UUID, error) {
	db := r.db.DB(ctx)
	var tier Tier
	if err := db.Where("is_default").Take(&tier).Error; err != nil {
		return uuid.Nil, err
	}
	u := User{
		ID: id.New(), Name: n.Name, Email: n.Email, Phone: n.Phone, PasswordHash: n.PasswordHash,
		TierID: tier.ID, Status: domain.StatusPendingVerification, PermVersion: 1,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return uuid.Nil, errs.New(errs.Conflict, "Email sudah terdaftar.")
		}
		return uuid.Nil, err
	}
	return u.ID, nil
}

func (r *Repo) FindByID(ctx context.Context, userID uuid.UUID) (domain.Credentials, error) {
	var row userRow
	err := r.base(ctx).Where("users.id = ?", userID).Take(&row).Error
	return mapFound(row, err)
}

func (r *Repo) FindByEmail(ctx context.Context, email string) (domain.Credentials, error) {
	var row userRow
	err := r.base(ctx).Where("users.email = ?", email).Take(&row).Error
	return mapFound(row, err)
}

func mapFound(row userRow, err error) (domain.Credentials, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Credentials{}, errs.New(errs.NotFound, "")
	}
	if err != nil {
		return domain.Credentials{}, err
	}
	return toCredentials(row), nil
}

// MarkEmailVerified mengisi email_verified_at dan mengaktifkan akun yang masih
// PENDING_VERIFICATION. Akun SUSPENDED tetap SUSPENDED. Idempoten: email yang
// sudah terverifikasi tidak diubah.
func (r *Repo) MarkEmailVerified(ctx context.Context, userID uuid.UUID, at time.Time) error {
	db := r.db.DB(ctx)
	err := db.Model(&User{}).
		Where("id = ? AND email_verified_at IS NULL", userID).
		Updates(map[string]any{"email_verified_at": at, "updated_at": at}).Error
	if err != nil {
		return err
	}
	return db.Model(&User{}).
		Where("id = ? AND status = ?", userID, domain.StatusPendingVerification).
		Updates(map[string]any{"status": domain.StatusActive, "updated_at": at}).Error
}

func (r *Repo) SetPasswordHash(ctx context.Context, userID uuid.UUID, hash string, at time.Time) error {
	res := r.db.DB(ctx).Model(&User{}).Where("id = ? AND deleted_at IS NULL", userID).
		Updates(map[string]any{"password_hash": hash, "updated_at": at})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.New(errs.NotFound, "")
	}
	return nil
}

func (r *Repo) SetTwoFactorEnabled(ctx context.Context, userID uuid.UUID, enabled bool, at time.Time) error {
	res := r.db.DB(ctx).Model(&User{}).Where("id = ? AND deleted_at IS NULL", userID).
		Updates(map[string]any{"two_factor_enabled": enabled, "updated_at": at})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.New(errs.NotFound, "")
	}
	return nil
}

func (r *Repo) TouchLastLogin(ctx context.Context, userID uuid.UUID, at time.Time) error {
	return r.db.DB(ctx).Model(&User{}).Where("id = ?", userID).
		Update("last_login_at", at).Error
}
