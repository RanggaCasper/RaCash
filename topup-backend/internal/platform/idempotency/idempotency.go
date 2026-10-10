// Package idempotency menerapkan header Idempotency-Key (API_CONTRACT §1.6):
// key sama + body sama → respons diputar ulang; body beda → 409
// IDEMPOTENCY_CONFLICT. Disimpan 24 jam di tabel idempotency_keys.
package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/clock"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/crypto"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/logger"
)

const (
	Header = "Idempotency-Key"
	// LocalsPrincipal diisi middleware auth (user_id / api_key id) agar scope per pemanggil.
	LocalsPrincipal = "principal"
)

type record struct {
	Scope          string `gorm:"primaryKey"`
	Key            string `gorm:"primaryKey"`
	RequestHash    string
	ResponseStatus *int16
	ResponseBody   datatypes.JSON
	LockedUntil    *time.Time
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

func (record) TableName() string { return "idempotency_keys" }

type Store struct {
	db      *gorm.DB
	clock   clock.Clock
	ttl     time.Duration
	lockTTL time.Duration
}

func NewStore(db *gorm.DB, c clock.Clock) *Store {
	return &Store{db: db, clock: c, ttl: 24 * time.Hour, lockTTL: 60 * time.Second}
}

type outcome int

const (
	proceed outcome = iota
	replay
)

// begin mencatat key atau memutuskan replay/konflik.
func (s *Store) begin(ctx context.Context, scope, key, hash string) (outcome, *record, error) {
	now := s.clock.Now()
	lock := now.Add(s.lockTTL)
	fresh := record{Scope: scope, Key: key, RequestHash: hash, LockedUntil: &lock, ExpiresAt: now.Add(s.ttl), CreatedAt: now}

	res := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&fresh)
	if res.Error != nil {
		return 0, nil, res.Error
	}
	if res.RowsAffected == 1 {
		return proceed, nil, nil
	}

	var out outcome
	var existing record
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("scope = ? AND key = ?", scope, key).First(&existing).Error; err != nil {
			return err
		}
		switch {
		case existing.ExpiresAt.Before(now):
			// Kedaluwarsa: anggap key baru.
			out = proceed
			if err := tx.Where("scope = ? AND key = ?", scope, key).Delete(&record{}).Error; err != nil {
				return err
			}
			return tx.Create(&fresh).Error
		case existing.RequestHash != hash:
			return errs.New(errs.IdempotencyConflict, "")
		case existing.ResponseStatus != nil:
			out = replay
			return nil
		case existing.LockedUntil != nil && existing.LockedUntil.After(now):
			return errs.New(errs.Conflict, "Request dengan Idempotency-Key ini sedang diproses.")
		default:
			// Lock habis tanpa respons (proses sebelumnya mati): ambil alih.
			out = proceed
			return tx.Model(&record{}).Where("scope = ? AND key = ?", scope, key).
				Update("locked_until", lock).Error
		}
	})
	return out, &existing, err
}

// complete menyimpan respons. Body disimpan sebagai string JSON di kolom jsonb
// karena jsonb menormalkan objek (urutan key & spasi); replay harus identik.
func (s *Store) complete(ctx context.Context, scope, key string, status int, body []byte) error {
	st := int16(status)
	encoded, err := json.Marshal(string(body))
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&record{}).Where("scope = ? AND key = ?", scope, key).
		Updates(map[string]any{"response_status": st, "response_body": datatypes.JSON(encoded), "locked_until": nil}).Error
}

func decodeBody(stored datatypes.JSON) ([]byte, error) {
	var s string
	if err := json.Unmarshal(stored, &s); err != nil {
		return nil, err
	}
	return []byte(s), nil
}

func (s *Store) release(ctx context.Context, scope, key string) error {
	return s.db.WithContext(ctx).Where("scope = ? AND key = ?", scope, key).Delete(&record{}).Error
}

// DeleteExpired dipakai job pembersihan berkala.
func (s *Store) DeleteExpired(ctx context.Context) (int64, error) {
	res := s.db.WithContext(ctx).Where("expires_at < ?", s.clock.Now()).Delete(&record{})
	return res.RowsAffected, res.Error
}

// Middleware mewajibkan (required=true) atau mengizinkan (false) Idempotency-Key.
// Hanya respons sukses (< 400) yang disimpan untuk diputar ulang; error melepas key
// agar klien bisa mencoba lagi dengan key yang sama.
func (s *Store) Middleware(required bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Get(Header)
		if key == "" {
			if required {
				return errs.Validation(errs.Detail{Field: Header, Code: "required", Message: "Header Idempotency-Key wajib diisi."})
			}
			return c.Next()
		}
		if _, err := uuid.Parse(key); err != nil {
			return errs.Validation(errs.Detail{Field: Header, Code: "uuid", Message: "Idempotency-Key harus UUID."})
		}

		ctx := c.UserContext()
		scope := Scope(c)
		hash := crypto.SHA256Hex(append([]byte(c.Method()+" "+c.OriginalURL()+"\n"), c.Body()...))

		out, rec, err := s.begin(ctx, scope, key, hash)
		if err != nil {
			return err
		}
		if out == replay {
			body, err := decodeBody(rec.ResponseBody)
			if err != nil {
				return errs.Wrap(errs.Internal, err, "")
			}
			c.Set("Idempotent-Replayed", "true")
			c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
			return c.Status(int(*rec.ResponseStatus)).Send(body)
		}

		if err := c.Next(); err != nil {
			s.releaseLogged(ctx, scope, key)
			return err
		}
		status := c.Response().StatusCode()
		if status >= 400 {
			s.releaseLogged(ctx, scope, key)
			return nil
		}
		if err := s.complete(ctx, scope, key, status, c.Response().Body()); err != nil {
			logger.FromContext(ctx).Error("gagal menyimpan respons idempotency", "err", err)
		}
		return nil
	}
}

func (s *Store) releaseLogged(ctx context.Context, scope, key string) {
	if err := s.release(ctx, scope, key); err != nil && !errors.Is(err, context.Canceled) {
		logger.FromContext(ctx).Error("gagal melepas idempotency key", "err", err)
	}
}

// Scope = hash(method + pola route + pemanggil), 64 karakter sesuai kolom scope.
func Scope(c *fiber.Ctx) string {
	principal := "ip:" + c.IP()
	if p, ok := c.Locals(LocalsPrincipal).(string); ok && p != "" {
		principal = p
	}
	route := c.Path()
	if r := c.Route(); r != nil && r.Path != "" {
		route = r.Path
	}
	return crypto.SHA256Hex([]byte(c.Method() + " " + route + "|" + principal))
}
