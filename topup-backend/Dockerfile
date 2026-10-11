# syntax=docker/dockerfile:1
# Satu image untuk api, worker, dan migrate (ARCHITECTURE §8). Command dipilih saat run:
#   /app/api | /app/worker | /app/migrate up
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/api ./cmd/worker ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/api /out/worker /out/migrate /app/
COPY migrations /app/migrations
EXPOSE 8080
ENTRYPOINT ["/app/api"]
