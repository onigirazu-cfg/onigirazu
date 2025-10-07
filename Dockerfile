# Build stage
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

# Build arguments for cross-compilation
ARG TARGETOS
ARG TARGETARCH

# Install git and ca-certificates (needed for go modules)
RUN apk add --no-cache git ca-certificates tzdata

# Create appuser
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build the binary with dynamic architecture
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o onigirazu ./cmd/onigirazu

# Final stage
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd

# Copy binary
COPY --from=builder /build/onigirazu /onigirazu

# Use an unprivileged user
USER appuser

# Expose port (if needed)
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/onigirazu"]
