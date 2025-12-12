# Multi-stage build for minimal, secure Go server image
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimization flags
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o mimir-aip-server .

# Final stage - minimal runtime image
FROM gcr.io/distroless/static-debian12

# Install CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Create non-root user
USER 65532:65532

# Set working directory
WORKDIR /app

# Copy binary and configuration
COPY --from=builder /app/mimir-aip-server /app/mimir-aip-server
COPY --from=builder /app/config.yaml /app/config.yaml

# Create directories for data and logs (using /bin/sh equivalent)
# Note: Directories will be created automatically by COPY commands if needed

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/app/mimir-aip-server", "--version"]

# Set environment variables
ENV MIMIR_PORT=8080
ENV MIMIR_LOG_LEVEL=INFO
ENV MIMIR_DATA_DIR=/app/data
ENV MIMIR_LOG_DIR=/app/logs
ENV MIMIR_PLUGIN_DIR=/app/plugins

# Entry point
ENTRYPOINT ["/app/mimir-aip-server"]

# Default command - can be overridden
CMD ["--server"]