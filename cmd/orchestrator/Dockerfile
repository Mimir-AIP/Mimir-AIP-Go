# Single stage - use same Go environment for building and running
# This ensures orchestrator binary and plugins are compiled with identical toolchain
FROM golang:1.25

# Install runtime dependencies (git for cloning plugins)
RUN apt-get update && apt-get install -y ca-certificates git && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (needed for plugin compilation)
COPY . .

# Build the orchestrator binary with CGO enabled (required for Go plugins)
# Use -trimpath to ensure reproducible builds for plugin compatibility
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -o /app/orchestrator ./cmd/orchestrator

# Expose port
EXPOSE 8080

# Run the orchestrator
CMD ["./orchestrator"]
