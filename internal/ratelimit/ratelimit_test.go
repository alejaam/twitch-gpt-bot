package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterAllow(t *testing.T) {
	now := time.Date(2026, time.July, 29, 22, 0, 0, 0, time.UTC)
	limiter := New(8 * time.Second)

	tests := []struct {
		name string
		user string
		at   time.Time
		want bool
	}{
		{name: "allows the first request", user: "alice", at: now, want: true},
		{name: "blocks a request within the cooldown", user: "alice", at: now.Add(7 * time.Second), want: false},
		{name: "allows a request when the cooldown expires", user: "alice", at: now.Add(8 * time.Second), want: true},
		{name: "tracks users independently", user: "bob", at: now.Add(7 * time.Second), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := limiter.Allow(tt.user, tt.at); got != tt.want {
				t.Fatalf("Allow(%q, %s) = %t, want %t", tt.user, tt.at, got, tt.want)
			}
		})
	}
}
