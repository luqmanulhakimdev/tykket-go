# =============================================================================
# Build Stage
# =============================================================================
FROM golang:alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Leverage Docker cache for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Compile lightweight static binaries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/migrate ./cmd/migrate && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/seed ./cmd/seed

# =============================================================================
# Final Minimal Runtime Stage
# =============================================================================
FROM alpine:3.20

WORKDIR /app

# Install runtime dependencies & create non-root user
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy binaries from builder
COPY --from=builder /bin/api /app/api
COPY --from=builder /bin/migrate /app/migrate
COPY --from=builder /bin/seed /app/seed

# Copy migrations and openapi documentation required at runtime
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/docs /app/docs

# Set ownership to non-root user
RUN mkdir -p /app/uploads && chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

CMD ["/app/api"]

