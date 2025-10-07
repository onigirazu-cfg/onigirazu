# Build stage
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

# Build arguments for cross-compilation
ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ARG COMMIT
ARG DATE

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
  go build -trimpath \
  -ldflags="-s -w -X github.com/onigirazu-cfg/onigirazu/internal/version.Version=${VERSION} -X github.com/onigirazu-cfg/onigirazu/internal/version.Commit=${COMMIT} -X github.com/onigirazu-cfg/onigirazu/internal/version.Date=${DATE}" \
  -o onigirazu ./cmd/onigirazu

# Create appuser
RUN adduser -D -g '' appuser

# Final stage
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /build/onigirazu /onigirazu

# Use an unprivileged user
USER appuser

# Expose port (if needed)
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/onigirazu"]
