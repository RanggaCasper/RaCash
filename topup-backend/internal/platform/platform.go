// Package platform adalah shared kernel: infrastruktur tanpa logika bisnis.
// Platform tidak boleh bergantung pada modul mana pun (RULES §3.1).
package platform

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/cache"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/clock"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/config"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/crypto"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/database"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/event"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/idempotency"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/metrics"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/queue"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/ratelimit"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/uow"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/validator"
)

// Platform berisi dependensi bersama yang diteruskan ke setiap modul.
type Platform struct {
	Config      config.Config
	Log         *slog.Logger
	Clock       clock.Clock
	DB          *gorm.DB
	UoW         *uow.Manager
	Redis       *redis.Client
	QueueOpt    asynq.RedisConnOpt
	Queue       *asynq.Client
	Outbox      *event.Outbox
	Keyring     *crypto.Keyring
	Validator   *validator.Validator
	RateLimit   *ratelimit.Limiter
	Idempotency *idempotency.Store
	Metrics     *metrics.Metrics
}

// New membuka seluruh koneksi infrastruktur.
func New(cfg config.Config, log *slog.Logger) (*Platform, error) {
	keyring, err := crypto.ParseKeyring(cfg.EncryptionKeys, cfg.EncryptionActiveKeyID)
	if err != nil {
		return nil, err
	}
	db, err := database.Open(cfg, log)
	if err != nil {
		return nil, err
	}
	cache.SetLogger(log)
	rdb, err := cache.New(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	qopt, err := queue.RedisOpt(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	clk := clock.Real{}
	um := uow.New(db)
	return &Platform{
		Config:      cfg,
		Log:         log,
		Clock:       clk,
		DB:          db,
		UoW:         um,
		Redis:       rdb,
		QueueOpt:    qopt,
		Queue:       queue.NewClient(qopt),
		Outbox:      event.NewOutbox(um),
		Keyring:     keyring,
		Validator:   validator.New(),
		RateLimit:   ratelimit.New(rdb, clk),
		Idempotency: idempotency.NewStore(db, clk),
		Metrics:     metrics.New(),
	}, nil
}

// Ready mengecek PostgreSQL & Redis (dipakai /readyz).
func (p *Platform) Ready(ctx context.Context) error {
	if err := database.Ping(ctx, p.DB); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	if err := p.Redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	return nil
}

// Close menutup seluruh koneksi.
func (p *Platform) Close() error {
	return errors.Join(p.Queue.Close(), p.Redis.Close(), database.Close(p.DB))
}
