package arch

import (
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
)

// TestArchitecture menjalankan aturan terhadap seluruh package di repo.
func TestArchitecture(t *testing.T) {
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = "../.."
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list gagal: %v", err)
	}
	var pkgs []Package
	dec := json.NewDecoder(strings.NewReader(string(out)))
	for {
		var p Package
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		pkgs = append(pkgs, p)
	}
	if len(pkgs) == 0 {
		t.Fatal("tidak ada package yang terbaca")
	}
	for _, v := range Check(pkgs) {
		t.Error(v)
	}
}

// TestRules memastikan aturan benar-benar menangkap pelanggaran.
func TestRules(t *testing.T) {
	m := func(s string) string { return modulesRoot + s }
	pkgs := []Package{
		// Diizinkan: order → payment (kontrak publik).
		{ImportPath: m("order/internal/service"), Imports: []string{m("payment"), "gorm.io/gorm"}},
		// Dilarang: payment → order (arah terbalik).
		{ImportPath: m("payment/internal/service"), Imports: []string{m("order")}},
		// Dilarang: import implementasi internal modul lain.
		{ImportPath: m("order/internal/service"), Imports: []string{m("payment/internal/repository")}},
		// Dilarang: akses DB tanpa GORM.
		{ImportPath: m("wallet/internal/repository"), Imports: []string{"database/sql", "github.com/jackc/pgx/v5"}},
		// Dilarang: domain import GORM.
		{ImportPath: m("order/internal/domain"), Imports: []string{"gorm.io/gorm"}},
		// Dilarang: platform import modul.
		{ImportPath: platformRoot + "/event", Imports: []string{m("order")}},
		// Dilarang: modul tak terdaftar.
		{ImportPath: m("loyalty"), Imports: nil},
	}
	got := Check(pkgs)
	wantSubstr := []string{
		"payment → order tidak diizinkan",
		"import implementasi internal modul payment",
		"dilarang import database/sql",
		"dilarang import github.com/jackc/pgx/v5",
		"layer domain tidak boleh import gorm.io/gorm",
		"platform tidak boleh import",
		`modul "loyalty" belum terdaftar`,
	}
	if len(got) != len(wantSubstr) {
		t.Fatalf("jumlah pelanggaran %d, want %d:\n%s", len(got), len(wantSubstr), strings.Join(got, "\n"))
	}
	for _, w := range wantSubstr {
		found := false
		for _, g := range got {
			if strings.Contains(g, w) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pelanggaran %q tidak terdeteksi; got:\n%s", w, strings.Join(got, "\n"))
		}
	}
}
