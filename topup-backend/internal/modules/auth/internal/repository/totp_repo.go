package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/id"
)

// TOTP adalah rahasia TOTP user (terenkripsi) beserta status konfirmasinya.
type TOTP struct {
	SecretEncrypted []byte
	SecretKeyID     string
	Confirmed       bool
	LastUsedStep    *int64
}

// UpsertPendingTOTP menyimpan rahasia baru yang belum dikonfirmasi. Rahasia yang
// belum dikonfirmasi boleh ditimpa (user mengulang setup); yang sudah
// dikonfirmasi tidak, dan mengembalikan errs.Conflict.
func (r *Repo) UpsertPendingTOTP(ctx context.Context, userID uuid.UUID, secretEnc []byte, keyID string, now time.Time) error {
	res := r.db.DB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		Where:     clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "user_totp.confirmed_at IS NULL"}}},
		DoUpdates: clause.Assignments(map[string]any{"secret_encrypted": secretEnc, "secret_key_id": keyID, "last_used_step": nil, "updated_at": now}),
	}).Create(&UserTOTP{UserID: userID, SecretEncrypted: secretEnc, SecretKeyID: keyID, CreatedAt: now, UpdatedAt: now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.New(errs.Conflict, "2FA sudah aktif.")
	}
	return nil
}

// FindTOTP mengambil rahasia TOTP user; errs.NotFound bila belum pernah setup.
func (r *Repo) FindTOTP(ctx context.Context, userID uuid.UUID) (TOTP, error) {
	var row UserTOTP
	err := r.db.DB(ctx).Where("user_id = ?", userID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return TOTP{}, errs.New(errs.NotFound, "")
	}
	if err != nil {
		return TOTP{}, err
	}
	return TOTP{
		SecretEncrypted: row.SecretEncrypted, SecretKeyID: row.SecretKeyID,
		Confirmed: row.ConfirmedAt != nil, LastUsedStep: row.LastUsedStep,
	}, nil
}

// ClaimStep mencatat langkah waktu yang baru dipakai. Atomik: hanya satu
// pemanggil yang berhasil untuk langkah yang sama, dan langkah yang tidak lebih
// baru dari last_used_step ditolak. Ini mencegah replay kode TOTP.
func (r *Repo) ClaimStep(ctx context.Context, userID uuid.UUID, step int64, now time.Time) (bool, error) {
	res := r.db.DB(ctx).Model(&UserTOTP{}).
		Where("user_id = ? AND (last_used_step IS NULL OR last_used_step < ?)", userID, step).
		Updates(map[string]any{"last_used_step": step, "updated_at": now})
	return res.RowsAffected > 0, res.Error
}

// ConfirmTOTP menandai rahasia sebagai aktif.
func (r *Repo) ConfirmTOTP(ctx context.Context, userID uuid.UUID, now time.Time) error {
	return r.db.DB(ctx).Model(&UserTOTP{}).
		Where("user_id = ? AND confirmed_at IS NULL", userID).
		Updates(map[string]any{"confirmed_at": now, "updated_at": now}).Error
}

// DeleteTOTP menghapus rahasia TOTP dan semua recovery code user.
func (r *Repo) DeleteTOTP(ctx context.Context, userID uuid.UUID) error {
	db := r.db.DB(ctx)
	if err := db.Where("user_id = ?", userID).Delete(&UserTOTP{}).Error; err != nil {
		return err
	}
	return db.Where("user_id = ?", userID).Delete(&UserRecoveryCode{}).Error
}

// ReplaceRecoveryCodes mengganti seluruh recovery code user dengan hash baru.
func (r *Repo) ReplaceRecoveryCodes(ctx context.Context, userID uuid.UUID, hashes []string, now time.Time) error {
	db := r.db.DB(ctx)
	if err := db.Where("user_id = ?", userID).Delete(&UserRecoveryCode{}).Error; err != nil {
		return err
	}
	rows := make([]UserRecoveryCode, len(hashes))
	for i, h := range hashes {
		rows[i] = UserRecoveryCode{ID: id.New(), UserID: userID, CodeHash: h, CreatedAt: now}
	}
	return db.Create(&rows).Error
}

// ConsumeRecoveryCode memakai satu recovery code. Atomik: kode yang sama tidak
// bisa dipakai dua kali.
func (r *Repo) ConsumeRecoveryCode(ctx context.Context, userID uuid.UUID, hash string, now time.Time) (bool, error) {
	res := r.db.DB(ctx).Model(&UserRecoveryCode{}).
		Where("user_id = ? AND code_hash = ? AND used_at IS NULL", userID, hash).
		Update("used_at", now)
	return res.RowsAffected > 0, res.Error
}

// CountRecoveryCodes menghitung recovery code yang belum terpakai.
func (r *Repo) CountRecoveryCodes(ctx context.Context, userID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.DB(ctx).Model(&UserRecoveryCode{}).
		Where("user_id = ? AND used_at IS NULL", userID).Count(&n).Error
	return n, err
}
