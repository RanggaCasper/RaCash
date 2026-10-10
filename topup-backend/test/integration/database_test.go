package integration

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/config"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/database"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/id"
)

func TestEnsureExistsCreatesOnce(t *testing.T) {
	if os.Getenv("TEST_DB_NAME") == "" {
		t.Skip("TEST_DB_NAME tidak di-set")
	}
	base, err := config.LoadDatabaseWithPrefix("TEST_")
	if err != nil {
		t.Fatal(err)
	}
	// Nama acak dengan huruf besar & tanda hubung untuk menguji quoting.
	name := "RaCash-ensure-" + strings.ReplaceAll(id.New().String()[:8], "-", "")
	cfg := base.WithName(name)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	t.Cleanup(func() {
		maint, err := gorm.Open(postgres.Open(base.WithName(base.MaintenanceName).URL()), &gorm.Config{})
		if err == nil {
			maint.Exec(`DROP DATABASE IF EXISTS "` + name + `"`)
			database.Close(maint)
		}
	})

	created, err := database.EnsureExists(ctx, cfg, log)
	if err != nil || !created {
		t.Fatalf("pemanggilan pertama harus membuat database: created=%v err=%v", created, err)
	}
	created, err = database.EnsureExists(ctx, cfg, log)
	if err != nil || created {
		t.Fatalf("pemanggilan kedua harus no-op: created=%v err=%v", created, err)
	}

	db, err := gorm.Open(postgres.Open(cfg.URL()), &gorm.Config{})
	if err != nil {
		t.Fatalf("database baru tidak bisa dibuka: %v", err)
	}
	database.Close(db)
}
