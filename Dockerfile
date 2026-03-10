# Build stage
FROM golang:1.25.1-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum ./

# Copy go-lib dependency (required by replace directive)
COPY ../go-lib /go-lib

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o semaphore \
    ./semaphore/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 semaphore && \
    adduser -D -u 1000 -G semaphore semaphore

# Copy binary from builder
COPY --from=builder /build/semaphore /app/semaphore

# Set ownership
RUN chown -R semaphore:semaphore /app

USER semaphore

# Expose ports (HTTP and gRPC)
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/semaphore"]
