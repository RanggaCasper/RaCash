package validator

import (
	"testing"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
)

type target struct {
	UserID string `json:"user_id" validate:"required"`
}

type createOrder struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	Target    target `json:"target"`
	Email     string `json:"email" validate:"omitempty,email"`
}

func TestValidationDetailsUseJSONPaths(t *testing.T) {
	err := New().Struct(createOrder{ProductID: "bukan-uuid", Email: "x"})
	e, ok := errs.As(err)
	if !ok || e.Code != errs.ValidationError {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
	fields := map[string]string{}
	for _, d := range e.Details {
		fields[d.Field] = d.Code
	}
	want := map[string]string{"product_id": "uuid", "target.user_id": "required", "email": "email"}
	for f, code := range want {
		if fields[f] != code {
			t.Errorf("field %s: got %q want %q (semua: %v)", f, fields[f], code, fields)
		}
	}
}

func TestValidStruct(t *testing.T) {
	ok := createOrder{ProductID: "0192f5a2-7c3e-7b2a-9d41-3f0c2b9a1e77", Target: target{UserID: "123"}}
	if err := New().Struct(ok); err != nil {
		t.Fatalf("struct valid ditolak: %v", err)
	}
}
