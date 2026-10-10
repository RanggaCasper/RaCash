// Package clock menyediakan sumber waktu yang bisa diganti saat test.
// Kode aplikasi mengambil waktu lewat Clock, bukan time.Now() langsung (RULES §2).
package clock

import (
	"sync"
	"time"
	_ "time/tzdata" // Asia/Jakarta tersedia walau OS tanpa tzdata
)

// WIB adalah zona tampilan (Asia/Jakarta). Penyimpanan selalu UTC.
var WIB = mustLoad("Asia/Jakarta")

type Clock interface {
	Now() time.Time
}

// Real mengembalikan waktu sistem dalam UTC.
type Real struct{}

func (Real) Now() time.Time { return time.Now().UTC() }

// Fixed adalah jam yang bisa diatur, untuk test.
type Fixed struct {
	mu sync.Mutex
	t  time.Time
}

func NewFixed(t time.Time) *Fixed { return &Fixed{t: t.UTC()} }

func (f *Fixed) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

func (f *Fixed) Set(t time.Time) {
	f.mu.Lock()
	f.t = t.UTC()
	f.mu.Unlock()
}

func (f *Fixed) Advance(d time.Duration) {
	f.mu.Lock()
	f.t = f.t.Add(d)
	f.mu.Unlock()
}

// ToWIB mengonversi waktu ke zona WIB untuk tampilan.
func ToWIB(t time.Time) time.Time { return t.In(WIB) }

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}
