# Multi-stage build for minimal production image.
#
# No `# syntax=docker/dockerfile:1.4` pin on purpose (see frontend/Dockerfile
# for rationale). Base image is vendored into our own GHCR namespace (mirrored
# from Docker Hub via `make mirror-bases`) and pinned by manifest-list digest
# so builds never hit Docker Hub (avoids "TLS handshake timeout" deploy
# failures). Manifest-list digest → BuildKit picks linux/amd64 automatically.
FROM ghcr.io/4rankng/golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files (for better caching)
COPY go.mod go.sum ./

# Download dependencies (cached if go.mod/go.sum unchanged)
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY . .

# Build the application
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /tmp/api-server \
    ./cmd/api-server

# Final stage - minimal runtime image
FROM scratch

# Import certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Pin the container timezone to the application's business timezone (Asia/Ho_Chi_Minh).
# Go's time.Local defaults to UTC in a scratch image (no /etc/localtime). Without this,
# time.Local diverges from clock.Now() (pinned to Asia/Ho_Chi_Minh) and from the MySQL
# driver's loc=Local, shifting DB DATE boundary comparisons by 7h in prod — e.g. BCC import
# rejected with "no active payrate found" on the payrate's effective day.
ENV TZ=Asia/Ho_Chi_Minh

# Create non-root user
COPY --from=builder /etc/passwd /etc/passwd
USER nobody

WORKDIR /app

# Copy the binary
COPY --from=builder /tmp/api-server /api-server

# Copy config files
COPY --from=builder /build/configs /app/configs

# Copy fonts directory
COPY --from=builder /build/fonts /app/fonts

# Copy templates directory
COPY --from=builder /build/templates /app/templates

# Create logs directory (will be created at runtime)
# Note: logs directory will be created by the application when needed

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/api-server", "healthcheck"]

# Run the application
ENTRYPOINT ["/api-server"]
