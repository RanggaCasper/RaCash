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
// sesinya, lalu menyimpan Principal di locals request.
func Authenticate(svc *service.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		scheme, raw, found := strings.Cut(c.Get(fiber.HeaderAuthorization), " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(raw) == "" {
			return errs.New(errs.Unauthenticated, "")
		}
		p, err := svc.Authenticate(c.UserContext(), strings.TrimSpace(raw))
		if err != nil {
			return err
		}
		c.Locals(principalKey{}, p)
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
