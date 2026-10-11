package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/rbac/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/id"
)

// Seed menyinkronkan katalog permission dan role sistem (RBAC-01). Idempoten:
//   - permission di-upsert berdasarkan code (deskripsi & flag sensitif ikut kode);
//     permission yang sudah dihapus dari kode tidak dihapus dari DB;
//   - role sistem yang belum ada dibuat beserta matriks awalnya; role yang
//     sudah ada tidak diubah agar pengaturan Super Admin tidak tertimpa;
//   - super_admin selalu dilengkapi seluruh katalog.
//
// Jalankan di dalam satu transaksi.
func Seed(ctx context.Context, db *gorm.DB, perms []domain.Permission, roles []domain.Role) error {
	db = db.WithContext(ctx)

	permRows := make([]Permission, len(perms))
	for i, p := range perms {
		permRows[i] = Permission{
			ID: id.New(), Code: p.Code, Resource: p.Resource(), Action: p.Action(),
			Description: p.Description, IsSensitive: p.Sensitive,
		}
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{"resource", "action", "description", "is_sensitive"}),
	}).Create(&permRows).Error; err != nil {
		return fmt.Errorf("seed permissions: %w", err)
	}

	// ID permission setelah upsert (baris lama mempertahankan ID-nya).
	var stored []Permission
	if err := db.Select("id", "code").Find(&stored).Error; err != nil {
		return fmt.Errorf("seed permissions: %w", err)
	}
	permID := make(map[string]uuid.UUID, len(stored))
	for _, p := range stored {
		permID[p.Code] = p.ID
	}

	var existing []Role
	if err := db.Select("id", "code").Where("code IN ?", roleCodes(roles)).Find(&existing).Error; err != nil {
		return fmt.Errorf("seed roles: %w", err)
	}
	roleID := make(map[string]uuid.UUID, len(existing))
	for _, r := range existing {
		roleID[r.Code] = r.ID
	}

	var grants []RolePermission
	for _, r := range roles {
		rid, exists := roleID[r.Code]
		if !exists {
			desc := r.Description
			row := Role{ID: id.New(), Code: r.Code, Name: r.Name, Description: &desc, Scope: r.Scope, IsSystem: true}
			if err := db.Create(&row).Error; err != nil {
				return fmt.Errorf("seed role %s: %w", r.Code, err)
			}
			rid = row.ID
		}
		if exists && r.Code != domain.RoleSuperAdmin {
			continue
		}
		for _, code := range r.Permissions {
			pid, ok := permID[code]
			if !ok {
				return fmt.Errorf("seed role %s: permission %q tidak ada di katalog", r.Code, code)
			}
			grants = append(grants, RolePermission{RoleID: rid, PermissionID: pid})
		}
	}
	if len(grants) == 0 {
		return nil
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&grants).Error; err != nil {
		return fmt.Errorf("seed role_permissions: %w", err)
	}
	return nil
}

func roleCodes(roles []domain.Role) []string {
	codes := make([]string, len(roles))
	for i, r := range roles {
		codes[i] = r.Code
	}
	return codes
}
