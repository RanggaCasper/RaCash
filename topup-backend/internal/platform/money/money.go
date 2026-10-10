// Package money menangani nilai rupiah sebagai integer (RULES §2: dilarang float untuk uang).
package money

import (
	"errors"
	"strconv"
	"strings"
)

// Percent adalah persentase dengan presisi 3 desimal, disimpan sebagai
// seperseribu persen (sesuai kolom numeric(6,3)). Contoh: 2,5% = 2500.
type Percent int64

const percentScale = 1000 // 1% = 1000

var ErrInvalidPercent = errors.New("money: persentase tidak valid")

// ParsePercent mengurai string desimal ("2.5", "0.7", "10") tanpa float.
func ParsePercent(s string) (Percent, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrInvalidPercent
	}
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	whole, frac, _ := strings.Cut(s, ".")
	if len(frac) > 3 {
		return 0, ErrInvalidPercent
	}
	frac += strings.Repeat("0", 3-len(frac))
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, ErrInvalidPercent
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, ErrInvalidPercent
	}
	p := Percent(w*percentScale + f)
	if neg {
		p = -p
	}
	return p, nil
}

// String mengembalikan bentuk desimal, mis. "2.5".
func (p Percent) String() string {
	sign := ""
	if p < 0 {
		sign, p = "-", -p
	}
	whole, frac := int64(p)/percentScale, int64(p)%percentScale
	if frac == 0 {
		return sign + strconv.FormatInt(whole, 10)
	}
	fs := strings.TrimRight(strconv.FormatInt(frac+percentScale, 10)[1:], "0")
	return sign + strconv.FormatInt(whole, 10) + "." + fs
}

// PercentOf menghitung p% dari amount, dibulatkan half-up ke rupiah terdekat.
func PercentOf(amount int64, p Percent) int64 {
	const denom = 100 * percentScale
	n := amount * int64(p)
	if n >= 0 {
		return (n + denom/2) / denom
	}
	return -((-n + denom/2) / denom)
}

// FormatIDR memformat rupiah untuk tampilan, mis. 15750 → "Rp15.750".
func FormatIDR(amount int64) string {
	sign := ""
	if amount < 0 {
		sign, amount = "-", -amount
	}
	s := strconv.FormatInt(amount, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return sign + "Rp" + b.String()
}
