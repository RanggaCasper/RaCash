package database

import (
	"database/sql/driver"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

// StringArray memetakan kolom array PostgreSQL (varchar[], uuid[], cidr[]) ke
// []string. Modul dilarang import pgx/lib/pq langsung (RULES §3.3), jadi
// konversinya disediakan di sini. nil disimpan sebagai NULL, slice kosong
// sebagai '{}'.
type StringArray []string

var arrayMap = pgtype.NewMap()

// Value meng-encode ke literal array teks, mis. {"a","b"}; PostgreSQL
// meng-cast-nya ke tipe kolom.
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	buf, err := arrayMap.Encode(pgtype.TextArrayOID, pgtype.TextFormatCode, []string(a), nil)
	if err != nil {
		return nil, fmt.Errorf("database: encode array: %w", err)
	}
	return string(buf), nil
}

// Scan membaca literal array teks dari driver.
func (a *StringArray) Scan(src any) error {
	if src == nil {
		*a = nil
		return nil
	}
	var buf []byte
	switch v := src.(type) {
	case string:
		buf = []byte(v)
	case []byte:
		buf = v
	default:
		return fmt.Errorf("database: tidak bisa scan %T ke StringArray", src)
	}
	var out []string
	if err := arrayMap.Scan(pgtype.TextArrayOID, pgtype.TextFormatCode, buf, &out); err != nil {
		return fmt.Errorf("database: scan array: %w", err)
	}
	if out == nil {
		out = []string{}
	}
	*a = out
	return nil
}
