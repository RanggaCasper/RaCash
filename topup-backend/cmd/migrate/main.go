// Command migrate menjalankan migrasi SQL bernomor (golang-migrate).
//
//	migrate up              terapkan semua migrasi
//	migrate down N          batalkan N migrasi terakhir (N wajib)
//	migrate version         tampilkan versi saat ini
//	migrate force V         set versi tanpa menjalankan (pemulihan state dirty)
//	migrate create NAME     buat pasangan file baru, NAME = <modul>_<deskripsi>
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/config"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/database"
)

type options struct {
	AppEnv         string `env:"APP_ENV" envDefault:"local"`
	MigrationsPath string `env:"MIGRATIONS_PATH" envDefault:"migrations"`
}

var (
	namePattern = regexp.MustCompile(`^[a-z]+_[a-z0-9_]+$`)
	filePattern = regexp.MustCompile(`^(\d+)_.*\.(up|down)\.sql$`)
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var opt options
	if err := env.Parse(&opt); err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("perintah: up | down N | version | force V | create NAME")
	}
	if args[0] == "create" {
		if len(args) != 2 {
			return errors.New("pemakaian: create <modul>_<deskripsi>")
		}
		return create(opt.MigrationsPath, args[1])
	}
	db, err := config.LoadDatabase()
	if err != nil {
		return err
	}
	if args[0] == "up" && db.AutoCreateEnabled(opt.AppEnv) {
		log := slog.New(slog.NewTextHandler(os.Stdout, nil))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := database.EnsureExists(ctx, db, log)
		cancel()
		if err != nil {
			return err
		}
	}

	m, err := migrate.New("file://"+filepath.ToSlash(opt.MigrationsPath), pgx5URL(db.URL()))
	if err != nil {
		return err
	}
	defer m.Close()

	switch args[0] {
	case "up":
		return ignoreNoChange(m.Up())
	case "down":
		n, err := intArg(args, "down N")
		if err != nil {
			return err
		}
		return ignoreNoChange(m.Steps(-n))
	case "force":
		v, err := intArg(args, "force V")
		if err != nil {
			return err
		}
		return m.Force(v)
	case "version":
		v, dirty, err := m.Version()
		if errors.Is(err, migrate.ErrNilVersion) {
			fmt.Println("belum ada migrasi")
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Printf("versi %d (dirty=%t)\n", v, dirty)
		return nil
	default:
		return fmt.Errorf("perintah tidak dikenal: %s", args[0])
	}
}

// create membuat file NNNN_<modul>_<deskripsi>.{up,down}.sql dengan nomor berikutnya.
func create(dir, name string) error {
	if !namePattern.MatchString(name) {
		return errors.New("NAME harus <modul>_<deskripsi>, huruf kecil, mis. user_init")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	next := 1
	for _, e := range entries {
		if mt := filePattern.FindStringSubmatch(e.Name()); mt != nil {
			if v, _ := strconv.Atoi(mt[1]); v >= next {
				next = v + 1
			}
		}
	}
	base := fmt.Sprintf("%04d_%s", next, name)
	for _, dirn := range []string{"up", "down"} {
		path := filepath.Join(dir, base+"."+dirn+".sql")
		if err := os.WriteFile(path, []byte("-- "+base+" ("+dirn+")\n"), 0o644); err != nil {
			return err
		}
		fmt.Println("dibuat:", path)
	}
	return nil
}

func intArg(args []string, usage string) (int, error) {
	if len(args) != 2 {
		return 0, errors.New("pemakaian: " + usage)
	}
	n, err := strconv.Atoi(args[1])
	if err != nil || n < 0 {
		return 0, errors.New("pemakaian: " + usage)
	}
	return n, nil
}

func ignoreNoChange(err error) error {
	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("tidak ada perubahan")
		return nil
	}
	return err
}

// pgx5URL mengubah postgres:// menjadi pgx5:// untuk driver golang-migrate.
func pgx5URL(u string) string { return "pgx5://" + strings.TrimPrefix(u, "postgres://") }
