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

func TestLoadValidatesAuthSettings(t *testing.T) {
	base := map[string]string{
		"DB_USER": "u", "DB_NAME": "n", "REDIS_URL": "redis://x",
		"ENCRYPTION_KEYS": "k1:x", "ENCRYPTION_ACTIVE_KEY_ID": "k1",
		"JWT_SECRET": "0123456789abcdef0123456789abcdef",
	}
	cases := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{"local tanpa captcha", map[string]string{"APP_ENV": "local"}, false},
		{"secret JWT terlalu pendek", map[string]string{"APP_ENV": "local", "JWT_SECRET": "pendek"}, true},
		{"production tanpa captcha", map[string]string{"APP_ENV": "production"}, true},
		{"production dengan captcha", map[string]string{"APP_ENV": "production", "CAPTCHA_SECRET": "s"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for k, v := range base {
				t.Setenv(k, v)
			}
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			_, err := Load()
			if (err != nil) != c.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, c.wantErr)
			}
		})
	}
}
