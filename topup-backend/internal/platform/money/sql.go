package money

import (
	"database/sql/driver"
	"fmt"
)

// Value menyimpan Percent ke kolom numeric sebagai string desimal (tanpa float).
func (p Percent) Value() (driver.Value, error) { return p.String(), nil }

// Scan membaca kolom numeric (mis. "2.500") ke Percent.
func (p *Percent) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	case int64:
		*p = Percent(v * percentScale)
		return nil
	default:
		return fmt.Errorf("money: tidak bisa scan %T ke Percent", src)
	}
	v, err := ParsePercent(s)
	if err != nil {
		return fmt.Errorf("money: scan %q: %w", s, err)
	}
	*p = v
	return nil
}
