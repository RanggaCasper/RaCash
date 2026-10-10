// Package uow (unit of work) menjalankan beberapa repository dalam satu
// transaksi GORM. Transaksi dibawa lewat context; repository mengambil
// *gorm.DB dari DBProvider dan tidak pernah membuka transaksi sendiri.
package uow

import (
	"context"

	"gorm.io/gorm"
)

// Tx dipakai service untuk membungkus use case dalam satu transaksi.
type Tx interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// DBProvider dipakai repository untuk mendapatkan *gorm.DB yang tepat:
// transaksi aktif bila ada, atau koneksi biasa.
type DBProvider interface {
	DB(ctx context.Context) *gorm.DB
}

type txKey struct{}

// Manager mengimplementasikan Tx dan DBProvider.
type Manager struct{ db *gorm.DB }

func New(db *gorm.DB) *Manager { return &Manager{db: db} }

// Do menjalankan fn dalam transaksi. Bila context sudah membawa transaksi,
// fn bergabung ke transaksi tersebut (tidak membuat transaksi bersarang).
func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// DB mengembalikan transaksi aktif atau koneksi biasa, selalu dengan context.
func (m *Manager) DB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return m.db.WithContext(ctx)
}

// InTx melaporkan apakah context sedang membawa transaksi.
func InTx(ctx context.Context) bool {
	_, ok := ctx.Value(txKey{}).(*gorm.DB)
	return ok
}
