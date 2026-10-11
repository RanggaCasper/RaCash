// Integration test modul auth: alur HTTP lengkap terhadap PostgreSQL dan Redis
// sungguhan, melewati bootstrap aplikasi yang sama dengan produksi.
// Butuh TEST_DB_* (lihat platform_test.go) dan TEST_REDIS_URL; tanpa
// keduanya test di-skip.
package integration

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/app"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/config"
)

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   struct {
		Code    string `json:"code"`
		Details []struct {
			Field string `json:"field"`
		} `json:"details"`
	} `json:"error"`
}

type authEnv struct {
	t    *testing.T
	app  *app.App
	http *fiber.App
}

const testJWTSecret = "integration-test-secret-0123456789abcdef"

func newAuthEnv(t *testing.T) *authEnv {
	t.Helper()
	if os.Getenv("TEST_DB_NAME") == "" || os.Getenv("TEST_REDIS_URL") == "" {
		t.Skip("TEST_DB_NAME / TEST_REDIS_URL tidak di-set")
	}
	for _, k := range []string{"HOST", "PORT", "USER", "PASSWORD", "NAME", "SSLMODE"} {
		if v := os.Getenv("TEST_DB_" + k); v != "" {
			t.Setenv("DB_"+k, v)
		}
	}
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	t.Setenv("APP_ENV", "local")
	t.Setenv("DB_AUTO_CREATE", "false")
	t.Setenv("REDIS_URL", os.Getenv("TEST_REDIS_URL"))
	t.Setenv("ENCRYPTION_KEYS", "k1:"+base64.StdEncoding.EncodeToString(key))
	t.Setenv("ENCRYPTION_ACTIVE_KEY_ID", "k1")
	t.Setenv("JWT_SECRET", testJWTSecret)
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	a, err := app.Bootstrap(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	e := &authEnv{t: t, app: a, http: a.HTTP()}
	e.flushRateLimits()
	return e
}

// flushRateLimits menghapus counter rate limit supaya test tidak saling mengganggu
// (semua request test datang dari IP yang sama).
func (e *authEnv) flushRateLimits() {
	e.t.Helper()
	ctx := context.Background()
	iter := e.app.Platform.Redis.Scan(ctx, 0, "rl:*", 200).Iterator()
	for iter.Next(ctx) {
		e.app.Platform.Redis.Del(ctx, iter.Val())
	}
	if err := iter.Err(); err != nil {
		e.t.Fatal(err)
	}
}

func (e *authEnv) call(method, path string, body any, bearer string) (int, envelope) {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = strings.NewReader(string(b))
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "integration-test")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := e.http.Test(req, 10000)
	if err != nil {
		e.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var env envelope
	raw, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(raw, &env); err != nil {
		e.t.Fatalf("%s %s: respons bukan envelope: %s", method, path, raw)
	}
	return resp.StatusCode, env
}

func (e *authEnv) wantCode(status int, env envelope, wantStatus int, wantCode string) {
	e.t.Helper()
	if status != wantStatus || env.Error.Code != wantCode {
		e.t.Fatalf("status=%d code=%q, want %d %q", status, env.Error.Code, wantStatus, wantCode)
	}
}

func (e *authEnv) wantOK(status int, env envelope) {
	e.t.Helper()
	if !env.Success || status >= 300 {
		e.t.Fatalf("status=%d success=%v code=%q, want sukses", status, env.Success, env.Error.Code)
	}
}

type loginData struct {
	TwoFactorRequired bool   `json:"two_factor_required"`
	AccessToken       string `json:"access_token"`
	RefreshToken      string `json:"refresh_token"`
	User              struct {
		ID    string   `json:"id"`
		Email string   `json:"email"`
		Tier  string   `json:"tier"`
		Roles []string `json:"roles"`
	} `json:"user"`
}

const goodPassword = "S3cure!pass"

func uniqueEmail() string { return "u-" + uuid.NewString()[:8] + "@example.com" }

// outboxToken membaca token mentah dari event outbox terbaru untuk email tersebut.
func (e *authEnv) outboxToken(eventType, email string) string {
	e.t.Helper()
	var payload string
	err := e.app.Platform.DB.Raw(`SELECT payload::text FROM outbox_events
		WHERE event_type = ? AND payload->'payload'->>'email' = ? ORDER BY created_at DESC LIMIT 1`,
		eventType, email).Scan(&payload).Error
	if err != nil || payload == "" {
		e.t.Fatalf("event %s untuk %s tidak ditemukan (err=%v)", eventType, email, err)
	}
	var env struct {
		Payload struct {
			Token string `json:"token"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(payload), &env); err != nil {
		e.t.Fatal(err)
	}
	return env.Payload.Token
}

func (e *authEnv) outboxCount(eventType, email string) int64 {
	e.t.Helper()
	var n int64
	err := e.app.Platform.DB.Raw(`SELECT count(*) FROM outbox_events
		WHERE event_type = ? AND payload->'payload'->>'email' = ?`, eventType, email).Scan(&n).Error
	if err != nil {
		e.t.Fatal(err)
	}
	return n
}

// registerVerified mendaftar lalu memverifikasi email lewat alur HTTP biasa.
func (e *authEnv) registerVerified(email string) {
	e.t.Helper()
	e.flushRateLimits()
	st, env := e.call("POST", "/api/v1/auth/register",
		map[string]any{"name": "Tester", "email": email, "phone": "081234567890", "password": goodPassword}, "")
	e.wantOK(st, env)
	st, env = e.call("POST", "/api/v1/auth/verify-email",
		map[string]any{"token": e.outboxToken("email_verification.requested", email)}, "")
	e.wantOK(st, env)
}

func (e *authEnv) login(email, password string) (int, envelope, loginData) {
	e.t.Helper()
	st, env := e.call("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": password}, "")
	var d loginData
	if env.Success {
		if err := json.Unmarshal(env.Data, &d); err != nil {
			e.t.Fatal(err)
		}
	}
	return st, env, d
}

func (e *authEnv) loginOK(email string) loginData {
	e.t.Helper()
	e.flushRateLimits()
	st, env, d := e.login(email, goodPassword)
	e.wantOK(st, env)
	return d
}

func (e *authEnv) refresh(rt string) (int, envelope, loginData) {
	e.t.Helper()
	st, env := e.call("POST", "/api/v1/auth/refresh", map[string]any{"refresh_token": rt}, "")
	var d loginData
	if env.Success {
		_ = json.Unmarshal(env.Data, &d)
	}
	return st, env, d
}

type sessionRow struct {
	ID        string `json:"id"`
	IsCurrent bool   `json:"is_current"`
}

func (e *authEnv) sessions(access string) []sessionRow {
	e.t.Helper()
	st, env := e.call("GET", "/api/v1/me/sessions", nil, access)
	e.wantOK(st, env)
	var rows []sessionRow
	if err := json.Unmarshal(env.Data, &rows); err != nil {
		e.t.Fatal(err)
	}
	return rows
}

func TestAuthRegisterVerifyLogin(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()

	st, env := e.call("POST", "/api/v1/auth/register",
		map[string]any{"name": "Rangga", "email": strings.ToUpper(email), "phone": "081234567890", "password": goodPassword}, "")
	e.wantOK(st, env)
	var reg struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(env.Data, &reg)
	if st != http.StatusCreated || reg.Status != "PENDING_VERIFICATION" {
		t.Fatalf("register: status=%d data=%s", st, env.Data)
	}

	st, env = e.call("POST", "/api/v1/auth/register",
		map[string]any{"name": "Dup", "email": email, "password": goodPassword}, "")
	e.wantCode(st, env, 409, "CONFLICT")

	// Belum diverifikasi: password benar tetap ditolak dengan EMAIL_NOT_VERIFIED.
	st, env, _ = e.login(email, goodPassword)
	e.wantCode(st, env, 403, "EMAIL_NOT_VERIFIED")

	token := e.outboxToken("email_verification.requested", email)
	st, env = e.call("POST", "/api/v1/auth/verify-email", map[string]any{"token": token}, "")
	e.wantOK(st, env)
	// Token sekali pakai.
	st, env = e.call("POST", "/api/v1/auth/verify-email", map[string]any{"token": token}, "")
	e.wantCode(st, env, 422, "VALIDATION_ERROR")

	st, env, d := e.login(email, goodPassword)
	e.wantOK(st, env)
	if d.TwoFactorRequired || d.AccessToken == "" || !strings.HasPrefix(d.RefreshToken, "rt_") {
		t.Fatalf("login: %+v", d)
	}
	if d.User.Email != email || d.User.Tier != "MEMBER" || len(d.User.Roles) != 1 || d.User.Roles[0] != "member" {
		t.Fatalf("user: %+v", d.User)
	}

	// Status akun menjadi ACTIVE dan last_login_at terisi.
	var row struct {
		Status      string
		LastLoginAt *time.Time
	}
	if err := e.app.Platform.DB.Raw("SELECT status::text AS status, last_login_at FROM users WHERE email = ?", email).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != "ACTIVE" || row.LastLoginAt == nil {
		t.Fatalf("users row: %+v", row)
	}
}

func TestAuthRegisterValidation(t *testing.T) {
	e := newAuthEnv(t)
	cases := []struct {
		name  string
		body  map[string]any
		field string
	}{
		{"password tanpa angka", map[string]any{"name": "A", "email": uniqueEmail(), "password": "abcdefghij"}, "password"},
		{"password pendek", map[string]any{"name": "A", "email": uniqueEmail(), "password": "ab1"}, "password"},
		{"email salah", map[string]any{"name": "A", "email": "bukan-email", "password": goodPassword}, "email"},
		{"telepon salah", map[string]any{"name": "A", "email": uniqueEmail(), "phone": "12345", "password": goodPassword}, "phone"},
		{"nama kosong", map[string]any{"name": "  ", "email": uniqueEmail(), "password": goodPassword}, "name"},
	}
	for _, c := range cases {
		e.flushRateLimits()
		st, env := e.call("POST", "/api/v1/auth/register", c.body, "")
		e.wantCode(st, env, 422, "VALIDATION_ERROR")
		if len(env.Error.Details) == 0 || env.Error.Details[0].Field != c.field {
			t.Errorf("%s: details=%+v, want field %q", c.name, env.Error.Details, c.field)
		}
	}
}

func TestAuthLoginErrors(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	e.registerVerified(email)
	e.flushRateLimits()

	st, env, _ := e.login(email, "SalahTotal1")
	e.wantCode(st, env, 401, "INVALID_CREDENTIALS")
	st, env, _ = e.login(uniqueEmail(), goodPassword)
	e.wantCode(st, env, 401, "INVALID_CREDENTIALS")

	if err := e.app.Platform.DB.Exec("UPDATE users SET status = 'SUSPENDED' WHERE email = ?", email).Error; err != nil {
		t.Fatal(err)
	}
	st, env, _ = e.login(email, goodPassword)
	e.wantCode(st, env, 403, "ACCOUNT_SUSPENDED")
	// Password salah pada akun suspended tidak boleh membocorkan statusnya.
	st, env, _ = e.login(email, "SalahTotal1")
	e.wantCode(st, env, 401, "INVALID_CREDENTIALS")
}

func TestAuthLoginRateLimitPerEmail(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	var last int
	for i := 0; i < 11; i++ {
		last, _, _ = e.login(email, "SalahTotal1")
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("percobaan ke-11: status %d, want 429", last)
	}
}

func TestAuthAccessTokenChecks(t *testing.T) {
	e := newAuthEnv(t)

	st, env := e.call("GET", "/api/v1/me/sessions", nil, "")
	e.wantCode(st, env, 401, "UNAUTHENTICATED")
	st, env = e.call("GET", "/api/v1/me/sessions", nil, "bukan.jwt.valid")
	e.wantCode(st, env, 401, "UNAUTHENTICATED")

	expired, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": uuid.NewString(), "sid": uuid.NewString(), "pv": 1,
		"iat": time.Now().Add(-time.Hour).Unix(), "exp": time.Now().Add(-time.Minute).Unix(),
	}).SignedString([]byte(testJWTSecret))
	st, env = e.call("GET", "/api/v1/me/sessions", nil, expired)
	e.wantCode(st, env, 401, "TOKEN_EXPIRED")

	// JWT valid, tetapi sesinya tidak ada.
	orphan, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": uuid.NewString(), "sid": uuid.NewString(), "pv": 1,
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(),
	}).SignedString([]byte(testJWTSecret))
	st, env = e.call("GET", "/api/v1/me/sessions", nil, orphan)
	e.wantCode(st, env, 401, "UNAUTHENTICATED")
}

func TestAuthRefreshRotationAndReuseDetection(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	e.registerVerified(email)
	first := e.loginOK(email)

	st, env, second := e.refresh(first.RefreshToken)
	e.wantOK(st, env)
	if second.RefreshToken == first.RefreshToken || second.AccessToken == "" {
		t.Fatalf("refresh harus merotasi token: %+v", second)
	}
	// Sesi tetap satu, token akses baru berlaku.
	if rows := e.sessions(second.AccessToken); len(rows) != 1 || !rows[0].IsCurrent {
		t.Fatalf("sessions: %+v", rows)
	}

	// Token lama dipakai ulang → ditolak, dan sesi dicabut sehingga token baru ikut mati.
	st, env, _ = e.refresh(first.RefreshToken)
	e.wantCode(st, env, 401, "UNAUTHENTICATED")
	st, env, _ = e.refresh(second.RefreshToken)
	e.wantCode(st, env, 401, "UNAUTHENTICATED")
	st, env = e.call("GET", "/api/v1/me/sessions", nil, second.AccessToken)
	e.wantCode(st, env, 401, "UNAUTHENTICATED")

	var reason string
	if err := e.app.Platform.DB.Raw("SELECT revoked_reason FROM sessions WHERE user_id = ?", first.User.ID).Scan(&reason).Error; err != nil {
		t.Fatal(err)
	}
	if reason != "REUSE_DETECTED" {
		t.Fatalf("revoked_reason = %q", reason)
	}

	st, env, _ = e.refresh("rt_tidak-dikenal")
	e.wantCode(st, env, 401, "UNAUTHENTICATED")
}

func TestAuthRefreshExpiredSession(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	e.registerVerified(email)
	d := e.loginOK(email)
	if err := e.app.Platform.DB.Exec("UPDATE sessions SET expires_at = now() - interval '1 minute' WHERE user_id = ?", d.User.ID).Error; err != nil {
		t.Fatal(err)
	}
	st, env, _ := e.refresh(d.RefreshToken)
	e.wantCode(st, env, 401, "UNAUTHENTICATED")
}

func TestAuthConcurrentRefreshAllowsOnlyOne(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	e.registerVerified(email)
	d := e.loginOK(email)
	e.flushRateLimits()

	const n = 6
	codes := make(chan int, n)
	for i := 0; i < n; i++ {
		go func() {
			st, _, _ := e.refresh(d.RefreshToken)
			codes <- st
		}()
	}
	okCount := 0
	for i := 0; i < n; i++ {
		if <-codes == http.StatusOK {
			okCount++
		}
	}
	if okCount != 1 {
		t.Fatalf("refresh paralel dengan token sama: %d sukses, want tepat 1", okCount)
	}
}

func TestAuthLogoutAndSessions(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	e.registerVerified(email)
	a := e.loginOK(email)
	b := e.loginOK(email)
	c := e.loginOK(email)

	rows := e.sessions(a.AccessToken)
	if len(rows) != 3 {
		t.Fatalf("want 3 sesi, got %+v", rows)
	}
	var otherID string
	currents := 0
	for _, r := range rows {
		if r.IsCurrent {
			currents++
		} else {
			otherID = r.ID
		}
	}
	if currents != 1 {
		t.Fatalf("harus tepat satu sesi current, got %d", currents)
	}

	// Mencabut sesi lain; sesi itu tidak bisa refresh lagi.
	st, env := e.call("DELETE", "/api/v1/me/sessions/"+otherID, nil, a.AccessToken)
	e.wantOK(st, env)
	if got := len(e.sessions(a.AccessToken)); got != 2 {
		t.Fatalf("setelah cabut: %d sesi, want 2", got)
	}
	// Mencabut dua kali, ID acak, dan ID bukan UUID → NOT_FOUND.
	st, env = e.call("DELETE", "/api/v1/me/sessions/"+otherID, nil, a.AccessToken)
	e.wantCode(st, env, 404, "NOT_FOUND")
	st, env = e.call("DELETE", "/api/v1/me/sessions/"+uuid.NewString(), nil, a.AccessToken)
	e.wantCode(st, env, 404, "NOT_FOUND")
	st, env = e.call("DELETE", "/api/v1/me/sessions/xyz", nil, a.AccessToken)
	e.wantCode(st, env, 404, "NOT_FOUND")

	// Sesi milik user lain tidak bisa dicabut (404, bukan 403).
	other := uniqueEmail()
	e.registerVerified(other)
	od := e.loginOK(other)
	victim := e.sessions(od.AccessToken)[0].ID
	st, env = e.call("DELETE", "/api/v1/me/sessions/"+victim, nil, a.AccessToken)
	e.wantCode(st, env, 404, "NOT_FOUND")
	if got := len(e.sessions(od.AccessToken)); got != 1 {
		t.Fatalf("sesi user lain terpengaruh: %d", got)
	}

	// Logout mencabut sesi saat ini: access token langsung mati walau belum kedaluwarsa.
	st, env = e.call("POST", "/api/v1/auth/logout", nil, a.AccessToken)
	e.wantOK(st, env)
	st, env = e.call("GET", "/api/v1/me/sessions", nil, a.AccessToken)
	e.wantCode(st, env, 401, "UNAUTHENTICATED")
	st, env, _ = e.refresh(a.RefreshToken)
	e.wantCode(st, env, 401, "UNAUTHENTICATED")

	// Logout-all mencabut sisanya.
	// Salah satu dari b/c sudah dicabut di atas; pakai yang masih hidup.
	live := b.AccessToken
	if st, _ := e.call("GET", "/api/v1/me/sessions", nil, live); st != http.StatusOK {
		live = c.AccessToken
	}
	st, env = e.call("POST", "/api/v1/auth/logout-all", nil, live)
	e.wantOK(st, env)
	for _, tok := range []string{b.AccessToken, c.AccessToken} {
		st, env = e.call("GET", "/api/v1/me/sessions", nil, tok)
		e.wantCode(st, env, 401, "UNAUTHENTICATED")
	}

	st, env = e.call("POST", "/api/v1/auth/logout", nil, "")
	e.wantCode(st, env, 401, "UNAUTHENTICATED")
}

func TestAuthResendVerification(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	e.flushRateLimits()
	st, env := e.call("POST", "/api/v1/auth/register", map[string]any{"name": "A", "email": email, "password": goodPassword}, "")
	e.wantOK(st, env)

	st, env = e.call("POST", "/api/v1/auth/resend-verification", map[string]any{"email": email}, "")
	e.wantOK(st, env)
	if n := e.outboxCount("email_verification.requested", email); n != 2 {
		t.Fatalf("event verifikasi: %d, want 2 (register + resend)", n)
	}

	// Email tidak terdaftar: tetap 200, tanpa event (tidak membocorkan keberadaan akun).
	ghost := uniqueEmail()
	st, env = e.call("POST", "/api/v1/auth/resend-verification", map[string]any{"email": ghost}, "")
	e.wantOK(st, env)
	if n := e.outboxCount("email_verification.requested", ghost); n != 0 {
		t.Fatalf("email tak dikenal menghasilkan %d event", n)
	}

	// Sudah terverifikasi: tidak ada event baru.
	st, env = e.call("POST", "/api/v1/auth/verify-email",
		map[string]any{"token": e.outboxToken("email_verification.requested", email)}, "")
	e.wantOK(st, env)
	st, env = e.call("POST", "/api/v1/auth/resend-verification", map[string]any{"email": email}, "")
	e.wantOK(st, env)
	if n := e.outboxCount("email_verification.requested", email); n != 2 {
		t.Fatalf("email terverifikasi menghasilkan event baru: %d", n)
	}
}

func TestAuthForgotAndResetPassword(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	e.registerVerified(email)
	old := e.loginOK(email)
	e.flushRateLimits()

	// Email tidak dikenal: 200 tanpa event.
	ghost := uniqueEmail()
	st, env := e.call("POST", "/api/v1/auth/forgot-password", map[string]any{"email": ghost}, "")
	e.wantOK(st, env)
	if n := e.outboxCount("password_reset.requested", ghost); n != 0 {
		t.Fatalf("email tak dikenal menghasilkan %d event", n)
	}

	st, env = e.call("POST", "/api/v1/auth/forgot-password", map[string]any{"email": email}, "")
	e.wantOK(st, env)
	token := e.outboxToken("password_reset.requested", email)

	// Password lemah ditolak dan tidak menghanguskan token.
	st, env = e.call("POST", "/api/v1/auth/reset-password", map[string]any{"token": token, "new_password": "lemah"}, "")
	e.wantCode(st, env, 422, "VALIDATION_ERROR")
	st, env = e.call("POST", "/api/v1/auth/reset-password", map[string]any{"token": "pr_salah", "new_password": "BaruAman123"}, "")
	e.wantCode(st, env, 422, "VALIDATION_ERROR")

	st, env = e.call("POST", "/api/v1/auth/reset-password", map[string]any{"token": token, "new_password": "BaruAman123"}, "")
	e.wantOK(st, env)
	st, env = e.call("POST", "/api/v1/auth/reset-password", map[string]any{"token": token, "new_password": "LainLagi123"}, "")
	e.wantCode(st, env, 422, "VALIDATION_ERROR")

	// Semua sesi lama dicabut; password lama mati, password baru hidup.
	st, env = e.call("GET", "/api/v1/me/sessions", nil, old.AccessToken)
	e.wantCode(st, env, 401, "UNAUTHENTICATED")
	st, env, _ = e.refresh(old.RefreshToken)
	e.wantCode(st, env, 401, "UNAUTHENTICATED")
	e.flushRateLimits()
	st, env, _ = e.login(email, goodPassword)
	e.wantCode(st, env, 401, "INVALID_CREDENTIALS")
	st, env, _ = e.login(email, "BaruAman123")
	e.wantOK(st, env)
}

func TestAuthResetTokenExpired(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	e.registerVerified(email)
	e.flushRateLimits()
	st, env := e.call("POST", "/api/v1/auth/forgot-password", map[string]any{"email": email}, "")
	e.wantOK(st, env)
	token := e.outboxToken("password_reset.requested", email)
	if err := e.app.Platform.DB.Exec(`UPDATE password_resets SET expires_at = now() - interval '1 second'
		WHERE user_id = (SELECT id FROM users WHERE email = ?)`, email).Error; err != nil {
		t.Fatal(err)
	}
	st, env = e.call("POST", "/api/v1/auth/reset-password", map[string]any{"token": token, "new_password": "BaruAman123"}, "")
	e.wantCode(st, env, 422, "VALIDATION_ERROR")
}

func TestAuthTokensAreStoredHashed(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	e.registerVerified(email)
	d := e.loginOK(email)
	var n int64
	if err := e.app.Platform.DB.Raw(`SELECT count(*) FROM sessions
		WHERE refresh_token_hash = ? OR previous_token_hash = ?`, d.RefreshToken, d.RefreshToken).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("refresh token tersimpan sebagai plaintext")
	}
	var hash string
	if err := e.app.Platform.DB.Raw("SELECT password_hash FROM users WHERE email = ?", email).Scan(&hash).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("password_hash bukan argon2id: %.12s", hash)
	}
}
