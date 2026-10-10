// Package config membaca konfigurasi dari environment variable (12-factor),
// sekali saat startup. Rahasia tidak pernah disimpan di repo.
package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
)

const (
	EnvLocal      = "local"
	EnvStaging    = "staging"
	EnvProduction = "production"
)

type Config struct {
	AppEnv   string `env:"APP_ENV" envDefault:"local"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	HTTPAddr        string        `env:"HTTP_ADDR" envDefault:":8080"`
	MetricsAddr     string        `env:"METRICS_ADDR" envDefault:":9090"`
	BodyLimitBytes  int           `env:"HTTP_BODY_LIMIT_BYTES" envDefault:"4194304"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`

	DB                Database
	DBMaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
	DBMaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
	DBConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"30m"`

	RedisURL string `env:"REDIS_URL,required"`

	// ENCRYPTION_KEYS: "k1:<base64 32 byte>,k2:<base64 32 byte>"
	EncryptionKeys        string `env:"ENCRYPTION_KEYS,required"`
	EncryptionActiveKeyID string `env:"ENCRYPTION_ACTIVE_KEY_ID,required"`

	WorkerConcurrency  int           `env:"WORKER_CONCURRENCY" envDefault:"10"`
	OutboxPollInterval time.Duration `env:"OUTBOX_POLL_INTERVAL" envDefault:"1s"`
	OutboxBatchSize    int           `env:"OUTBOX_BATCH_SIZE" envDefault:"100"`
}

// Load membaca dan memvalidasi konfigurasi.
func Load() (Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return c, fmt.Errorf("config: %w", err)
	}
	switch c.AppEnv {
	case EnvLocal, EnvStaging, EnvProduction:
	default:
		return c, fmt.Errorf("config: APP_ENV %q tidak dikenal (local|staging|production)", c.AppEnv)
	}
	return c, nil
}

func (c Config) IsLocal() bool      { return c.AppEnv == EnvLocal }
func (c Config) IsProduction() bool { return c.AppEnv == EnvProduction }

// Database adalah koneksi PostgreSQL dalam variabel terpisah (bukan satu URL),
// sehingga password dengan karakter khusus tidak perlu di-encode manual.
type Database struct {
	Host     string `env:"DB_HOST" envDefault:"localhost"`
	Port     int    `env:"DB_PORT" envDefault:"5432"`
	User     string `env:"DB_USER,required"`
	Password string `env:"DB_PASSWORD"`
	Name     string `env:"DB_NAME,required"`
	SSLMode  string `env:"DB_SSLMODE" envDefault:"disable"`

	// AutoCreate membuat DB_NAME bila belum ada. Kosong = otomatis hanya di APP_ENV=local.
	AutoCreate *bool `env:"DB_AUTO_CREATE"`
	// MaintenanceName adalah database yang dipakai untuk mengecek/membuat DB_NAME.
	MaintenanceName string `env:"DB_MAINTENANCE_NAME" envDefault:"postgres"`
}

// AutoCreateEnabled: nilai DB_AUTO_CREATE bila di-set, selain itu true hanya di local.
func (d Database) AutoCreateEnabled(appEnv string) bool {
	if d.AutoCreate != nil {
		return *d.AutoCreate
	}
	return appEnv == EnvLocal
}

// WithName mengembalikan salinan konfigurasi dengan nama database lain.
func (d Database) WithName(name string) Database {
	d.Name = name
	return d
}

// LoadDatabase hanya membaca variabel DB_* (dipakai cmd/migrate).
func LoadDatabase() (Database, error) { return LoadDatabaseWithPrefix("") }

// LoadDatabaseWithPrefix membaca <prefix>DB_*, mis. "TEST_" → TEST_DB_HOST, TEST_DB_NAME, ...
func LoadDatabaseWithPrefix(prefix string) (Database, error) {
	var d Database
	if err := env.ParseWithOptions(&d, env.Options{Prefix: prefix}); err != nil {
		return d, fmt.Errorf("config: %w", err)
	}
	return d, nil
}

// URL menyusun connection string postgres:// dengan user & password ter-escape.
func (d Database) URL() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     net.JoinHostPort(d.Host, strconv.Itoa(d.Port)),
		Path:     "/" + d.Name,
		RawQuery: url.Values{"sslmode": {d.SSLMode}}.Encode(),
	}
	if d.Password == "" {
		u.User = url.User(d.User)
	}
	return u.String()
}
