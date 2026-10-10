// Command keygen membuat kunci AES-256 untuk ENCRYPTION_KEYS.
//
//	go run ./cmd/keygen                 # kunci baru dengan id "k1"
//	go run ./cmd/keygen -id k2          # kunci untuk rotasi
//	go run ./cmd/keygen -id k2 -existing "k1:..."   # gabungkan dengan kunci lama
//
// Saat rotasi, kunci lama WAJIB tetap dicantumkan agar data lama bisa didekripsi;
// hanya ENCRYPTION_ACTIVE_KEY_ID yang dipindah ke kunci baru.
package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var idPattern = regexp.MustCompile(`^[a-z0-9_]{1,32}$`)

func main() {
	id := flag.String("id", "k1", "key_id untuk kunci baru (huruf kecil/angka, maks. 32)")
	existing := flag.String("existing", "", "nilai ENCRYPTION_KEYS saat ini (opsional, untuk rotasi)")
	flag.Parse()

	if !idPattern.MatchString(*id) {
		fail("id tidak valid: %q", *id)
	}
	for _, part := range strings.Split(*existing, ",") {
		if kid, _, _ := strings.Cut(strings.TrimSpace(part), ":"); kid == *id {
			fail("key_id %q sudah dipakai di -existing", *id)
		}
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		fail("gagal membuat kunci: %v", err)
	}
	entry := *id + ":" + base64.StdEncoding.EncodeToString(key)

	keys := entry
	if e := strings.TrimSpace(*existing); e != "" {
		keys = e + "," + entry
	}
	fmt.Printf("ENCRYPTION_KEYS=%s\n", keys)
	fmt.Printf("ENCRYPTION_ACTIVE_KEY_ID=%s\n", *id)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "keygen: "+format+"\n", args...)
	os.Exit(1)
}
