// Package cache menyediakan klien Redis untuk cache, lock, dan rate limit.
// Prefix key mengikuti ARCHITECTURE §4 (cache:*, rbac:perm:*, lock:*, rl:*, cb:*).
package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrLockNotAcquired = errors.New("cache: lock sedang dipegang proses lain")

// SetLogger mengarahkan log internal go-redis ke slog (bawaannya ke stderr teks).
func SetLogger(l *slog.Logger) { redis.SetLogger(redisLogger{l.With("component", "redis")}) }

type redisLogger struct{ l *slog.Logger }

func (r redisLogger) Printf(_ context.Context, format string, v ...any) {
	r.l.Warn(fmt.Sprintf(format, v...))
}

// New membuat klien dari URL redis://.
func New(url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("cache: %w", err)
	}
	return redis.NewClient(opt), nil
}

// Lock adalah distributed lock sederhana (SET NX PX) dengan token pemilik.
type Lock struct {
	rdb   *redis.Client
	key   string
	token string
}

var unlockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0`)

// Acquire mengambil lock key (mis. "lock:order:<id>") selama ttl.
func Acquire(ctx context.Context, rdb *redis.Client, key, token string, ttl time.Duration) (*Lock, error) {
	ok, err := rdb.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrLockNotAcquired
	}
	return &Lock{rdb: rdb, key: key, token: token}, nil
}

// Release melepas lock hanya bila masih milik pemanggil.
func (l *Lock) Release(ctx context.Context) error {
	return unlockScript.Run(ctx, l.rdb, []string{l.key}, l.token).Err()
}
