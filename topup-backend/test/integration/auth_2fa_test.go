package integration

import (
	"encoding/base32"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/crypto"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/totp"
)

// totpAt menghitung kode TOTP dari secret base32 untuk langkah waktu sekarang + offset.
func totpAt(t *testing.T, secretB32 string, offset int64) string {
	t.Helper()
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secretB32)
	if err != nil {
		t.Fatal(err)
	}
	return totp.Code(secret, totp.Step(time.Now())+offset)
}

// resetStep melonggarkan penjaga replay agar satu test bisa memakai kode di langkah
// waktu yang sama berkali-kali (test replay sendiri tidak memanggil ini).
func (e *authEnv) resetStep(userID string) {
	e.t.Helper()
	if err := e.app.Platform.DB.Exec("UPDATE user_totp SET last_used_step = NULL WHERE user_id = ?", userID).Error; err != nil {
		e.t.Fatal(err)
	}
}

type setupData struct {
	OtpauthURL string `json:"otpauth_url"`
	Secret     string `json:"secret"`
}

// enable2FA menjalankan setup + enable dan mengembalikan secret dan recovery code.
func (e *authEnv) enable2FA(access, userID string) (string, []string) {
	e.t.Helper()
	e.flushRateLimits()
	st, env := e.call("POST", "/api/v1/auth/2fa/setup", nil, access)
	e.wantOK(st, env)
	var s setupData
	_ = json.Unmarshal(env.Data, &s)
	if !strings.HasPrefix(s.OtpauthURL, "otpauth://totp/RaCash:") || s.Secret == "" {
		e.t.Fatalf("setup: %+v", s)
	}
	st, env = e.call("POST", "/api/v1/auth/2fa/enable", map[string]any{"code": totpAt(e.t, s.Secret, 0)}, access)
	e.wantOK(st, env)
	var r struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	_ = json.Unmarshal(env.Data, &r)
	if len(r.RecoveryCodes) != 10 {
		e.t.Fatalf("recovery codes: %v", r.RecoveryCodes)
	}
	e.resetStep(userID)
	return s.Secret, r.RecoveryCodes
}

func (e *authEnv) status2FA(access string) (enabled, required bool, remaining int) {
	e.t.Helper()
	st, env := e.call("GET", "/api/v1/auth/2fa/status", nil, access)
	e.wantOK(st, env)
	var d struct {
		Enabled   bool `json:"enabled"`
		Required  bool `json:"required"`
		Remaining int  `json:"recovery_codes_remaining"`
	}
	_ = json.Unmarshal(env.Data, &d)
	return d.Enabled, d.Required, d.Remaining
}

type challengeData struct {
	TwoFactorRequired bool   `json:"two_factor_required"`
	ChallengeToken    string `json:"challenge_token"`
	AccessToken       string `json:"access_token"`
}

func (e *authEnv) loginChallenge(email string) challengeData {
	e.t.Helper()
	e.flushRateLimits()
	st, env := e.call("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": goodPassword}, "")
	e.wantOK(st, env)
	var d challengeData
	_ = json.Unmarshal(env.Data, &d)
	return d
}

func (e *authEnv) login2FA(challenge, code string) (int, envelope, loginData) {
	e.t.Helper()
	st, env := e.call("POST", "/api/v1/auth/login/2fa", map[string]any{"challenge_token": challenge, "code": code}, "")
	var d loginData
	if env.Success {
		_ = json.Unmarshal(env.Data, &d)
	}
	return st, env, d
}

func (e *authEnv) stepUp(access, code string) (int, envelope, string) {
	e.t.Helper()
	st, env := e.call("POST", "/api/v1/auth/step-up", map[string]any{"code": code}, access)
	var d struct {
		Token string `json:"step_up_token"`
	}
	if env.Success {
		_ = json.Unmarshal(env.Data, &d)
	}
	return st, env, d.Token
}

func (e *authEnv) callWithStepUp(method, path, access, stepUpToken string) (int, envelope) {
	e.t.Helper()
	return e.callHeaders(method, path, nil, access, map[string]string{"X-Step-Up-Token": stepUpToken})
}

func TestTwoFactorFullFlowForMember(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	e.registerVerified(email)
	d := e.loginOK(email)

	if en, req, _ := e.status2FA(d.AccessToken); en || req {
		t.Fatalf("status awal: enabled=%v required=%v", en, req)
	}

	// Enable sebelum setup, dan dengan kode salah, ditolak.
	e.flushRateLimits()
	st, env := e.call("POST", "/api/v1/auth/2fa/enable", map[string]any{"code": "123456"}, d.AccessToken)
	e.wantCode(st, env, 422, "VALIDATION_ERROR")

	secret, codes := e.enable2FA(d.AccessToken, d.User.ID)
	if en, _, rem := e.status2FA(d.AccessToken); !en || rem != 10 {
		t.Fatalf("status setelah enable: enabled=%v remaining=%d", en, rem)
	}

	// Setup ulang saat sudah aktif: 409; recovery code tersimpan sebagai hash.
	st, env = e.call("POST", "/api/v1/auth/2fa/setup", nil, d.AccessToken)
	e.wantCode(st, env, 409, "CONFLICT")
	var plain int64
	if err := e.app.Platform.DB.Raw("SELECT count(*) FROM user_recovery_codes WHERE code_hash = ?", codes[0]).Scan(&plain).Error; err != nil || plain != 0 {
		t.Fatalf("recovery code tersimpan sebagai plaintext (n=%d err=%v)", plain, err)
	}
	var enc int64
	if err := e.app.Platform.DB.Raw("SELECT count(*) FROM user_totp WHERE user_id = ? AND position(convert_to(?, 'UTF8') in secret_encrypted) > 0", d.User.ID, secret).Scan(&enc).Error; err != nil || enc != 0 {
		t.Fatalf("secret TOTP tersimpan sebagai plaintext (n=%d err=%v)", enc, err)
	}

	// Login sekarang butuh challenge; belum ada token.
	ch := e.loginChallenge(email)
	if !ch.TwoFactorRequired || !strings.HasPrefix(ch.ChallengeToken, "ch_") || ch.AccessToken != "" {
		t.Fatalf("challenge: %+v", ch)
	}
	st, env, _ = e.login2FA(ch.ChallengeToken, "000000")
	e.wantCode(st, env, 401, "INVALID_CREDENTIALS")
	st, env, full := e.login2FA(ch.ChallengeToken, totpAt(t, secret, 0))
	e.wantOK(st, env)
	if full.AccessToken == "" || full.User.Email != email {
		t.Fatalf("login 2fa: %+v", full)
	}
	// Challenge sekali pakai.
	e.resetStep(d.User.ID)
	st, env, _ = e.login2FA(ch.ChallengeToken, totpAt(t, secret, 0))
	e.wantCode(st, env, 401, "UNAUTHENTICATED")

	// Step-up: endpoint ★ ditolak tanpa/dengan token salah, diterima dengan token valid.
	e.resetStep(d.User.ID)
	st, env = e.call("POST", "/api/v1/auth/2fa/disable", nil, full.AccessToken)
	e.wantCode(st, env, 403, "STEP_UP_REQUIRED")
	st, env = e.callWithStepUp("POST", "/api/v1/auth/2fa/disable", full.AccessToken, "su_salah")
	e.wantCode(st, env, 403, "STEP_UP_REQUIRED")

	e.flushRateLimits()
	st, env, _ = e.stepUp(full.AccessToken, "000000")
	e.wantCode(st, env, 422, "VALIDATION_ERROR")
	st, env, su := e.stepUp(full.AccessToken, totpAt(t, secret, 0))
	e.wantOK(st, env)
	if !strings.HasPrefix(su, "su_") {
		t.Fatalf("step_up_token: %q", su)
	}

	// Token step-up terikat ke sesi pembuatnya: sesi lain dari user yang sama ditolak.
	other := e.loginChallenge(email)
	e.resetStep(d.User.ID)
	st, env, second := e.login2FA(other.ChallengeToken, totpAt(t, secret, 0))
	e.wantOK(st, env)
	st, env = e.callWithStepUp("POST", "/api/v1/auth/2fa/disable", second.AccessToken, su)
	e.wantCode(st, env, 403, "STEP_UP_REQUIRED")

	st, env = e.callWithStepUp("POST", "/api/v1/auth/2fa/disable", full.AccessToken, su)
	e.wantOK(st, env)
	if en, _, _ := e.status2FA(full.AccessToken); en {
		t.Fatal("2FA masih aktif setelah disable")
	}
	var left int64
	if err := e.app.Platform.DB.Raw("SELECT (SELECT count(*) FROM user_totp WHERE user_id = ?) + (SELECT count(*) FROM user_recovery_codes WHERE user_id = ?)", d.User.ID, d.User.ID).Scan(&left).Error; err != nil || left != 0 {
		t.Fatalf("sisa data 2FA setelah disable: %d (err=%v)", left, err)
	}
	// Login kembali tanpa challenge.
	if d2 := e.loginOK(email); d2.AccessToken == "" {
		t.Fatal("login setelah disable harus langsung dapat token")
	}
}

func TestTwoFactorReplayAndRecoveryCodes(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	e.registerVerified(email)
	d := e.loginOK(email)

	// Tanpa resetStep: kode yang dipakai untuk enable tidak boleh dipakai lagi.
	e.flushRateLimits()
	st, env := e.call("POST", "/api/v1/auth/2fa/setup", nil, d.AccessToken)
	e.wantOK(st, env)
	var s setupData
	_ = json.Unmarshal(env.Data, &s)
	code := totpAt(t, s.Secret, 0)
	st, env = e.call("POST", "/api/v1/auth/2fa/enable", map[string]any{"code": code}, d.AccessToken)
	e.wantOK(st, env)
	var r struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	_ = json.Unmarshal(env.Data, &r)

	ch := e.loginChallenge(email)
	st, env, _ = e.login2FA(ch.ChallengeToken, code)
	e.wantCode(st, env, 401, "INVALID_CREDENTIALS")

	// Recovery code: format dilonggarkan (huruf kecil, tanpa tanda hubung), sekali pakai.
	rc := r.RecoveryCodes[0]
	loose := strings.ToLower(strings.ReplaceAll(rc, "-", " "))
	st, env, _ = e.login2FA(ch.ChallengeToken, loose)
	e.wantOK(st, env)
	fresh := e.loginChallenge(email)
	if st, _, ok := e.login2FA(fresh.ChallengeToken, r.RecoveryCodes[1]); st != http.StatusOK || ok.AccessToken == "" {
		t.Fatal("recovery code kedua harus bisa dipakai")
	}
	ch2 := e.loginChallenge(email)
	st, env, _ = e.login2FA(ch2.ChallengeToken, rc)
	e.wantCode(st, env, 401, "INVALID_CREDENTIALS")

	// Step-up tidak menerima recovery code.
	st, env, _ = e.stepUp(d.AccessToken, r.RecoveryCodes[2])
	e.wantCode(st, env, 422, "VALIDATION_ERROR")
}

func TestTwoFactorChallengeAttemptsExhausted(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	e.registerVerified(email)
	d := e.loginOK(email)
	secret, _ := e.enable2FA(d.AccessToken, d.User.ID)

	ch := e.loginChallenge(email)
	e.flushRateLimits()
	for i := 0; i < 5; i++ {
		st, env, _ := e.login2FA(ch.ChallengeToken, "000000")
		e.wantCode(st, env, 401, "INVALID_CREDENTIALS")
	}
	// Setelah 5 kali salah, challenge hangus walau kodenya benar.
	st, env, _ := e.login2FA(ch.ChallengeToken, totpAt(t, secret, 0))
	e.wantCode(st, env, 401, "UNAUTHENTICATED")

	st, env, _ = e.login2FA("ch_tidak-ada", "123456")
	e.wantCode(st, env, 401, "UNAUTHENTICATED")
}

func TestStepUpRequiresTwoFactorAndIsBoundToUser(t *testing.T) {
	e := newAuthEnv(t)
	a, b := uniqueEmail(), uniqueEmail()
	e.registerVerified(a)
	e.registerVerified(b)
	da, db := e.loginOK(a), e.loginOK(b)

	// Tanpa 2FA aktif, step-up tidak mungkin.
	e.flushRateLimits()
	st, env, _ := e.stepUp(da.AccessToken, "123456")
	e.wantCode(st, env, 422, "VALIDATION_ERROR")

	secret, _ := e.enable2FA(da.AccessToken, da.User.ID)
	st, env, su := e.stepUp(da.AccessToken, totpAt(t, secret, 0))
	e.wantOK(st, env)

	// Token milik A dipakai B → ditolak.
	e.enable2FA(db.AccessToken, db.User.ID)
	st, env = e.callWithStepUp("POST", "/api/v1/auth/2fa/disable", db.AccessToken, su)
	e.wantCode(st, env, 403, "STEP_UP_REQUIRED")

	// Token kedaluwarsa (kunci Redis hilang) → ditolak.
	if err := e.app.Platform.Redis.Del(t.Context(), "auth:stepup:"+crypto.SHA256Hex([]byte(su))).Err(); err != nil {
		t.Fatal(err)
	}
	st, env = e.callWithStepUp("POST", "/api/v1/auth/2fa/disable", da.AccessToken, su)
	e.wantCode(st, env, 403, "STEP_UP_REQUIRED")
}

func TestInternalUserMustSetUpTwoFactor(t *testing.T) {
	e := newAuthEnv(t)
	email := uniqueEmail()
	e.registerVerified(email)
	var userID string
	if err := e.app.Platform.DB.Raw("SELECT id::text FROM users WHERE email = ?", email).Scan(&userID).Error; err != nil {
		t.Fatal(err)
	}
	if err := e.app.Platform.DB.Exec(`INSERT INTO user_roles (user_id, role_id, assigned_at)
		SELECT ?, id, now() FROM roles WHERE code = 'admin'`, userID).Error; err != nil {
		t.Fatal(err)
	}

	e.flushRateLimits()
	st, env, d := e.login(email, goodPassword)
	e.wantOK(st, env)
	var raw struct {
		SetupRequired bool `json:"two_factor_setup_required"`
	}
	_ = json.Unmarshal(env.Data, &raw)
	if !raw.SetupRequired || d.AccessToken == "" {
		t.Fatalf("login staf tanpa 2FA: setup_required=%v token=%v", raw.SetupRequired, d.AccessToken != "")
	}

	// Token terbatas: hanya /auth/2fa/* (dan logout).
	st, env = e.call("GET", "/api/v1/me/sessions", nil, d.AccessToken)
	e.wantCode(st, env, 403, "PERMISSION_DENIED")
	st, env = e.call("POST", "/api/v1/auth/step-up", map[string]any{"code": "123456"}, d.AccessToken)
	e.wantCode(st, env, 403, "PERMISSION_DENIED")
	if en, req, _ := e.status2FA(d.AccessToken); en || !req {
		t.Fatalf("status: enabled=%v required=%v", en, req)
	}

	// Refresh tidak boleh melonggarkan pembatasan.
	st, env, rd := e.refresh(d.RefreshToken)
	e.wantOK(st, env)
	_ = json.Unmarshal(env.Data, &raw)
	if !raw.SetupRequired {
		t.Fatal("refresh melonggarkan token terbatas")
	}
	st, env = e.call("GET", "/api/v1/me/sessions", nil, rd.AccessToken)
	e.wantCode(st, env, 403, "PERMISSION_DENIED")

	// Setelah 2FA aktif, token yang sama langsung berlaku penuh.
	secret, _ := e.enable2FA(rd.AccessToken, userID)
	st, env = e.call("GET", "/api/v1/me/sessions", nil, rd.AccessToken)
	e.wantOK(st, env)

	// Staf internal tidak boleh mematikan 2FA, walau punya token step-up.
	st, env, su := e.stepUp(rd.AccessToken, totpAt(t, secret, 0))
	e.wantOK(st, env)
	st, env = e.callWithStepUp("POST", "/api/v1/auth/2fa/disable", rd.AccessToken, su)
	e.wantCode(st, env, 403, "PERMISSION_DENIED")
	if en, _, _ := e.status2FA(rd.AccessToken); !en {
		t.Fatal("2FA staf internal termatikan")
	}

	// Login berikutnya: challenge dulu, lalu token penuh tanpa flag setup.
	ch := e.loginChallenge(email)
	e.resetStep(userID)
	st, env, full := e.login2FA(ch.ChallengeToken, totpAt(t, secret, 0))
	e.wantOK(st, env)
	var after struct {
		SetupRequired bool `json:"two_factor_setup_required"`
	}
	_ = json.Unmarshal(env.Data, &after)
	if after.SetupRequired || full.AccessToken == "" {
		t.Fatalf("login 2fa staf: %+v", after)
	}
	if !slices.Contains(full.User.Roles, "admin") {
		t.Fatalf("roles: %v", full.User.Roles)
	}
}
