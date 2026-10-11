// Package domain berisi entity dan aturan bisnis murni modul catalog.
package domain

// Category adalah kategori dasar yang di-seed (CAT-01).
type Category struct {
	Slug      string
	Name      string
	SortOrder int32
}

// BaseCategories di-seed saat migrasi. Seed hanya menambah slug yang belum
// ada, sehingga kategori yang diubah atau dihapus admin tidak dibuat ulang.
var BaseCategories = []Category{
	{Slug: "game", Name: "Game", SortOrder: 1},
	{Slug: "voucher", Name: "Voucher", SortOrder: 2},
	{Slug: "pulsa", Name: "Pulsa", SortOrder: 3},
	{Slug: "data", Name: "Data", SortOrder: 4},
	{Slug: "pln", Name: "PLN", SortOrder: 5},
	{Slug: "e-wallet", Name: "E-Wallet", SortOrder: 6},
}
