package database

import (
	"slices"
	"testing"
)

func TestStringArrayRoundTrip(t *testing.T) {
	for _, in := range []StringArray{{}, {"a"}, {"10.0.0.0/8", "192.168.1.1/32"}, {`koma, "kutip"`, `back\slash`, "NULL"}} {
		v, err := in.Value()
		if err != nil {
			t.Fatal(err)
		}
		var out StringArray
		if err := out.Scan(v); err != nil {
			t.Fatalf("Scan(%v): %v", v, err)
		}
		if !slices.Equal(in, out) || out == nil {
			t.Errorf("round trip %q → %v → %q", in, v, out)
		}
	}
	var nilArr StringArray
	if v, _ := nilArr.Value(); v != nil {
		t.Errorf("nil harus jadi NULL, got %v", v)
	}
	var out StringArray
	if err := out.Scan(nil); err != nil || out != nil {
		t.Errorf("Scan(nil) = %v, %v", out, err)
	}
}
