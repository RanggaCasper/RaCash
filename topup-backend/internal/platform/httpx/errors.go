package httpx

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/logger"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/validator"
)

type errorBody struct {
	Code    errs.Code     `json:"code"`
	Message string        `json:"message"`
	Details []errs.Detail `json:"details,omitempty"`
}

type errorEnvelope struct {
	Success   bool      `json:"success"`
	Error     errorBody `json:"error"`
	RequestID string    `json:"request_id"`
}

// ErrorHandler memetakan error apa pun ke envelope gagal. Pesan internal dan
// error vendor tidak pernah dikirim ke klien (RULES §3.6).
func ErrorHandler(c *fiber.Ctx, err error) error {
	e := toError(err)
	status := e.HTTPStatus()
	if status >= 500 {
		logger.FromContext(c.UserContext()).Error("request gagal", "code", e.Code, "err", err)
	}
	if e.Code == errs.RateLimited && c.GetRespHeader(fiber.HeaderRetryAfter) == "" {
		c.Set(fiber.HeaderRetryAfter, "60")
	}
	return c.Status(status).JSON(errorEnvelope{
		Success:   false,
		Error:     errorBody{Code: e.Code, Message: e.Message, Details: e.Details},
		RequestID: RequestID(c),
	})
}

func toError(err error) *errs.Error {
	if e, ok := errs.As(err); ok {
		if e.Code == errs.Internal {
			// Jangan bocorkan pesan internal yang mungkin diisi pemanggil.
			return errs.New(errs.Internal, "")
		}
		return e
	}
	var fe *fiber.Error
	if errors.As(err, &fe) {
		switch fe.Code {
		case fiber.StatusNotFound:
			return errs.New(errs.NotFound, "")
		case fiber.StatusMethodNotAllowed, fiber.StatusBadRequest,
			fiber.StatusRequestEntityTooLarge, fiber.StatusUnsupportedMediaType:
			return errs.New(errs.BadRequest, "")
		case fiber.StatusTooManyRequests:
			return errs.New(errs.RateLimited, "")
		case fiber.StatusServiceUnavailable:
			return errs.New(errs.ServiceUnavailable, "")
		}
	}
	return errs.New(errs.Internal, "")
}

// Bind mengurai body JSON ke out lalu memvalidasinya.
func Bind(c *fiber.Ctx, v *validator.Validator, out any) error {
	if err := c.BodyParser(out); err != nil {
		return errs.Wrap(errs.BadRequest, err, "Body request bukan JSON yang valid.")
	}
	return v.Struct(out)
}
