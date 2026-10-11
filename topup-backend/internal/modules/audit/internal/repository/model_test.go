package repository

import (
	"testing"

	"github.com/RanggaCasper/RaCash/topup-backend/test/dbtest"
)

func TestModelsMatchSchema(t *testing.T) {
	dbtest.AssertModelsMatchSchema(t, dbtest.Open(t),
		&AuditLog{},
	)
}
