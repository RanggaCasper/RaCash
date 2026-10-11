package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/httpx"
)

func (h *Handler) login2FA(c *fiber.Ctx) error {
	var req login2FARequest
	if err := httpx.Bind(c, h.val, &req); err != nil {
		return err
	}
	res, err := h.svc.Login2FA(c.UserContext(), req.ChallengeToken, req.Code, meta(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, toLoginResponse(res))
}

func (h *Handler) twoFactorStatus(c *fiber.Ctx) error {
	st, err := h.svc.TwoFactorStatus(c.UserContext(), PrincipalFrom(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, fiber.Map{
		"enabled":                  st.Enabled,
		"required":                 st.Required,
		"recovery_codes_remaining": st.RecoveryCodesRemaining,
	})
}

func (h *Handler) twoFactorSetup(c *fiber.Ctx) error {
	res, err := h.svc.SetupTwoFactor(c.UserContext(), PrincipalFrom(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, fiber.Map{"otpauth_url": res.OtpauthURL, "secret": res.Secret})
}

func (h *Handler) twoFactorEnable(c *fiber.Ctx) error {
	var req totpCodeRequest
	if err := httpx.Bind(c, h.val, &req); err != nil {
		return err
	}
	codes, err := h.svc.EnableTwoFactor(c.UserContext(), PrincipalFrom(c), req.Code)
	if err != nil {
		return err
	}
	return httpx.OK(c, fiber.Map{"recovery_codes": codes})
}

func (h *Handler) twoFactorDisable(c *fiber.Ctx) error {
	if err := h.svc.DisableTwoFactor(c.UserContext(), PrincipalFrom(c)); err != nil {
		return err
	}
	return ok(c)
}

func (h *Handler) stepUpToken(c *fiber.Ctx) error {
	var req totpCodeRequest
	if err := httpx.Bind(c, h.val, &req); err != nil {
		return err
	}
	token, exp, err := h.svc.StepUp(c.UserContext(), PrincipalFrom(c), req.Code)
	if err != nil {
		return err
	}
	return httpx.OK(c, fiber.Map{"step_up_token": token, "expires_at": exp.UTC().Format(time.RFC3339)})
}
