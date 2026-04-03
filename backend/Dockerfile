# ─── Stage 1: Build ──────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /app/prds-server ./cmd/server

# ─── Stage 2: Runtime ────────────────────────────────────────────────────────
FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata curl

# Non-root user
RUN addgroup -S prds && adduser -S prds -G prds

WORKDIR /app

COPY --from=builder /app/prds-server .
COPY --from=builder /app/migrations ./migrations

RUN chown -R prds:prds /app
USER prds

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

CMD ["./prds-server"]
