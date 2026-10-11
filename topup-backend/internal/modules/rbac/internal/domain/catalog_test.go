package domain

import (
	"regexp"
	"slices"
	"testing"
)

var codePattern = regexp.MustCompile(`^[a-z0-9_]+:[a-z0-9_]+$`)

func TestPermissionCatalog(t *testing.T) {
	seen := map[string]bool{}
	var sensitive []string
	for _, p := range Permissions {
		if !codePattern.MatchString(p.Code) {
			t.Errorf("kode %q bukan resource:action", p.Code)
		}
		if seen[p.Code] {
			t.Errorf("kode %q duplikat", p.Code)
		}
		seen[p.Code] = true
		if p.Description == "" || len(p.Description) > 255 {
			t.Errorf("deskripsi %q kosong atau > 255", p.Code)
		}
		if p.Resource()+":"+p.Action() != p.Code {
			t.Errorf("Resource/Action %q salah", p.Code)
		}
		if p.Sensitive {
			sensitive = append(sensitive, p.Code)
		}
	}
	// Permission ★ di PRD §7.4.
	want := []string{
		"balance:adjust", "orders:mark_status", "orders:refund", "payments:manage_credentials",
		"providers:manage_credentials", "roles:write", "settings:write", "users:assign_role",
	}
	slices.Sort(sensitive)
	if !slices.Equal(sensitive, want) {
		t.Errorf("permission sensitif = %v; want %v", sensitive, want)
	}
}

func TestSystemRoles(t *testing.T) {
	known := map[string]Permission{}
	for _, p := range Permissions {
		known[p.Code] = p
	}
	roles := map[string]Role{}
	for _, r := range SystemRoles() {
		roles[r.Code] = r
		if r.Scope != ScopeInternal && r.Scope != ScopeCustomer {
			t.Errorf("%s: scope %q tidak valid", r.Code, r.Scope)
		}
		seen := map[string]bool{}
		for _, c := range r.Permissions {
			if _, ok := known[c]; !ok {
				t.Errorf("%s: permission %q tidak ada di katalog", r.Code, c)
			}
			if seen[c] {
				t.Errorf("%s: permission %q duplikat", r.Code, c)
			}
			seen[c] = true
			// Permission customer hanya untuk role customer, dan sebaliknya.
			isCustomerPerm := known[c].Resource() == "self" || known[c].Resource() == "h2h"
			if r.Code != RoleSuperAdmin && isCustomerPerm != (r.Scope == ScopeCustomer) {
				t.Errorf("%s (%s) tidak boleh memegang %q", r.Code, r.Scope, c)
			}
		}
	}
	if len(roles) != 7 {
		t.Fatalf("jumlah role sistem %d; want 7", len(roles))
	}
	if got := len(roles[RoleSuperAdmin].Permissions); got != len(Permissions) {
		t.Errorf("super_admin memegang %d dari %d permission", got, len(Permissions))
	}
	// Hanya super_admin yang memegang kredensial, roles:write, dan settings:write.
	for _, c := range []string{"providers:manage_credentials", "payments:manage_credentials", "roles:write", "settings:write"} {
		for code, r := range roles {
			if code != RoleSuperAdmin && slices.Contains(r.Permissions, c) {
				t.Errorf("%s tidak boleh memegang %q", code, c)
			}
		}
	}
	if slices.Contains(roles[RoleCustomerService].Permissions, "customers:view_pii") {
		t.Error("customer_service tidak boleh melihat PII (RBAC-09)")
	}
	member, reseller := roles[RoleMember].Permissions, roles[RoleReseller].Permissions
	for _, c := range member {
		if !slices.Contains(reseller, c) {
			t.Errorf("reseller harus memegang semua permission member, kurang %q", c)
		}
	}
	if !slices.Contains(reseller, "h2h:access") || slices.Contains(member, "h2h:access") {
		t.Error("h2h:access hanya untuk reseller")
	}
}
