package repository

import (
	"slices"
	"testing"

	"gorm.io/gorm"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/database"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/id"
	"github.com/RanggaCasper/RaCash/topup-backend/test/dbtest"
)

func TestModelsMatchSchema(t *testing.T) {
	dbtest.AssertModelsMatchSchema(t, dbtest.Open(t),
		&APIKey{},
		&ResellerSetting{},
		&ResellerCallback{},
	)
}

func TestArrayAndInetRoundTrip(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Rollback(t, db, func(tx *gorm.DB) {
		var tierID string
		if err := tx.Raw("SELECT id FROM tiers WHERE code = 'RESELLER_SILVER'").Scan(&tierID).Error; err != nil || tierID == "" {
			t.Fatalf("tier belum di-seed: %v", err)
		}
		userID := id.New()
		if err := tx.Exec(`INSERT INTO users (id, name, email, password_hash, tier_id) VALUES (?, 'R', ?, 'x', ?)`,
			userID, userID.String()+"@racash.test", tierID).Error; err != nil {
			t.Fatal(err)
		}
		in := ResellerSetting{UserID: userID, IPWhitelist: database.StringArray{"10.0.0.0/8", "203.0.113.7/32"}, RateLimitPerMinute: 600}
		if err := tx.Create(&in).Error; err != nil {
			t.Fatal(err)
		}
		ip := "203.0.113.7"
		key := APIKey{ID: id.New(), UserID: userID, KeyPrefix: "pk_test", KeyHash: id.New().String(),
			SecretEncrypted: []byte{1}, SecretKeyID: "k1", LastUsedIP: &ip}
		if err := tx.Create(&key).Error; err != nil {
			t.Fatal(err)
		}
		var out ResellerSetting
		if err := tx.First(&out, "user_id = ?", userID).Error; err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(out.IPWhitelist, in.IPWhitelist) {
			t.Errorf("ip_whitelist = %v; want %v", out.IPWhitelist, in.IPWhitelist)
		}
		var gotKey APIKey
		if err := tx.First(&gotKey, "id = ?", key.ID).Error; err != nil {
			t.Fatal(err)
		}
		if gotKey.LastUsedIP == nil || *gotKey.LastUsedIP != ip {
			t.Errorf("last_used_ip = %v; want %s", gotKey.LastUsedIP, ip)
		}
	})
}
