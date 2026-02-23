# ─── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies (needed for CGO — postgres driver uses pure Go, no CGO needed)
RUN apk add --no-cache git

# Copy dependency files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/bin/api \
    ./cmd/api/...

# ─── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM alpine:3.21 AS runtime

WORKDIR /app

# Install CA certificates for HTTPS calls (OTEL exporters)
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/bin/api .

# Expose default port
EXPOSE 3000

# Run as non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

ENTRYPOINT ["./api"]
