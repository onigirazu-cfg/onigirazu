# Prepare base image with certificates and user
FROM alpine:3.22 AS builder

# Install ca-certificates and tzdata
RUN apk add --no-cache ca-certificates tzdata

# Create appuser
RUN adduser -D -g '' appuser

# Final stage
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd

# Copy the pre-built binary from GoReleaser
COPY onigirazu /onigirazu

# Use an unprivileged user
USER appuser

# Expose port (if needed)
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/onigirazu"]
