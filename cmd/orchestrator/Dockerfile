# Build stage
FROM golang:1.25 AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the orchestrator binary with CGO enabled (required for Go plugins)
# Use -trimpath to ensure reproducible builds for plugin compatibility
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/orchestrator ./cmd/orchestrator

# Runtime stage - smaller base image with just what we need
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates git && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy built binary from builder
COPY --from=builder /app/orchestrator .

# Copy source code (needed for potential plugin compilation at runtime)
COPY --from=builder /app/go.mod /app/go.sum ./
COPY --from=builder /app/pkg ./pkg
COPY --from=builder /app/cmd ./cmd

# Expose port
EXPOSE 8080

# Run the orchestrator
CMD ["./orchestrator"]
