// Package arch memeriksa aturan batas modul (ARCHITECTURE §2.2 & §2.4, RULES §3).
package arch

import (
	"fmt"
	"sort"
	"strings"
)

const (
	ModulePath   = "github.com/RanggaCasper/RaCash/topup-backend"
	modulesRoot  = ModulePath + "/internal/modules/"
	platformRoot = ModulePath + "/internal/platform"
	appRoot      = ModulePath + "/internal/app"
)

// Allowed adalah arah dependensi antar modul (ARCHITECTURE §2.4).
// Modul baru WAJIB ditambahkan di sini dan di dokumen.
var Allowed = map[string][]string{
	"user":         {},
	"rbac":         {"user"},
	"auth":         {"user", "rbac"},
	"media":        {},
	"catalog":      {"media"},
	"provider":     {"catalog"},
	"pricing":      {"catalog", "provider", "user"},
	"promo":        {},
	"payment":      {"media"},
	"wallet":       {"payment"},
	"order":        {"catalog", "pricing", "promo", "payment", "wallet"},
	"fulfillment":  {"order", "provider"},
	"reseller":     {"order", "wallet", "pricing", "user"},
	"notification": {},
	"audit":        {},
	"report":       {},
	"setting":      {},
}

// Akses database selain GORM dilarang di kode modul (RULES §3.3).
var forbiddenInModules = []string{
	"database/sql",
	"github.com/jmoiron/sqlx",
	"github.com/jackc/pgx",
	"github.com/lib/pq",
}

// Layer domain harus murni: tanpa GORM dan Fiber (ARCHITECTURE §2.7).
var forbiddenInDomain = []string{"gorm.io/", "github.com/gofiber/"}

type Package struct {
	ImportPath string
	Imports    []string
}

// Check mengembalikan daftar pelanggaran, terurut.
func Check(pkgs []Package) []string {
	var v []string
	for _, p := range pkgs {
		switch {
		case strings.HasPrefix(p.ImportPath, platformRoot):
			for _, imp := range p.Imports {
				if strings.HasPrefix(imp, modulesRoot) || strings.HasPrefix(imp, appRoot) {
					v = append(v, fmt.Sprintf("%s: platform tidak boleh import %s", p.ImportPath, imp))
				}
			}
		case strings.HasPrefix(p.ImportPath, modulesRoot):
			v = append(v, checkModule(p)...)
		}
	}
	sort.Strings(v)
	return v
}

func checkModule(p Package) []string {
	var v []string
	mod, rest := splitModule(p.ImportPath)
	allowed, known := Allowed[mod]
	if !known {
		return []string{fmt.Sprintf("%s: modul %q belum terdaftar di tabel dependensi (ARCHITECTURE §2.4)", p.ImportPath, mod)}
	}
	isDomain := strings.HasPrefix(rest, "internal/domain")

	for _, imp := range p.Imports {
		if strings.HasPrefix(imp, appRoot) {
			v = append(v, fmt.Sprintf("%s: modul tidak boleh import internal/app", p.ImportPath))
		}
		if strings.HasPrefix(imp, modulesRoot) {
			other, otherRest := splitModule(imp)
			if other != mod {
				if !contains(allowed, other) {
					v = append(v, fmt.Sprintf("%s: %s → %s tidak diizinkan (pakai domain event)", p.ImportPath, mod, other))
				}
				if strings.HasPrefix(otherRest, "internal") {
					v = append(v, fmt.Sprintf("%s: import implementasi internal modul %s (%s)", p.ImportPath, other, imp))
				}
			}
		}
		for _, f := range forbiddenInModules {
			if imp == f || strings.HasPrefix(imp, f+"/") {
				v = append(v, fmt.Sprintf("%s: akses DB wajib lewat GORM, dilarang import %s", p.ImportPath, imp))
			}
		}
		if isDomain {
			for _, f := range forbiddenInDomain {
				if strings.HasPrefix(imp, f) {
					v = append(v, fmt.Sprintf("%s: layer domain tidak boleh import %s", p.ImportPath, imp))
				}
			}
		}
	}
	return v
}

// splitModule: ".../internal/modules/order/internal/service" → ("order", "internal/service").
func splitModule(path string) (string, string) {
	rel := strings.TrimPrefix(path, modulesRoot)
	mod, rest, _ := strings.Cut(rel, "/")
	return mod, rest
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
