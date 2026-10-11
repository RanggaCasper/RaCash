.PHONY: run-api run-worker migrate-up migrate-down migrate-version migrate-create keygen keygen-rotate test test-integration arch vet build tidy

# Variabel env dibaca dari .env bila ada (lihat .env.example).
ifneq (,$(wildcard .env))
include .env
export
endif

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

migrate-up:
	go run ./cmd/migrate up

# make migrate-down N=1
migrate-down:
	go run ./cmd/migrate down $(N)

migrate-version:
	go run ./cmd/migrate version

# make migrate-create NAME=user_init
migrate-create:
	go run ./cmd/migrate create $(NAME)

# Kunci enkripsi baru untuk instalasi awal. Salin dua baris output ke .env.
# make keygen            → id k1
# make keygen ID=k2      → id lain
ID ?= k1
keygen:
	@go run ./cmd/keygen -id $(ID)

# Rotasi: tambah kunci baru ke ENCRYPTION_KEYS yang ada di .env dan jadikan aktif.
# make keygen-rotate ID=k2
keygen-rotate:
	@go run ./cmd/keygen -id $(ID) -existing "$(ENCRYPTION_KEYS)"

test:
	go test ./...

# Butuh PostgreSQL terpisah yang sudah dimigrasi: TEST_DB_HOST, TEST_DB_USER, TEST_DB_NAME, ...
test-integration:
	go test -count=1 ./test/integration/...

arch:
	go test -count=1 ./test/arch/...

vet:
	go vet ./...

build:
	go build -o bin/ ./cmd/...

tidy:
	go mod tidy
