package repository

import (
	"testing"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/id"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/money"
	"github.com/RanggaCasper/RaCash/topup-backend/test/dbtest"
)

func TestModelsMatchSchema(t *testing.T) {
	dbtest.AssertModelsMatchSchema(t, dbtest.Open(t),
		&PaymentGateway{},
		&PaymentMethod{},
		&PaymentMethodGatewayMap{},
		&Payment{},
	)
}

func TestNumericRoundTrip(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Rollback(t, db, func(tx *gorm.DB) {
		share := money.Percent(40500) // 40,5%
		in := PaymentMethod{ID: id.New(), Code: "T_" + id.New().String()[:8], Name: "X", Group: "QRIS",
			FeePercent: 700, FeeBearer: "SPLIT", FeeCustomerShare: &share, ExpiryMinutes: 60,
			Instructions: datatypes.JSON("[]"), IsActive: true}
		if err := tx.Create(&in).Error; err != nil {
			t.Fatal(err)
		}
		var out PaymentMethod
		if err := tx.First(&out, "id = ?", in.ID).Error; err != nil {
			t.Fatal(err)
		}
		if out.FeePercent != 700 || out.FeeCustomerShare == nil || *out.FeeCustomerShare != share {
			t.Errorf("fee_percent=%v fee_customer_share=%v; want 0.7 dan 40.5", out.FeePercent, out.FeeCustomerShare)
		}
	})
}
