package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/service"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
)

type principalKey struct{}

// Authenticate memverifikasi header Authorization: Bearer <JWT> beserta
// sesinya, lalu menyimpan Principal di locals request. Token terbatas (user
// internal yang belum punya 2FA) ditolak kecuali allowRestricted.
func Authenticate(svc *service.Service, allowRestricted bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		scheme, raw, found := strings.Cut(c.Get(fiber.HeaderAuthorization), " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(raw) == "" {
			return errs.New(errs.Unauthenticated, "")
		}
		p, err := svc.Authenticate(c.UserContext(), strings.TrimSpace(raw))
		if err != nil {
			return err
		}
		if p.Restricted && !allowRestricted {
			return errs.New(errs.PermissionDenied, "Aktifkan 2FA terlebih dahulu untuk melanjutkan.").
				WithDetails(errs.Detail{Code: "two_factor_setup_required"})
		}
		c.Locals(principalKey{}, p)
		return c.Next()
	}
}

// HeaderStepUp adalah header token step-up (API_CONTRACT §1.6).
const HeaderStepUp = "X-Step-Up-Token"

// StepUp mewajibkan X-Step-Up-Token yang valid untuk user dan sesi pemanggil.
// Harus dipasang setelah Authenticate.
func StepUp(svc *service.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		p, ok := LookupPrincipal(c)
		if !ok {
			return errs.New(errs.Unauthenticated, "")
		}
		if err := svc.CheckStepUp(c.UserContext(), p, strings.TrimSpace(c.Get(HeaderStepUp))); err != nil {
			return err
		}
		return c.Next()
	}
}

// LookupPrincipal mengembalikan identitas request yang sudah lolos Authenticate.
func LookupPrincipal(c *fiber.Ctx) (domain.Principal, bool) {
	p, ok := c.Locals(principalKey{}).(domain.Principal)
	return p, ok
}

// PrincipalFrom seperti LookupPrincipal, tetapi untuk handler di balik
// Authenticate; panic bila dipanggil di route tanpa Authenticate (bug wiring).
func PrincipalFrom(c *fiber.Ctx) domain.Principal {
	p, ok := LookupPrincipal(c)
	if !ok {
		panic("auth: route tanpa middleware Authenticate")
	}
	return p
}
