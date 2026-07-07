# --- Build stage ---
FROM golang:1.24-alpine AS builder

# sqlite driver (mattn/go-sqlite3) needs cgo + gcc
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/...

# --- Run stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/internal/db/migrations ./internal/db/migrations

# SQLite data lives here — mount a volume to this path so data survives rebuilds
RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./server"]