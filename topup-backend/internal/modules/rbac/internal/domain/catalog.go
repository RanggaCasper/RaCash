// Package domain berisi entity dan aturan bisnis murni modul rbac.
package domain

import "strings"

// Permission adalah entri katalog permission (PRD §7.4). Katalog hanya berasal
// dari kode dan di-seed saat migrasi; UI tidak bisa membuat permission (RBAC-01).
type Permission struct {
	Code        string // resource:action
	Description string
	Sensitive   bool // ★: wajib 2FA step-up dan selalu diaudit (RBAC-08)
}

func (p Permission) Resource() string { r, _, _ := strings.Cut(p.Code, ":"); return r }
func (p Permission) Action() string   { _, a, _ := strings.Cut(p.Code, ":"); return a }

// Permissions adalah katalog permission awal (PRD §7.4).
var Permissions = []Permission{
	{Code: "dashboard:view", Description: "Melihat dashboard"},
	{Code: "dashboard:view_profit", Description: "Melihat profit di dashboard"},
	{Code: "catalog:read", Description: "Melihat kategori, brand, produk, dan banner"},
	{Code: "catalog:write", Description: "Mengelola kategori, brand, produk, dan banner"},
	{Code: "catalog:sync", Description: "Menjalankan sinkronisasi produk dan harga dari provider"},
	{Code: "media:read", Description: "Melihat pustaka media"},
	{Code: "media:write", Description: "Mengunggah dan menghapus media"},
	{Code: "pricing:read", Description: "Melihat aturan harga"},
	{Code: "pricing:write", Description: "Mengelola aturan harga"},
	{Code: "providers:read", Description: "Melihat provider, routing, dan mapping produk"},
	{Code: "providers:write", Description: "Mengelola provider, routing, dan mapping produk"},
	{Code: "providers:manage_credentials", Description: "Mengelola kredensial provider", Sensitive: true},
	{Code: "payments:read", Description: "Melihat payment gateway, metode bayar, dan pembayaran"},
	{Code: "payments:write", Description: "Mengelola payment gateway dan metode bayar"},
	{Code: "payments:manage_credentials", Description: "Mengelola kredensial payment gateway", Sensitive: true},
	{Code: "orders:read", Description: "Melihat transaksi"},
	{Code: "orders:retry", Description: "Mengulang fulfillment transaksi"},
	{Code: "orders:switch_provider", Description: "Memindahkan transaksi ke provider lain"},
	{Code: "orders:mark_status", Description: "Menandai status transaksi secara manual", Sensitive: true},
	{Code: "orders:refund", Description: "Me-refund transaksi", Sensitive: true},
	{Code: "users:read", Description: "Melihat user"},
	{Code: "users:write", Description: "Mengelola data user dan undangan staf"},
	{Code: "users:suspend", Description: "Menangguhkan dan mengaktifkan kembali user"},
	{Code: "users:assign_role", Description: "Memberi dan mencabut role user", Sensitive: true},
	{Code: "customers:view_pii", Description: "Melihat data pribadi pembeli tanpa masking"},
	{Code: "balance:read", Description: "Melihat saldo dan mutasi saldo user"},
	{Code: "balance:adjust", Description: "Menyesuaikan saldo user secara manual", Sensitive: true},
	{Code: "promos:read", Description: "Melihat promo"},
	{Code: "promos:write", Description: "Mengelola promo"},
	{Code: "reports:read", Description: "Melihat laporan"},
	{Code: "reports:export", Description: "Mengekspor laporan"},
	{Code: "reconciliation:run", Description: "Menjalankan rekonsiliasi"},
	{Code: "notifications:resend", Description: "Mengirim ulang notifikasi"},
	{Code: "roles:read", Description: "Melihat role dan permission"},
	{Code: "roles:write", Description: "Mengelola role dan permission-nya", Sensitive: true},
	{Code: "audit:read", Description: "Melihat log audit"},
	{Code: "settings:write", Description: "Mengubah pengaturan sistem", Sensitive: true},
	{Code: "self:orders", Description: "Melihat dan membuat transaksi milik sendiri"},
	{Code: "self:balance", Description: "Melihat saldo dan mutasi saldo milik sendiri"},
	{Code: "self:deposit", Description: "Melakukan deposit saldo"},
	{Code: "h2h:access", Description: "Mengakses API H2H reseller"},
}

// Scope role.
const (
	ScopeInternal = "internal"
	ScopeCustomer = "customer"
)

// Kode role sistem (PRD §7.3).
const (
	RoleSuperAdmin      = "super_admin"
	RoleAdmin           = "admin"
	RoleOperator        = "operator"
	RoleFinance         = "finance"
	RoleCustomerService = "customer_service"
	RoleReseller        = "reseller"
	RoleMember          = "member"
)

// Role adalah role sistem (is_system) beserta permission awalnya.
type Role struct {
	Code        string
	Name        string
	Description string
	Scope       string
	Permissions []string
}

var memberPermissions = []string{"self:orders", "self:balance", "self:deposit"}

// SystemRoles adalah role bawaan dengan matriks permission awal (PRD §7.5).
// super_admin selalu memegang seluruh katalog (wildcard). Matriks role lain
// hanya nilai awal: diterapkan saat role pertama kali dibuat, setelah itu
// diatur Super Admin dari panel.
func SystemRoles() []Role {
	all := make([]string, len(Permissions))
	for i, p := range Permissions {
		all[i] = p.Code
	}
	return []Role{
		{
			Code: RoleSuperAdmin, Name: "Super Admin", Scope: ScopeInternal, Permissions: all,
			Description: "Akses penuh. Satu-satunya yang bisa kelola role dan kredensial provider/PG.",
		},
		{
			Code: RoleAdmin, Name: "Admin", Scope: ScopeInternal,
			Description: "Operasional harian penuh, kecuali kredensial, manajemen role, dan pengaturan sistem.",
			Permissions: []string{
				"dashboard:view", "dashboard:view_profit",
				"catalog:read", "catalog:write", "catalog:sync", "pricing:read", "pricing:write",
				"media:read", "media:write",
				"providers:read", "providers:write",
				"payments:read", "payments:write",
				"orders:read", "orders:retry", "orders:switch_provider", "orders:mark_status", "orders:refund",
				"users:read", "users:write", "users:suspend", "users:assign_role",
				"customers:view_pii",
				"balance:read",
				"promos:read", "promos:write",
				"reports:read", "reports:export", "reconciliation:run",
				"notifications:resend",
				"roles:read",
				"audit:read",
			},
		},
		{
			Code: RoleOperator, Name: "Operator", Scope: ScopeInternal,
			Description: "Kelola katalog, harga, mapping provider, dan menangani transaksi bermasalah.",
			Permissions: []string{
				"dashboard:view",
				"catalog:read", "catalog:write", "catalog:sync", "pricing:read", "pricing:write",
				"media:read", "media:write",
				"providers:read", "providers:write",
				"payments:read",
				"orders:read", "orders:retry", "orders:switch_provider", "orders:mark_status",
				"notifications:resend",
			},
		},
		{
			Code: RoleFinance, Name: "Finance", Scope: ScopeInternal,
			Description: "Refund, penyesuaian saldo, rekonsiliasi, dan laporan keuangan.",
			Permissions: []string{
				"dashboard:view", "dashboard:view_profit",
				"catalog:read", "pricing:read",
				"providers:read",
				"payments:read", "payments:write",
				"orders:read", "orders:refund",
				"users:read",
				"customers:view_pii",
				"balance:read", "balance:adjust",
				"promos:read",
				"reports:read", "reports:export", "reconciliation:run",
				"audit:read",
			},
		},
		{
			Code: RoleCustomerService, Name: "Customer Service", Scope: ScopeInternal,
			Description: "Lihat transaksi dan user, kirim ulang notifikasi, retry transaksi. Data pribadi pembeli tersamar.",
			Permissions: []string{
				"dashboard:view",
				"catalog:read", "pricing:read",
				"orders:read", "orders:retry",
				"users:read",
				"balance:read",
				"promos:read",
				"notifications:resend",
			},
		},
		{
			Code: RoleReseller, Name: "Reseller", Scope: ScopeCustomer,
			Description: "Fitur member, harga reseller, dan akses API H2H.",
			Permissions: append(append([]string{}, memberPermissions...), "h2h:access"),
		},
		{
			Code: RoleMember, Name: "Member", Scope: ScopeCustomer,
			Description: "Dashboard pribadi, saldo, dan riwayat transaksi sendiri.",
			Permissions: append([]string{}, memberPermissions...),
		},
	}
}

// RoleRef adalah ringkasan role yang dimiliki user.
type RoleRef struct {
	Code  string
	Name  string
	Scope string
}

// IsInternal menyatakan role staf internal (wajib 2FA).
func (r RoleRef) IsInternal() bool { return r.Scope == ScopeInternal }
