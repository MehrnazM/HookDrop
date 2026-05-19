package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	LifetimeLimit = 10000
	RatePerSecond = 100
)

// lifetimeScript atomically increments the per-drop lifetime counter.
// On first increment it mirrors the drop key's TTL so the counter expires with the drop.
const lifetimeScript = `
local count = redis.call('INCR', KEYS[1])
if count == 1 then
    local ttl = redis.call('TTL', KEYS[2])
    if ttl > 0 then
        redis.call('EXPIRE', KEYS[1], ttl)
    else
        redis.call('EXPIRE', KEYS[1], 86400)
    end
end
return count
`

// rateScript atomically maintains a 1-second sliding-window sorted set.
// Removes entries older than 1 second, adds the current request, returns the window size.
const rateScript = `
local now_ms  = tonumber(ARGV[1])
local cutoff  = now_ms - 1000
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', tostring(cutoff))
redis.call('ZADD', KEYS[1], tostring(now_ms), ARGV[2])
local total = redis.call('ZCARD', KEYS[1])
redis.call('EXPIRE', KEYS[1], 2)
return total
`

type rateLimitResult int

const (
	rateLimitOK       rateLimitResult = iota
	rateLimitLifetime                 // lifetime cap exceeded
	rateLimitRate                     // sustained rate cap exceeded
)

type evalClient interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
}

// checkLimits runs the lifetime cap then the sliding-window rate cap for the given slug.
// It returns rateLimitOK when both pass, or the first limit that fired.
// On Redis error it returns rateLimitOK so a transient failure does not block ingestion.
func checkLimits(ctx context.Context, client evalClient, slug string) (rateLimitResult, error) {
	lifetimeKey := fmt.Sprintf("drop:%s:lifetime_count", slug)
	dropKey := fmt.Sprintf("drop:%s", slug)

	// --- lifetime cap (INCR + compare) ---
	res, err := client.Eval(ctx, lifetimeScript, []string{lifetimeKey, dropKey}).Result()
	if err != nil {
		return rateLimitOK, fmt.Errorf("lifetime eval: %w", err)
	}
	count, ok := res.(int64)
	if !ok {
		return rateLimitOK, fmt.Errorf("unexpected lifetime result type %T", res)
	}
	if count > LifetimeLimit {
		return rateLimitLifetime, nil
	}

	// --- sliding-window rate cap ---
	nowMs := time.Now().UnixMilli()
	// member is unique per request: ms timestamp + lifetime counter (monotonically increasing per slug)
	member := strconv.FormatInt(nowMs, 10) + ":" + strconv.FormatInt(count, 10)
	rateKey := fmt.Sprintf("drop:%s:rate_window", slug)

	res, err = client.Eval(ctx, rateScript, []string{rateKey}, nowMs, member).Result()
	if err != nil {
		return rateLimitOK, fmt.Errorf("rate eval: %w", err)
	}
	windowCount, ok := res.(int64)
	if !ok {
		return rateLimitOK, fmt.Errorf("unexpected rate result type %T", res)
	}
	if windowCount > RatePerSecond {
		return rateLimitRate, nil
	}

	return rateLimitOK, nil
}
