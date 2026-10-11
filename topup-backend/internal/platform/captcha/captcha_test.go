package captcha

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/httpx"
)

type stub struct{ err error }

func (s stub) Verify(context.Context, string, string) error { return s.err }

func run(t *testing.T, v Verifier, token string) int {
	t.Helper()
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	app.Post("/", Middleware(v), func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if token != "" {
		req.Header.Set(HeaderToken, token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode
}

func TestMiddleware(t *testing.T) {
	cases := []struct {
		name  string
		v     Verifier
		token string
		want  int
	}{
		{"disabled tanpa token", Disabled{}, "", 200},
		{"token kosong", stub{}, "", 422},
		{"token valid", stub{}, "abc", 200},
		{"token ditolak", stub{ErrRejected}, "abc", 422},
		{"vendor mati → fail closed", stub{context.DeadlineExceeded}, "abc", 503},
	}
	for _, c := range cases {
		if got := run(t, c.v, c.token); got != c.want {
			t.Errorf("%s: status %d, want %d", c.name, got, c.want)
		}
	}
}

func TestTurnstileVerify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("response") == "good" {
			_, _ = w.Write([]byte(`{"success":true}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":false}`))
	}))
	defer srv.Close()
	tv := NewTurnstile("secret", srv.URL)
	if err := tv.Verify(context.Background(), "good", "1.1.1.1"); err != nil {
		t.Fatalf("token benar ditolak: %v", err)
	}
	if err := tv.Verify(context.Background(), "bad", ""); err != ErrRejected {
		t.Fatalf("token salah: %v", err)
	}
}
