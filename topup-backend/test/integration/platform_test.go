// Integration test platform terhadap PostgreSQL sungguhan yang sudah dimigrasi.
// Koneksi dibaca dari TEST_DB_HOST, TEST_DB_PORT, TEST_DB_USER, TEST_DB_PASSWORD,
// TEST_DB_NAME, TEST_DB_SSLMODE (sengaja terpisah dari DB_* development).
// Tanpa TEST_DB_NAME, test di-skip.
package integration

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/clock"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/config"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/event"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/httpx"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/id"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/idempotency"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/uow"
)

func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	if os.Getenv("TEST_DB_NAME") == "" {
		t.Skip("TEST_DB_NAME tidak di-set")
	}
	cfg, err := config.LoadDatabaseWithPrefix("TEST_")
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.Open(cfg.URL()), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func countOutbox(t *testing.T, db *gorm.DB, aggID uuid.UUID, status string) int64 {
	t.Helper()
	var n int64
	q := db.Table("outbox_events").Where("aggregate_id = ?", aggID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func newEvent(t *testing.T, aggID uuid.UUID) event.Event {
	t.Helper()
	e, err := event.New("order", "order.paid", "order", aggID, map[string]any{"invoice_no": "RC-20261010-TEST01"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestOutboxRequiresTransaction(t *testing.T) {
	db := openDB(t)
	um := uow.New(db)
	ob := event.NewOutbox(um)
	if err := ob.Publish(context.Background(), newEvent(t, id.New())); !errors.Is(err, event.ErrNotInTx) {
		t.Fatalf("Publish di luar transaksi harus ditolak, got %v", err)
	}
}

func TestOutboxCommitAndRollback(t *testing.T) {
	db := openDB(t)
	um := uow.New(db)
	ob := event.NewOutbox(um)
	ctx := context.Background()

	committed := id.New()
	if err := um.Do(ctx, func(ctx context.Context) error { return ob.Publish(ctx, newEvent(t, committed)) }); err != nil {
		t.Fatal(err)
	}
	if n := countOutbox(t, db, committed, event.StatusPending); n != 1 {
		t.Fatalf("event commit: want 1 PENDING, got %d", n)
	}

	rolledBack := id.New()
	boom := errors.New("gagal setelah publish")
	err := um.Do(ctx, func(ctx context.Context) error {
		if err := ob.Publish(ctx, newEvent(t, rolledBack)); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("want boom, got %v", err)
	}
	if n := countOutbox(t, db, rolledBack, ""); n != 0 {
		t.Fatalf("event harus ikut rollback, got %d", n)
	}
}

func TestRelayMarksPublishedWithoutSubscribers(t *testing.T) {
	db := openDB(t)
	um := uow.New(db)
	ob := event.NewOutbox(um)
	ctx := context.Background()

	agg := id.New()
	if err := um.Do(ctx, func(ctx context.Context) error { return ob.Publish(ctx, newEvent(t, agg)) }); err != nil {
		t.Fatal(err)
	}
	// Tanpa subscriber, relay tidak menyentuh Redis sehingga klien dummy cukup.
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: "127.0.0.1:1"})
	defer client.Close()
	relay := event.NewRelay(db, client, event.NewRegistry(), clock.Real{}, slog.New(slog.NewTextHandler(io.Discard, nil)), 1000, time.Second)
	if _, err := relay.ProcessBatch(ctx); err != nil {
		t.Fatal(err)
	}
	if n := countOutbox(t, db, agg, event.StatusPublished); n != 1 {
		t.Fatalf("want PUBLISHED, got %d", n)
	}
}

func TestIdempotencyMiddleware(t *testing.T) {
	db := openDB(t)
	store := idempotency.NewStore(db, clock.Real{})
	calls := 0

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler, DisableStartupMessage: true})
	app.Use(httpx.RequestIDMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil))))
	app.Post("/orders", store.Middleware(true), func(c *fiber.Ctx) error {
		calls++
		if strings.Contains(string(c.Body()), "fail") {
			return errs.New(errs.ProductUnavailable, "")
		}
		return httpx.Created(c, fiber.Map{"call": calls})
	})

	post := func(key, body string) (int, string, string) {
		req := httptest.NewRequest("POST", "/orders", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if key != "" {
			req.Header.Set(idempotency.Header, key)
		}
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b), res.Header.Get("Idempotent-Replayed")
	}

	if st, _, _ := post("", `{}`); st != 422 {
		t.Fatalf("tanpa key harus 422, got %d", st)
	}

	key := uuid.NewString()
	st1, body1, _ := post(key, `{"a":1}`)
	st2, body2, replayed := post(key, `{"a":1}`)
	if st1 != 201 || st2 != 201 || replayed != "true" || calls != 1 {
		t.Fatalf("replay gagal: st1=%d st2=%d replayed=%q calls=%d", st1, st2, replayed, calls)
	}
	// Replay harus identik byte-per-byte, termasuk request_id respons asli.
	if body1 != body2 {
		t.Fatalf("body replay berbeda:\n%s\n%s", body1, body2)
	}
	if st, body, _ := post(key, `{"a":2}`); st != 409 || !strings.Contains(body, "IDEMPOTENCY_CONFLICT") {
		t.Fatalf("body berbeda harus 409 IDEMPOTENCY_CONFLICT, got %d %s", st, body)
	}

	// Error melepas key sehingga klien bisa mencoba lagi dengan key sama.
	failKey := uuid.NewString()
	if st, _, _ := post(failKey, `{"fail":true}`); st != 422 {
		t.Fatalf("want 422, got %d", st)
	}
	before := calls
	if st, _, _ := post(failKey, `{"fail":true}`); st != 422 || calls != before+1 {
		t.Fatalf("key harus dilepas setelah error (calls %d → %d)", before, calls)
	}
}
