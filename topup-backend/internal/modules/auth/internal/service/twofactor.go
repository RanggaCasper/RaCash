package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/user"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/crypto"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/totp"
)

// Batas percobaan kode 2FA per user, di luar batas per IP dan per challenge.
const (
	codeUserLimit  = 10
	codeUserWindow = 5 * time.Minute
)

var errBadCode = errors.New("kode 2FA salah")

func invalidCode() error {
	return errs.Validation(errs.Detail{Field: "code", Code: "invalid_code", Message: "Kode salah atau sudah dipakai."})
}

// redisKey memakai hash token sehingga token mentah tidak pernah tersimpan di Redis.
func redisKey(kind, token string) string {
	return "auth:" + kind + ":" + crypto.SHA256Hex([]byte(token))
}

// ---- login dengan 2FA ----

type challengeData struct {
	UserID uuid.UUID `json:"user_id"`
}

// beginChallenge menyimpan challenge login di Redis; belum ada sesi/token.
func (s *Service) beginChallenge(ctx context.Context, u user.User) (domain.LoginResult, error) {
	raw, err := crypto.RandomToken(domain.ChallengePrefix, 32)
	if err != nil {
		return domain.LoginResult{}, err
	}
	val, err := json.Marshal(challengeData{UserID: u.ID})
	if err != nil {
		return domain.LoginResult{}, err
	}
	if err := s.rdb.Set(ctx, redisKey("2fa:ch", raw), val, domain.ChallengeTTL).Err(); err != nil {
		return domain.LoginResult{}, errs.Wrap(errs.ServiceUnavailable, err, "")
	}
	return domain.LoginResult{
		TwoFactorRequired: true, ChallengeToken: raw,
		ChallengeExpiresAt: s.clock.Now().Add(domain.ChallengeTTL),
	}, nil
}

var errChallengeExpired = errs.New(errs.Unauthenticated, "Sesi login 2FA kedaluwarsa. Silakan masuk ulang.")

// Login2FA menyelesaikan login dengan kode TOTP atau recovery code.
func (s *Service) Login2FA(ctx context.Context, challenge, code string, meta domain.SessionMeta) (domain.LoginResult, error) {
	key := redisKey("2fa:ch", challenge)
	raw, err := s.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return domain.LoginResult{}, errChallengeExpired
	}
	if err != nil {
		return domain.LoginResult{}, errs.Wrap(errs.ServiceUnavailable, err, "")
	}
	var ch challengeData
	if err := json.Unmarshal(raw, &ch); err != nil {
		return domain.LoginResult{}, err
	}

	// Batas percobaan per challenge: setelah habis, challenge dihanguskan.
	attKey := key + ":att"
	n, err := s.rdb.Incr(ctx, attKey).Result()
	if err != nil {
		return domain.LoginResult{}, errs.Wrap(errs.ServiceUnavailable, err, "")
	}
	if n == 1 {
		s.rdb.Expire(ctx, attKey, domain.ChallengeTTL)
	}
	if n > domain.MaxChallengeAttempts {
		s.rdb.Del(ctx, key, attKey)
		return domain.LoginResult{}, errChallengeExpired
	}
	if !s.allow(ctx, "2fa_user", ch.UserID.String(), codeUserLimit, codeUserWindow) {
		return domain.LoginResult{}, errs.New(errs.RateLimited, "")
	}

	u, err := s.users.Get(ctx, ch.UserID)
	if errs.Is(err, errs.NotFound) {
		return domain.LoginResult{}, errChallengeExpired
	}
	if err != nil {
		return domain.LoginResult{}, err
	}
	if u.Status == user.StatusSuspended {
		return domain.LoginResult{}, errs.New(errs.AccountSuspended, "")
	}

	switch err := s.checkCode(ctx, u.ID, code, true, true); {
	case errors.Is(err, errBadCode):
		return domain.LoginResult{}, errs.New(errs.InvalidCredentials, "Kode 2FA salah.")
	case err != nil:
		return domain.LoginResult{}, err
	}

	// Hanya satu request yang boleh menukar challenge ini dengan sesi.
	deleted, err := s.rdb.Del(ctx, key).Result()
	if err != nil {
		return domain.LoginResult{}, errs.Wrap(errs.ServiceUnavailable, err, "")
	}
	if deleted == 0 {
		return domain.LoginResult{}, errChallengeExpired
	}
	s.rdb.Del(ctx, attKey)

	roles, err := s.roles.RolesOf(ctx, u.ID)
	if err != nil {
		return domain.LoginResult{}, err
	}
	var res domain.LoginResult
	err = s.tx.Do(ctx, func(ctx context.Context) error {
		var err error
		res, err = s.startSession(ctx, u, roles, meta, false)
		if err != nil {
			return err
		}
		return s.users.RecordLogin(ctx, u.ID)
	})
	return res, err
}

// ---- verifikasi kode ----

// checkCode memverifikasi kode TOTP (6 digit) atau, bila allowRecovery, recovery
// code. wantConfirmed membedakan rahasia aktif (login, step-up) dari rahasia yang
// sedang dikonfirmasi (enable). Kode yang sama tidak bisa dipakai dua kali.
// Mengembalikan errBadCode untuk kode yang salah.
func (s *Service) checkCode(ctx context.Context, userID uuid.UUID, code string, allowRecovery, wantConfirmed bool) error {
	code = strings.TrimSpace(code)
	row, err := s.repo.FindTOTP(ctx, userID)
	if errs.Is(err, errs.NotFound) {
		return errBadCode
	}
	if err != nil {
		return err
	}
	if row.Confirmed != wantConfirmed {
		return errBadCode
	}
	now := s.clock.Now()

	if domain.IsTOTPCode(code) {
		secret, err := s.keyring.Decrypt(row.SecretEncrypted, row.SecretKeyID)
		if err != nil {
			return err
		}
		step, ok := totp.Match(secret, code, now, domain.TOTPSkew)
		if !ok {
			return errBadCode
		}
		claimed, err := s.repo.ClaimStep(ctx, userID, step, now)
		if err != nil {
			return err
		}
		if !claimed {
			return errBadCode // replay: langkah ini (atau yang lebih baru) sudah dipakai
		}
		return nil
	}

	if allowRecovery && wantConfirmed {
		norm := domain.NormalizeRecoveryCode(code)
		if len(norm) != 10 {
			return errBadCode
		}
		used, err := s.repo.ConsumeRecoveryCode(ctx, userID, crypto.SHA256Hex([]byte(norm)), now)
		if err != nil {
			return err
		}
		if used {
			return nil
		}
	}
	return errBadCode
}

// ---- setup, enable, disable ----

type TwoFactorSetup struct {
	OtpauthURL string
	Secret     string
}

// SetupTwoFactor membuat rahasia TOTP baru (belum aktif). Boleh diulang selama belum di-enable.
func (s *Service) SetupTwoFactor(ctx context.Context, p domain.Principal) (TwoFactorSetup, error) {
	u, err := s.users.Get(ctx, p.UserID)
	if err != nil {
		return TwoFactorSetup{}, err
	}
	if u.TwoFactorEnabled {
		return TwoFactorSetup{}, errs.New(errs.Conflict, "2FA sudah aktif.")
	}
	secret, err := totp.GenerateSecret()
	if err != nil {
		return TwoFactorSetup{}, err
	}
	enc, keyID, err := s.keyring.Encrypt(secret)
	if err != nil {
		return TwoFactorSetup{}, err
	}
	if err := s.repo.UpsertPendingTOTP(ctx, p.UserID, enc, keyID, s.clock.Now()); err != nil {
		return TwoFactorSetup{}, err
	}
	return TwoFactorSetup{OtpauthURL: totp.URL(domain.TOTPIssuer, u.Email, secret), Secret: totp.EncodeSecret(secret)}, nil
}

// EnableTwoFactor mengaktifkan 2FA dengan kode pertama dan mengembalikan
// recovery code (hanya ditampilkan sekali; di DB hanya hash-nya).
func (s *Service) EnableTwoFactor(ctx context.Context, p domain.Principal, code string) ([]string, error) {
	if !s.allow(ctx, "2fa_user", p.UserID.String(), codeUserLimit, codeUserWindow) {
		return nil, errs.New(errs.RateLimited, "")
	}
	var codes []string
	err := s.tx.Do(ctx, func(ctx context.Context) error {
		row, err := s.repo.FindTOTP(ctx, p.UserID)
		if errs.Is(err, errs.NotFound) {
			return errs.Validation(errs.Detail{Field: "code", Code: "two_factor_not_set_up", Message: "Jalankan setup 2FA terlebih dahulu."})
		}
		if err != nil {
			return err
		}
		if row.Confirmed {
			return errs.New(errs.Conflict, "2FA sudah aktif.")
		}
		switch err := s.checkCode(ctx, p.UserID, code, false, false); {
		case errors.Is(err, errBadCode):
			return invalidCode()
		case err != nil:
			return err
		}

		now := s.clock.Now()
		if err := s.repo.ConfirmTOTP(ctx, p.UserID, now); err != nil {
			return err
		}
		if err := s.users.SetTwoFactorEnabled(ctx, p.UserID, true); err != nil {
			return err
		}
		hashes := make([]string, domain.RecoveryCodeCount)
		codes = make([]string, domain.RecoveryCodeCount)
		for i := range codes {
			c, err := domain.NewRecoveryCode()
			if err != nil {
				return err
			}
			codes[i] = c
			hashes[i] = crypto.SHA256Hex([]byte(domain.NormalizeRecoveryCode(c)))
		}
		return s.repo.ReplaceRecoveryCodes(ctx, p.UserID, hashes, now)
	})
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// DisableTwoFactor mematikan 2FA. Pemanggil harus sudah melewati middleware
// StepUp. User internal tidak boleh mematikan 2FA (USR-06).
func (s *Service) DisableTwoFactor(ctx context.Context, p domain.Principal) error {
	roles, err := s.roles.RolesOf(ctx, p.UserID)
	if err != nil {
		return err
	}
	if hasInternal(roles) {
		return errs.New(errs.PermissionDenied, "2FA wajib untuk akun internal dan tidak dapat dimatikan.")
	}
	return s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.DeleteTOTP(ctx, p.UserID); err != nil {
			return err
		}
		return s.users.SetTwoFactorEnabled(ctx, p.UserID, false)
	})
}

type TwoFactorStatus struct {
	Enabled                bool
	Required               bool
	RecoveryCodesRemaining int
}

// TwoFactorStatus melaporkan status 2FA; boleh dipanggil dengan token terbatas.
func (s *Service) TwoFactorStatus(ctx context.Context, p domain.Principal) (TwoFactorStatus, error) {
	u, err := s.users.Get(ctx, p.UserID)
	if err != nil {
		return TwoFactorStatus{}, err
	}
	roles, err := s.roles.RolesOf(ctx, p.UserID)
	if err != nil {
		return TwoFactorStatus{}, err
	}
	st := TwoFactorStatus{Enabled: u.TwoFactorEnabled, Required: hasInternal(roles)}
	if u.TwoFactorEnabled {
		n, err := s.repo.CountRecoveryCodes(ctx, p.UserID)
		if err != nil {
			return TwoFactorStatus{}, err
		}
		st.RecoveryCodesRemaining = int(n)
	}
	return st, nil
}

// ---- step-up ----

type stepUpData struct {
	UserID    uuid.UUID `json:"user_id"`
	SessionID uuid.UUID `json:"session_id"`
}

// StepUp menukar kode TOTP dengan step_up_token (5 menit) untuk endpoint ★.
// Recovery code sengaja tidak diterima di sini (kontrak: TotpCode 6 digit).
func (s *Service) StepUp(ctx context.Context, p domain.Principal, code string) (string, time.Time, error) {
	u, err := s.users.Get(ctx, p.UserID)
	if err != nil {
		return "", time.Time{}, err
	}
	if !u.TwoFactorEnabled {
		return "", time.Time{}, errs.Validation(errs.Detail{Field: "code", Code: "two_factor_not_enabled", Message: "Aktifkan 2FA terlebih dahulu."})
	}
	if !s.allow(ctx, "2fa_user", p.UserID.String(), codeUserLimit, codeUserWindow) {
		return "", time.Time{}, errs.New(errs.RateLimited, "")
	}
	switch err := s.checkCode(ctx, p.UserID, code, false, true); {
	case errors.Is(err, errBadCode):
		return "", time.Time{}, invalidCode()
	case err != nil:
		return "", time.Time{}, err
	}

	raw, err := crypto.RandomToken(domain.StepUpPrefix, 32)
	if err != nil {
		return "", time.Time{}, err
	}
	val, err := json.Marshal(stepUpData{UserID: p.UserID, SessionID: p.SessionID})
	if err != nil {
		return "", time.Time{}, err
	}
	if err := s.rdb.Set(ctx, redisKey("stepup", raw), val, domain.StepUpTTL).Err(); err != nil {
		return "", time.Time{}, errs.Wrap(errs.ServiceUnavailable, err, "")
	}
	return raw, s.clock.Now().Add(domain.StepUpTTL), nil
}

// CheckStepUp memastikan token step-up masih berlaku dan milik user dan sesi yang sama.
func (s *Service) CheckStepUp(ctx context.Context, p domain.Principal, token string) error {
	if token == "" {
		return errs.New(errs.StepUpRequired, "")
	}
	raw, err := s.rdb.Get(ctx, redisKey("stepup", token)).Bytes()
	if errors.Is(err, redis.Nil) {
		return errs.New(errs.StepUpRequired, "")
	}
	if err != nil {
		return errs.Wrap(errs.ServiceUnavailable, err, "")
	}
	var d stepUpData
	if err := json.Unmarshal(raw, &d); err != nil || d.UserID != p.UserID || d.SessionID != p.SessionID {
		return errs.New(errs.StepUpRequired, "")
	}
	return nil
}
