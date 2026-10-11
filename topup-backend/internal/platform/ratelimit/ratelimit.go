// Package ratelimit: rate limit fixed-window di Redis (key rl:{scope}:{key}),
// konsisten antar instance. Batas default ada di API_CONTRACT §1.8.
package ratelimit

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/clock"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/logger"
)

type Limiter struct {
	rdb   *redis.Client
	clock clock.Clock
}

func New(rdb *redis.Client, c clock.Clock) *Limiter { return &Limiter{rdb: rdb, clock: c} }

type Result struct {
	Allowed    bool
	Limit      int
	Remaining  int
	RetryAfter time.Duration
}

// Allow menghitung satu hit untuk key pada jendela berjalan.
func (l *Limiter) Allow(ctx context.Context, scope, key string, limit int, window time.Duration) (Result, error) {
	now := l.clock.Now()
	start := now.Truncate(window)
	k := "rl:" + scope + ":" + key + ":" + strconv.FormatInt(start.Unix(), 10)

	var incr *redis.IntCmd
	_, err := l.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		incr = p.Incr(ctx, k)
		p.Expire(ctx, k, window+time.Second)
		return nil
	})
	if err != nil {
		return Result{Allowed: true, Limit: limit, Remaining: limit}, err
	}
	n := int(incr.Val())
	res := Result{Allowed: n <= limit, Limit: limit, Remaining: max(limit-n, 0)}
	if !res.Allowed {
		res.RetryAfter = start.Add(window).Sub(now)
	}
	return res, nil
}

// KeyFunc menentukan identitas yang dibatasi (IP, user, API key).
type KeyFunc func(c *fiber.Ctx) string

// ByIP membatasi per alamat IP.
func ByIP(c *fiber.Ctx) string { return c.IP() }

// Middleware menerapkan batas dan header X-RateLimit-*. Bila Redis gagal,
// request tetap diloloskan (fail-open) dan dicatat.
func (l *Limiter) Middleware(scope string, limit int, window time.Duration, key KeyFunc) fiber.Handler {
	return func(c *fiber.Ctx) error {
		res, err := l.Allow(c.UserContext(), scope, key(c), limit, window)
		if err != nil {
			logger.FromContext(c.UserContext()).Warn("rate limit tidak tersedia", "scope", scope, "err", err)
			return c.Next()
		}
		c.Set("X-RateLimit-Limit", strconv.Itoa(res.Limit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		if !res.Allowed {
			secs := int(res.RetryAfter.Seconds()) + 1
			c.Set(fiber.HeaderRetryAfter, strconv.Itoa(secs))
			return errs.New(errs.RateLimited, "")
		}
		return c.Next()
	}
}
