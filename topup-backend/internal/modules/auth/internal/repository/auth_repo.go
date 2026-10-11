package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/id"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/uow"
)

// Repo adalah satu-satunya akses ke tabel sessions, email_verifications, dan password_resets.
type Repo struct{ db uow.DBProvider }

func New(db uow.DBProvider) *Repo { return &Repo{db: db} }

func toSession(s Session) domain.Session {
	return domain.Session{
		ID: s.ID, UserID: s.UserID, UserAgent: s.UserAgent, IP: s.IP,
		ExpiresAt: s.ExpiresAt, LastUsedAt: s.LastUsedAt, RevokedAt: s.RevokedAt, CreatedAt: s.CreatedAt,
	}
}

// ---- sesi ----

type NewSession struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash string
	UserAgent        *string
	IP               *string
	ExpiresAt        time.Time
	Now              time.Time
}

func (r *Repo) CreateSession(ctx context.Context, n NewSession) error {
	return r.db.DB(ctx).Create(&Session{
		ID: n.ID, UserID: n.UserID, RefreshTokenHash: n.RefreshTokenHash,
		UserAgent: n.UserAgent, IP: n.IP, ExpiresAt: n.ExpiresAt, LastUsedAt: &n.Now, CreatedAt: n.Now,
	}).Error
}

func (r *Repo) FindSession(ctx context.Context, sessionID uuid.UUID) (domain.Session, error) {
	var s Session
	err := r.db.DB(ctx).Where("id = ?", sessionID).Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Session{}, errs.New(errs.NotFound, "")
	}
	return toSession(s), err
}

// RefreshMatch adalah hasil pencarian sesi berdasarkan hash refresh token.
type RefreshMatch struct {
	Session domain.Session
	// IsPrevious true bila token yang dikirim adalah token sebelum rotasi terakhir
	// (tanda pemakaian ulang).
	IsPrevious bool
}

// LockByRefreshHash mengunci sesi (SELECT ... FOR UPDATE) yang refresh token
// saat ini atau sebelumnya cocok dengan hash. Harus dipanggil di dalam transaksi.
func (r *Repo) LockByRefreshHash(ctx context.Context, hash string) (RefreshMatch, error) {
	var s Session
	err := r.db.DB(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("refresh_token_hash = ? OR previous_token_hash = ?", hash, hash).
		Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return RefreshMatch{}, errs.New(errs.NotFound, "")
	}
	if err != nil {
		return RefreshMatch{}, err
	}
	return RefreshMatch{Session: toSession(s), IsPrevious: s.RefreshTokenHash != hash}, nil
}

// Rotate mengganti refresh token sesi: hash lama disimpan sebagai previous_token_hash.
func (r *Repo) Rotate(ctx context.Context, sessionID uuid.UUID, oldHash, newHash string, expiresAt, now time.Time) error {
	return r.db.DB(ctx).Model(&Session{}).Where("id = ?", sessionID).Updates(map[string]any{
		"refresh_token_hash":  newHash,
		"previous_token_hash": oldHash,
		"expires_at":          expiresAt,
		"last_used_at":        now,
	}).Error
}

// Revoke mencabut satu sesi milik user. Mengembalikan false bila sesi tidak
// ada, bukan milik user, atau sudah dicabut.
func (r *Repo) Revoke(ctx context.Context, userID, sessionID uuid.UUID, reason string, now time.Time) (bool, error) {
	res := r.db.DB(ctx).Model(&Session{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", sessionID, userID).
		Updates(map[string]any{"revoked_at": now, "revoked_reason": reason})
	return res.RowsAffected > 0, res.Error
}

// RevokeByID mencabut sesi tanpa memeriksa pemilik (dipakai deteksi reuse yang sudah mengunci sesi).
func (r *Repo) RevokeByID(ctx context.Context, sessionID uuid.UUID, reason string, now time.Time) error {
	return r.db.DB(ctx).Model(&Session{}).
		Where("id = ? AND revoked_at IS NULL", sessionID).
		Updates(map[string]any{"revoked_at": now, "revoked_reason": reason}).Error
}

// RevokeAll mencabut semua sesi aktif user.
func (r *Repo) RevokeAll(ctx context.Context, userID uuid.UUID, reason string, now time.Time) error {
	return r.db.DB(ctx).Model(&Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]any{"revoked_at": now, "revoked_reason": reason}).Error
}

// ListActive mengembalikan sesi aktif user, terbaru lebih dulu.
func (r *Repo) ListActive(ctx context.Context, userID uuid.UUID, now time.Time) ([]domain.Session, error) {
	var rows []Session
	err := r.db.DB(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, now).
		Order("created_at DESC, id DESC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.Session, len(rows))
	for i, s := range rows {
		out[i] = toSession(s)
	}
	return out, nil
}

// ---- token sekali pakai (verifikasi email & reset password) ----

// OneTime adalah token sekali pakai yang valid.
type OneTime struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

func (r *Repo) CreateEmailVerification(ctx context.Context, userID uuid.UUID, hash string, expiresAt, now time.Time) (uuid.UUID, error) {
	row := EmailVerification{ID: id.New(), UserID: userID, TokenHash: hash, ExpiresAt: expiresAt, CreatedAt: now}
	return row.ID, r.db.DB(ctx).Create(&row).Error
}

// ConsumeEmailVerification menandai token terpakai bila masih valid. Atomik:
// UPDATE ... WHERE used_at IS NULL AND expires_at > now, sehingga dua request
// paralel tidak bisa memakai token yang sama.
func (r *Repo) ConsumeEmailVerification(ctx context.Context, hash string, now time.Time) (OneTime, error) {
	var row EmailVerification
	res := r.db.DB(ctx).Model(&row).Clauses(clause.Returning{}).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, now).
		Update("used_at", now)
	if res.Error != nil {
		return OneTime{}, res.Error
	}
	if res.RowsAffected == 0 {
		return OneTime{}, errs.New(errs.NotFound, "")
	}
	return OneTime{ID: row.ID, UserID: row.UserID}, nil
}

func (r *Repo) CreatePasswordReset(ctx context.Context, userID uuid.UUID, hash string, ip *string, expiresAt, now time.Time) (uuid.UUID, error) {
	row := PasswordReset{ID: id.New(), UserID: userID, TokenHash: hash, ExpiresAt: expiresAt, IP: ip, CreatedAt: now}
	return row.ID, r.db.DB(ctx).Create(&row).Error
}

// ConsumePasswordReset sama seperti ConsumeEmailVerification untuk token reset.
func (r *Repo) ConsumePasswordReset(ctx context.Context, hash string, now time.Time) (OneTime, error) {
	var row PasswordReset
	res := r.db.DB(ctx).Model(&row).Clauses(clause.Returning{}).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, now).
		Update("used_at", now)
	if res.Error != nil {
		return OneTime{}, res.Error
	}
	if res.RowsAffected == 0 {
		return OneTime{}, errs.New(errs.NotFound, "")
	}
	return OneTime{ID: row.ID, UserID: row.UserID}, nil
}

// InvalidateResets menghanguskan semua token reset user yang belum dipakai.
func (r *Repo) InvalidateResets(ctx context.Context, userID uuid.UUID, now time.Time) error {
	return r.db.DB(ctx).Model(&PasswordReset{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Update("used_at", now).Error
}
