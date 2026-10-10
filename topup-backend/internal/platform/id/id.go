// Package id membuat primary key UUID v7 (terurut waktu) di sisi aplikasi.
package id

import "github.com/google/uuid"

// New mengembalikan UUID v7 baru.
func New() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// Parse memvalidasi string UUID.
func Parse(s string) (uuid.UUID, error) { return uuid.Parse(s) }
