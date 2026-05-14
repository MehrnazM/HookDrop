package main

import (
	"sync"

	"golang.org/x/time/rate"
)

// DropRateLimiter enforces a per-slug token bucket to protect the ingestion endpoint.
// Limiters are created on first use and retained for the process lifetime — acceptable
// because slugs are short-lived (24h TTL) and the process restarts periodically.
type DropRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	burst    int
}

func NewDropRateLimiter(r rate.Limit, burst int) *DropRateLimiter {
	return &DropRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		burst:    burst,
	}
}

func (d *DropRateLimiter) Allow(slug string) bool {
	d.mu.Lock()
	l, ok := d.limiters[slug]
	if !ok {
		l = rate.NewLimiter(d.r, d.burst)
		d.limiters[slug] = l
	}
	d.mu.Unlock()
	return l.Allow()
}
