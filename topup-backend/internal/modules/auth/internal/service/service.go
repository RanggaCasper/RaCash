// Package service berisi use case modul auth: registrasi, verifikasi email,
// login, sesi dengan rotasi refresh token, dan reset password.
package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/repository"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/token"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/rbac"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/user"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/clock"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/crypto"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/event"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/id"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/ratelimit"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/uow"
)

// Batas per email, di luar batas per IP di route (mencegah serangan terdistribusi
// ke satu akun). Window dihitung tetap oleh ratelimit.Limiter.
const (
	loginEmailLimit  = 10
	loginEmailWindow = 15 * time.Minute
	mailEmailLimit   = 3
	mailEmailWindow  = time.Hour
)

type Service struct {
	repo    *repository.Repo
	users   user.Service
	roles   rbac.Service
	tx      uow.Tx
	outbox  *event.Outbox
	tokens  *token.Issuer
	limiter *ratelimit.Limiter
	clock   clock.Clock
	log     *slog.Logger

	// dummyHash dipakai agar login ke email yang tidak ada memakan waktu yang
	// sama dengan email yang ada (mencegah enumerasi lewat timing).
	dummyHash string
}

type Deps struct {
	Repo    *repository.Repo
	Users   user.Service
	Roles   rbac.Service
	Tx      uow.Tx
	Outbox  *event.Outbox
	Tokens  *token.Issuer
	Limiter *ratelimit.Limiter
	Clock   clock.Clock
	Log     *slog.Logger
}

func New(d Deps) (*Service, error) {
	dummy, err := crypto.HashPassword("dummy-password-for-timing")
	if err != nil {
		return nil, err
	}
	return &Service{
		repo: d.Repo, users: d.Users, roles: d.Roles, tx: d.Tx, outbox: d.Outbox,
		tokens: d.Tokens, limiter: d.Limiter, clock: d.Clock, log: d.Log, dummyHash: dummy,
	}, nil
}

// ---- registrasi & verifikasi email ----

type RegisterInput struct {
	Name     string
	Email    string
	Phone    *string
	Password string
}

// Register membuat member PENDING_VERIFICATION, memberinya role member, dan
// menerbitkan event verifikasi email — semuanya dalam satu transaksi.
func (s *Service) Register(ctx context.Context, in RegisterInput) (user.User, error) {
	hash, err := crypto.HashPassword(in.Password)
	if err != nil {
		return user.User{}, err
	}
	var u user.User
	err = s.tx.Do(ctx, func(ctx context.Context) error {
		var err error
		u, err = s.users.Create(ctx, user.NewUser{
			Name: strings.TrimSpace(in.Name), Email: in.Email, Phone: in.Phone, PasswordHash: hash,
		})
		if err != nil {
			return err
		}
		if err := s.roles.AssignRole(ctx, u.ID, rbac.RoleMember); err != nil {
			return err
		}
		return s.issueVerification(ctx, u)
	})
	return u, err
}

// issueVerification membuat token verifikasi dan menerbitkan event. Harus di dalam transaksi.
func (s *Service) issueVerification(ctx context.Context, u user.User) error {
	raw, hash, err := newOneTimeToken(domain.EmailVerifyTokenPrefix)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	exp := now.Add(domain.EmailVerificationTTL)
	rowID, err := s.repo.CreateEmailVerification(ctx, u.ID, hash, exp, now)
	if err != nil {
		return err
	}
	return s.publishToken(ctx, domain.EventEmailVerificationReq, domain.AggregateEmailVerification, rowID, u, raw, exp, now)
}

func (s *Service) publishToken(ctx context.Context, typ, aggregate string, rowID uuid.UUID, u user.User, raw string, exp, now time.Time) error {
	ev, err := event.New(domain.EventModule, typ, aggregate, rowID, domain.TokenEventPayload{
		UserID: u.ID, Name: u.Name, Email: u.Email, Token: raw, ExpiresAt: exp,
	}, now)
	if err != nil {
		return err
	}
	return s.outbox.Publish(ctx, ev)
}

var errInvalidToken = errs.Validation(errs.Detail{
	Field: "token", Code: "invalid_token", Message: "Token tidak valid atau sudah kedaluwarsa.",
})

// VerifyEmail memakai token verifikasi (sekali pakai) dan mengaktifkan akun.
func (s *Service) VerifyEmail(ctx context.Context, rawToken string) error {
	return s.tx.Do(ctx, func(ctx context.Context) error {
		ot, err := s.repo.ConsumeEmailVerification(ctx, crypto.SHA256Hex([]byte(rawToken)), s.clock.Now())
		if errs.Is(err, errs.NotFound) {
			return errInvalidToken
		}
		if err != nil {
			return err
		}
		return s.users.MarkEmailVerified(ctx, ot.UserID)
	})
}

// ResendVerification selalu sukses agar tidak membocorkan email yang terdaftar.
// Dibatasi mailEmailLimit per jam per email; kelebihannya diabaikan diam-diam.
func (s *Service) ResendVerification(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	if !s.allow(ctx, "resend_email", email, mailEmailLimit, mailEmailWindow) {
		return nil
	}
	u, err := s.users.FindByEmail(ctx, email)
	if errs.Is(err, errs.NotFound) || (err == nil && u.EmailVerifiedAt != nil) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.tx.Do(ctx, func(ctx context.Context) error { return s.issueVerification(ctx, u) })
}

// ---- login & sesi ----

// Login memverifikasi kredensial lalu membuat sesi baru.
func (s *Service) Login(ctx context.Context, email, password string, meta domain.SessionMeta) (domain.LoginResult, error) {
	email = normalizeEmail(email)
	if !s.allow(ctx, "login_email", email, loginEmailLimit, loginEmailWindow) {
		return domain.LoginResult{}, errs.New(errs.RateLimited, "")
	}

	creds, err := s.users.CredentialsByEmail(ctx, email)
	hash := creds.PasswordHash
	if err != nil {
		if !errs.Is(err, errs.NotFound) {
			return domain.LoginResult{}, err
		}
		hash = s.dummyHash
	}
	ok, verr := crypto.VerifyPassword(password, hash)
	if verr != nil {
		s.log.ErrorContext(ctx, "hash password rusak", "user_id", creds.ID, "err", verr)
	}
	if err != nil || verr != nil || !ok {
		return domain.LoginResult{}, errs.New(errs.InvalidCredentials, "")
	}

	switch {
	case creds.Status == user.StatusSuspended:
		return domain.LoginResult{}, errs.New(errs.AccountSuspended, "")
	case creds.EmailVerifiedAt == nil:
		return domain.LoginResult{}, errs.New(errs.EmailNotVerified, "")
	}

	roles, err := s.roles.RolesOf(ctx, creds.ID)
	if err != nil {
		return domain.LoginResult{}, err
	}
	// Alur 2FA (challenge, setup wajib untuk role internal) belum diimplementasikan.
	// Gagal tertutup: jangan terbitkan token penuh yang seharusnya butuh 2FA.
	if creds.TwoFactorEnabled || hasInternal(roles) {
		return domain.LoginResult{}, errs.New(errs.ServiceUnavailable, "Login dengan 2FA belum tersedia.")
	}

	var res domain.LoginResult
	err = s.tx.Do(ctx, func(ctx context.Context) error {
		var err error
		res, err = s.startSession(ctx, creds.User, roles, meta)
		if err != nil {
			return err
		}
		return s.users.RecordLogin(ctx, creds.ID)
	})
	return res, err
}

func hasInternal(roles []rbac.RoleRef) bool {
	for _, r := range roles {
		if r.Scope == rbac.ScopeInternal {
			return true
		}
	}
	return false
}

// startSession membuat sesi dengan refresh token baru dan menandatangani access token.
func (s *Service) startSession(ctx context.Context, u user.User, roles []rbac.RoleRef, meta domain.SessionMeta) (domain.LoginResult, error) {
	rawRT, hashRT, err := newOneTimeToken(domain.RefreshTokenPrefix)
	if err != nil {
		return domain.LoginResult{}, err
	}
	now := s.clock.Now()
	sid := id.New()
	rtExp := now.Add(domain.RefreshTokenTTL)
	if err := s.repo.CreateSession(ctx, repository.NewSession{
		ID: sid, UserID: u.ID, RefreshTokenHash: hashRT,
		UserAgent: clip(meta.UserAgent, 512), IP: nonEmpty(meta.IP), ExpiresAt: rtExp, Now: now,
	}); err != nil {
		return domain.LoginResult{}, err
	}
	return s.result(u, roles, sid, rawRT, rtExp)
}

func (s *Service) result(u user.User, roles []rbac.RoleRef, sid uuid.UUID, rawRT string, rtExp time.Time) (domain.LoginResult, error) {
	at, atExp, err := s.tokens.Sign(u.ID, sid, u.PermVersion)
	if err != nil {
		return domain.LoginResult{}, err
	}
	codes := make([]string, len(roles))
	for i, r := range roles {
		codes[i] = r.Code
	}
	return domain.LoginResult{
		Tokens: domain.Tokens{AccessToken: at, AccessExpiresAt: atExp, RefreshToken: rawRT, RefreshExpiresAt: rtExp},
		User:   domain.AuthUser{ID: u.ID, Name: u.Name, Email: u.Email, Tier: u.TierCode, Roles: codes},
	}, nil
}

// Refresh menukar refresh token dengan pasangan token baru (rotasi). Memakai
// ulang token yang sudah dirotasi dianggap pencurian: sesinya dicabut.
func (s *Service) Refresh(ctx context.Context, rawRT string) (domain.LoginResult, error) {
	oldHash := crypto.SHA256Hex([]byte(rawRT))
	var (
		res    domain.LoginResult
		reused bool
	)
	err := s.tx.Do(ctx, func(ctx context.Context) error {
		m, err := s.repo.LockByRefreshHash(ctx, oldHash)
		if err != nil {
			return err
		}
		now := s.clock.Now()
		if m.IsPrevious {
			reused = true
			// Commit pencabutan; error dikembalikan setelah transaksi selesai.
			return s.repo.RevokeByID(ctx, m.Session.ID, domain.RevokedReuse, now)
		}
		if !m.Session.Active(now) {
			return errs.New(errs.Unauthenticated, "")
		}
		u, err := s.users.Get(ctx, m.Session.UserID)
		if err != nil {
			return err
		}
		if u.Status == user.StatusSuspended {
			return errs.New(errs.AccountSuspended, "")
		}
		roles, err := s.roles.RolesOf(ctx, u.ID)
		if err != nil {
			return err
		}
		newRT, newHash, err := newOneTimeToken(domain.RefreshTokenPrefix)
		if err != nil {
			return err
		}
		exp := now.Add(domain.RefreshTokenTTL)
		if err := s.repo.Rotate(ctx, m.Session.ID, oldHash, newHash, exp, now); err != nil {
			return err
		}
		res, err = s.result(u, roles, m.Session.ID, newRT, exp)
		return err
	})
	switch {
	case reused:
		s.log.WarnContext(ctx, "refresh token dipakai ulang; sesi dicabut")
		return domain.LoginResult{}, errs.New(errs.Unauthenticated, "")
	case errs.Is(err, errs.NotFound):
		return domain.LoginResult{}, errs.New(errs.Unauthenticated, "")
	}
	return res, err
}

// Authenticate memverifikasi access token beserta sesinya. Sesi yang dicabut
// langsung menolak token yang belum kedaluwarsa.
func (s *Service) Authenticate(ctx context.Context, rawJWT string) (domain.Principal, error) {
	c, err := s.tokens.Parse(rawJWT)
	if err != nil {
		return domain.Principal{}, err
	}
	sess, err := s.repo.FindSession(ctx, c.SessionID)
	if errs.Is(err, errs.NotFound) {
		return domain.Principal{}, errs.New(errs.Unauthenticated, "")
	}
	if err != nil {
		return domain.Principal{}, err
	}
	if sess.UserID != c.UserID || !sess.Active(s.clock.Now()) {
		return domain.Principal{}, errs.New(errs.Unauthenticated, "")
	}
	u, err := s.users.Get(ctx, c.UserID)
	if errs.Is(err, errs.NotFound) {
		return domain.Principal{}, errs.New(errs.Unauthenticated, "")
	}
	if err != nil {
		return domain.Principal{}, err
	}
	if u.Status == user.StatusSuspended {
		return domain.Principal{}, errs.New(errs.AccountSuspended, "")
	}
	return domain.Principal{UserID: c.UserID, SessionID: c.SessionID, PermVersion: c.PermVersion}, nil
}

func (s *Service) Logout(ctx context.Context, p domain.Principal) error {
	_, err := s.repo.Revoke(ctx, p.UserID, p.SessionID, domain.RevokedLogout, s.clock.Now())
	return err
}

func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.repo.RevokeAll(ctx, userID, domain.RevokedLogoutAll, s.clock.Now())
}

func (s *Service) ListSessions(ctx context.Context, p domain.Principal) ([]domain.SessionView, error) {
	rows, err := s.repo.ListActive(ctx, p.UserID, s.clock.Now())
	if err != nil {
		return nil, err
	}
	out := make([]domain.SessionView, len(rows))
	for i, r := range rows {
		out[i] = domain.SessionView{Session: r, IsCurrent: r.ID == p.SessionID}
	}
	return out, nil
}

// RevokeSession mencabut satu sesi milik user. Sesi orang lain atau yang tidak
// ada menghasilkan NOT_FOUND (RULES §5.4).
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	ok, err := s.repo.Revoke(ctx, userID, sessionID, domain.RevokedUser, s.clock.Now())
	if err != nil {
		return err
	}
	if !ok {
		return errs.New(errs.NotFound, "")
	}
	return nil
}

// ---- lupa & reset password ----

// ForgotPassword selalu sukses agar tidak membocorkan email yang terdaftar.
func (s *Service) ForgotPassword(ctx context.Context, email, ip string) error {
	email = normalizeEmail(email)
	if !s.allow(ctx, "forgot_email", email, mailEmailLimit, mailEmailWindow) {
		return nil
	}
	u, err := s.users.FindByEmail(ctx, email)
	if errs.Is(err, errs.NotFound) || (err == nil && u.Status != user.StatusActive) {
		return nil
	}
	if err != nil {
		return err
	}
	raw, hash, err := newOneTimeToken(domain.PasswordResetPrefix)
	if err != nil {
		return err
	}
	return s.tx.Do(ctx, func(ctx context.Context) error {
		now := s.clock.Now()
		exp := now.Add(domain.PasswordResetTTL)
		rowID, err := s.repo.CreatePasswordReset(ctx, u.ID, hash, nonEmpty(ip), exp, now)
		if err != nil {
			return err
		}
		return s.publishToken(ctx, domain.EventPasswordResetReq, domain.AggregatePasswordReset, rowID, u, raw, exp, now)
	})
}

// ResetPassword memakai token reset (sekali pakai), mengganti password, dan
// mencabut seluruh sesi user.
func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if !domain.ValidPassword(newPassword) {
		return errs.Validation(errs.Detail{Field: "new_password", Code: "password", Message: "Password minimal 8 karakter dan mengandung huruf dan angka."})
	}
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.tx.Do(ctx, func(ctx context.Context) error {
		now := s.clock.Now()
		ot, err := s.repo.ConsumePasswordReset(ctx, crypto.SHA256Hex([]byte(rawToken)), now)
		if errs.Is(err, errs.NotFound) {
			return errInvalidToken
		}
		if err != nil {
			return err
		}
		if err := s.users.SetPasswordHash(ctx, ot.UserID, hash); err != nil {
			return err
		}
		if err := s.repo.InvalidateResets(ctx, ot.UserID, now); err != nil {
			return err
		}
		return s.repo.RevokeAll(ctx, ot.UserID, domain.RevokedPasswordSet, now)
	})
}

// ---- util ----

// allow mengecek batas per kunci. Gagal terbuka bila Redis bermasalah: batas
// per IP di route tetap berlaku, dan login tidak boleh mati karena Redis.
func (s *Service) allow(ctx context.Context, scope, key string, limit int, window time.Duration) bool {
	res, err := s.limiter.Allow(ctx, scope, crypto.SHA256Hex([]byte(key))[:32], limit, window)
	if err != nil {
		s.log.WarnContext(ctx, "rate limit tidak tersedia", "scope", scope, "err", err)
		return true
	}
	return res.Allowed
}

// newOneTimeToken membuat token acak 256-bit beserta hash SHA-256-nya. Hanya
// hash yang disimpan; token mentah cukup entropinya sehingga tidak perlu salt.
func newOneTimeToken(prefix string) (raw, hash string, err error) {
	raw, err = crypto.RandomToken(prefix, 32)
	if err != nil {
		return "", "", err
	}
	return raw, crypto.SHA256Hex([]byte(raw)), nil
}

func normalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func clip(s string, n int) *string {
	if s == "" {
		return nil
	}
	if len(s) > n {
		s = strings.ToValidUTF8(s[:n], "")
	}
	return &s
}
