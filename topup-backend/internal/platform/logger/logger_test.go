package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestSensitiveFieldsMasked(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "info", "local", "test")
	l.Info("login", "email", "a@b.c", "password", "hunter2", "refresh_token", "rt_abc", "serial_number", "SN123")
	out := buf.String()
	for _, leak := range []string{"hunter2", "rt_abc", "SN123"} {
		if strings.Contains(out, leak) {
			t.Fatalf("nilai sensitif %q bocor ke log: %s", leak, out)
		}
	}
	if !strings.Contains(out, "a@b.c") {
		t.Fatalf("field biasa ikut dimasking: %s", out)
	}
}
