# Build stage
FROM golang:1.25 AS builder

# Install dependencies for CGO (required for Go plugins and some ML libraries)
RUN apt-get update && apt-get install -y ca-certificates git && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the worker binary with CGO enabled (required for Go plugins and gonum)
# Use -trimpath to ensure reproducible builds for plugin compatibility
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/worker ./cmd/worker

# Runtime stage - smaller base image with just what we need
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates git && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy built binary from builder
COPY --from=builder /app/worker .

# Copy source code (needed for potential plugin compilation at runtime)
COPY --from=builder /app/go.mod /app/go.sum ./
COPY --from=builder /app/pkg ./pkg
COPY --from=builder /app/cmd ./cmd

# Create directory for model artifacts
RUN mkdir -p /tmp/models /tmp/plugins

# Run the worker
CMD ["./worker"]
