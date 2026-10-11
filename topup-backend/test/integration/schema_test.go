package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/rbac"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/id"
	"github.com/RanggaCasper/RaCash/topup-backend/test/dbtest"
)

// exec menjalankan SQL di savepoint; bila ditolak database, transaksi
// dikembalikan ke savepoint agar bisa dipakai lagi.
func exec(tx *gorm.DB, sql string, args ...any) error {
	sp := "sp_" + id.New().String()[:8]
	tx.SavePoint(sp)
	err := tx.Exec(sql, args...).Error
	if err != nil {
		tx.RollbackTo(sp)
	}
	return err
}

func mustExec(t *testing.T, tx *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := exec(tx, sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func rejects(t *testing.T, tx *gorm.DB, name, sql string, args ...any) {
	t.Helper()
	if err := exec(tx, sql, args...); err == nil {
		t.Errorf("%s: seharusnya ditolak database", name)
	}
}

func seededID(t *testing.T, tx *gorm.DB, sql string) uuid.UUID {
	t.Helper()
	var s string
	if err := tx.Raw(sql).Scan(&s).Error; err != nil || s == "" {
		t.Fatalf("data seed tidak ada (jalankan migrate up): %s: %v", sql, err)
	}
	return uuid.MustParse(s)
}

// Data uji dibuat di transaksi yang selalu di-rollback agar data seed bersama
// tidak berubah.
func TestSchemaConstraints(t *testing.T) {
	db := openDB(t)
	dbtest.Rollback(t, db, func(tx *gorm.DB) {
		tierID := seededID(t, tx, "SELECT id FROM tiers WHERE code = 'MEMBER'")
		catID := seededID(t, tx, "SELECT id FROM categories WHERE slug = 'game'")

		userID := id.New()
		mustExec(t, tx, `INSERT INTO users (id, name, email, password_hash, tier_id) VALUES (?, 'A', ?, 'x', ?)`,
			userID, userID.String()+"@racash.test", tierID)
		rejects(t, tx, "email uppercase", `INSERT INTO users (id, name, email, password_hash, tier_id) VALUES (?, 'B', 'B@X.TEST', 'x', ?)`, id.New(), tierID)
		rejects(t, tx, "saldo negatif", `UPDATE users SET balance = -1 WHERE id = ?`, userID)
		rejects(t, tx, "tier default kedua", `INSERT INTO tiers (id, code, name, is_default) VALUES (?, 'X_DEFAULT', 'X', true)`, id.New())
		rejects(t, tx, "kode role tidak valid", `INSERT INTO roles (id, code, name, scope) VALUES (?, 'Bad-Code', 'X', 'internal')`, id.New())
		rejects(t, tx, "kode promo lowercase", `INSERT INTO promos (id, code, name, type, value, starts_at, ends_at) VALUES (?, 'hemat', 'X', 'FLAT', 1, now(), now())`, id.New())

		// pricing_rules: UNIQUE NULLS NOT DISTINCT (scope, scope_id, tier_id).
		mustExec(t, tx, `DELETE FROM pricing_rules WHERE scope = 'global'`)
		mustExec(t, tx, `INSERT INTO pricing_rules (id, scope, margin_type, margin_value) VALUES (?, 'global', 'PERCENT', 5)`, id.New())
		rejects(t, tx, "aturan global ganda", `INSERT INTO pricing_rules (id, scope, margin_type, margin_value) VALUES (?, 'global', 'FLAT', 100)`, id.New())
		rejects(t, tx, "scope category tanpa scope_id", `INSERT INTO pricing_rules (id, scope, margin_type, margin_value) VALUES (?, 'category', 'FLAT', 100)`, id.New())
		rejects(t, tx, "scope global dengan scope_id", `INSERT INTO pricing_rules (id, scope, scope_id, tier_id, margin_type, margin_value) VALUES (?, 'global', ?, ?, 'FLAT', 100)`, id.New(), catID, tierID)

		// payment_methods: fee_customer_share wajib & 0..100 bila SPLIT.
		pmID := id.New()
		rejects(t, tx, "SPLIT tanpa share", `INSERT INTO payment_methods (id, code, name, "group", fee_bearer) VALUES (?, 'X1', 'X', 'QRIS', 'SPLIT')`, id.New())
		rejects(t, tx, "share > 100", `INSERT INTO payment_methods (id, code, name, "group", fee_bearer, fee_customer_share) VALUES (?, 'X2', 'X', 'QRIS', 'SPLIT', 101)`, id.New())
		mustExec(t, tx, `INSERT INTO payment_methods (id, code, name, "group", fee_bearer, fee_customer_share) VALUES (?, ?, 'X', 'QRIS', 'SPLIT', 50)`, pmID, "T_"+pmID.String()[:8])

		// orders: partial unique (user_id, reseller_ref_id) WHERE source = H2H.
		brandID, productID := id.New(), id.New()
		mustExec(t, tx, `INSERT INTO brands (id, category_id, slug, name) VALUES (?, ?, ?, 'B')`, brandID, catID, brandID.String())
		mustExec(t, tx, `INSERT INTO products (id, brand_id, sku, name) VALUES (?, ?, ?, 'P')`, productID, brandID, productID.String())
		insertOrder := func(source string, ref *string, price, discount, fee, total int64) (uuid.UUID, error) {
			oid := id.New()
			return oid, exec(tx, `INSERT INTO orders (id, invoice_no, source, user_id, reseller_ref_id, product_id, product_snapshot,
				target_data, target_display, payment_method_id, price, estimated_cost, discount, fee, total, expires_at)
				VALUES (?, ?, ?, ?, ?, ?, '{}', '{}', 't', ?, ?, 0, ?, ?, ?, now())`,
				oid, oid.String()[:32], source, userID, ref, productID, pmID, price, discount, fee, total)
		}
		ref := "REF-1"
		orderID, err := insertOrder("H2H", &ref, 100, 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := insertOrder("H2H", &ref, 100, 0, 0, 100); err == nil {
			t.Error("reseller_ref_id H2H ganda seharusnya ditolak")
		}
		if _, err := insertOrder("WEB", &ref, 100, 0, 0, 100); err != nil {
			t.Errorf("partial unique hanya untuk H2H: %v", err)
		}
		if _, err := insertOrder("WEB", nil, 100, 10, 5, 100); err == nil {
			t.Error("total ≠ price - discount + fee seharusnya ditolak")
		}

		// payments: tepat satu referensi sesuai purpose.
		rejects(t, tx, "payment tanpa referensi", `INSERT INTO payments (id, purpose, payment_method_id, amount) VALUES (?, 'ORDER', ?, 100)`, id.New(), pmID)
		rejects(t, tx, "payment ORDER dengan deposit_id", `INSERT INTO payments (id, purpose, order_id, deposit_id, payment_method_id, amount) VALUES (?, 'ORDER', ?, ?, ?, 100)`, id.New(), orderID, id.New(), pmID)
		mustExec(t, tx, `INSERT INTO payments (id, purpose, order_id, payment_method_id, amount) VALUES (?, 'ORDER', ?, ?, 100)`, id.New(), orderID, pmID)

		// balance_ledger: append-only, saldo akhir ≥ 0, referensi unik per tipe.
		ledgerID, refID := id.New(), id.New()
		ledger := `INSERT INTO balance_ledger (id, user_id, type, amount, balance_before, balance_after, reference_type, reference_id, description)
			VALUES (?, ?, ?, ?, ?, ?, 'order', ?, 'test')`
		mustExec(t, tx, ledger, ledgerID, userID, "DEPOSIT", 100, 0, 100, refID)
		rejects(t, tx, "ledger referensi ganda", ledger, id.New(), userID, "DEPOSIT", 100, 100, 200, refID)
		rejects(t, tx, "ledger saldo negatif", ledger, id.New(), userID, "PURCHASE", -200, 100, -100, id.New())
		rejects(t, tx, "ledger aritmetika salah", ledger, id.New(), userID, "PURCHASE", -50, 100, 60, id.New())
		rejects(t, tx, "UPDATE ledger", `UPDATE balance_ledger SET description = 'x' WHERE id = ?`, ledgerID)
		rejects(t, tx, "DELETE ledger", `DELETE FROM balance_ledger WHERE id = ?`, ledgerID)
		rejects(t, tx, "TRUNCATE ledger", `TRUNCATE balance_ledger`)

		// audit_logs: append-only.
		auditID := id.New()
		mustExec(t, tx, `INSERT INTO audit_logs (id, actor_roles, action, entity) VALUES (?, '{super_admin}', 'test.run', 'test')`, auditID)
		rejects(t, tx, "UPDATE audit", `UPDATE audit_logs SET reason = 'x' WHERE id = ?`, auditID)
		rejects(t, tx, "DELETE audit", `DELETE FROM audit_logs WHERE id = ?`, auditID)
	})
}

func TestAppendOnlyPrivileges(t *testing.T) {
	db := openDB(t)
	for _, table := range []string{"balance_ledger", "audit_logs"} {
		var granted bool
		err := db.Raw(`SELECT has_table_privilege('public', ?, 'UPDATE') OR has_table_privilege('public', ?, 'DELETE')`,
			table, table).Scan(&granted).Error
		if err != nil {
			t.Fatal(err)
		}
		if granted {
			t.Errorf("%s: UPDATE/DELETE masih diberikan ke PUBLIC", table)
		}
	}
}

func TestRBACSeed(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()
	countPerms := func(tx *gorm.DB, role string) int64 {
		var n int64
		tx.Table("role_permissions rp").Joins("JOIN roles r ON r.id = rp.role_id").Where("r.code = ?", role).Count(&n)
		return n
	}
	dbtest.Rollback(t, db, func(tx *gorm.DB) {
		var total int64
		tx.Table("permissions").Count(&total)
		if total == 0 || countPerms(tx, "super_admin") != total {
			t.Fatalf("super_admin harus memegang seluruh %d permission", total)
		}
		operator := countPerms(tx, "operator")

		// Ubahan Super Admin pada role lain tidak tertimpa seed berikutnya,
		// sedangkan super_admin selalu dilengkapi.
		mustExec(t, tx, `DELETE FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE code = 'operator')
			AND permission_id = (SELECT id FROM permissions WHERE code = 'orders:retry')`)
		mustExec(t, tx, `DELETE FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE code = 'super_admin')
			AND permission_id = (SELECT id FROM permissions WHERE code = 'orders:retry')`)
		if err := rbac.Seed(ctx, tx); err != nil {
			t.Fatal(err)
		}
		if got := countPerms(tx, "operator"); got != operator-1 {
			t.Errorf("seed menimpa permission operator: %d, want %d", got, operator-1)
		}
		if got := countPerms(tx, "super_admin"); got != total {
			t.Errorf("super_admin tidak dilengkapi ulang: %d, want %d", got, total)
		}
		var roles int64
		tx.Table("roles").Where("is_system").Count(&roles)
		if roles != 7 {
			t.Errorf("role sistem = %d, want 7", roles)
		}
	})
}
