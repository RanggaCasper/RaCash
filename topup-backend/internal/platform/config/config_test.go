package config

import "testing"

func TestDatabaseURLEscapesCredentials(t *testing.T) {
	d := Database{Host: "db.local", Port: 5433, User: "racash", Password: "p@ss:w/rd#1", Name: "racash", SSLMode: "require"}
	want := "postgres://racash:p%40ss%3Aw%2Frd%231@db.local:5433/racash?sslmode=require"
	if got := d.URL(); got != want {
		t.Fatalf("URL()\n got %s\nwant %s", got, want)
	}
}

func TestDatabaseURLWithoutPassword(t *testing.T) {
	d := Database{Host: "localhost", Port: 5432, User: "racash", Name: "racash", SSLMode: "disable"}
	if got, want := d.URL(), "postgres://racash@localhost:5432/racash?sslmode=disable"; got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestAutoCreateEnabled(t *testing.T) {
	yes, no := true, false
	cases := []struct {
		flag   *bool
		appEnv string
		want   bool
	}{
		{nil, EnvLocal, true},
		{nil, EnvStaging, false},
		{nil, EnvProduction, false},
		{&yes, EnvProduction, true},
		{&no, EnvLocal, false},
	}
	for _, c := range cases {
		if got := (Database{AutoCreate: c.flag}).AutoCreateEnabled(c.appEnv); got != c.want {
			t.Errorf("flag=%v env=%s: got %v want %v", c.flag, c.appEnv, got, c.want)
		}
	}
}

func TestLoadDatabaseFromEnv(t *testing.T) {
	t.Setenv("DB_USER", "u")
	t.Setenv("DB_NAME", "n")
	t.Setenv("DB_PORT", "6543")
	d, err := LoadDatabase()
	if err != nil {
		t.Fatal(err)
	}
	if d.Host != "localhost" || d.Port != 6543 || d.SSLMode != "disable" {
		t.Fatalf("default/override salah: %+v", d)
	}
}
