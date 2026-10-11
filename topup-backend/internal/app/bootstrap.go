package app

import (
	"log/slog"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/rbac"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/user"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/config"
)

// App adalah hasil wiring: platform + modul terdaftar.
type App struct {
	Platform *platform.Platform
	Registry *Registry
}

// Bootstrap merangkai platform dan modul secara eksplisit (tanpa container
// reflection) agar urutan dependensi mudah dibaca.
func Bootstrap(cfg config.Config, log *slog.Logger) (*App, error) {
	p, err := platform.New(cfg, log)
	if err != nil {
		return nil, err
	}
	reg := NewRegistry()

	// Modul didaftarkan sesuai urutan dependensi (ARCHITECTURE §2.6). Modul
	// berikutnya (media, catalog, ...) ditambahkan di bawah authM, mis.:
	//   mediaM   := media.New(p)
	//   catalogM := catalog.New(p, mediaM.Service())
	userM := user.New(p)
	rbacM := rbac.New(p)
	authM, err := auth.New(p, userM.Service(), rbacM.Service())
	if err != nil {
		_ = p.Close()
		return nil, err
	}
	if err := reg.Add(userM, rbacM, authM); err != nil {
		_ = p.Close()
		return nil, err
	}

	return &App{Platform: p, Registry: reg}, nil
}

func (a *App) Close() error { return a.Platform.Close() }
