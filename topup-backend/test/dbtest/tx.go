package dbtest

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

var errRollback = errors.New("dbtest: rollback")

// Rollback menjalankan fn di transaksi yang selalu di-rollback.
func Rollback(t *testing.T, db *gorm.DB, fn func(tx *gorm.DB)) {
	t.Helper()
	if err := db.Transaction(func(tx *gorm.DB) error {
		fn(tx)
		return errRollback
	}); !errors.Is(err, errRollback) {
		t.Fatal(err)
	}
}
