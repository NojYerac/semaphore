# Build stage
FROM golang:1.25.1-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w \
      -X github.com/nojyerac/go-lib/version.semVer=$(cat VERSION 2>/dev/null || echo 0.0.0) \
      -X github.com/nojyerac/go-lib/version.gitSHA=$(git rev-list -1 HEAD 2>/dev/null || echo unknown)" \
    -o /build/semaphore \
    ./semaphore/main.go

# Final stage
FROM scratch

# Copy CA certificates for HTTPS connections
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /build/semaphore /semaphore

# Expose HTTP and gRPC ports
EXPOSE 8080 9090

# Run the binary
ENTRYPOINT ["/semaphore"]
