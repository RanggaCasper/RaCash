package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
)

func newApp() *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler, DisableStartupMessage: true})
	app.Use(RequestIDMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil))))
	app.Get("/ok", func(c *fiber.Ctx) error { return OK(c, fiber.Map{"x": 1}) })
	app.Get("/conflict", func(c *fiber.Ctx) error { return errs.New(errs.VersionConflict, "") })
	app.Get("/boom", func(c *fiber.Ctx) error { return errors.New("db password=hunter2 bocor") })
	app.Get("/page", func(c *fiber.Ctx) error {
		p, err := ParsePage(c)
		if err != nil {
			return err
		}
		return Page(c, []int{}, NewPageMeta(p, 134))
	})
	return app
}

func do(t *testing.T, app *fiber.App, path string, header map[string]string) (int, map[string]any, string) {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	_ = json.NewDecoder(res.Body).Decode(&body)
	return res.StatusCode, body, res.Header.Get(HeaderRequestID)
}

func TestEnvelopeSuccess(t *testing.T) {
	status, body, rid := do(t, newApp(), "/ok", nil)
	if status != 200 || body["success"] != true || body["request_id"] != rid || rid == "" {
		t.Fatalf("status=%d body=%v rid=%s", status, body, rid)
	}
}

func TestRequestIDPassthrough(t *testing.T) {
	_, body, rid := do(t, newApp(), "/ok", map[string]string{HeaderRequestID: "req_client123"})
	if rid != "req_client123" || body["request_id"] != "req_client123" {
		t.Fatalf("request id klien tidak dipakai: %s", rid)
	}
}

func TestDomainErrorMapping(t *testing.T) {
	status, body, _ := do(t, newApp(), "/conflict", nil)
	e := body["error"].(map[string]any)
	if status != 409 || body["success"] != false || e["code"] != "VERSION_CONFLICT" {
		t.Fatalf("status=%d body=%v", status, body)
	}
}

func TestUnknownErrorIsHidden(t *testing.T) {
	status, body, _ := do(t, newApp(), "/boom", nil)
	e := body["error"].(map[string]any)
	if status != 500 || e["code"] != "INTERNAL_ERROR" || e["message"] == "db password=hunter2 bocor" {
		t.Fatalf("pesan internal bocor: status=%d body=%v", status, body)
	}
}

func TestNotFoundRoute(t *testing.T) {
	status, body, _ := do(t, newApp(), "/tidak-ada", nil)
	if status != 404 || body["error"].(map[string]any)["code"] != "NOT_FOUND" {
		t.Fatalf("status=%d body=%v", status, body)
	}
}

func TestPagination(t *testing.T) {
	_, body, _ := do(t, newApp(), "/page?page=2&per_page=20", nil)
	meta := body["meta"].(map[string]any)
	if meta["page"].(float64) != 2 || meta["total_pages"].(float64) != 7 {
		t.Fatalf("meta=%v", meta)
	}
	status, _, _ := do(t, newApp(), "/page?per_page=101", nil)
	if status != 422 {
		t.Fatalf("per_page > 100 harus 422, got %d", status)
	}
}
