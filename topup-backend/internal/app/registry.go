package app

import "fmt"

// Registry menyimpan modul sesuai urutan registrasi (urutan dependensi,
// ARCHITECTURE §2.6). Nama modul harus unik.
type Registry struct {
	modules []Module
	names   map[string]struct{}
}

func NewRegistry() *Registry { return &Registry{names: map[string]struct{}{}} }

func (r *Registry) Add(ms ...Module) error {
	for _, m := range ms {
		if _, dup := r.names[m.Name()]; dup {
			return fmt.Errorf("app: modul %q terdaftar dua kali", m.Name())
		}
		r.names[m.Name()] = struct{}{}
		r.modules = append(r.modules, m)
	}
	return nil
}

func (r *Registry) Modules() []Module { return r.modules }
