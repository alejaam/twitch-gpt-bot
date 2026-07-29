package ratelimit

import (
	"sync"
	"time"
)

// Limiter keeps per-user cooldown state in process memory. It intentionally does
// not coordinate across replicas; this service is designed for a single container.
type Limiter struct {
	mu       sync.Mutex
	cooldown time.Duration
	lastSeen map[string]time.Time
}

func New(cooldown time.Duration) *Limiter {
	return &Limiter{
		cooldown: cooldown,
		lastSeen: make(map[string]time.Time),
	}
}

// Allow records a request only when the user is outside the configured cooldown.
func (l *Limiter) Allow(user string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	lastRequest, exists := l.lastSeen[user]
	if exists && now.Sub(lastRequest) < l.cooldown {
		return false
	}

	l.lastSeen[user] = now
	return true
}
