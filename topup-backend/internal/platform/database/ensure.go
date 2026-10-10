package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/config"
)

// SQLSTATE 42P04 duplicate_database: proses lain (api/worker) membuatnya bersamaan.
const sqlStateDuplicateDatabase = "42P04"

// EnsureExists membuat database d.Name bila belum ada, lewat koneksi ke
// database maintenance (default "postgres"). Mengembalikan true bila dibuat.
// User DB_USER butuh hak CREATEDB.
func EnsureExists(ctx context.Context, d config.Database, log *slog.Logger) (bool, error) {
	maint, err := gorm.Open(postgres.Open(d.WithName(d.MaintenanceName).URL()), &gorm.Config{
		Logger: gormlogger.Discard,
	})
	if err != nil {
		return false, fmt.Errorf("database: konek ke database maintenance %q: %w", d.MaintenanceName, err)
	}
	defer Close(maint)

	var exists bool
	if err := maint.WithContext(ctx).
		Raw("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = ?)", d.Name).
		Scan(&exists).Error; err != nil {
		return false, fmt.Errorf("database: cek keberadaan %q: %w", d.Name, err)
	}
	if exists {
		return false, nil
	}

	// CREATE DATABASE tidak menerima parameter, jadi nama di-quote sebagai identifier.
	if err := maint.WithContext(ctx).Exec("CREATE DATABASE " + quoteIdent(d.Name)).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == sqlStateDuplicateDatabase {
			return false, nil
		}
		return false, fmt.Errorf("database: membuat %q: %w", d.Name, err)
	}
	log.Info("database dibuat otomatis", "database", d.Name)
	return true, nil
}

// quoteIdent meng-quote identifier PostgreSQL ("nama" dengan " di-escape).
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
