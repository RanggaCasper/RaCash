// Package validator membungkus go-playground/validator dan mengubah hasilnya
// menjadi VALIDATION_ERROR dengan detail per field (nama field = tag json).
package validator

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
)

type Validator struct{ v *validator.Validate }

func New() *Validator {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		if name == "" {
			return f.Name
		}
		return name
	})
	return &Validator{v: v}
}

// Struct memvalidasi s; mengembalikan *errs.Error (VALIDATION_ERROR) bila gagal.
func (x *Validator) Struct(s any) error {
	err := x.v.Struct(s)
	if err == nil {
		return nil
	}
	var ves validator.ValidationErrors
	if !errors.As(err, &ves) {
		return errs.Wrap(errs.BadRequest, err, "")
	}
	details := make([]errs.Detail, 0, len(ves))
	for _, fe := range ves {
		details = append(details, errs.Detail{
			Field:   fieldPath(fe.Namespace()),
			Code:    fe.Tag(),
			Message: message(fe),
		})
	}
	return errs.Validation(details...)
}

// Engine membuka akses ke validator asli untuk mendaftarkan aturan kustom.
func (x *Validator) Engine() *validator.Validate { return x.v }

// fieldPath membuang nama struct root: "CreateOrderRequest.target.user_id" → "target.user_id".
func fieldPath(ns string) string {
	if _, rest, ok := strings.Cut(ns, "."); ok {
		return rest
	}
	return ns
}

func message(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required", "required_without", "required_if":
		return "Wajib diisi."
	case "email":
		return "Format email tidak valid."
	case "uuid", "uuid4", "uuid7":
		return "Format ID tidak valid."
	case "min":
		return "Minimal " + fe.Param() + "."
	case "max":
		return "Maksimal " + fe.Param() + "."
	case "oneof":
		return "Harus salah satu dari: " + fe.Param() + "."
	case "url", "http_url":
		return "Format URL tidak valid."
	default:
		return "Nilai tidak valid."
	}
}
