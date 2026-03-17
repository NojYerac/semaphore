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
    -o ./build/semaphore \
    -ldflags="-s -w \
        -X github.com/nojyerac/go-lib/version.semVer=$(cat VERSION 2>/dev/null || echo 0.0.0) \
        -X github.com/nojyerac/go-lib/version.gitSHA=$(git rev-parse HEAD 2>/dev/null || echo unknown) \
        -X github.com/nojyerac/go-lib/version.serviceName=semaphore" \
    ./semaphore

RUN adduser -u 1000 -D semaphore && \
    chown semaphore:semaphore ./build/semaphore

# Final stage
FROM scratch

COPY --from=builder /etc/passwd /etc/passwd
# Copy the binary
COPY --from=builder /build/build/semaphore /semaphore
USER semaphore

# Run the binary
ENTRYPOINT ["/semaphore"]
