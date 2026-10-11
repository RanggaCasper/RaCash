package money

import "testing"

func TestParsePercent(t *testing.T) {
	cases := map[string]Percent{"2.5": 2500, "0.7": 700, "10": 10000, "0.125": 125, "-1.5": -1500}
	for in, want := range cases {
		got, err := ParsePercent(in)
		if err != nil || got != want {
			t.Errorf("ParsePercent(%q) = %d, %v; want %d", in, got, err, want)
		}
		if got.String() != in {
			t.Errorf("Percent(%d).String() = %q; want %q", got, got.String(), in)
		}
	}
	for _, bad := range []string{"", "abc", "1.2345", "1.x"} {
		if _, err := ParsePercent(bad); err == nil {
			t.Errorf("ParsePercent(%q) expected error", bad)
		}
	}
}

func TestPercentOf(t *testing.T) {
	cases := []struct {
		amount int64
		p      Percent
		want   int64
	}{
		{21500, 700, 151},    // 150.5 → 151 (half-up)
		{21500, 10000, 2150}, // 10%
		{19487, 0, 0},
		{-21500, 700, -151},
	}
	for _, c := range cases {
		if got := PercentOf(c.amount, c.p); got != c.want {
			t.Errorf("PercentOf(%d, %s) = %d; want %d", c.amount, c.p, got, c.want)
		}
	}
}

func TestFormatIDR(t *testing.T) {
	cases := map[int64]string{0: "Rp0", 999: "Rp999", 15750: "Rp15.750", 1000000: "Rp1.000.000", -50000: "-Rp50.000"}
	for in, want := range cases {
		if got := FormatIDR(in); got != want {
			t.Errorf("FormatIDR(%d) = %q; want %q", in, got, want)
		}
	}
}

func TestPercentSQL(t *testing.T) {
	cases := []struct {
		src  any
		want Percent
	}{{"2.500", 2500}, {[]byte("100.00"), 100000}, {int64(3), 3000}, {"-0.125", -125}}
	for _, c := range cases {
		var p Percent
		if err := p.Scan(c.src); err != nil || p != c.want {
			t.Errorf("Scan(%v) = %d, %v; want %d", c.src, p, err, c.want)
		}
	}
	var p Percent
	if err := p.Scan(1.5); err == nil {
		t.Error("Scan(float64) harus ditolak")
	}
	if v, _ := Percent(2500).Value(); v != "2.5" {
		t.Errorf("Value() = %v; want 2.5", v)
	}
}
