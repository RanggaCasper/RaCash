package domain

import (
	"testing"
	"time"
)

func TestValidPassword(t *testing.T) {
	cases := []struct {
		pw   string
		want bool
	}{
		{"S3cure!pass", true},
		{"abcdefg1", true},
		{"abcdefgh", false}, // tanpa angka
		{"12345678", false}, // tanpa huruf
		{"abc123", false},   // < 8
		{"", false},
		{"päßwörd1", true},                        // huruf non-ASCII
		{string(make([]byte, 200)) + "a1", false}, // > 128
	}
	for _, c := range cases {
		if got := ValidPassword(c.pw); got != c.want {
			t.Errorf("ValidPassword(%q) = %v, want %v", c.pw, got, c.want)
		}
	}
}

func TestSessionActive(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	revoked := now.Add(-time.Minute)
	cases := []struct {
		name string
		s    Session
		want bool
	}{
		{"aktif", Session{ExpiresAt: now.Add(time.Hour)}, true},
		{"kedaluwarsa", Session{ExpiresAt: now}, false},
		{"dicabut", Session{ExpiresAt: now.Add(time.Hour), RevokedAt: &revoked}, false},
	}
	for _, c := range cases {
		if got := c.s.Active(now); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
