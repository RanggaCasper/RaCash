package app

import (
	"log/slog"

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

	// Modul didaftarkan di sini sesuai urutan dependensi (ARCHITECTURE §2.6), mis.:
	//   userM    := user.New(p)
	//   rbacM    := rbac.New(p, userM.Service())
	//   authM    := auth.New(p, userM.Service(), rbacM.Authorizer())
	//   mediaM   := media.New(p)
	//   catalogM := catalog.New(p, mediaM.Service())
	//   ...
	//   if err := reg.Add(userM, rbacM, authM, mediaM, catalogM); err != nil { ... }

	return &App{Platform: p, Registry: reg}, nil
}

func (a *App) Close() error { return a.Platform.Close() }
