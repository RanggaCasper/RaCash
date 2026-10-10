package database

import "testing"

func TestQuoteIdent(t *testing.T) {
	cases := map[string]string{
		"racash":      `"racash"`,
		"Ra-Cash dev": `"Ra-Cash dev"`,
		`x"; DROP --`: `"x""; DROP --"`,
	}
	for in, want := range cases {
		if got := quoteIdent(in); got != want {
			t.Errorf("quoteIdent(%q) = %s; want %s", in, got, want)
		}
	}
}
