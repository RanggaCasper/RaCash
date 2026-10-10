// Package database membuka koneksi GORM ke PostgreSQL. Seluruh akses data
// aplikasi wajib lewat *gorm.DB (RULES §3.3).
package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/config"
)

// Open membuka koneksi dan mengatur pool. Bila auto-create aktif (default hanya
// di APP_ENV=local), database dibuat lebih dulu bila belum ada.
func Open(cfg config.Config, log *slog.Logger) (*gorm.DB, error) {
	if cfg.DB.AutoCreateEnabled(cfg.AppEnv) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := EnsureExists(ctx, cfg.DB, log)
		cancel()
		if err != nil {
			return nil, err
		}
	}
	level := gormlogger.Warn
	if cfg.IsLocal() && cfg.LogLevel == "debug" {
		level = gormlogger.Info
	}
	db, err := gorm.Open(postgres.Open(cfg.DB.URL()), &gorm.Config{
		Logger: gormlogger.New(slogWriter{log.With("component", "gorm")}, gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  level,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true, // nilai parameter (bisa berisi PII) tidak ikut dicatat
		}),
		NowFunc:                func() time.Time { return time.Now().UTC() },
		TranslateError:         true, // unique violation → gorm.ErrDuplicatedKey, dst.
		SkipDefaultTransaction: true, // transaksi eksplisit lewat uow
	})
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
	return db, nil
}

// Ping dipakai /readyz.
func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close menutup pool koneksi.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

type slogWriter struct{ l *slog.Logger }

func (w slogWriter) Printf(format string, args ...any) {
	w.l.Warn(fmt.Sprintf(format, args...))
}
