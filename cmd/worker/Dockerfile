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

# Runtime stage — includes the Go toolchain because pipeline and ML provider plugins
# are compiled with `go build -buildmode=plugin` inside worker pods.
FROM golang:1.25

WORKDIR /app

# Copy built binary from builder
COPY --from=builder /app/worker .

# Copy source code (needed for potential plugin compilation at runtime)
COPY --from=builder /app/go.mod /app/go.sum ./
COPY --from=builder /app/pkg ./pkg
COPY --from=builder /app/cmd ./cmd

# Create directories for model artifacts and runtime plugin caches
RUN mkdir -p /tmp/models /tmp/plugins /tmp/plugins/ml

# Run the worker
CMD ["./worker"]
