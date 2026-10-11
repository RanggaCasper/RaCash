// Package handler berisi route HTTP modul auth: parsing, validasi, dan mapping
// DTO. Tanpa logika bisnis (RULES §3.2).
package handler

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	govalidator "github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/service"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/httpx"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/validator"
)

// Batas per IP untuk endpoint publik auth (per menit).
const (
	limitWindow   = time.Minute
	limitRegister = 5
	limitLogin    = 10
	limitMail     = 5 // resend-verification & forgot-password
	limitToken    = 10
	limitRefresh  = 30
)

var phonePattern = regexp.MustCompile(`^(\+62|62|0)8[0-9]{8,12}$`)

type Handler struct {
	svc *service.Service
	val *validator.Validator
	// authenticate memasang Principal ke context request.
	authenticate fiber.Handler
}

func New(svc *service.Service, val *validator.Validator, authenticate fiber.Handler) (*Handler, error) {
	if err := val.Engine().RegisterValidation("password", func(fl govalidator.FieldLevel) bool {
		return domain.ValidPassword(fl.Field().String())
	}); err != nil {
		return nil, err
	}
	if err := val.Engine().RegisterValidation("phone_id", func(fl govalidator.FieldLevel) bool {
		return phonePattern.MatchString(fl.Field().String())
	}); err != nil {
		return nil, err
	}
	return &Handler{svc: svc, val: val, authenticate: authenticate}, nil
}

// Register memasang route di /api/v1.
func (h *Handler) Register(r fiber.Router, mw httpx.Middlewares) {
	a := r.Group(httpx.PrefixPublic + "/auth")
	a.Post("/register", mw.RateLimitIP("auth_register", limitRegister, limitWindow), mw.Captcha, h.register)
	a.Post("/verify-email", mw.RateLimitIP("auth_verify", limitToken, limitWindow), h.verifyEmail)
	a.Post("/resend-verification", mw.RateLimitIP("auth_resend", limitMail, limitWindow), mw.Captcha, h.resendVerification)
	a.Post("/login", mw.RateLimitIP("auth_login", limitLogin, limitWindow), mw.Captcha, h.login)
	a.Post("/refresh", mw.RateLimitIP("auth_refresh", limitRefresh, limitWindow), h.refresh)
	a.Post("/logout", h.authenticate, h.logout)
	a.Post("/logout-all", h.authenticate, h.logoutAll)
	a.Post("/forgot-password", mw.RateLimitIP("auth_forgot", limitMail, limitWindow), mw.Captcha, h.forgotPassword)
	a.Post("/reset-password", mw.RateLimitIP("auth_reset", limitToken, limitWindow), h.resetPassword)

	me := r.Group(httpx.PrefixPublic+"/me", h.authenticate)
	me.Get("/sessions", h.listSessions)
	me.Delete("/sessions/:id", h.revokeSession)
}

// ---- request & response ----

type registerRequest struct {
	Name     string  `json:"name" validate:"required,max=120"`
	Email    string  `json:"email" validate:"required,email,max=254"`
	Phone    *string `json:"phone" validate:"omitempty,phone_id"`
	Password string  `json:"password" validate:"required,password"`
}

type emailRequest struct {
	Email string `json:"email" validate:"required,email,max=254"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email,max=254"`
	Password string `json:"password" validate:"required,max=128"`
}

type tokenRequest struct {
	Token string `json:"token" validate:"required,max=256"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required,max=256"`
}

type resetRequest struct {
	Token       string `json:"token" validate:"required,max=256"`
	NewPassword string `json:"new_password" validate:"required,password"`
}

type authUserResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Tier  string    `json:"tier"`
	Roles []string  `json:"roles"`
}

type loginResponse struct {
	TwoFactorRequired    bool             `json:"two_factor_required"`
	AccessToken          string           `json:"access_token"`
	AccessTokenExpiresAt time.Time        `json:"access_token_expires_at"`
	RefreshToken         string           `json:"refresh_token"`
	RefreshTokenExpires  time.Time        `json:"refresh_token_expires_at"`
	User                 authUserResponse `json:"user"`
}

func toLoginResponse(r domain.LoginResult) loginResponse {
	roles := r.User.Roles
	if roles == nil {
		roles = []string{}
	}
	return loginResponse{
		AccessToken:          r.AccessToken,
		AccessTokenExpiresAt: r.AccessExpiresAt.UTC(),
		RefreshToken:         r.RefreshToken,
		RefreshTokenExpires:  r.RefreshExpiresAt.UTC(),
		User:                 authUserResponse{ID: r.User.ID, Name: r.User.Name, Email: r.User.Email, Tier: r.User.Tier, Roles: roles},
	}
}

type sessionResponse struct {
	ID         uuid.UUID  `json:"id"`
	UserAgent  *string    `json:"user_agent"`
	IP         *string    `json:"ip"`
	IsCurrent  bool       `json:"is_current"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

// ok adalah respons untuk aksi tanpa data (components/responses/Ok).
func ok(c *fiber.Ctx) error { return httpx.OK(c, nil) }

func meta(c *fiber.Ctx) domain.SessionMeta {
	return domain.SessionMeta{UserAgent: c.Get(fiber.HeaderUserAgent), IP: c.IP()}
}

// ---- handler ----

func (h *Handler) register(c *fiber.Ctx) error {
	var req registerRequest
	if err := httpx.Bind(c, h.val, &req); err != nil {
		return err
	}
	var phone *string
	if req.Phone != nil && strings.TrimSpace(*req.Phone) != "" {
		p := strings.TrimSpace(*req.Phone)
		phone = &p
	}
	if utf8.RuneCountInString(strings.TrimSpace(req.Name)) == 0 {
		return errs.Validation(errs.Detail{Field: "name", Code: "required", Message: "Wajib diisi."})
	}
	u, err := h.svc.Register(c.UserContext(), service.RegisterInput{
		Name: req.Name, Email: req.Email, Phone: phone, Password: req.Password,
	})
	if err != nil {
		return err
	}
	return httpx.Created(c, fiber.Map{"user_id": u.ID, "status": u.Status})
}

func (h *Handler) verifyEmail(c *fiber.Ctx) error {
	var req tokenRequest
	if err := httpx.Bind(c, h.val, &req); err != nil {
		return err
	}
	if err := h.svc.VerifyEmail(c.UserContext(), req.Token); err != nil {
		return err
	}
	return ok(c)
}

func (h *Handler) resendVerification(c *fiber.Ctx) error {
	var req emailRequest
	if err := httpx.Bind(c, h.val, &req); err != nil {
		return err
	}
	if err := h.svc.ResendVerification(c.UserContext(), req.Email); err != nil {
		return err
	}
	return ok(c)
}

func (h *Handler) login(c *fiber.Ctx) error {
	var req loginRequest
	if err := httpx.Bind(c, h.val, &req); err != nil {
		return err
	}
	res, err := h.svc.Login(c.UserContext(), req.Email, req.Password, meta(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, toLoginResponse(res))
}

func (h *Handler) refresh(c *fiber.Ctx) error {
	var req refreshRequest
	if err := httpx.Bind(c, h.val, &req); err != nil {
		return err
	}
	res, err := h.svc.Refresh(c.UserContext(), req.RefreshToken)
	if err != nil {
		return err
	}
	return httpx.OK(c, toLoginResponse(res))
}

func (h *Handler) logout(c *fiber.Ctx) error {
	if err := h.svc.Logout(c.UserContext(), PrincipalFrom(c)); err != nil {
		return err
	}
	return ok(c)
}

func (h *Handler) logoutAll(c *fiber.Ctx) error {
	if err := h.svc.LogoutAll(c.UserContext(), PrincipalFrom(c).UserID); err != nil {
		return err
	}
	return ok(c)
}

func (h *Handler) forgotPassword(c *fiber.Ctx) error {
	var req emailRequest
	if err := httpx.Bind(c, h.val, &req); err != nil {
		return err
	}
	if err := h.svc.ForgotPassword(c.UserContext(), req.Email, c.IP()); err != nil {
		return err
	}
	return ok(c)
}

func (h *Handler) resetPassword(c *fiber.Ctx) error {
	var req resetRequest
	if err := httpx.Bind(c, h.val, &req); err != nil {
		return err
	}
	if err := h.svc.ResetPassword(c.UserContext(), req.Token, req.NewPassword); err != nil {
		return err
	}
	return ok(c)
}

func (h *Handler) listSessions(c *fiber.Ctx) error {
	rows, err := h.svc.ListSessions(c.UserContext(), PrincipalFrom(c))
	if err != nil {
		return err
	}
	out := make([]sessionResponse, len(rows))
	for i, s := range rows {
		out[i] = sessionResponse{
			ID: s.ID, UserAgent: s.UserAgent, IP: s.IP, IsCurrent: s.IsCurrent,
			CreatedAt: s.CreatedAt.UTC(), LastUsedAt: s.LastUsedAt,
		}
	}
	return httpx.OK(c, out)
}

func (h *Handler) revokeSession(c *fiber.Ctx) error {
	sid, err := uuid.Parse(c.Params("id"))
	if err != nil {
		// ID yang bukan UUID pasti bukan milik user: NOT_FOUND, bukan 422.
		return errs.New(errs.NotFound, "")
	}
	if err := h.svc.RevokeSession(c.UserContext(), PrincipalFrom(c).UserID, sid); err != nil {
		return err
	}
	return ok(c)
}
